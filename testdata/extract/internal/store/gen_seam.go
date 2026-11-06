package store

import "os"

// genWrite is a repo-specific seam: the raw call lives here, callers get
// via=genWrite attribution only once the manifest declares it.
func genWrite(path string) error {
	return os.WriteFile(path, []byte("x"), 0o600)
}

func callGenWrite(path string) error {
	return genWrite(path)
}
