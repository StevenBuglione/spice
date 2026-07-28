package vet

// ListModel is the immutable veterinarian list view model.
type ListModel struct {
	Vets         []Vet
	CurrentPage  int
	TotalPages   int
	TotalItems   int
	PreviousPage int
	NextPage     int
	HasPrevious  bool
	HasNext      bool
}
