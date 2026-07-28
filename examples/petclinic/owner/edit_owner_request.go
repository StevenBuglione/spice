package owner

// EditOwnerRequest binds a protected path identity and editable fields.
type EditOwnerRequest struct {
	OwnerID   int    `path:"ownerId"`
	FirstName string `form:"firstName,required"`
	LastName  string `form:"lastName,required"`
	Address   string `form:"address,required"`
	City      string `form:"city,required"`
	Telephone string `form:"telephone,required"`
}

// Apply updates editable fields while preserving identity and pets.
func (request EditOwnerRequest) Apply(value *Owner) {
	if value == nil {
		return
	}
	value.FirstName = request.FirstName
	value.LastName = request.LastName
	value.Address = request.Address
	value.City = request.City
	value.Telephone = request.Telephone
}
