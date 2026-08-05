package vet

import "github.com/spice-framework/spice/examples/petclinic/presentation"

// ListModel is the immutable veterinarian list view model.
type ListModel struct {
	presentation.Page
	Vets         []Vet
	CurrentPage  int
	TotalPages   int
	TotalItems   int
	PreviousPage int
	NextPage     int
	HasPrevious  bool
	HasNext      bool
}
