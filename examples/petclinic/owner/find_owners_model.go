package owner

import (
	"github.com/StevenBuglione/spice/examples/petclinic/presentation"
	"github.com/StevenBuglione/spice/validation"
)

// FindOwnersModel renders owner search input and failures.
type FindOwnersModel struct {
	presentation.Page
	LastName string
	Errors   []validation.Violation
}
