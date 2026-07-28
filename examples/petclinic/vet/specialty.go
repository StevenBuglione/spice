// Package vet owns Petclinic veterinarians and specialties.
package vet

import "github.com/StevenBuglione/spice/examples/petclinic/model"

// Specialty identifies an area of veterinary expertise.
type Specialty struct {
	model.NamedEntity
}
