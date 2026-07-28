// Package owner owns Petclinic owners, pets, visits, and pet types.
package owner

import (
	"errors"
	"slices"
	"strings"

	"github.com/StevenBuglione/spice/examples/petclinic/model"
	"github.com/StevenBuglione/spice/validation"
)

// Owner is a person responsible for zero or more pets.
type Owner struct {
	model.Person
	Address   string
	City      string
	Telephone string
	Pets      []Pet
}

// Clone returns a deep value copy suitable for repository boundaries.
func (owner *Owner) Clone() Owner {
	if owner == nil {
		return Owner{}
	}
	result := *owner
	result.Pets = clonePets(owner.Pets)
	return result
}

// Validate returns the Petclinic owner form constraints in stable field order.
func (owner *Owner) Validate() (validation.Errors, error) {
	if owner == nil {
		return validation.New(validation.Violation{
			Code:    "required",
			Message: "owner must be provided",
		})
	}
	personErrors, err := owner.Person.Validate()
	if err != nil {
		return validation.Errors{}, err
	}
	violations := personErrors.All()
	violations = appendRequired(
		violations,
		"address",
		owner.Address,
	)
	violations = appendRequired(violations, "city", owner.City)
	if !validTelephone(owner.Telephone) {
		violations = append(violations, validation.Violation{
			Field:   "telephone",
			Code:    "telephone.invalid",
			Message: "must contain exactly 10 digits",
		})
	}
	return validation.New(violations...)
}

// PetByID returns a defensive copy of one persisted pet.
func (owner *Owner) PetByID(id model.ID) (Pet, bool) {
	if owner == nil {
		return Pet{}, false
	}
	for _, pet := range owner.Pets {
		if pet.ID == id && id.Valid() {
			return pet.Clone(), true
		}
	}
	return Pet{}, false
}

// PetByName returns a case-insensitive defensive copy. New pets can be omitted
// while validating an edit against already-persisted siblings.
func (owner *Owner) PetByName(
	name string,
	ignoreNew bool,
) (Pet, bool) {
	if owner == nil {
		return Pet{}, false
	}
	for _, pet := range owner.Pets {
		if strings.EqualFold(pet.Name, name) &&
			(!ignoreNew || !pet.New()) {
			return pet.Clone(), true
		}
	}
	return Pet{}, false
}

// AddPet adds one new pet and rejects nil receivers or persisted duplicates.
func (owner *Owner) AddPet(pet Pet) error {
	if owner == nil {
		return errors.New("add pet: owner is nil")
	}
	if !pet.New() {
		return errors.New("add pet: pet is already persisted")
	}
	owner.Pets = append(owner.Pets, pet.Clone())
	slices.SortStableFunc(owner.Pets, comparePets)
	return nil
}

// AddVisit adds a visit to an existing pet.
func (owner *Owner) AddVisit(
	petID model.ID,
	visit Visit,
) error {
	if owner == nil {
		return errors.New("add visit: owner is nil")
	}
	for index := range owner.Pets {
		if owner.Pets[index].ID == petID && petID.Valid() {
			return owner.Pets[index].AddVisit(visit)
		}
	}
	return errors.New("add visit: pet was not found")
}

func appendRequired(
	violations []validation.Violation,
	field string,
	value string,
) []validation.Violation {
	if strings.TrimSpace(value) != "" {
		return violations
	}
	return append(violations, validation.Violation{
		Field:   field,
		Code:    "required",
		Message: "must not be blank",
	})
}

func validTelephone(value string) bool {
	if len(value) != 10 {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}
