package absorb

import "github.com/larsartmann/go-error-family"

// codeCapabilityType anchors every unreadable-Capslock-type failure; the
// sentinel and the site-level rejections share it so errors.Is matches by
// code+family.
const codeCapabilityType = "capability.type"

// ErrCapabilityType means a Capslock report entry carries a capability type
// this schema cannot read: the field is empty or newer than the pinned
// report format. Run fails hard rather than guessing a depth.
var ErrCapabilityType error = errorfamily.NewRejection(codeCapabilityType, "unreadable capability type")
