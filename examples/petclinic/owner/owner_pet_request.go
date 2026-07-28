package owner

// OwnerPetRequest binds one pet inside its owner aggregate.
type OwnerPetRequest struct {
	OwnerID int `path:"ownerId"`
	PetID   int `path:"petId"`
}
