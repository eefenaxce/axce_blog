package thymeleaf

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

// ─── Template Loader ───

// TemplateLoader loads template content from a source.
type TemplateLoader interface {
	Load(name string) (string, error)
}

// FileSystemLoader loads templates from a directory on disk.
type FileSystemLoader struct {
	baseDir string
	mu      sync.RWMutex
	cache   map[string]string
}

func NewFileSystemLoader(baseDir string) *FileSystemLoader {
	return &FileSystemLoader{
		baseDir: baseDir,
		cache:   make(map[string]string),
	}
}

func (l *FileSystemLoader) Load(name string) (string, error) {
	l.mu.RLock()
	if cached, ok := l.cache[name]; ok {
		l.mu.RUnlock()
		return cached, nil
	}
	l.mu.RUnlock()

	// Try .html extension if not present
	path := filepath.Join(l.baseDir, name)
	if !strings.HasSuffix(name, ".html") {
		path += ".html"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// Try without .html extension added
		path2 := filepath.Join(l.baseDir, name)
		data2, err2 := os.ReadFile(path2)
		if err2 != nil {
			return "", fmt.Errorf("template %q not found: %w", name, err)
		}
		data = data2
	}

	content := string(data)
	l.mu.Lock()
	l.cache[name] = content
	l.mu.Unlock()
	return content, nil
}

// ClearCache clears the template cache (e.g., on theme switch).
func (l *FileSystemLoader) ClearCache() {
	l.mu.Lock()
	l.cache = make(map[string]string)
	l.mu.Unlock()
}

// ─── Message Resolver ───

// MessageResolver resolves #{...} message expressions (i18n).
type MessageResolver interface {
	// Resolve looks up a message by key. Returns the message and whether it was found.
	// Parameters are already-evaluated string values for {0}, {1}, ... substitution.
	Resolve(key string, params []string) (string, bool)
}

// PropertiesMessageResolver loads messages from Java .properties files.
// Multiple files can be loaded; later loads override earlier ones for the same key.
type PropertiesMessageResolver struct {
	messages map[string]string
}

func NewPropertiesMessageResolver() *PropertiesMessageResolver {
	return &PropertiesMessageResolver{messages: make(map[string]string)}
}

// Load reads a .properties file content and merges it into the resolver.
func (r *PropertiesMessageResolver) Load(content string) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		// Split on first = or :
		idx := strings.IndexAny(line, "=:")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		r.messages[key] = val
	}
}

// LoadFile reads a .properties file from disk and merges it.
func (r *PropertiesMessageResolver) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	r.Load(string(data))
	return nil
}

// Resolve looks up a message by key and substitutes {0}, {1}, ... placeholders.
func (r *PropertiesMessageResolver) Resolve(key string, params []string) (string, bool) {
	msg, ok := r.messages[key]
	if !ok {
		return "??[" + key + "]??", false
	}
	for i, p := range params {
		placeholder := "{" + strconv.Itoa(i) + "}"
		msg = strings.ReplaceAll(msg, placeholder, p)
	}
	return msg, true
}

// ─── Context ───

// Context holds variables available during template rendering.
type Context struct {
	vars     map[string]any
	parent   *Context
	messages MessageResolver
}

func NewContext() *Context {
	return &Context{vars: make(map[string]any)}
}

func (c *Context) Set(name string, value any) {
	c.vars[name] = value
}

func (c *Context) Get(name string) (any, bool) {
	if v, ok := c.vars[name]; ok {
		return v, true
	}
	if c.parent != nil {
		return c.parent.Get(name)
	}
	return nil, false
}

// SetMessageResolver sets the i18n message resolver on this context.
func (c *Context) SetMessageResolver(r MessageResolver) {
	c.messages = r
}

// GetMessageResolver returns the message resolver (inherited from parent).
func (c *Context) GetMessageResolver() MessageResolver {
	if c.messages != nil {
		return c.messages
	}
	if c.parent != nil {
		return c.parent.GetMessageResolver()
	}
	return nil
}

func (c *Context) Child() *Context {
	return &Context{
		vars:   make(map[string]any),
		parent: c,
	}
}

// ─── Engine ───

// Engine is the main Thymeleaf-compatible template engine.
type Engine struct {
	loader    TemplateLoader
	processor *Processor
}

func NewEngine(loader TemplateLoader) *Engine {
	e := &Engine{loader: loader}
	e.processor = NewProcessor(e)
	return e
}

// Render renders a template by name with the given context.
func (e *Engine) Render(templateName string, ctx *Context) (string, error) {
	return e.RenderFragment(templateName, "", ctx)
}

// RenderFragment renders a specific fragment from a template.
func (e *Engine) RenderFragment(templateName, fragmentName string, ctx *Context) (string, error) {
	doc, err := e.loadTemplate(templateName)
	if err != nil {
		return "", err
	}

	var renderNodes []*html.Node
	if fragmentName != "" {
		renderNodes = e.processor.findFragment(doc, fragmentName)
		if len(renderNodes) == 0 {
			return "", fmt.Errorf("fragment %q not found in template %q", fragmentName, templateName)
		}
	} else {
		// Render entire document body
		renderNodes = []*html.Node{doc}
	}

	var processedNodes []*html.Node
	for _, node := range renderNodes {
		processed, err := e.processor.processNode(node, ctx)
		if err != nil {
			return "", err
		}
		processedNodes = append(processedNodes, processed...)
	}
	// Fix body-level content that the HTML parser placed inside <head>
	// because <template> (the synthetic stand-in for <th:block>) is allowed
	// in <head>. This is essential for layout delegation patterns where a
	// th:replace inside <head> produces a <body> tag.
	for _, n := range processedNodes {
		ensureBodyContentOutsideHead(n)
	}

	var sb strings.Builder
	for _, n := range processedNodes {
		renderNode(&sb, n)
	}
	return sb.String(), nil
}

// RenderFragmentNodes renders a specific fragment and returns the processed
// nodes directly (without serializing to string and re-parsing). This avoids
// the lossy round-trip through html.Parse that would otherwise discard the
// <html>/<head> structure when the fragment is a full-page layout template.
func (e *Engine) RenderFragmentNodes(templateName, fragmentName string, ctx *Context) ([]*html.Node, error) {
	doc, err := e.loadTemplate(templateName)
	if err != nil {
		return nil, err
	}

	var renderNodes []*html.Node
	if fragmentName != "" {
		renderNodes = e.processor.findFragment(doc, fragmentName)
		if len(renderNodes) == 0 {
			return nil, fmt.Errorf("fragment %q not found in template %q", fragmentName, templateName)
		}
	} else {
		// When no fragment name is given (e.g. ~{modules/widgets/popular-posts}),
		// html.Parse wraps a bare fragment in a full <html><head></head><body>...</body></html>
		// document. Return the body's children (cloned and detached) so the inserted
		// content does not include the synthetic wrapper and can be safely appended
		// to the calling document.
		if body := findBody(doc); body != nil {
			for c := body.FirstChild; c != nil; c = c.NextSibling {
				renderNodes = append(renderNodes, cloneNode(c))
			}
		}
		if len(renderNodes) == 0 {
			renderNodes = []*html.Node{doc}
		}
	}

	var result []*html.Node
	for _, node := range renderNodes {
		processed, err := e.processor.processNode(node, ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, processed...)
	}
	return result, nil
}

// loadTemplate parses a template file into an HTML node tree.
func (e *Engine) loadTemplate(name string) (*html.Node, error) {
	content, err := e.loader.Load(name)
	if err != nil {
		return nil, err
	}
	// Pre-process th:block tags before parsing.
	// 1. Fix self-closing <th:block .../> → <th:block ...></th:block>
	// 2. Replace <th:block> with <template data-th-block> so the HTML5 parser
	//    doesn't close <head> prematurely. Per the HTML5 spec, unknown non-void
	//    elements inside <head> trigger an implicit </head><body>. <template>
	//    is one of the few elements explicitly allowed in <head>.
	content = fixSelfClosingTags(content)
	content = replaceThBlockTags(content)
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %q: %w", name, err)
	}
	return doc, nil
}

// selfClosingThBlockRe matches <th:block ... /> self-closing tags.
var selfClosingThBlockRe = regexp.MustCompile(`<th:block\b([^>]*?)\s*/>`)

// fixSelfClosingTags converts self-closing <th:block .../> tags into
// explicit <th:block ...></th:block> pairs so the HTML5 parser doesn't
// nest subsequent content inside them.
func fixSelfClosingTags(content string) string {
	return selfClosingThBlockRe.ReplaceAllString(content, "<th:block$1></th:block>")
}

// thBlockOpenRe matches <th:block ...> opening tags (not self-closing).
var thBlockOpenRe = regexp.MustCompile(`<th:block\b([^>]*?)>`)

// thBlockCloseRe matches </th:block> closing tags.
var thBlockCloseRe = regexp.MustCompile(`</th:block\s*>`)

// ensureBodyContentOutsideHead moves any body-level content that the HTML5
// parser placed inside <head> (because <template> is allowed in <head>) out to
// a proper <body> sibling. This fixes layout delegation where a th:replace
// inside <head> produces a <body> tag.
func ensureBodyContentOutsideHead(root *html.Node) {
	var htmlNode *html.Node
	var findHTML func(n *html.Node)
	findHTML = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "html" {
			htmlNode = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findHTML(c)
		}
	}
	findHTML(root)
	if htmlNode == nil {
		return
	}

	var headNode, bodyNode *html.Node
	for c := htmlNode.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			if c.Data == "head" {
				headNode = c
			} else if c.Data == "body" {
				bodyNode = c
			}
		}
	}
	if headNode == nil {
		return
	}

	// Collect children of <head> that do not belong there.
	var toMove []*html.Node
	for c := headNode.FirstChild; c != nil; {
		next := c.NextSibling
		if shouldMoveOutOfHead(c) {
			headNode.RemoveChild(c)
			toMove = append(toMove, c)
		}
		c = next
	}
	if len(toMove) == 0 {
		return
	}

	// Ensure a real <body> exists after </head>.
	if bodyNode == nil {
		bodyNode = &html.Node{Type: html.ElementNode, Data: "body"}
		if headNode.NextSibling != nil {
			htmlNode.InsertBefore(bodyNode, headNode.NextSibling)
		} else {
			htmlNode.AppendChild(bodyNode)
		}
	}

	// Move nodes into the real <body>. If a node is itself a <body>, unwrap it.
	for _, n := range toMove {
		if n.Type == html.ElementNode && n.Data == "body" {
			for c := n.FirstChild; c != nil; {
				next := c.NextSibling
				n.RemoveChild(c)
				bodyNode.AppendChild(c)
				c = next
			}
		} else {
			bodyNode.AppendChild(n)
		}
	}
}

// shouldMoveOutOfHead reports whether a node should not remain inside <head>.
func shouldMoveOutOfHead(n *html.Node) bool {
	switch n.Type {
	case html.ElementNode:
		switch n.Data {
		case "base", "link", "meta", "noscript", "script", "style",
			"template", "title", "th:block", "th:block/":
			return false
		}
		return true
	case html.TextNode:
		return strings.TrimSpace(n.Data) != ""
	}
	return false
}

// replaceThBlockTags replaces <th:block> with <template data-th-block> and
// </th:block> with </template>, but only inside <head>...</head>. Outside
// <head>, th:block is kept as-is because <template> is explicitly allowed in
// <head> per HTML5: a <template> token in "after head" mode switches back to
// "in head" and stays inside <head>. Keeping body-level th:block as th:block
// prevents body content from being pulled back into <head>.
func replaceThBlockTags(content string) string {
	var result strings.Builder
	inHead := false
	i := 0
	for i < len(content) {
		if content[i] != '<' {
			result.WriteByte(content[i])
			i++
			continue
		}

		// Detect <head ...> and </head ...> to track head state.
		if isTagStart(content, i, "head") {
			inHead = true
			end := findTagEnd(content, i)
			result.WriteString(content[i : end+1])
			i = end + 1
			continue
		}
		if isEndTagStart(content, i, "head") {
			inHead = false
			end := findTagEnd(content, i)
			result.WriteString(content[i : end+1])
			i = end + 1
			continue
		}

		// Handle th:block tags.
		if isTagStart(content, i, "th:block") {
			end, selfClosing := findTagEndWithSelfClose(content, i)
			// The text between '<th:block' and '>' (exclusive of both).
			attrs := content[i+9 : end]
			if selfClosing {
				if inHead {
					result.WriteString("<template data-th-block")
					result.WriteString(attrs)
					result.WriteString("></template>")
				} else {
					result.WriteString("<th:block")
					result.WriteString(attrs)
					result.WriteString("></th:block>")
				}
			} else {
				if inHead {
					result.WriteString("<template data-th-block")
					result.WriteString(attrs)
					result.WriteString(">")
				} else {
					result.WriteString(content[i : end+1])
				}
			}
			i = end + 1
			continue
		}
		if isEndTagStart(content, i, "th:block") {
			end := findTagEnd(content, i)
			if inHead {
				result.WriteString("</template>")
			} else {
				result.WriteString(content[i : end+1])
			}
			i = end + 1
			continue
		}

		// Copy any other tag unchanged.
		end := findTagEnd(content, i)
		result.WriteString(content[i : end+1])
		i = end + 1
	}
	return result.String()
}

// isTagStart reports whether a start tag for the given lowercase tag name
// begins at position i in content (case-insensitive).
func isTagStart(content string, i int, tag string) bool {
	if i+1+len(tag) > len(content) {
		return false
	}
	if content[i] != '<' {
		return false
	}
	for j := 0; j < len(tag); j++ {
		if toLower(content[i+1+j]) != tag[j] {
			return false
		}
	}
	next := i + 1 + len(tag)
	if next >= len(content) {
		return false
	}
	c := content[next]
	return c == '>' || c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '/'
}

// isEndTagStart reports whether an end tag for the given lowercase tag name
// begins at position i in content (case-insensitive).
func isEndTagStart(content string, i int, tag string) bool {
	if i+2+len(tag) > len(content) {
		return false
	}
	if content[i] != '<' || content[i+1] != '/' {
		return false
	}
	for j := 0; j < len(tag); j++ {
		if toLower(content[i+2+j]) != tag[j] {
			return false
		}
	}
	next := i + 2 + len(tag)
	if next >= len(content) {
		return false
	}
	c := content[next]
	return c == '>' || c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// findTagEnd returns the index of the '>' that closes the tag starting at i.
func findTagEnd(content string, i int) int {
	inQuote := byte(0)
	for j := i + 1; j < len(content); j++ {
		ch := content[j]
		if inQuote != 0 {
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			inQuote = ch
			continue
		}
		if ch == '>' {
			return j
		}
	}
	return len(content) - 1
}

// findTagEndWithSelfClose returns the index of the '>' that closes the tag
// starting at i, and whether it is a self-closing tag ('/>').
func findTagEndWithSelfClose(content string, i int) (int, bool) {
	end := findTagEnd(content, i)
	selfClosing := false
	for j := end - 1; j > i; j-- {
		if content[j] == '/' {
			selfClosing = true
			break
		}
		if content[j] != ' ' && content[j] != '\t' && content[j] != '\n' && content[j] != '\r' {
			break
		}
	}
	return end, selfClosing
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// GetProcessor returns the engine's processor (used internally).
func (e *Engine) GetProcessor() *Processor {
	return e.processor
}

// GetLoader returns the engine's loader.
func (e *Engine) GetLoader() TemplateLoader {
	return e.loader
}
