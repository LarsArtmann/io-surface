package manifest

import "errors"

var (
	// ErrNotInitialized means the manifest directory does not exist.
	ErrNotInitialized = errors.New("no io-surface manifest")
	// ErrEmpty means the package loaded but declares no sites.
	ErrEmpty = errors.New("manifest declares no sites")
	// ErrConstraints means a disciplines.cue rule failed: the violations
	// list is non-empty (or conflicts with a declared default). Load reports
	// the actual id/msg entries alongside this sentinel.
	ErrConstraints = errors.New("manifest constraints failed")
)
