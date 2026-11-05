package scaffold

import "errors"

// ErrExists means a manifest package is already present.
var ErrExists = errors.New("manifest already exists")
