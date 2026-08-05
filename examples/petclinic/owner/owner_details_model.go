package owner

import "github.com/spice-framework/spice/examples/petclinic/presentation"

// OwnerDetailsModel renders one complete owner aggregate.
type OwnerDetailsModel struct {
	presentation.Page
	Owner Owner
}
