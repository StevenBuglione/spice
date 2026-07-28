package owner

import "github.com/StevenBuglione/spice/validation"

// OwnerFormModel renders create and edit forms.
type OwnerFormModel struct {
	Owner    Owner
	Errors   []validation.Violation
	Creating bool
}
