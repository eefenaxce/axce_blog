package thymeleaf

import (
	"strings"

	"golang.org/x/net/html"
)

// renderNode renders an HTML node to a string builder.
func renderNode(sb *strings.Builder, n *html.Node) {
	if n == nil {
		return
	}
	switch n.Type {
	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			renderNode(sb, c)
		}
	case html.ElementNode:
		renderElement(sb, n)
	case html.TextNode:
		sb.WriteString(n.Data)
	case html.CommentNode:
		// Skip comments
	case html.DoctypeNode:
		sb.WriteString("<!doctype html>")
	case html.RawNode:
		sb.WriteString(n.Data)
	}
}

func renderElement(sb *strings.Builder, n *html.Node) {
	// th:block is a synthetic element - render children only.
	// Also handle <template data-th-block> which was converted from <th:block>
	// during pre-processing to prevent HTML5 parser from closing <head>.
	if n.Data == "th:block" || n.Data == "th:block/" {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			renderNode(sb, c)
		}
		return
	}
	if n.Data == "template" {
		if _, ok := getAttr(n, "data-th-block"); ok {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				renderNode(sb, c)
			}
			return
		}
	}

	sb.WriteByte('<')
	sb.WriteString(n.Data)

	for _, attr := range n.Attr {
		// Skip th:* attributes, xmlns:th, and data-th-block (synthetic marker)
		if strings.HasPrefix(attr.Key, "th:") || attr.Key == "xmlns:th" || attr.Key == "data-th-block" {
			continue
		}
		sb.WriteByte(' ')
		sb.WriteString(attr.Key)
		if attr.Val != "" {
			sb.WriteString(`="`)
			val := attr.Val
			// URL attributes may already contain HTML entities (e.g. user pasted &amp; in config).
			// Decode first, then escape once, to avoid double encoding like &amp;amp;.
			if isURLAttribute(attr.Key) {
				val = html.UnescapeString(val)
			}
			sb.WriteString(html.EscapeString(val))
			sb.WriteString(`"`)
		} else if attr.Key != "" {
			// Valueless attribute
		}
	}

	// Self-closing tags
	if isVoidElement(n.Data) && n.FirstChild == nil {
		sb.WriteString(">")
		return
	}

	sb.WriteString(">")

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		renderNode(sb, c)
	}

	sb.WriteString("</")
	sb.WriteString(n.Data)
	sb.WriteString(">")
}

var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true,
	"embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "param": true, "source": true,
	"track": true, "wbr": true,
}

func isVoidElement(tag string) bool {
	return voidElements[tag]
}

// isURLAttribute reports whether the attribute value is a URL reference.
func isURLAttribute(key string) bool {
	switch strings.ToLower(key) {
	case "src", "href", "data-src", "srcset", "action", "cite", "poster", "formaction", "longdesc", "profile", "usemap":
		return true
	}
	return false
}

// getAttr returns the value of a th: attribute or a regular attribute.
func getAttr(n *html.Node, key string) (string, bool) {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}
	return "", false
}

// getThAttr returns the value of a th:xxx attribute.
func getThAttr(n *html.Node, suffix string) (string, bool) {
	return getAttr(n, "th:"+suffix)
}

// removeAttr removes an attribute from a node.
func removeAttr(n *html.Node, key string) {
	var filtered []html.Attribute
	for _, attr := range n.Attr {
		if attr.Key != key {
			filtered = append(filtered, attr)
		}
	}
	n.Attr = filtered
}

// setAttr sets or replaces an attribute value.
func setAttr(n *html.Node, key, val string) {
	for i, attr := range n.Attr {
		if attr.Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

// cloneNode creates a deep copy of an HTML node (without parent).
func cloneNode(n *html.Node) *html.Node {
	if n == nil {
		return nil
	}
	clone := &html.Node{
		Type: n.Type,
		Data: n.Data,
		Attr: append([]html.Attribute{}, n.Attr...),
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		clone.AppendChild(cloneNode(c))
	}
	return clone
}
