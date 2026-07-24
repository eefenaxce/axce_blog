package transport

import (
	_ "embed"
	"strings"
)

//go:embed halo_widgets.js
var haloWidgetsJS string

// injectHaloWidgets inserts the Halo-compatible comment and search widget
// scripts into the rendered HTML. Halo themes rely on the <halo-comment>
// custom element and window.SearchWidget; without these scripts the comment
// area shows "closed" and the search button does nothing.
func injectHaloWidgets(html string) string {
	if !strings.Contains(html, "</head>") && !strings.Contains(html, "</HEAD>") {
		return html
	}

	script := "<script>" + haloWidgetsJS + "</script>"

	// Place the script early in <head> so the custom element is registered
	// before any body scripts try to query halo-comment shadow roots.
	headRe := strings.NewReplacer("</head>", script+"\n</head>", "</HEAD>", script+"\n</HEAD>")
	html = headRe.Replace(html)

	return html
}
