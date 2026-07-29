package owner

import (
	"github.com/StevenBuglione/spice/examples/petclinic/presentation"
	"github.com/StevenBuglione/spice/validation"
)

// VisitFormModel is the immutable new-visit view model.
type VisitFormModel struct {
	presentation.Page
	Owner  Owner
	Pet    Pet
	Visit  Visit
	Errors []validation.Violation
}
