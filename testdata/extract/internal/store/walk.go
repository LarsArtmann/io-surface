package store

import (
	"io/fs"
	"path/filepath"
)

func WalkFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			out = append(out, path)
		}

		return nil
	})

	return out, err
}
