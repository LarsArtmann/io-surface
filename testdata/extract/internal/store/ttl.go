package store

import "os"

type ttl struct{ path string }

// Sweep reads the ttl file: a method on an UNEXPORTED type. Its ID
// (internal/store.ttl.Sweep) is dotted but legal — not an id-form error.
func (t ttl) Sweep() error {
	_, err := os.ReadFile(t.path)

	return err
}
