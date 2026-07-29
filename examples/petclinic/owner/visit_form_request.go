package owner

// VisitFormRequest binds a new visit and protected aggregate identities.
type VisitFormRequest struct {
	OwnerID     int    `path:"ownerId"`
	PetID       int    `path:"petId"`
	Language    string `header:"Accept-Language"`
	Date        string `form:"date,required"`
	Description string `form:"description,required"`
}
