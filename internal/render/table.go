package render

import (
	"strings"
)

// table accumulates a GFM table and renders it with formatter-canonical
// aligned columns. Formatters such as dprint and prettier align markdown
// tables (pipe padding, separator dashes sized to the column) on every pass,
// so a renderer that emits compact tables fights the formatter: the generated
// block drifts on the next formatter run and the render --check gate fails on
// a doc nobody hand-edited. Emitting the aligned shape directly makes the
// generated block a formatter no-op.
type table struct {
	headers []string
	right   []bool
	rows    [][]string
}

func newTable(headers []string, right []bool) *table {
	return &table{headers: headers, right: right}
}

// row appends one body row; short rows are padded with empty cells.
func (t *table) row(cells ...string) {
	for len(cells) < len(t.headers) {
		cells = append(cells, "")
	}
	t.rows = append(t.rows, cells)
}

// render draws the header row, the alignment separator, and the body with
// every column padded to its display width. A column is at least three dashes
// wide so it always parses as an alignment row; right-aligned columns carry
// the trailing colon and right-align their cells, exactly the shape dprint
// and prettier produce.
func (t *table) render() string {
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = max(displayWidth(h), 3)
	}
	for _, r := range t.rows {
		for i, c := range r {
			widths[i] = max(widths[i], displayWidth(c))
		}
	}

	var b strings.Builder
	b.WriteString("|")
	for i, h := range t.headers {
		b.WriteString(" " + pad(h, widths[i], t.right[i]) + " |")
	}
	b.WriteString("\n|")
	for i := range t.headers {
		sep := strings.Repeat("-", widths[i])
		if t.right[i] {
			sep = sep[:len(sep)-1] + ":"
		}
		b.WriteString(" " + sep + " |")
	}
	b.WriteString("\n")
	for _, r := range t.rows {
		b.WriteString("|")
		for i, c := range r {
			b.WriteString(" " + pad(c, widths[i], t.right[i]) + " |")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// pad right- or left-fills a cell to the column's display width.
func pad(s string, width int, right bool) string {
	spaces := width - displayWidth(s)
	if spaces <= 0 {
		return s
	}
	if right {
		return strings.Repeat(" ", spaces) + s
	}

	return s + strings.Repeat(" ", spaces)
}

// displayWidth is the terminal column count of s: one per rune, two for East
// Asian wide and fullwidth runes, so claims prose in any script keeps the
// columns aligned the way a formatter would lay them out.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}

	return w
}

// runeWidth reports the terminal columns of one rune, two for the Unicode
// East Asian Width wide/fullwidth ranges. Everything else, including the
// em-dash placeholder and CJK punctuation look-alikes that are narrow, is one
// column.
func runeWidth(r rune) int {
	switch {
	case r >= 0x1100 && r <= 0x115F, // Hangul Jamo
		r >= 0x2E80 && r <= 0x303E, // CJK radicals, Kangxi, CJK symbols
		r >= 0x3041 && r <= 0x33FF, // Hiragana through CJK compatibility
		r >= 0x3400 && r <= 0x4DBF, // CJK extension A
		r >= 0x4E00 && r <= 0x9FFF, // CJK unified ideographs
		r >= 0xA000 && r <= 0xA4CF, // Yi syllables and radicals
		r >= 0xAC00 && r <= 0xD7A3, // Hangul syllables
		r >= 0xF900 && r <= 0xFAFF, // CJK compatibility ideographs
		r >= 0xFE30 && r <= 0xFE4F, // CJK compatibility forms
		r >= 0xFF00 && r <= 0xFF60, // fullwidth forms
		r >= 0xFFE0 && r <= 0xFFE6, // fullwidth signs
		r >= 0x20000 && r <= 0x3FFFD: // extensions B and beyond
		return 2
	default:
		return 1
	}
}
