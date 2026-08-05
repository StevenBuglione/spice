package owner

import "github.com/spice-framework/spice/examples/petclinic/presentation"

// OwnersListModel renders one deterministic owner result page.
type OwnersListModel struct {
	presentation.Page
	Owners       []Owner
	LastName     string
	CurrentPage  int
	TotalPages   int
	TotalItems   int
	PreviousPage int
	NextPage     int
	HasPrevious  bool
	HasNext      bool
}
