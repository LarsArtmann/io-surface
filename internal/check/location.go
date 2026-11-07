package check

import (
	"os"
	"path/filepath"
	"strings"
)

// symbolVisible reports whether any declared location's file exists and still
// mentions the symbol name. It is a cheap, deliberately loose heuristic used
// only to classify a non-extracting declared site: a file that no longer
// contains the name is a lying claim, a file that does is an extractor blind
// spot.
func symbolVisible(root string, locations []string, symbol string) bool {
	if symbol == "" || len(locations) == 0 {
		return false
	}
	// Declared locations of one site usually share a file; reading each
	// unique path once keeps the loop from re-reading it per location.
	read := map[string][]byte{}
	for _, loc := range locations {
		path, _, ok := splitLocation(loc)
		if !ok {
			continue
		}
		src, cached := read[path]
		if !cached {
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
			if err != nil {
				read[path] = nil

				continue
			}
			read[path] = data
			src = data
		}
		if src != nil && strings.Contains(string(src), symbol) {
			return true
		}
	}

	return false
}

// splitLocation splits "internal/x/y.go:123" (also "y.go:123-140") into path
// and line.
func splitLocation(loc string) (string, string, bool) {
	i := strings.LastIndex(loc, ":")
	if i <= 0 || i == len(loc)-1 {
		return "", "", false
	}

	return loc[:i], loc[i+1:], true
}
