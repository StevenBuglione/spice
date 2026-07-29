package owner

// NewPetRequest binds the owner identity for a new pet form.
type NewPetRequest struct {
	OwnerID  int    `path:"ownerId"`
	Language string `header:"Accept-Language"`
}
