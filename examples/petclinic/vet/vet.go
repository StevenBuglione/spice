package vet

import (
	"slices"
	"strings"

	"github.com/StevenBuglione/spice/examples/petclinic/model"
	"github.com/StevenBuglione/spice/validation"
)

// Vet is a veterinarian and their ordered specialties.
type Vet struct {
	model.Person
	Specialties []Specialty
}

// Clone returns a deep value copy.
func (vet *Vet) Clone() Vet {
	if vet == nil {
		return Vet{}
	}
	result := *vet
	result.Specialties = slices.Clone(vet.Specialties)
	return result
}

// AddSpecialty adds unique reference data in display order.
func (vet *Vet) AddSpecialty(specialty Specialty) {
	if vet == nil {
		return
	}
	for _, existing := range vet.Specialties {
		if existing.ID == specialty.ID {
			return
		}
	}
	vet.Specialties = append(vet.Specialties, specialty)
	slices.SortStableFunc(vet.Specialties, func(left, right Specialty) int {
		return strings.Compare(
			strings.ToLower(left.Name),
			strings.ToLower(right.Name),
		)
	})
}

// Validate applies the shared person constraints.
func (vet *Vet) Validate() (validation.Errors, error) {
	if vet == nil {
		return validation.New(validation.Violation{
			Code:    "required",
			Message: "veterinarian must be provided",
		})
	}
	return vet.Person.Validate()
}
