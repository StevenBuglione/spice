package owner

import (
	"github.com/StevenBuglione/spice/examples/petclinic/presentation"
	"github.com/StevenBuglione/spice/validation"
)

// OwnerFormModel renders create and edit forms.
type OwnerFormModel struct {
	presentation.Page
	Owner    Owner
	Errors   []validation.Violation
	Creating bool
}
