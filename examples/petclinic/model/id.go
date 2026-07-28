package model

// ID is a persistent Petclinic entity identity.
type ID int64

// Valid reports whether the identity denotes a persisted entity.
func (id ID) Valid() bool {
	return id > 0
}
