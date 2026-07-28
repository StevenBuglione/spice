package model

import (
	"strings"

	"github.com/StevenBuglione/spice/validation"
)

// Person is the shared name data for owners and veterinarians.
type Person struct {
	BaseEntity
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// Validate returns stable, presentation-safe person violations.
func (person Person) Validate() (validation.Errors, error) {
	var violations []validation.Violation
	violations = appendNameViolation(
		violations,
		"firstName",
		person.FirstName,
	)
	violations = appendNameViolation(
		violations,
		"lastName",
		person.LastName,
	)
	return validation.New(violations...)
}

func appendNameViolation(
	violations []validation.Violation,
	field string,
	value string,
) []validation.Violation {
	switch {
	case strings.TrimSpace(value) == "":
		return append(violations, validation.Violation{
			Field:   field,
			Code:    "required",
			Message: "must not be blank",
		})
	case len(value) > 30:
		return append(violations, validation.Violation{
			Field:   field,
			Code:    "size",
			Message: "must contain at most 30 characters",
		})
	default:
		return violations
	}
}
