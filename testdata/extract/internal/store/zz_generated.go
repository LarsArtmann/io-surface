package store

import "os"

// GeneratedCode simulates machine-written code: templ/protobuf generators
// re-create this file, so hand-declaring its I/O would be inventory noise.
func GeneratedSave(path string) error {
	return os.WriteFile(path, []byte("gen"), 0o600)
}
