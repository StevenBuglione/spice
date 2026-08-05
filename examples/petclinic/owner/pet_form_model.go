package owner

import (
	"github.com/spice-framework/spice/examples/petclinic/presentation"
	"github.com/spice-framework/spice/validation"
)

// PetFormModel is the immutable create/edit pet view model.
type PetFormModel struct {
	presentation.Page
	Owner    Owner
	Pet      Pet
	PetTypes []PetType
	Errors   []validation.Violation
	Creating bool
}
