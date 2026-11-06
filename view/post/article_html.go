package post

import "strings"

// wrapTables wraps top-level HTML <table> elements in a keyboard-scrollable
// region container so wide Markdown tables scroll horizontally within the
// article instead of overflowing the page viewport.
func wrapTables(rawHTML string) string {
	if !strings.Contains(rawHTML, "<table") {
		return rawHTML
	}
	withOpen := strings.ReplaceAll(
		rawHTML,
		"<table>",
		`<div tabindex="0" role="region" aria-label="Scrollable data table" class="table-wrapper"><table>`,
	)
	return strings.ReplaceAll(withOpen, "</table>", "</table></div>")
}
