package thymeleaf

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// DebugMode controls verbose logging of expression evaluation.
const DebugMode = false

// ─── Expression Evaluator ───
// Handles ${...} variable expressions, |...| literal substitutions,
// @{...} URL expressions, and standard expression operators.

// evalStandard evaluates a Thymeleaf standard expression (the value of a th:* attribute).
// Examples:
//
//	${site.title}
//	${a} and ${b}
//	|text ${expr} text|
//	@{/path/to/resource}
//	${expr} != ''
//	${cond ? 'a' : 'b'}
func evalStandard(expr string, ctx *Context) (any, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", nil
	}

	// Single ${...} expression
	if strings.HasPrefix(expr, "${") && strings.HasSuffix(expr, "}") && balancedBraces(expr) == 1 {
		return evalExpr(expr[2:len(expr)-1], ctx)
	}

	// Single |...| literal substitution
	if strings.HasPrefix(expr, "|") && strings.HasSuffix(expr, "|") && len(expr) >= 2 {
		return evalLiteralSubstitution(expr[1:len(expr)-1], ctx)
	}

	// Single @{...} URL expression
	if strings.HasPrefix(expr, "@{") && strings.HasSuffix(expr, "}") {
		return evalURLExpr(expr[2:len(expr)-1], ctx)
	}

	// Single #{...} message (i18n) expression
	if strings.HasPrefix(expr, "#{") && strings.HasSuffix(expr, "}") {
		return evalMessageExpr(expr[2:len(expr)-1], ctx)
	}

	// Single ~{...} fragment expression - handled separately by processor
	if strings.HasPrefix(expr, "~{") {
		return expr, nil // Return as-is, processor handles fragment expressions
	}

	// Complex expression with operators - tokenize and parse
	return evalComplexExpr(expr, ctx)
}

// balancedBraces counts ${...} brace pairs at top level
func balancedBraces(s string) int {
	depth := 0
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '{' {
			depth++
		} else if s[i] == '}' {
			depth--
			if depth == 0 {
				count++
			}
		}
	}
	return count
}

// evalExpr evaluates the content inside ${...}
func evalExpr(expr string, ctx *Context) (any, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", nil
	}
	p := newExprParser(expr)
	ast, err := p.parseExpression()
	if err != nil {
		return nil, fmt.Errorf("parse error in %q: %w", expr, err)
	}
	return ast.eval(ctx)
}

// evalLiteralSubstitution handles |text ${expr} text|
func evalLiteralSubstitution(content string, ctx *Context) (any, error) {
	var sb strings.Builder
	i := 0
	for i < len(content) {
		if i+1 < len(content) && content[i] == '$' && content[i+1] == '{' {
			// Find matching }
			end := findMatchingBrace(content, i+1)
			if end == -1 {
				sb.WriteByte(content[i])
				i++
				continue
			}
			inner := content[i+2 : end]
			val, err := evalExpr(inner, ctx)
			if err != nil {
				return nil, err
			}
			sb.WriteString(toLiteralString(val))
			i = end + 1
		} else if i+1 < len(content) && content[i] == '#' && content[i+1] == '{' {
			// Find matching }
			end := findMatchingBrace(content, i+1)
			if end == -1 {
				sb.WriteByte(content[i])
				i++
				continue
			}
			inner := content[i+2 : end]
			val, err := evalMessageExpr(inner, ctx)
			if err != nil {
				return nil, err
			}
			sb.WriteString(toLiteralString(val))
			i = end + 1
		} else {
			sb.WriteByte(content[i])
			i++
		}
	}
	return sb.String(), nil
}

// evalMessageExpr evaluates the content inside #{...}.
// Supports: key, key(param1, param2) where params are ${...} or literals.
func evalMessageExpr(content string, ctx *Context) (any, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", nil
	}

	// Split key and optional params
	var key string
	var params []string

	// Find opening paren for params (if any)
	// Key is everything before the first ( that's at depth 0
	depth := 0
	parenIdx := -1
	for i := 0; i < len(content); i++ {
		ch := content[i]
		if ch == '(' && depth == 0 {
			parenIdx = i
			break
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
		}
	}

	if parenIdx >= 0 {
		key = strings.TrimSpace(content[:parenIdx])
		// Extract params between ( and )
		paramsContent := content[parenIdx+1:]
		// Remove trailing )
		if strings.HasSuffix(paramsContent, ")") {
			paramsContent = paramsContent[:len(paramsContent)-1]
		}
		params = splitMessageParams(paramsContent)
	} else {
		key = content
	}

	// Evaluate each parameter
	evaluatedParams := make([]string, 0, len(params))
	for _, p := range params {
		p = strings.TrimSpace(p)
		val, err := evalStandard(p, ctx)
		if err != nil {
			evaluatedParams = append(evaluatedParams, p)
		} else {
			evaluatedParams = append(evaluatedParams, toStr(val))
		}
	}

	// Resolve the message
	resolver := ctx.GetMessageResolver()
	if resolver == nil {
		// No resolver — return key as fallback
		return key, nil
	}
	msg, _ := resolver.Resolve(key, evaluatedParams)
	return msg, nil
}

// splitMessageParams splits comma-separated params, respecting ${...} nesting.
func splitMessageParams(s string) []string {
	var params []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
		} else if ch == ',' && depth == 0 {
			params = append(params, strings.TrimSpace(s[start:i]))
			start = i + 1
		}
	}
	if start < len(s) {
		rest := strings.TrimSpace(s[start:])
		if rest != "" {
			params = append(params, rest)
		}
	}
	return params
}

// evalURLExpr handles @{/path} or @{${expr}}
func evalURLExpr(content string, ctx *Context) (any, error) {
	// Handle @{${expr}} - evaluate inner expression
	if strings.HasPrefix(content, "${") {
		return evalExpr(content[2:len(content)-1], ctx)
	}
	// Handle @{/path} - prepend theme base path for /assets/ URLs.
	// In Halo, theme templates reference assets via @{/assets/...} which
	// resolves to /themes/<theme-id>/assets/...
	if strings.HasPrefix(content, "/assets/") {
		if base, ok := ctx.Get("__theme_base_path"); ok {
			baseStr := toStr(base)
			if baseStr != "" {
				return baseStr + content, nil
			}
		}
	}
	// Other paths (e.g. /categories, /search) are returned as-is
	return content, nil
}

func findMatchingBrace(s string, start int) int {
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '{' {
			depth++
		} else if s[i] == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// ─── Tokenizer ───

type tokenType int

const (
	tokEOF tokenType = iota
	tokIdent
	tokNumber
	tokString
	tokOp       // ==, !=, >=, <=, >, <, +, -, *, /, !
	tokLParen   // (
	tokRParen   // )
	tokComma    // ,
	tokDot      // .
	tokColon    // :
	tokQuestion // ?
	tokSafeNav  // ?.
	tokElvis    // ?:
	tokLBrack   // [
	tokRBrack   // ]
	tokLBrace   // {
	tokRBrace   // }
	tokVarExpr  // ${...}
	tokUrlExpr  // @{...}
	tokDollar   // $ (standalone, shouldn't happen)
	tokHash     // # (utility prefix)
	tokTrue
	tokFalse
	tokNull
	tokAnd // and
	tokOr  // or
	tokNot // not
	tokNew // new
)

type token struct {
	typ   tokenType
	value string
}

type tokenizer struct {
	input []rune
	pos   int
}

func newTokenizer(input string) *tokenizer {
	return &tokenizer{input: []rune(input)}
}

func (t *tokenizer) tokenize() ([]token, error) {
	var tokens []token
	for t.pos < len(t.input) {
		t.skipWhitespace()
		if t.pos >= len(t.input) {
			break
		}

		ch := t.input[t.pos]

		// ${...} expression
		if ch == '$' && t.peek(1) == '{' {
			start := t.pos
			t.pos += 2
			depth := 1
			for t.pos < len(t.input) && depth > 0 {
				if t.input[t.pos] == '{' {
					depth++
				} else if t.input[t.pos] == '}' {
					depth--
				}
				if depth > 0 {
					t.pos++
				}
			}
			if depth != 0 {
				return nil, fmt.Errorf("unmatched ${ in expression")
			}
			t.pos++ // skip }
			tokens = append(tokens, token{tokVarExpr, string(t.input[start:t.pos])})
			continue
		}

		// URL expression @{...}
		if ch == '@' && t.peek(1) == '{' {
			start := t.pos
			t.pos += 2
			depth := 1
			for t.pos < len(t.input) && depth > 0 {
				if t.input[t.pos] == '{' {
					depth++
				} else if t.input[t.pos] == '}' {
					depth--
				}
				if depth > 0 {
					t.pos++
				}
			}
			if depth != 0 {
				return nil, fmt.Errorf("unmatched @{ in expression")
			}
			t.pos++ // skip }
			tokens = append(tokens, token{tokUrlExpr, string(t.input[start:t.pos])})
			continue
		}

		// String literal (Thymeleaf style: '' inside single quotes is an escaped ')
		if ch == '\'' {
			t.pos++
			var sb strings.Builder
			for t.pos < len(t.input) {
				if t.input[t.pos] == '\'' {
					// Escaped quote: '' -> '
					if t.pos+1 < len(t.input) && t.input[t.pos+1] == '\'' {
						sb.WriteRune('\'')
						t.pos += 2
						continue
					}
					break
				}
				sb.WriteRune(t.input[t.pos])
				t.pos++
			}
			if t.pos >= len(t.input) {
				return nil, fmt.Errorf("unterminated string")
			}
			tokens = append(tokens, token{tokString, sb.String()})
			t.pos++ // skip closing '
			continue
		}

		// Number
		if unicode.IsDigit(ch) {
			start := t.pos
			for t.pos < len(t.input) && (unicode.IsDigit(t.input[t.pos]) || t.input[t.pos] == '.') {
				t.pos++
			}
			tokens = append(tokens, token{tokNumber, string(t.input[start:t.pos])})
			continue
		}

		// Utility prefix
		if ch == '#' {
			t.pos++
			start := t.pos
			for t.pos < len(t.input) && (unicode.IsLetter(t.input[t.pos]) || t.input[t.pos] == '_') {
				t.pos++
			}
			if t.pos > start {
				tokens = append(tokens, token{tokHash, "#" + string(t.input[start:t.pos])})
			} else {
				tokens = append(tokens, token{tokHash, "#"})
			}
			continue
		}

		// Identifier or keyword
		if unicode.IsLetter(ch) || ch == '_' {
			start := t.pos
			for t.pos < len(t.input) && (unicode.IsLetter(t.input[t.pos]) || unicode.IsDigit(t.input[t.pos]) || t.input[t.pos] == '_' || t.input[t.pos] == '-') {
				t.pos++
			}
			word := string(t.input[start:t.pos])
			switch word {
			case "and":
				tokens = append(tokens, token{tokAnd, word})
			case "or":
				tokens = append(tokens, token{tokOr, word})
			case "not":
				tokens = append(tokens, token{tokNot, word})
			case "true":
				tokens = append(tokens, token{tokTrue, word})
			case "false":
				tokens = append(tokens, token{tokFalse, word})
			case "null":
				tokens = append(tokens, token{tokNull, word})
			case "new":
				tokens = append(tokens, token{tokNew, word})
			// Thymeleaf textual comparison operators
			case "gt":
				tokens = append(tokens, token{tokOp, ">"})
			case "lt":
				tokens = append(tokens, token{tokOp, "<"})
			case "ge":
				tokens = append(tokens, token{tokOp, ">="})
			case "le":
				tokens = append(tokens, token{tokOp, "<="})
			case "eq":
				tokens = append(tokens, token{tokOp, "=="})
			case "ne", "neq":
				tokens = append(tokens, token{tokOp, "!="})
			default:
				tokens = append(tokens, token{tokIdent, word})
			}
			continue
		}

		// Operators
		switch ch {
		case '(':
			tokens = append(tokens, token{tokLParen, "("})
			t.pos++
		case ')':
			tokens = append(tokens, token{tokRParen, ")"})
			t.pos++
		case ',':
			tokens = append(tokens, token{tokComma, ","})
			t.pos++
		case '.':
			tokens = append(tokens, token{tokDot, "."})
			t.pos++
		case '[':
			tokens = append(tokens, token{tokLBrack, "["})
			t.pos++
		case ']':
			tokens = append(tokens, token{tokRBrack, "]"})
			t.pos++
		case '{':
			tokens = append(tokens, token{tokLBrace, "{"})
			t.pos++
		case '}':
			tokens = append(tokens, token{tokRBrace, "}"})
			t.pos++
		case ':':
			tokens = append(tokens, token{tokColon, ":"})
			t.pos++
		case '?':
			if t.peek(1) == '.' {
				tokens = append(tokens, token{tokSafeNav, "?."})
				t.pos += 2
			} else if t.peek(1) == ':' {
				tokens = append(tokens, token{tokElvis, "?:"})
				t.pos += 2
			} else {
				tokens = append(tokens, token{tokQuestion, "?"})
				t.pos++
			}
		case '=':
			if t.peek(1) == '=' {
				tokens = append(tokens, token{tokOp, "=="})
				t.pos += 2
			} else {
				tokens = append(tokens, token{tokOp, "="})
				t.pos++
			}
		case '!':
			if t.peek(1) == '=' {
				tokens = append(tokens, token{tokOp, "!="})
				t.pos += 2
			} else {
				tokens = append(tokens, token{tokOp, "!"})
				t.pos++
			}
		case '>':
			if t.peek(1) == '=' {
				tokens = append(tokens, token{tokOp, ">="})
				t.pos += 2
			} else {
				tokens = append(tokens, token{tokOp, ">"})
				t.pos++
			}
		case '<':
			if t.peek(1) == '=' {
				tokens = append(tokens, token{tokOp, "<="})
				t.pos += 2
			} else {
				tokens = append(tokens, token{tokOp, "<"})
				t.pos++
			}
		case '&':
			if t.peek(1) == '&' {
				tokens = append(tokens, token{tokAnd, "&&"})
				t.pos += 2
			} else {
				return nil, fmt.Errorf("unexpected character %q at position %d", ch, t.pos)
			}
		case '|':
			if t.peek(1) == '|' {
				tokens = append(tokens, token{tokOr, "||"})
				t.pos += 2
			} else {
				return nil, fmt.Errorf("unexpected character %q at position %d", ch, t.pos)
			}
		case '+', '-', '*', '/':
			tokens = append(tokens, token{tokOp, string(ch)})
			t.pos++
		default:
			return nil, fmt.Errorf("unexpected character %q at position %d", ch, t.pos)
		}
	}
	tokens = append(tokens, token{tokEOF, ""})
	return tokens, nil
}

func (t *tokenizer) peek(offset int) rune {
	pos := t.pos + offset
	if pos < len(t.input) {
		return t.input[pos]
	}
	return 0
}

func (t *tokenizer) skipWhitespace() {
	for t.pos < len(t.input) && unicode.IsSpace(t.input[t.pos]) {
		t.pos++
	}
}

// ─── AST ───

type exprNode interface {
	eval(ctx *Context) (any, error)
}

// literalNode holds a literal value (string, number, bool, nil)
type literalNode struct {
	value any
}

func (n *literalNode) eval(ctx *Context) (any, error) {
	return n.value, nil
}

// varExprNode holds a ${...} expression
type varExprNode struct {
	inner string
}

func (n *varExprNode) eval(ctx *Context) (any, error) {
	return evalExpr(n.inner, ctx)
}

// urlExprNode holds a @{...} URL expression
type urlExprNode struct {
	inner string
}

func (n *urlExprNode) eval(ctx *Context) (any, error) {
	val, err := evalURLExpr(n.inner, ctx)
	if err != nil {
		return nil, err
	}
	// 在复杂表达式中与其他路径片段拼接时，去掉尾部斜杠可避免产生 //
	if s, ok := val.(string); ok && len(s) > 1 && strings.HasSuffix(s, "/") {
		return strings.TrimSuffix(s, "/"), nil
	}
	return val, nil
}

// identNode holds an identifier (variable name)
type identNode struct {
	name string
}

func (n *identNode) eval(ctx *Context) (any, error) {
	val, ok := ctx.Get(n.name)
	if !ok {
		return nil, nil
	}
	return val, nil
}

// hashNode holds a utility reference (#strings, #lists, etc.)
type hashNode struct {
	name string
}

func (n *hashNode) eval(ctx *Context) (any, error) {
	return getUtility(n.name, ctx)
}

// propertyNode holds a.b.c property access
type propertyNode struct {
	target exprNode
	name   string
	safe   bool // ?. safe navigation
}

func (n *propertyNode) eval(ctx *Context) (any, error) {
	target, err := n.target.eval(ctx)
	if err != nil {
		return nil, err
	}
	if n.safe && isNil(target) {
		return nil, nil
	}
	return getProperty(target, n.name)
}

// methodCallNode holds a method call like obj.method(args)
type methodCallNode struct {
	target exprNode
	name   string
	args   []exprNode
	safe   bool
}

func (n *methodCallNode) eval(ctx *Context) (any, error) {
	target, err := n.target.eval(ctx)
	if err != nil {
		return nil, err
	}
	if n.safe && isNil(target) {
		return nil, nil
	}
	args := make([]any, len(n.args))
	for i, arg := range n.args {
		v, err := arg.eval(ctx)
		if err != nil {
			return nil, err
		}
		args[i] = v
	}
	return callMethod(target, n.name, args)
}

// binaryOpNode holds binary operations
type binaryOpNode struct {
	op    string
	left  exprNode
	right exprNode
}

func (n *binaryOpNode) eval(ctx *Context) (any, error) {
	left, err := n.left.eval(ctx)
	if err != nil {
		return nil, err
	}
	right, err := n.right.eval(ctx)
	if err != nil {
		return nil, err
	}

	// For equality/inequality, treat unresolved bare identifiers as their name.
	// This supports the FreeMarker-ported theme convention of writing
	// comparisons like ${x} == num without quotes.
	if n.op == "==" || n.op == "!=" {
		if left == nil {
			if ident, ok := n.left.(*identNode); ok {
				left = ident.name
			}
		}
		if right == nil {
			if ident, ok := n.right.(*identNode); ok {
				right = ident.name
			}
		}
	}

	return evalBinaryOp(n.op, left, right)
}

// unaryOpNode holds unary operations (not, !)
type unaryOpNode struct {
	op      string
	operand exprNode
}

func (n *unaryOpNode) eval(ctx *Context) (any, error) {
	val, err := n.operand.eval(ctx)
	if err != nil {
		return nil, err
	}
	switch n.op {
	case "not", "!":
		return !toBool(val), nil
	}
	return nil, fmt.Errorf("unknown unary op %q", n.op)
}

// ternaryNode holds condition ? trueExpr : falseExpr
type ternaryNode struct {
	cond exprNode
	then exprNode
	els  exprNode
}

func (n *ternaryNode) eval(ctx *Context) (any, error) {
	cond, err := n.cond.eval(ctx)
	if err != nil {
		return nil, err
	}
	if toBool(cond) {
		return n.then.eval(ctx)
	}
	return n.els.eval(ctx)
}

// elvisNode holds expr ?: defaultVal
type elvisNode struct {
	target     exprNode
	defaultVal exprNode
}

func (n *elvisNode) eval(ctx *Context) (any, error) {
	val, err := n.target.eval(ctx)
	if err != nil {
		return nil, err
	}
	if isNil(val) || val == "" {
		return n.defaultVal.eval(ctx)
	}
	return val, nil
}

// indexNode holds a[index] access
type indexNode struct {
	target exprNode
	index  exprNode
}

func (n *indexNode) eval(ctx *Context) (any, error) {
	target, err := n.target.eval(ctx)
	if err != nil {
		return nil, err
	}
	idx, err := n.index.eval(ctx)
	if err != nil {
		return nil, err
	}
	return getIndex(target, idx)
}

// mapLiteralNode holds a {key: value, key2: value2} map literal.
type mapLiteralNode struct {
	pairs []mapPair
}

type mapPair struct {
	key   exprNode
	value exprNode
}

func (n *mapLiteralNode) eval(ctx *Context) (any, error) {
	result := make(map[string]any, len(n.pairs))
	for _, p := range n.pairs {
		v, err := p.value.eval(ctx)
		if err != nil {
			return nil, err
		}
		var keyStr string
		if ident, ok := p.key.(*identNode); ok {
			// Map literal keys like {page: 1} should use the identifier
			// name as the key string, not resolve it as a variable.
			keyStr = ident.name
		} else {
			kv, err := p.key.eval(ctx)
			if err != nil {
				return nil, err
			}
			keyStr = toStr(kv)
		}
		result[keyStr] = v
	}
	return result, nil
}

// newExprNode handles Java `new ClassName(args)` expressions.
// Currently only java.util.Date is supported (returns a JavaDate wrapping now).
type newExprNode struct {
	className string
}

// JavaDate wraps a millisecond timestamp and exposes getTime()/toString()
// so Thymeleaf expressions like `new java.util.Date().getTime()` work.
type JavaDate struct {
	millis int64
}

func (d JavaDate) GetTime() int64 {
	return d.millis
}

func (n *newExprNode) eval(ctx *Context) (any, error) {
	switch n.className {
	case "java.util.Date", "Date":
		return JavaDate{millis: time.Now().UnixMilli()}, nil
	default:
		return nil, nil
	}
}

// ─── Parser (recursive descent) ───

type exprParser struct {
	tokens []token
	pos    int
}

func newExprParser(input string) *exprParser {
	t := newTokenizer(input)
	tokens, err := t.tokenize()
	if err != nil {
		// Return a parser with just EOF - eval will return error
		return &exprParser{tokens: []token{{tokEOF, ""}}}
	}
	return &exprParser{tokens: tokens}
}

func (p *exprParser) current() token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return token{tokEOF, ""}
}

func (p *exprParser) advance() token {
	tok := p.current()
	p.pos++
	return tok
}

func (p *exprParser) parseExpression() (exprNode, error) {
	return p.parseTernary()
}

func (p *exprParser) parseTernary() (exprNode, error) {
	cond, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	if p.current().typ == tokElvis {
		p.advance()
		def, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		return &elvisNode{target: cond, defaultVal: def}, nil
	}

	if p.current().typ == tokQuestion {
		p.advance()
		then, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.current().typ != tokColon {
			return nil, fmt.Errorf("expected ':' in ternary")
		}
		p.advance()
		els, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		return &ternaryNode{cond: cond, then: then, els: els}, nil
	}

	return cond, nil
}

func (p *exprParser) parseOr() (exprNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.current().typ == tokOr {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &binaryOpNode{op: "or", left: left, right: right}
	}
	return left, nil
}

func (p *exprParser) parseAnd() (exprNode, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}
	for p.current().typ == tokAnd {
		p.advance()
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		left = &binaryOpNode{op: "and", left: left, right: right}
	}
	return left, nil
}

func (p *exprParser) parseEquality() (exprNode, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.current().typ == tokOp && (p.current().value == "==" || p.current().value == "!=") {
		op := p.advance().value
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &binaryOpNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *exprParser) parseComparison() (exprNode, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	for p.current().typ == tokOp && (p.current().value == ">" || p.current().value == "<" || p.current().value == ">=" || p.current().value == "<=") {
		op := p.advance().value
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		left = &binaryOpNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *exprParser) parseAdditive() (exprNode, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for p.current().typ == tokOp && (p.current().value == "+" || p.current().value == "-") {
		op := p.advance().value
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = &binaryOpNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *exprParser) parseMultiplicative() (exprNode, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.current().typ == tokOp && (p.current().value == "*" || p.current().value == "/") {
		op := p.advance().value
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &binaryOpNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *exprParser) parseUnary() (exprNode, error) {
	if p.current().typ == tokNot || (p.current().typ == tokOp && p.current().value == "!") {
		op := p.advance().value
		if op == "!" {
			op = "not"
		}
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &unaryOpNode{op: op, operand: operand}, nil
	}
	return p.parsePostfix()
}

func (p *exprParser) parsePostfix() (exprNode, error) {
	node, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		switch p.current().typ {
		case tokDot:
			p.advance()
			if p.current().typ != tokIdent {
				return nil, fmt.Errorf("expected identifier after '.'")
			}
			name := p.advance().value
			if p.current().typ == tokLParen {
				// Method call
				p.advance()
				args, err := p.parseArgs()
				if err != nil {
					return nil, err
				}
				if p.current().typ != tokRParen {
					return nil, fmt.Errorf("expected ')' after method args")
				}
				p.advance()
				node = &methodCallNode{target: node, name: name, args: args}
			} else {
				node = &propertyNode{target: node, name: name}
			}
		case tokSafeNav:
			p.advance()
			if p.current().typ != tokIdent {
				return nil, fmt.Errorf("expected identifier after '?.'")
			}
			name := p.advance().value
			if p.current().typ == tokLParen {
				p.advance()
				args, err := p.parseArgs()
				if err != nil {
					return nil, err
				}
				if p.current().typ != tokRParen {
					return nil, fmt.Errorf("expected ')' after method args")
				}
				p.advance()
				node = &methodCallNode{target: node, name: name, args: args, safe: true}
			} else {
				node = &propertyNode{target: node, name: name, safe: true}
			}
		case tokLBrack:
			p.advance()
			idx, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if p.current().typ != tokRBrack {
				return nil, fmt.Errorf("expected ']'")
			}
			p.advance()
			node = &indexNode{target: node, index: idx}
		default:
			return node, nil
		}
	}
}

func (p *exprParser) parseArgs() ([]exprNode, error) {
	var args []exprNode
	if p.current().typ == tokRParen {
		return args, nil
	}
	for {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.current().typ == tokComma {
			p.advance()
			continue
		}
		break
	}
	return args, nil
}

func (p *exprParser) parsePrimary() (exprNode, error) {
	tok := p.current()
	switch tok.typ {
	case tokVarExpr:
		p.advance()
		return &varExprNode{inner: tok.value[2 : len(tok.value)-1]}, nil
	case tokUrlExpr:
		p.advance()
		return &urlExprNode{inner: tok.value[2 : len(tok.value)-1]}, nil
	case tokString:
		p.advance()
		return &literalNode{value: tok.value}, nil
	case tokNumber:
		p.advance()
		if strings.Contains(tok.value, ".") {
			f, _ := strconv.ParseFloat(tok.value, 64)
			return &literalNode{value: f}, nil
		}
		i, _ := strconv.Atoi(tok.value)
		return &literalNode{value: i}, nil
	case tokTrue:
		p.advance()
		return &literalNode{value: true}, nil
	case tokFalse:
		p.advance()
		return &literalNode{value: false}, nil
	case tokNull:
		p.advance()
		return &literalNode{value: nil}, nil
	case tokIdent:
		p.advance()
		return &identNode{name: tok.value}, nil
	case tokHash:
		p.advance()
		return &hashNode{name: tok.value}, nil
	case tokLParen:
		p.advance()
		node, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.current().typ != tokRParen {
			return nil, fmt.Errorf("expected ')'")
		}
		p.advance()
		return node, nil
	case tokNew:
		// Parse: new <dotted.class.Name>(args)
		// Currently only java.util.Date is supported (returns current time).
		p.advance()
		var classNameParts []string
		for {
			if p.current().typ != tokIdent {
				return nil, fmt.Errorf("expected class name after 'new'")
			}
			classNameParts = append(classNameParts, p.advance().value)
			if p.current().typ == tokDot {
				p.advance()
				continue
			}
			break
		}
		className := strings.Join(classNameParts, ".")
		// Parse constructor args (if any)
		if p.current().typ == tokLParen {
			p.advance()
			// Skip args - we don't need them for java.util.Date
			depth := 1
			for p.current().typ != tokEOF && depth > 0 {
				if p.current().typ == tokLParen {
					depth++
				} else if p.current().typ == tokRParen {
					depth--
				}
				if depth > 0 {
					p.advance()
				}
			}
			if p.current().typ != tokRParen {
				return nil, fmt.Errorf("expected ')' after constructor args")
			}
			p.advance()
		}
		return &newExprNode{className: className}, nil
	case tokLBrace:
		// Map literal: {key: value, key2: value2}
		// Also handles set literal: {val1, val2} (stored as map with stringified vals as keys)
		p.advance()
		var pairs []mapPair
		if p.current().typ == tokRBrace {
			p.advance()
			return &mapLiteralNode{pairs: pairs}, nil
		}
		for {
			keyNode, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			var valNode exprNode
			if p.current().typ == tokColon {
				p.advance()
				valNode, err = p.parseExpression()
				if err != nil {
					return nil, err
				}
			} else {
				// Set-style: {a, b} → treat as map with key==value
				valNode = keyNode
			}
			pairs = append(pairs, mapPair{key: keyNode, value: valNode})
			if p.current().typ == tokComma {
				p.advance()
				continue
			}
			break
		}
		if p.current().typ != tokRBrace {
			return nil, fmt.Errorf("expected '}' in map literal")
		}
		p.advance()
		return &mapLiteralNode{pairs: pairs}, nil
	case tokEOF:
		return nil, fmt.Errorf("unexpected end of expression")
	default:
		return nil, fmt.Errorf("unexpected token %q", tok.value)
	}
}

// ─── Complex expression evaluation ───
// Handles expressions like: ${a} and ${b}, ${expr} != '', etc.

func evalComplexExpr(expr string, ctx *Context) (any, error) {
	// Use the tokenizer to parse the full expression
	t := newTokenizer(expr)
	tokens, err := t.tokenize()
	if err != nil {
		return nil, fmt.Errorf("tokenize error in %q: %w", expr, err)
	}
	p := &exprParser{tokens: tokens}
	node, err := p.parseExpression()
	if err != nil {
		return nil, fmt.Errorf("parse error in %q: %w", expr, err)
	}
	if p.current().typ != tokEOF {
		return nil, fmt.Errorf("unexpected token %q after expression %q", p.current().value, expr)
	}
	return node.eval(ctx)
}

// ─── Value helpers ───

func toBool(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != "" && val != "false" && val != "0"
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	case []any:
		return len(val) > 0
	case map[string]any:
		return len(val) > 0
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			return rv.Len() > 0
		}
		return true
	}
}

func toStr(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// toLiteralString converts a value for insertion into a literal substitution (|...|).
// Slices and maps are serialized as JSON so they can be used directly in JS expressions.
func toLiteralString(v any) string {
	if v == nil {
		return ""
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		b, err := json.Marshal(v)
		if err != nil {
			return toStr(v)
		}
		return string(b)
	default:
		return toStr(v)
	}
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface:
		return rv.IsNil()
	case reflect.Slice, reflect.Map, reflect.Chan:
		return rv.IsNil()
	}
	return false
}

func evalBinaryOp(op string, left, right any) (any, error) {
	switch op {
	case "and":
		return toBool(left) && toBool(right), nil
	case "or":
		return toBool(left) || toBool(right), nil
	case "==":
		return reflect.DeepEqual(left, right) || toStr(left) == toStr(right), nil
	case "!=":
		return !(reflect.DeepEqual(left, right) || toStr(left) == toStr(right)), nil
	case "+":
		// String concatenation
		if isStringLike(left) || isStringLike(right) {
			return toStr(left) + toStr(right), nil
		}
		// Numeric addition
		ln, lok := toFloat(left)
		rn, rok := toFloat(right)
		if lok && rok {
			return ln + rn, nil
		}
		return toStr(left) + toStr(right), nil
	case "-":
		ln, lok := toFloat(left)
		rn, rok := toFloat(right)
		if lok && rok {
			return ln - rn, nil
		}
		// Gracefully treat nil/missing values as 0 (matches template engines' lenient
		// behavior for expressions like contributors.size()-1 when contributors is absent).
		if !lok {
			ln = 0
		}
		if !rok {
			rn = 0
		}
		return ln - rn, nil
	case "*":
		ln, _ := toFloat(left)
		rn, _ := toFloat(right)
		return ln * rn, nil
	case "/":
		ln, _ := toFloat(left)
		rn, _ := toFloat(right)
		if rn == 0 {
			return 0, nil
		}
		return ln / rn, nil
	case ">", "<", ">=", "<=":
		ln, _ := toFloat(left)
		rn, _ := toFloat(right)
		switch op {
		case ">":
			return ln > rn, nil
		case "<":
			return ln < rn, nil
		case ">=":
			return ln >= rn, nil
		case "<=":
			return ln <= rn, nil
		}
	}
	return nil, fmt.Errorf("unknown operator %q", op)
}

func isStringLike(v any) bool {
	switch v.(type) {
	case string:
		return true
	}
	return false
}

func toFloat(v any) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float64:
		return val, true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	case bool:
		if val {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// getProperty accesses a property on a value (map, struct, etc.)
func getProperty(target any, name string) (any, error) {
	if target == nil {
		return nil, nil
	}

	rv := reflect.ValueOf(target)
	isPtr := rv.Kind() == reflect.Ptr
	if isPtr {
		if rv.IsNil() {
			return nil, nil
		}
	}

	// For maps, dereference if needed
	mapRv := rv
	if isPtr {
		mapRv = rv.Elem()
	}
	if mapRv.Kind() == reflect.Map {
		if mapVal := mapRv.MapIndex(reflect.ValueOf(name)); mapVal.IsValid() {
			val := mapVal.Interface()
			if DebugMode {
				log.Printf("[DEBUG] getProperty(map, %q) → %v(type:%T)", name, val, val)
			}
			return val, nil
		}
		if DebugMode {
			log.Printf("[DEBUG] getProperty(map, %q) → nil (key not found)", name)
		}
		return nil, nil
	}

	// For structs, try field then method (getter)
	structRv := mapRv
	if structRv.Kind() == reflect.Struct {
		// Try struct field (case-insensitive: camelCase → PascalCase)
		field := findField(structRv, name)
		if field.IsValid() && field.CanInterface() {
			val := field.Interface()
			if DebugMode {
				log.Printf("[DEBUG] getProperty(struct field, %q) → %v(type:%T)", name, val, val)
			}
			return val, nil
		}
		// Try method on pointer first (pointer-receiver methods)
		if isPtr {
			method := findMethod(rv, name)
			if method.IsValid() {
				results := method.Call(nil)
				if len(results) > 0 {
					val := results[0].Interface()
					if DebugMode {
						log.Printf("[DEBUG] getProperty(ptr-method, %q on %T) → %v(type:%T)", name, target, val, val)
					}
					return val, nil
				}
				if DebugMode {
					log.Printf("[DEBUG] getProperty(ptr-method, %q on %T) → nil (no results)", name, target)
				}
				return nil, nil
			}
		}
		// Try method on struct value (value-receiver methods)
		method := findMethod(structRv, name)
		if method.IsValid() {
			results := method.Call(nil)
			if len(results) > 0 {
				val := results[0].Interface()
				if DebugMode {
					log.Printf("[DEBUG] getProperty(value-method, %q on %T) → %v(type:%T)", name, target, val, val)
				}
				return val, nil
			}
			if DebugMode {
				log.Printf("[DEBUG] getProperty(value-method, %q on %T) → nil (no results)", name, target)
			}
			return nil, nil
		}
		if DebugMode {
			log.Printf("[DEBUG] getProperty(struct, %q on %T) → nil (no field/method)", name, target)
		}
		return nil, nil
	}

	// For slices/arrays
	if mapRv.Kind() == reflect.Slice || mapRv.Kind() == reflect.Array {
		if name == "length" || name == "size" || name == "len" {
			return mapRv.Len(), nil
		}
	}

	// Fallback: try method on pointer or value
	if isPtr {
		method := findMethod(rv, name)
		if method.IsValid() && method.Type().NumIn() == 0 {
			results := method.Call(nil)
			if len(results) > 0 {
				return results[0].Interface(), nil
			}
		}
	}
	return nil, nil
}

// findMethod finds a method on a reflect.Value, trying exact name first,
// then the name with the first letter uppercased (camelCase → PascalCase).
// This allows Thymeleaf templates to call e.g. isEmpty() which maps to Go's IsEmpty().
func findMethod(rv reflect.Value, name string) reflect.Value {
	// Try exact name first
	method := rv.MethodByName(name)
	if method.IsValid() {
		return method
	}
	// Try with first letter uppercased (camelCase → PascalCase)
	if len(name) > 0 {
		upperName := strings.ToUpper(name[:1]) + name[1:]
		method = rv.MethodByName(upperName)
		if method.IsValid() {
			return method
		}
	}
	return reflect.Value{}
}

// findField finds a struct field by name, trying exact name first,
// then the name with the first letter uppercased (camelCase → PascalCase).
// This allows Thymeleaf templates to access e.g. hasNext which maps to Go's HasNext.
func findField(rv reflect.Value, name string) reflect.Value {
	// Try exact name first
	field := rv.FieldByName(name)
	if field.IsValid() {
		return field
	}
	// Try with first letter uppercased (camelCase → PascalCase)
	if len(name) > 0 {
		upperName := strings.ToUpper(name[:1]) + name[1:]
		field = rv.FieldByName(upperName)
		if field.IsValid() {
			return field
		}
	}
	return reflect.Value{}
}

// callMethod calls a method on a value
func callMethod(target any, name string, args []any) (any, error) {
	if target == nil {
		if DebugMode {
			log.Printf("[DEBUG] callMethod(nil, %q, %v) → nil", name, args)
		}
		return nil, nil
	}

	rv := reflect.ValueOf(target)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			if DebugMode {
				log.Printf("[DEBUG] callMethod(nil-ptr, %q, %v) → nil", name, args)
			}
			return nil, nil
		}
	}

	// Support pseudo-methods size()/length()/len() on slices/arrays/maps/strings,
	// which Thymeleaf templates commonly use (e.g. list.size()-1).
	if len(args) == 0 && (name == "size" || name == "length" || name == "len") {
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
			return rv.Len(), nil
		}
	}

	// Try method on the value as-is (works for both value and pointer receivers
	// when rv is a pointer, and for value receivers when rv is a struct)
	method := findMethod(rv, name)
	if !method.IsValid() && rv.Kind() == reflect.Ptr {
		// Try method on the dereferenced struct (value receiver methods)
		method = findMethod(rv.Elem(), name)
	}
	if !method.IsValid() {
		if DebugMode {
			log.Printf("[DEBUG] callMethod(%T, %q, %v) → nil (method not found)", target, name, args)
		}
		return nil, nil
	}

	mt := method.Type()
	if mt.NumIn() != len(args) {
		// Try variadic
		if !mt.IsVariadic() || mt.NumIn()-1 > len(args) {
			if DebugMode {
				log.Printf("[DEBUG] callMethod(%T, %q, %v) → nil (arg count mismatch: %d in vs %d expected)", target, name, args, len(args), mt.NumIn())
			}
			return nil, nil
		}
	}

	callArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		callArgs[i] = reflect.ValueOf(arg)
		if !callArgs[i].IsValid() {
			callArgs[i] = reflect.Zero(mt.In(i))
		}
	}

	// Recover from panics during method call
	var result any
	var panicked bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				if DebugMode {
					log.Printf("[DEBUG] callMethod(%T, %q, %v) → PANIC: %v", target, name, args, r)
				}
			}
		}()
		results := method.Call(callArgs)
		if len(results) > 0 {
			result = results[0].Interface()
		}
	}()

	if DebugMode && !panicked {
		log.Printf("[DEBUG] callMethod(%T, %q, %v) → %v(type:%T)", target, name, args, result, result)
	}
	return result, nil
}

// getIndex accesses an index on a slice/array/map
func getIndex(target any, idx any) (any, error) {
	if target == nil {
		return nil, nil
	}

	rv := reflect.ValueOf(target)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		i, ok := toFloat(idx)
		if !ok {
			return nil, nil
		}
		index := int(i)
		if index < 0 || index >= rv.Len() {
			return nil, nil
		}
		return rv.Index(index).Interface(), nil
	case reflect.Map:
		key := reflect.ValueOf(idx)
		if mapVal := rv.MapIndex(key); mapVal.IsValid() {
			return mapVal.Interface(), nil
		}
		return nil, nil
	}
	return nil, nil
}
