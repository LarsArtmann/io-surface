package render

import "github.com/larsartmann/go-error-family"

// codeDocBlock anchors every marker-structure failure (missing, unbalanced,
// duplicated); the sentinel and the site-level rejections share it so
// errors.Is matches by code+family.
const codeDocBlock = "render.doc-block"

// ErrDocBlock means the document's io-surface:begin/end marker structure is
// not exactly one balanced pair: missing, unbalanced, or duplicated. The
// renderer refuses to guess where generated content belongs.
var ErrDocBlock error = errorfamily.NewRejection(codeDocBlock, "malformed doc block markers")
