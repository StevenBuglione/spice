package owner

// OwnersListModel renders one deterministic owner result page.
type OwnersListModel struct {
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
