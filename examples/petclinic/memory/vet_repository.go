package memory

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/StevenBuglione/spice/examples/petclinic/vet"
)

// @import { Implements, Repository } from "github.com/StevenBuglione/spice/annotation/core"

// VetRepository reads memory-backed veterinarians.
//
// @Repository(constructor=NewVetRepository)
// @Implements(vet.Repository)
type VetRepository struct {
	database *Database
}

// NewVetRepository constructs a memory veterinarian repository.
func NewVetRepository(database *Database) (*VetRepository, error) {
	if database == nil {
		return nil, errors.New(
			"construct memory vet repository: database is nil",
		)
	}
	return &VetRepository{database: database}, nil
}

// FindAll returns defensive veterinarian copies in display order.
func (repository *VetRepository) FindAll(
	ctx context.Context,
) ([]vet.Vet, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	if repository == nil || repository.database == nil {
		return nil, errors.New(
			"find veterinarians: repository is nil",
		)
	}
	repository.database.mu.RLock()
	defer repository.database.mu.RUnlock()
	return cloneVets(repository.database.vets), nil
}

func cloneVets(values []vet.Vet) []vet.Vet {
	result := make([]vet.Vet, len(values))
	for index, value := range values {
		result[index] = value.Clone()
	}
	slices.SortStableFunc(result, func(left, right vet.Vet) int {
		if compared := strings.Compare(
			strings.ToLower(left.LastName),
			strings.ToLower(right.LastName),
		); compared != 0 {
			return compared
		}
		return strings.Compare(
			strings.ToLower(left.FirstName),
			strings.ToLower(right.FirstName),
		)
	})
	return result
}

func validateVets(values []vet.Vet) error {
	for _, value := range values {
		if !value.ID.Valid() {
			return errors.New(
				"construct memory database: veterinarian IDs must be positive",
			)
		}
		result, err := value.Validate()
		if err != nil {
			return err
		}
		if !result.Valid() {
			return errors.New(
				"construct memory database: veterinarian validation failed",
			)
		}
	}
	return nil
}
