package render

import (
	"strings"
	"testing"
)

// TestDisplayWidth pins the column math the dprint-stable renderer lives on:
// ASCII is one column, East-Asian wide runes are two, and the mixed sum is
// exact — a wrong count silently misaligns every rendered table.
func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"ascii", "os.ReadFile", 11},
		{"cjk ideographs", "読み書き", 8},
		{"mixed", "読 os", 5},
		{"fullwidth punctuation", "（ｘ）", 6},
		{"hangul jamo", "한", 2},
		{"em dash placeholder", "—", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := displayWidth(tc.in); got != tc.want {
				t.Errorf("displayWidth(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// TestPadNarrowColumn covers the min-3 separator path: a one-character header
// still yields a parseable alignment row (three dashes, or `--:` right),
// because a two-dash column is not a valid markdown table separator.
func TestPadNarrowColumn(t *testing.T) {
	rightTable := newTable([]string{"f", "family-long-header"}, []bool{true, false})
	rightTable.row("x", "v")
	if out := rightTable.render(); !strings.Contains(out, " --: |") {
		t.Errorf("right-aligned narrow column lost its `--:` separator:\n%s", out)
	}

	leftTable := newTable([]string{"f", "family-long-header"}, []bool{false, false})
	leftTable.row("x", "v")
	if out := leftTable.render(); !strings.Contains(out, " --- |") {
		t.Errorf("narrow left column did not get the min-3 dashes:\n%s", out)
	}

	padded := pad("x", 3, true)
	if padded != "  x" {
		t.Errorf("pad(x, 3, right) = %q, want %q", padded, "  x")
	}
	if got := pad("wide-cell", 3, false); got != "wide-cell" {
		t.Errorf("pad must never truncate, got %q", got)
	}
}
