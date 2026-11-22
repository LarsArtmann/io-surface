package absorb

import "errors"

// ErrCapabilityType means a Capslock report entry carries a capability type
// this schema cannot read: the field is empty or newer than the pinned
// report format. Run fails hard rather than guessing a depth.
var ErrCapabilityType = errors.New("unreadable capability type")
