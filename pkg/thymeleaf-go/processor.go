package thymeleaf

import (
	"fmt"
	stdhtml "html"
	"reflect"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// Processor handles th:* attribute processing on HTML nodes.
type Processor struct {
	engine *Engine
}

func NewProcessor(engine *Engine) *Processor {
	return &Processor{engine: engine}
}

// processNode processes a single node and returns the resulting nodes.
func (p *Processor) processNode(n *html.Node, ctx *Context) ([]*html.Node, error) {
	if n == nil {
		return nil, nil
	}

	switch n.Type {
	case html.DocumentNode:
		var result []*html.Node
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			processed, err := p.processNode(c, ctx)
			if err != nil {
				return nil, err
			}
			result = append(result, processed...)
		}
		return result, nil

	case html.ElementNode:
		return p.processElement(n, ctx)

	case html.TextNode:
		// Process inline expressions in text
		text := p.processInlineText(n.Data, ctx)
		if text != n.Data {
			// Create a new text node with processed content
			return []*html.Node{{Type: html.TextNode, Data: text}}, nil
		}
		return []*html.Node{n}, nil

	case html.CommentNode:
		// Check if it's a JS inline comment with expression
		if expr := extractCommentExpr(n.Data); expr != "" {
			val, err := evalStandard(expr, ctx)
			if err == nil {
				return []*html.Node{{Type: html.TextNode, Data: toStr(val)}}, nil
			}
		}
		// Skip comments
		return nil, nil

	case html.DoctypeNode:
		return []*html.Node{n}, nil

	case html.RawNode:
		text := p.processInlineText(n.Data, ctx)
		return []*html.Node{{Type: html.RawNode, Data: text}}, nil

	default:
		return []*html.Node{n}, nil
	}
}

// processElement handles th:* attributes on an element node.
func (p *Processor) processElement(n *html.Node, ctx *Context) (result []*html.Node, err error) {
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
			panic(r)
		}
	}()
	// Make a working copy so we don't mutate the cached template
	node := cloneNode(n)

	// 1. th:each - iteration (processed before th:if so that th:if on the
	// same element is evaluated per-iteration, matching Thymeleaf precedence).
	if val, ok := getThAttr(node, "each"); ok {
		removeAttr(node, "th:each")
		return p.processEach(node, val, ctx)
	}

	// 2. th:if / th:unless - conditional rendering
	if val, ok := getThAttr(node, "if"); ok {
		result, err := evalStandard(val, ctx)
		if err != nil {
			return nil, err
		}
		if !toBool(result) {
			return nil, nil
		}
		removeAttr(node, "th:if")
	}

	if val, ok := getThAttr(node, "unless"); ok {
		result, err := evalStandard(val, ctx)
		if err != nil {
			return nil, err
		}
		if toBool(result) {
			return nil, nil
		}
		removeAttr(node, "th:unless")
	}

	// Handle sec:authorize (Spring Security dialect)
	// isAuthenticated() → false (no authenticated user in theme context)
	// isAnonymous() → true (user is anonymous by default)
	if val, ok := getAttr(node, "sec:authorize"); ok {
		val = strings.TrimSpace(val)
		if strings.Contains(val, "isAuthenticated()") {
			removeAttr(node, "sec:authorize")
			return nil, nil // Remove element (user is not authenticated)
		} else if strings.Contains(val, "isAnonymous()") {
			removeAttr(node, "sec:authorize")
			// Keep element (user is anonymous)
		} else {
			removeAttr(node, "sec:authorize")
		}
	}

	// 3. th:switch / th:case - conditional rendering of first matching child
	if val, ok := getThAttr(node, "switch"); ok {
		removeAttr(node, "th:switch")
		switchResult, err := evalStandard(val, ctx)
		if err != nil {
			return nil, err
		}
		switchValue := toStr(switchResult)
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				if caseVal, ok := getThAttr(c, "case"); ok {
					// Thymeleaf th:case values are literals unless wrapped in ${...} / #{...}
				var caseResult any
				if strings.HasPrefix(caseVal, "${") || strings.HasPrefix(caseVal, "#{") {
					caseResult, err = evalStandard(caseVal, ctx)
					if err != nil {
						return nil, err
					}
				} else {
					// Strip surrounding quotes so th:case="'card'" matches the string "card".
					caseResult = strings.Trim(caseVal, "'\"")
				}
				if toStr(caseResult) == switchValue {
					removeAttr(c, "th:case")
					return p.processNode(c, ctx)
				}
				}
			}
		}
		return nil, nil
	}

	// 4. th:with - local variable assignment
	if val, ok := getThAttr(node, "with"); ok {
		removeAttr(node, "th:with")
		childCtx := ctx.Child()
		if err := p.processWith(val, childCtx); err != nil {
			return nil, err
		}
		ctx = childCtx
	}

	// 5. th:replace - fragment replacement
	if val, ok := getThAttr(node, "replace"); ok {
		nodes, err := p.processReplace(node, val, ctx)
		if err != nil {
			return nil, err
		}
		// If the th:replace node is inside <head> (this can happen because
		// <th:block> is converted to <template> which HTML5 allows in <head>),
		// move the replacement result out of <head> and place it right after
		// </head>. This is essential for layout delegation patterns where the
		// replaced content includes a <body> tag.
		//
		// Important: check the original node n, not the clone node, because the
		// clone's parent chain is nil until it is inserted back into the tree.
		wasInHead := inHead(n)
		if wasInHead {
			moved := false
			for parent := n.Parent; parent != nil; parent = parent.Parent {
				if parent.Type == html.ElementNode && parent.Data == "html" {
					head := findHead(parent)
					if head != nil && head.Parent != nil {
						// Insert all nodes before the original head.NextSibling so they
						// end up right after </head> and in the correct order.
						insertBefore := head.NextSibling
						for _, newNode := range nodes {
							parent.InsertBefore(newNode, insertBefore)
						}
						moved = true
					}
					break
				}
			}
			if moved {
				return nil, nil
			}
			// Could not find a rooted <html>/<head> to insert after; fall
			// through so the replacement nodes are returned normally. A
			// post-processing step in Engine.Render will move body-level
			// content out of <head> if it ended up there.
		}
		return nodes, nil
	}

	// 5b. th:insert - insert fragment contents into the current element.
	if val, ok := getThAttr(node, "insert"); ok {
		removeAttr(node, "th:insert")
		nodes, err := p.processReplace(node, val, ctx)
		if err != nil {
			return nil, err
		}
		// Append the rendered fragment nodes as children of the current element,
		// then return the element itself (unlike th:replace which discards it).
		for _, child := range nodes {
			node.AppendChild(child)
		}
		return []*html.Node{node}, nil
	}

	// 5. th:fragment - if this is a fragment definition and we're rendering
	// the full document (not targeting this fragment), skip the wrapper but
	// keep children. Actually, in Thymeleaf, th:fragment just marks the element;
	// when rendering normally, the element is kept.
	if _, ok := getThAttr(node, "fragment"); ok {
		removeAttr(node, "th:fragment")
	}

	// 6. th:text / th:utext - text content
	if val, ok := getThAttr(node, "text"); ok {
		removeAttr(node, "th:text")
		result, err := evalStandard(val, ctx)
		if err != nil {
			return nil, err
		}
		// Clear children and set text
		for c := node.FirstChild; c != nil; {
			next := c.NextSibling
			node.RemoveChild(c)
			c = next
		}
		node.AppendChild(&html.Node{Type: html.TextNode, Data: stdhtml.EscapeString(toStr(result))})
	} else if val, ok := getThAttr(node, "utext"); ok {
		removeAttr(node, "th:utext")
		result, err := evalStandard(val, ctx)
		if err != nil {
			return nil, err
		}
		for c := node.FirstChild; c != nil; {
			next := c.NextSibling
			node.RemoveChild(c)
			c = next
		}
		node.AppendChild(&html.Node{Type: html.TextNode, Data: toStr(result)})
	}

	// 7. Attribute processors
	p.processAttributes(node, ctx)

	// 8. Process children
	var newChildren []*html.Node
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		processed, err := p.processNode(c, ctx)
		if err != nil {
			return nil, err
		}
		newChildren = append(newChildren, processed...)
	}

	// Remove all existing children and add processed ones
	for c := node.FirstChild; c != nil; {
		next := c.NextSibling
		node.RemoveChild(c)
		c = next
	}
	for _, child := range newChildren {
		node.AppendChild(child)
	}

	return []*html.Node{node}, nil
}

// processEach handles th:each="var, status : ${iterable}"
func (p *Processor) processEach(node *html.Node, expr string, ctx *Context) ([]*html.Node, error) {
	// Parse "var, status : ${iterable}" or "var : ${iterable}"
	colonIdx := strings.Index(expr, ":")
	if colonIdx == -1 {
		return nil, fmt.Errorf("invalid th:each: missing ':'")
	}

	leftPart := strings.TrimSpace(expr[:colonIdx])
	iterExpr := strings.TrimSpace(expr[colonIdx+1:])

	var varName, statusName string
	if commaIdx := strings.Index(leftPart, ","); commaIdx != -1 {
		varName = strings.TrimSpace(leftPart[:commaIdx])
		statusName = strings.TrimSpace(leftPart[commaIdx+1:])
	} else {
		varName = strings.TrimSpace(leftPart)
	}

	// Evaluate the iterable
	iterable, err := evalStandard(iterExpr, ctx)
	if err != nil {
		return nil, err
	}

	items := toSlice(iterable)
	if items == nil {
		return nil, nil
	}

	var result []*html.Node
	for i, item := range items {
		childCtx := ctx.Child()
		childCtx.Set(varName, item)
		if statusName != "" {
			childCtx.Set(statusName, &IterationStatus{
				Index: i,
				Count: i + 1,
				Size:  len(items),
				Even:  i%2 == 0,
				Odd:   i%2 != 0,
				First: i == 0,
				Last:  i == len(items)-1,
			})
		}
		// Clone the node for each iteration
		nodeCopy := cloneNode(node)
		processed, err := p.processElement(nodeCopy, childCtx)
		if err != nil {
			return nil, err
		}
		result = append(result, processed...)
	}
	return result, nil
}

// IterationStatus holds iteration metadata.
type IterationStatus struct {
	Index int  `json:"index"`
	Count int  `json:"count"`
	Size  int  `json:"size"`
	Even  bool `json:"even"`
	Odd   bool `json:"odd"`
	First bool `json:"first"`
	Last  bool `json:"last"`
}

// processWith handles th:with="var1 = ${expr}, var2 = ${expr}"
func (p *Processor) processWith(expr string, ctx *Context) error {
	// Split by commas, but respect parentheses and ${}
	assignments := splitByComma(expr)
	for _, assignment := range assignments {
		assignment = strings.TrimSpace(assignment)
		eqIdx := strings.Index(assignment, "=")
		if eqIdx == -1 {
			continue
		}
		name := strings.TrimSpace(assignment[:eqIdx])
		valExpr := strings.TrimSpace(assignment[eqIdx+1:])
		val, err := evalStandard(valExpr, ctx)
		if err != nil {
			return err
		}
		ctx.Set(name, val)
	}
	return nil
}

// processReplace handles th:replace="~{template :: fragment(params)}" or "${variable}"
func (p *Processor) processReplace(node *html.Node, expr string, ctx *Context) ([]*html.Node, error) {
	expr = strings.TrimSpace(expr)

	// Fragment expression: ~{template :: fragment(params)}
	if strings.HasPrefix(expr, "~{") {
		return p.processFragmentReplace(node, expr, ctx)
	}

	// Variable expression: ${variable} - the variable holds a fragment reference
	if strings.HasPrefix(expr, "${") {
		val, err := evalStandard(expr, ctx)
		if err != nil {
			return nil, err
		}
		if fragRef, ok := val.(*FragmentRef); ok {
			// currentNode is nil here because local fragments from the calling
			// template were already captured when the ~{...} was processed.
			return p.renderFragmentRef(fragRef, ctx, nil)
		}
		// Not a fragment ref - just render as text
		return []*html.Node{{Type: html.TextNode, Data: toStr(val)}}, nil
	}

	// Literal substitution or other
	val, err := evalStandard(expr, ctx)
	if err != nil {
		return nil, err
	}
	return []*html.Node{{Type: html.TextNode, Data: toStr(val)}}, nil
}

// FragmentRef holds a reference to a fragment with parameters.
type FragmentRef struct {
	Template  string
	Fragment  string
	Params    map[string]string // raw param expressions
	LocalNode *html.Node        // for ~{::fragment} local references
}

func (f *FragmentRef) String() string {
	return fmt.Sprintf("FragmentRef{template=%s, fragment=%s}", f.Template, f.Fragment)
}

// processFragmentReplace parses and renders a ~{template :: fragment(params)} expression.
func (p *Processor) processFragmentReplace(node *html.Node, expr string, ctx *Context) ([]*html.Node, error) {
	fragRef, err := parseFragmentExpr(expr, node, ctx)
	if err != nil {
		return nil, err
	}
	return p.renderFragmentRef(fragRef, ctx, node)
}

// parseFragmentExpr parses ~{template :: fragment(param1=val1, param2=val2)}
// If ctx is non-nil and the template part is a dynamic expression (e.g.
// 'modules/widgets/'+${widget.value}), it is evaluated against ctx to produce
// the final template name.
func parseFragmentExpr(expr string, currentNode *html.Node, ctx *Context) (*FragmentRef, error) {
	// Strip ~{ and }
	if !strings.HasPrefix(expr, "~{") || !strings.HasSuffix(expr, "}") {
		return nil, fmt.Errorf("invalid fragment expression: %s", expr)
	}
	inner := strings.TrimSpace(expr[2 : len(expr)-1])

	// Check for :: separator
	parts := strings.SplitN(inner, "::", 2)
	ref := &FragmentRef{Params: make(map[string]string)}

	if len(parts) == 1 {
		// ~{template} or ~{::fragment}
		trimmed := strings.TrimSpace(parts[0])
		if strings.HasPrefix(trimmed, "::") {
			// Local fragment reference: ~{::fragment}
			ref.Fragment = strings.TrimSpace(trimmed[2:])
			ref.LocalNode = currentNode
		} else {
			ref.Template = evalTemplateName(trimmed, ctx)
		}
		return ref, nil
	}

	// ~{template :: fragment} or ~{template :: fragment(params)}
	ref.Template = evalTemplateName(strings.TrimSpace(parts[0]), ctx)
	fragPart := strings.TrimSpace(parts[1])

	// Check for parameters
	parenIdx := strings.Index(fragPart, "(")
	if parenIdx == -1 {
		ref.Fragment = fragPart
		return ref, nil
	}

	ref.Fragment = strings.TrimSpace(fragPart[:parenIdx])

	// Extract parameters
	if !strings.HasSuffix(fragPart, ")") {
		return nil, fmt.Errorf("unclosed parenthesis in fragment expression: %s", expr)
	}
	paramsStr := fragPart[parenIdx+1 : len(fragPart)-1]
	if paramsStr != "" {
		params := splitByComma(paramsStr)
		for _, param := range params {
			param = strings.TrimSpace(param)
			eqIdx := indexParamAssign(param)
			if eqIdx == -1 {
				// Positional parameter (not supported, skip)
				continue
			}
			name := strings.TrimSpace(param[:eqIdx])
			value := strings.TrimSpace(param[eqIdx+1:])
			ref.Params[name] = value
		}
	}

	return ref, nil
}

// evalTemplateName resolves a template name that may be a dynamic expression
// (e.g. 'modules/widgets/'+${widget.value}) into a plain string. Simple
// template names (e.g. "modules/layout") are returned as-is.
func evalTemplateName(name string, ctx *Context) string {
	// If it contains a ${ or a single-quote (string literal), evaluate it as
	// a standard expression to produce the final template path.
	if ctx != nil && (strings.Contains(name, "${") || strings.Contains(name, "'")) {
		if val, err := evalStandard(name, ctx); err == nil {
			return toStr(val)
		}
	}
	return name
}

// renderFragmentRef renders a fragment reference.
// currentNode is the node where th:replace was declared (nil when called from
// the ${variable} path, since local fragments were already captured upstream).
func (p *Processor) renderFragmentRef(ref *FragmentRef, ctx *Context, currentNode *html.Node) ([]*html.Node, error) {
	// Handle local fragment reference (~{::fragment})
	if ref.LocalNode != nil || (ref.Template == "" && ref.Fragment != "") {
		// Capture local fragments from the current document so ~{::fragmentName}
		// can find them in the context.
		if currentNode != nil {
			p.captureLocalFragments(currentNode, ctx)
		}
		return p.findAndRenderLocalFragment(ref.Fragment, ctx)
	}

	// Load the template and render the fragment
	fragCtx := ctx.Child()

	// Capture local fragments from the calling template so they can be
	// referenced via ~{::fragmentName} or th:replace="${paramName}" in the
	// target template (layout delegation pattern).
	if currentNode != nil {
		p.captureLocalFragments(currentNode, fragCtx)
	}

	// Evaluate and set parameters
	for name, expr := range ref.Params {
		val, err := p.evalParam(expr, ctx)
		if err != nil {
			return nil, err
		}
		fragCtx.Set(name, val)
	}

	// Use RenderFragmentNodes to get processed nodes directly, avoiding the
	// lossy string round-trip that would discard <html>/<head> structure.
	nodes, err := p.engine.RenderFragmentNodes(ref.Template, ref.Fragment, fragCtx)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// captureLocalFragments finds all th:fragment definitions in the current
// document and stores their cloned child nodes in the context under
// __local_fragment_<name> keys. This enables ~{::fragmentName} references
// and th:replace="${paramName}" (where paramName holds a FragmentRef) to
// locate and render fragments defined in the calling template.
func (p *Processor) captureLocalFragments(node *html.Node, ctx *Context) {
	if node == nil {
		return
	}
	// Find the root of the document
	root := node
	for root.Parent != nil {
		root = root.Parent
	}
	// Walk the document and find all th:fragment definitions
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if val, ok := getThAttr(n, "fragment"); ok {
				name := strings.TrimSpace(val)
				if parenIdx := strings.Index(name, "("); parenIdx != -1 {
					name = strings.TrimSpace(name[:parenIdx])
				}
				if name != "" {
					// Clone child nodes so the original tree isn't mutated during processing
					var children []*html.Node
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						children = append(children, cloneNode(c))
					}
					ctx.Set("__local_fragment_"+name, children)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
}

// evalParam evaluates a fragment parameter value.
// This handles both ${expr} and ~{::fragment} references.
// Bare identifiers are treated as string literals, matching Thymeleaf's
// fragment parameter semantics and the theme's convention of writing
// htmlType=index without quotes. Variables must be passed explicitly as
// ${varName} to avoid shadowing by context variables with the same name.
func (p *Processor) evalParam(expr string, ctx *Context) (any, error) {
	expr = strings.TrimSpace(expr)

	// ~{::fragment} or ~{::content} - local fragment reference
	if strings.HasPrefix(expr, "~{") {
		fragRef, err := parseFragmentExpr(expr, nil, ctx)
		if err != nil {
			return nil, err
		}
		return fragRef, nil
	}

	// Explicit variable/message expressions are evaluated normally.
	if strings.HasPrefix(expr, "${") || strings.HasPrefix(expr, "#{") {
		return evalStandard(expr, ctx)
	}

	// Bare identifiers are string literals in fragment parameters. This avoids
	// a name clash when the identifier happens to be a context variable
	// (e.g. htmlType=archives colliding with the archives data object).
	if isBareIdentifierLiteral(expr) {
		return expr, nil
	}

	// Numbers, quoted strings, and other literals.
	return evalStandard(expr, ctx)
}

var bareIdentRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func isBareIdentifierLiteral(expr string) bool {
	s := strings.TrimSpace(expr)
	if s == "null" || s == "true" || s == "false" {
		return false
	}
	return bareIdentRe.MatchString(s)
}

// findAndRenderLocalFragment finds a fragment in the current rendering context.
// For ~{::content}, this returns the fragment reference so the caller can
// render it later.
func (p *Processor) findAndRenderLocalFragment(fragmentName string, ctx *Context) ([]*html.Node, error) {
	// Check if we have a local fragment stored in context
	key := "__local_fragment_" + fragmentName
	if val, ok := ctx.Get(key); ok {
		if nodes, ok := val.([]*html.Node); ok {
			// Process these nodes with the current context
			var result []*html.Node
			for _, n := range nodes {
				processed, err := p.processNode(n, ctx)
				if err != nil {
					return nil, err
				}
				result = append(result, processed...)
			}
			return result, nil
		}
	}
	return nil, nil
}

// findFragment searches for a th:fragment definition in a document.
func (p *Processor) findFragment(doc *html.Node, fragmentName string) []*html.Node {
	var found []*html.Node
	var search func(n *html.Node)
	search = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if val, ok := getThAttr(n, "fragment"); ok {
				// Extract fragment name (may have parameters: "html(title,header,content)")
				name := strings.TrimSpace(val)
				if parenIdx := strings.Index(name, "("); parenIdx != -1 {
					name = strings.TrimSpace(name[:parenIdx])
				}
				if name == fragmentName {
					found = append(found, n)
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			search(c)
		}
	}
	search(doc)
	return found
}

// processAttributes handles th:href, th:src, th:content, th:classappend, etc.
func (p *Processor) processAttributes(node *html.Node, ctx *Context) {
	// Get th:inline setting
	inlineMode := ""
	if val, ok := getThAttr(node, "inline"); ok {
		inlineMode = val
		removeAttr(node, "th:inline")
	}

	var attrsToRemove []string
	var attrsToSet []html.Attribute

	for _, attr := range node.Attr {
		if !strings.HasPrefix(attr.Key, "th:") {
			continue
		}

		suffix := attr.Key[3:]
		val := attr.Val

		switch {
		case suffix == "text" || suffix == "utext" || suffix == "if" || suffix == "unless" ||
			suffix == "each" || suffix == "with" || suffix == "replace" || suffix == "fragment" ||
			suffix == "include" || suffix == "insert":
			// Already processed
			continue

		case suffix == "classappend":
			result, err := evalStandard(val, ctx)
			if err == nil {
				appendStr := toStr(result)
				if appendStr != "" {
					existing, _ := getAttr(node, "class")
					if existing != "" {
						setAttr(node, "class", existing+" "+appendStr)
					} else {
						setAttr(node, "class", appendStr)
					}
				}
			}
			attrsToRemove = append(attrsToRemove, attr.Key)

		case suffix == "styleappend":
			result, err := evalStandard(val, ctx)
			if err == nil {
				appendStr := toStr(result)
				if appendStr != "" {
					existing, _ := getAttr(node, "style")
					if existing != "" {
						setAttr(node, "style", existing+appendStr)
					} else {
						setAttr(node, "style", appendStr)
					}
				}
			}
			attrsToRemove = append(attrsToRemove, attr.Key)

		case suffix == "attr":
			// th:attr="rel=${expr}, data-x=|literal ${expr}|"
			// Parse raw value into name=expression pairs first, then evaluate each value.
			pairs := splitByComma(val)
			for _, pair := range pairs {
				pair = strings.TrimSpace(pair)
				eqIdx := indexParamAssign(pair)
				if eqIdx == -1 {
					continue
				}
				attrName := strings.TrimSpace(pair[:eqIdx])
				attrExpr := strings.TrimSpace(pair[eqIdx+1:])
				if attrExpr == "" {
					attrsToRemove = append(attrsToRemove, attrName)
					continue
				}
				result, err := evalStandard(attrExpr, ctx)
				if err != nil {
					continue
				}
				attrVal := toStr(result)
				if attrVal == "null" || attrVal == "" {
					attrsToRemove = append(attrsToRemove, attrName)
				} else {
					attrsToSet = append(attrsToSet, html.Attribute{Key: attrName, Val: attrVal})
				}
			}
			attrsToRemove = append(attrsToRemove, attr.Key)

		case strings.HasPrefix(suffix, "data-"):
			// th:data-src, th:data-next, etc.
			result, err := evalStandard(val, ctx)
			if err == nil {
				attrsToSet = append(attrsToSet, html.Attribute{Key: suffix, Val: toStr(result)})
			}
			attrsToRemove = append(attrsToRemove, attr.Key)

		default:
		// Generic attribute replacement: th:href, th:src, th:content, etc.
		result, err := evalStandard(val, ctx)
		if err == nil {
			attrsToSet = append(attrsToSet, html.Attribute{Key: suffix, Val: toStr(result)})
		}
		attrsToRemove = append(attrsToRemove, attr.Key)
	}
	}

	// Remove processed th: attributes
	for _, key := range attrsToRemove {
		removeAttr(node, key)
	}

	// Set evaluated attributes
	for _, attr := range attrsToSet {
		setAttr(node, attr.Key, attr.Val)
	}

	// If inline mode is javascript, process script content
	if inlineMode == "javascript" {
		p.processJSInline(node, ctx)
	}
	// If inline mode is text, process [# th:if=...] textual blocks
	if inlineMode == "text" {
		p.processTextInline(node, ctx)
	}
}

// processInlineText processes [[${expr}]] and [(${expr})] in text content.
func (p *Processor) processInlineText(text string, ctx *Context) string {
	if !strings.Contains(text, "[[") && !strings.Contains(text, "[(") {
		return text
	}

	var sb strings.Builder
	i := 0
	for i < len(text) {
		// [[${expr}]] - escaped inline text
		if i+1 < len(text) && text[i] == '[' && text[i+1] == '[' {
			end := findInlineEnd(text, i+2, "]]")
			if end != -1 {
				inner := text[i+2 : end]
				val, err := evalStandard(inner, ctx)
				if err == nil {
					sb.WriteString(stdhtml.EscapeString(toStr(val)))
				}
				i = end + 2
				continue
			}
		}

		// [(${expr})] - unescaped inline text
		if i+1 < len(text) && text[i] == '[' && text[i+1] == '(' {
			end := findInlineEnd(text, i+2, ")]")
			if end != -1 {
				inner := text[i+2 : end]
				val, err := evalStandard(inner, ctx)
				if err == nil {
					sb.WriteString(toStr(val))
				}
				i = end + 2
				continue
			}
		}

		sb.WriteByte(text[i])
		i++
	}
	return sb.String()
}

// processJSInline processes JavaScript inline comments /*[[${expr}]]*/
// and bare [[${expr}]] expressions in th:inline="javascript" context.
func (p *Processor) processJSInline(node *html.Node, ctx *Context) {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			c.Data = processJSCommentExprs(c.Data, ctx)
			c.Data = processJSBareInlineExprs(c.Data, ctx)
		}
	}
}

// processTextInline processes [# th:if=...] textual blocks in
// th:inline="text" context (used in <style> and similar elements).
func (p *Processor) processTextInline(node *html.Node, ctx *Context) {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			c.Data = processTextBlocks(c.Data, ctx)
		}
	}
}

// processTextBlocks processes [# th:if="${expr}"] ... [/] textual blocks.
// These are Thymeleaf's textual block syntax used in <style>, <script> etc.
// If the condition is true, the block content is kept (without markers).
// If false, the content is removed. Supports nesting.
func processTextBlocks(text string, ctx *Context) string {
	if !strings.Contains(text, "[# th:") {
		return text
	}
	var sb strings.Builder
	i := 0
	for i < len(text) {
		// Look for [# th:if="..." ] or [# th:unless="..." ]
		if i+6 < len(text) && text[i] == '[' && text[i+1] == '#' && text[i+2] == ' ' {
			end := strings.Index(text[i:], "]")
			if end != -1 {
				tagContent := text[i+2 : i+end]
				tagContent = strings.TrimSpace(tagContent)
				// Parse th:if or th:unless
				if strings.HasPrefix(tagContent, "th:") {
					// Find matching [/]
					blockStart := i + end + 1
					blockEnd := findTextBlockEnd(text, blockStart)
					if blockEnd != -1 {
						inner := text[blockStart:blockEnd]
						// Evaluate condition
						keep := evalTextBlockCondition(tagContent, ctx)
						if keep {
							// Recursively process nested blocks
							sb.WriteString(processTextBlocks(inner, ctx))
						}
						// Skip past [/]
						i = blockEnd + 3 // length of "[/]"
						continue
					}
				}
			}
		}
		sb.WriteByte(text[i])
		i++
	}
	return sb.String()
}

// findTextBlockEnd finds the matching [/] for a [# ...] block, accounting for nesting.
func findTextBlockEnd(text string, start int) int {
	depth := 1
	i := start
	for i < len(text) {
		if i+5 < len(text) && text[i] == '[' && text[i+1] == '#' && text[i+2] == ' ' {
			// Nested block start
			depth++
			// Skip to end of this tag
			end := strings.Index(text[i:], "]")
			if end != -1 {
				i += end + 1
				continue
			}
		}
		if i+2 < len(text) && text[i] == '[' && text[i+1] == '/' && text[i+2] == ']' {
			depth--
			if depth == 0 {
				return i
			}
			i += 3
			continue
		}
		i++
	}
	return -1
}

// evalTextBlockCondition evaluates the condition in a [# th:if="..." ] or
// [# th:unless="..." ] tag. Returns true if the content should be kept.
func evalTextBlockCondition(tag string, ctx *Context) bool {
	// tag is like "th:if=\"${expr}\"" or "th:unless=\"${expr}\""
	if strings.HasPrefix(tag, "th:if=") {
		val := strings.TrimPrefix(tag, "th:if=")
		val = strings.Trim(val, "\"'")
		result, err := evalStandard(val, ctx)
		if err != nil {
			return false
		}
		return toBool(result)
	}
	if strings.HasPrefix(tag, "th:unless=") {
		val := strings.TrimPrefix(tag, "th:unless=")
		val = strings.Trim(val, "\"'")
		result, err := evalStandard(val, ctx)
		if err != nil {
			return true
		}
		return !toBool(result)
	}
	// Unknown tag type — keep content
	return true
}

// processJSBareInlineExprs processes bare [[${expr}]] in JavaScript context.
// Unlike processInlineText which outputs raw values, this formats them as
// JavaScript literals: strings get double-quoted, numbers/booleans stay raw,
// null becomes "null".
func processJSBareInlineExprs(text string, ctx *Context) string {
	if !strings.Contains(text, "[[") {
		return text
	}
	var sb strings.Builder
	i := 0
	for i < len(text) {
		if i+1 < len(text) && text[i] == '[' && text[i+1] == '[' {
			end := findInlineEnd(text, i+2, "]]")
			if end != -1 {
				inner := text[i+2 : end]
				val, err := evalStandard(inner, ctx)
				if err == nil {
					sb.WriteString(jsFormatValue(val))
					i = end + 2
					continue
				}
			}
		}
		sb.WriteByte(text[i])
		i++
	}
	return sb.String()
}

// jsFormatValue formats a Go value as a JavaScript literal.
// Strings are double-quoted with escape sequences, numbers/booleans are raw,
// nil becomes "null".
func jsFormatValue(val any) string {
	if val == nil {
		return "null"
	}
	switch v := val.(type) {
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int, int32, int64, float32, float64:
		return toStr(val)
	default:
		str := toStr(val)
		if isNumericOrBool(str) {
			return str
		}
		// Quote and escape string
		return `"` + strings.ReplaceAll(str, `"`, `\"`) + `"`
	}
}

// processJSCommentExprs processes /*[[${expr}]]*/ in JavaScript content.
// In Thymeleaf's JavaScript natural templates, the pattern is:
//   var x = /*[[${expr}]]*/ "default";
// When processed, both the comment AND the default value are replaced with
// the evaluated expression result, producing:
//   var x = "evaluated_value";
func processJSCommentExprs(text string, ctx *Context) string {
	if !strings.Contains(text, "/*[[") {
		return text
	}

	var sb strings.Builder
	i := 0
	for i < len(text) {
		if i+3 < len(text) && text[i] == '/' && text[i+1] == '*' && text[i+2] == '[' && text[i+3] == '[' {
			// Find closing ]]*/
			end := strings.Index(text[i:], "]]*/")
			if end != -1 {
				inner := text[i+4 : i+end]
				val, err := evalStandard(inner, ctx)
				if err == nil {
					// In JS context, output as JSON-like value
					if val == nil {
						sb.WriteString("null")
					} else {
						str := toStr(val)
						if isNumericOrBool(str) {
							sb.WriteString(str)
						} else {
							// Values may already contain HTML entities (e.g. user pasted &amp; in config).
							// Decode them so JS strings contain the intended raw characters.
							str = stdhtml.UnescapeString(str)
							sb.WriteString(`"` + strings.ReplaceAll(str, `"`, `\"`) + `"`)
						}
					}
					// Skip past ]]*/ and consume the default value token that follows
					i = i + end + 4
					// Skip whitespace
					for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
						i++
					}
					// Consume one JS token (the default value)
					i = skipJSToken(text, i)
					continue
				}
				i = i + end + 4
				continue
			}
		}
		sb.WriteByte(text[i])
		i++
	}
	return sb.String()
}

// skipJSToken skips one JavaScript literal token starting at position i.
// Handles: strings ("..." or '...'), numbers, booleans, null, identifiers,
// arrays [...], and objects {...}.
func skipJSToken(text string, i int) int {
	if i >= len(text) {
		return i
	}
	ch := text[i]
	switch ch {
	case '"', '\'':
		// String literal
		quote := ch
		i++
		for i < len(text) && text[i] != quote {
			if text[i] == '\\' && i+1 < len(text) {
				i += 2
			} else {
				i++
			}
		}
		if i < len(text) {
			i++ // skip closing quote
		}
	case '[', '{':
		// Array or object - find matching bracket
		open := ch
		var close byte
		if open == '[' {
			close = ']'
		} else {
			close = '}'
		}
		depth := 1
		i++
		for i < len(text) && depth > 0 {
			if text[i] == open {
				depth++
			} else if text[i] == close {
				depth--
			} else if text[i] == '"' || text[i] == '\'' {
				// Skip string contents inside
				q := text[i]
				i++
				for i < len(text) && text[i] != q {
					if text[i] == '\\' && i+1 < len(text) {
						i += 2
					} else {
						i++
					}
				}
			}
			i++
		}
	default:
		// Number, boolean, null, or identifier
		for i < len(text) && !isJSDelimiter(text[i]) {
			i++
		}
	}
	return i
}

// isJSDelimiter returns true if the character is a JS token delimiter.
func isJSDelimiter(ch byte) bool {
	switch ch {
	case ' ', '\t', '\n', '\r', ';', ',', ')', ']', '}', '+', '-', '*', '/', '%', '=', '<', '>', '!', '&', '|', '?', ':':
		return true
	}
	return false
}

func isNumericOrBool(s string) bool {
	if s == "true" || s == "false" {
		return true
	}
	// strconv.ParseFloat requires the entire string to be a valid number,
	// unlike fmt.Sscanf which only parses a numeric prefix.
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

// extractCommentExpr checks if a comment node contains [[${expr}]]
func extractCommentExpr(comment string) string {
	comment = strings.TrimSpace(comment)
	if strings.HasPrefix(comment, "[[") && strings.HasSuffix(comment, "]]") {
		return comment
	}
	return ""
}

// findInlineEnd finds the closing bracket for [[ or [( expressions
func findInlineEnd(text string, start int, closing string) int {
	depth := 1
	for i := start; i < len(text); i++ {
		if i+1 < len(text) && text[i] == '[' && (text[i+1] == '[' || text[i+1] == '(') {
			depth++
		} else if i+1 < len(text) {
			if text[i] == ']' && text[i+1] == ']' && closing == "]]" {
				depth--
				if depth == 0 {
					return i
				}
			} else if text[i] == ')' && text[i+1] == ']' && closing == ")]" {
				depth--
				if depth == 0 {
					return i
				}
			}
		}
	}
	return -1
}

// findBody finds the <body> element in a document
func findBody(doc *html.Node) *html.Node {
	var find func(n *html.Node) *html.Node
	find = func(n *html.Node) *html.Node {
		if n.Type == html.ElementNode && n.Data == "body" {
			return n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if found := find(c); found != nil {
				return found
			}
		}
		return nil
	}
	return find(doc)
}

// findHead finds the <head> element in a document
func findHead(doc *html.Node) *html.Node {
	var find func(n *html.Node) *html.Node
	find = func(n *html.Node) *html.Node {
		if n.Type == html.ElementNode && n.Data == "head" {
			return n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if found := find(c); found != nil {
				return found
			}
		}
		return nil
	}
	return find(doc)
}

// inHead reports whether n is a descendant of a <head> element.
func inHead(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == "head" {
			return true
		}
	}
	return false
}

// splitByComma splits by commas, respecting (), ${}, [], '', ""
func splitByComma(s string) []string {
	var parts []string
	depth := 0
	inStr := false
	strCh := byte(0)
	start := 0

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			if ch == strCh {
				inStr = false
			}
			continue
		}
		switch ch {
		case '\'', '"':
			inStr = true
			strCh = ch
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

// indexParamAssign returns the index of the '=' that separates a fragment
// parameter name from its value. It skips '==' and '!=' so that comparison
// expressions such as ${list_layout == 'single' ? ...} are not split in the
// middle of the operator. It also ignores '=' inside string literals and
// bracket/brace/parenthesis groups.
func indexParamAssign(s string) int {
	depth := 0
	inStr := false
	strCh := byte(0)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			if ch == strCh {
				inStr = false
			}
			continue
		}
		switch ch {
		case '\'', '"':
			inStr = true
			strCh = ch
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case '=':
			if depth == 0 {
				// Skip '==' (and the '=' in '!=' has already been passed)
				if i+1 < len(s) && s[i+1] == '=' {
					i++
					continue
				}
				return i
			}
		}
	}
	return -1
}

// toSlice converts a value to a []any slice
func toSlice(val any) []any {
	if val == nil {
		return nil
	}
	if v, ok := val.([]any); ok {
		return v
	}
	// Use reflection for typed slices
	r := reflect.ValueOf(val)
	switch r.Kind() {
	case reflect.Slice, reflect.Array:
		result := make([]any, r.Len())
		for i := 0; i < r.Len(); i++ {
			result[i] = r.Index(i).Interface()
		}
		return result
	}
	return nil
}
