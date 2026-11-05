package manifest

import "errors"

var (
	// ErrNotInitialized means the manifest directory does not exist.
	ErrNotInitialized = errors.New("no io-surface manifest")
	// ErrEmpty means the package loaded but declares no sites.
	ErrEmpty = errors.New("manifest declares no sites")
)
