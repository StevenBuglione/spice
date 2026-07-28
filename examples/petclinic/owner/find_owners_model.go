package owner

import "github.com/StevenBuglione/spice/validation"

// FindOwnersModel renders owner search input and failures.
type FindOwnersModel struct {
	LastName string
	Errors   []validation.Violation
}
