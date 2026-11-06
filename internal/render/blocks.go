package render

import (
	"fmt"
	"strings"

	"github.com/larsartmann/io-surface/internal/manifest"
)

// Doc markers delimit a generated inventory block inside a hand-written
// markdown document. Everything between the markers is owned by the renderer;
// the surrounding prose stays human-authored and is never touched. HTML
// comments are invisible in rendered markdown, so the block leaves no visual
// trace in the published document.
const (
	BeginMarker = "<!-- io-surface:begin -->"
	EndMarker   = "<!-- io-surface:end -->"
)

// Block renders the generated content for embedding between the doc markers:
// the same sections Markdown produces, without the standalone title.
func Block(m *manifest.Manifest) string {
	return sections(m)
}

// ApplyToDoc replaces the content between the doc markers in doc with a fresh
// render of m and returns the updated document. The prose outside the markers
// is preserved byte-for-byte. An absent, unbalanced, or duplicated marker
// pair is an error: guessing where generated content belongs is how
// generators eat hand-written documents.
func ApplyToDoc(doc string, m *manifest.Manifest) (string, error) {
	start, end, err := docBlock(doc)
	if err != nil {
		return "", err
	}

	return doc[:start] + BeginMarker + "\n\n" + Block(m) + doc[end:], nil
}

// CheckDoc reports whether the generated block in doc matches a fresh render
// of m, returning the stored block and the fresh block for diagnostics.
func CheckDoc(doc string, m *manifest.Manifest) (stored, fresh string, err error) {
	start, end, err := docBlock(doc)
	if err != nil {
		return "", "", err
	}
	stored = strings.Trim(doc[start+len(BeginMarker):end], "\n")
	fresh = strings.Trim(Block(m), "\n")

	return stored, fresh, nil
}

// docBlock locates the marker pair and returns the offsets of BeginMarker
// start and EndMarker start in doc.
func docBlock(doc string) (start, end int, err error) {
	start = strings.Index(doc, BeginMarker)
	if start < 0 {
		return 0, 0, fmt.Errorf("no %s marker: wrap the generated inventory in %s/%s", BeginMarker, BeginMarker, EndMarker)
	}
	end = strings.Index(doc[start:], EndMarker)
	if end < 0 {
		return 0, 0, fmt.Errorf("unbalanced block: %s without %s", BeginMarker, EndMarker)
	}
	end += start
	if strings.Contains(doc[end+len(EndMarker):], BeginMarker) {
		return 0, 0, fmt.Errorf("multiple %s blocks: one document carries one inventory", BeginMarker)
	}

	return start, end, nil
}
