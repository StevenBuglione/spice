package owner

// NewVisitRequest binds the owner and pet identities for a visit form.
type NewVisitRequest struct {
	OwnerID  int    `path:"ownerId"`
	PetID    int    `path:"petId"`
	Language string `header:"Accept-Language"`
}
