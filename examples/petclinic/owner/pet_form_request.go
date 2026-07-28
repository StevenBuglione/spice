package owner

// PetFormRequest binds a new pet and its protected owner identity.
type PetFormRequest struct {
	OwnerID   int    `path:"ownerId"`
	Name      string `form:"name,required"`
	BirthDate string `form:"birthDate,required"`
	TypeID    int    `form:"type,required"`
}
