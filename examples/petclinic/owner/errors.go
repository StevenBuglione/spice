package owner

import "errors"

// ErrOwnerNotFound reports a missing owner identity.
var ErrOwnerNotFound = errors.New("owner not found")

// ErrPetNotFound reports a pet identity outside its requested owner.
var ErrPetNotFound = errors.New("pet not found")
