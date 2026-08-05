package owner

import (
	"github.com/spice-framework/spice/examples/petclinic/presentation"
	"github.com/spice-framework/spice/validation"
)

// OwnerFormModel renders create and edit forms.
type OwnerFormModel struct {
	presentation.Page
	Owner    Owner
	Errors   []validation.Violation
	Creating bool
}
