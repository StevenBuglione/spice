package owner

import (
	"github.com/spice-framework/spice/examples/petclinic/presentation"
	"github.com/spice-framework/spice/validation"
)

// FindOwnersModel renders owner search input and failures.
type FindOwnersModel struct {
	presentation.Page
	LastName string
	Errors   []validation.Violation
}
