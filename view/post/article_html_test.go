package post

import (
	"strings"
	"testing"
)

func TestWrapTables(t *testing.T) {
	input := `<h2>Comparison</h2><table><thead><tr><th>A</th><th>B</th></tr></thead><tbody><tr><td>1</td><td>2</td></tr></tbody></table><p>Done</p>`
	got := wrapTables(input)
	wantPrefix := `<h2>Comparison</h2><div tabindex="0" role="region" aria-label="Scrollable data table" class="table-wrapper"><table>`
	wantSuffix := `</table></div><p>Done</p>`
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("expected prefix %q, got %q", wantPrefix, got)
	}
	if !strings.HasSuffix(got, wantSuffix) {
		t.Fatalf("expected suffix %q, got %q", wantSuffix, got)
	}
}
