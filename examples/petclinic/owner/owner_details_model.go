package owner

import "github.com/StevenBuglione/spice/examples/petclinic/presentation"

// OwnerDetailsModel renders one complete owner aggregate.
type OwnerDetailsModel struct {
	presentation.Page
	Owner Owner
}
