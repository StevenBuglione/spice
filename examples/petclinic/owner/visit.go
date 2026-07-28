package owner

import (
	"strings"
	"time"

	"github.com/StevenBuglione/spice/examples/petclinic/model"
	"github.com/StevenBuglione/spice/validation"
)

// Visit records veterinary care for one pet.
type Visit struct {
	model.BaseEntity
	Date        time.Time
	Description string
}

// Validate returns stable visit constraints.
func (visit Visit) Validate() (validation.Errors, error) {
	var violations []validation.Violation
	if visit.Date.IsZero() {
		violations = append(violations, validation.Violation{
			Field:   "date",
			Code:    "required",
			Message: "must be provided",
		})
	}
	if strings.TrimSpace(visit.Description) == "" {
		violations = append(violations, validation.Violation{
			Field:   "description",
			Code:    "required",
			Message: "must not be blank",
		})
	}
	return validation.New(violations...)
}

func compareVisits(left, right Visit) int {
	if compared := left.Date.Compare(right.Date); compared != 0 {
		return compared
	}
	return compareID(left.ID, right.ID)
}

func compareID(left, right model.ID) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}
