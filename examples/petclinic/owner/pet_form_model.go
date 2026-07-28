package owner

import "github.com/StevenBuglione/spice/validation"

// PetFormModel is the immutable create/edit pet view model.
type PetFormModel struct {
	Owner    Owner
	Pet      Pet
	PetTypes []PetType
	Errors   []validation.Violation
	Creating bool
}
