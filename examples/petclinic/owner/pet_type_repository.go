package owner

import "context"

// PetTypeRepository lists the stable pet-type reference catalog.
type PetTypeRepository interface {
	FindAll(context.Context) ([]PetType, error)
}
