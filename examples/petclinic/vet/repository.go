package vet

import "context"

// Repository lists veterinarians in stable display order.
type Repository interface {
	FindAll(context.Context) ([]Vet, error)
}
