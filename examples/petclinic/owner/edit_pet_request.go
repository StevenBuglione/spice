package owner

// EditPetRequest binds an existing pet and its editable fields.
type EditPetRequest struct {
	OwnerID   int    `path:"ownerId"`
	PetID     int    `path:"petId"`
	Name      string `form:"name,required"`
	BirthDate string `form:"birthDate,required"`
	TypeID    int    `form:"type,required"`
}
