package owner

import "github.com/spice-framework/spice/examples/petclinic/model"

// OwnerFormRequest binds only the editable owner fields.
type OwnerFormRequest struct {
	Language  string `header:"Accept-Language"`
	FirstName string `form:"firstName,required"`
	LastName  string `form:"lastName,required"`
	Address   string `form:"address,required"`
	City      string `form:"city,required"`
	Telephone string `form:"telephone,required"`
}

// Owner returns a new aggregate without accepting caller-owned identities.
func (request OwnerFormRequest) Owner() Owner {
	return Owner{
		Person: model.Person{
			FirstName: request.FirstName,
			LastName:  request.LastName,
		},
		Address:   request.Address,
		City:      request.City,
		Telephone: request.Telephone,
	}
}
