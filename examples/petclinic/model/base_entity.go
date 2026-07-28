// Package model contains the shared Petclinic domain building blocks.
package model

// BaseEntity gives persistent Petclinic entities a stable identity. A zero ID
// identifies an entity that has not been persisted.
type BaseEntity struct {
	ID ID
}

// New reports whether the entity has not yet been persisted.
func (entity BaseEntity) New() bool {
	return entity.ID == 0
}
