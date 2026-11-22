package render

import "errors"

// ErrDocBlock means the document's io-surface:begin/end marker structure is
// not exactly one balanced pair: missing, unbalanced, or duplicated. The
// renderer refuses to guess where generated content belongs.
var ErrDocBlock = errors.New("malformed doc block markers")
