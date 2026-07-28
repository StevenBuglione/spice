package model

// NamedEntity is the shared shape for reference data such as pet types and
// veterinary specialties.
type NamedEntity struct {
	BaseEntity
	Name string `json:"name"`
}
