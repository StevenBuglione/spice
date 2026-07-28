package memory

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/StevenBuglione/spice/examples/petclinic/owner"
)

// PetTypeRepository reads memory-backed pet type reference data.
type PetTypeRepository struct {
	database *Database
}

// NewPetTypeRepository constructs a memory pet type repository.
func NewPetTypeRepository(
	database *Database,
) (*PetTypeRepository, error) {
	if database == nil {
		return nil, errors.New(
			"construct memory pet type repository: database is nil",
		)
	}
	return &PetTypeRepository{database: database}, nil
}

// FindAll returns pet types ordered by display name.
func (repository *PetTypeRepository) FindAll(
	ctx context.Context,
) ([]owner.PetType, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	if repository == nil || repository.database == nil {
		return nil, errors.New(
			"find pet types: repository is nil",
		)
	}
	repository.database.mu.RLock()
	defer repository.database.mu.RUnlock()
	return clonePetTypes(repository.database.petTypes), nil
}

func clonePetTypes(values []owner.PetType) []owner.PetType {
	result := slices.Clone(values)
	slices.SortStableFunc(result, func(
		left owner.PetType,
		right owner.PetType,
	) int {
		return strings.Compare(
			strings.ToLower(left.Name),
			strings.ToLower(right.Name),
		)
	})
	return result
}

func validatePetTypes(values []owner.PetType) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		name := strings.ToLower(strings.TrimSpace(value.Name))
		if !value.ID.Valid() || name == "" {
			return errors.New(
				"construct memory database: pet types require positive IDs and names",
			)
		}
		if _, duplicate := seen[name]; duplicate {
			return errors.New(
				"construct memory database: pet type names must be unique",
			)
		}
		seen[name] = struct{}{}
	}
	return nil
}
