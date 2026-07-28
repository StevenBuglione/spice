package owner

import "errors"

// ErrOwnerNotFound reports a missing owner identity.
var ErrOwnerNotFound = errors.New("owner not found")
