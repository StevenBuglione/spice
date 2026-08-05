package memory

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/spice-framework/spice/examples/petclinic/model"
	"github.com/spice-framework/spice/examples/petclinic/owner"
)

// @import { Implements, Repository } from "github.com/spice-framework/spice/annotation/core"

// OwnerRepository persists owner aggregates in one Database.
//
// @Repository(constructor=NewOwnerRepository)
// @Implements(owner.Repository)
type OwnerRepository struct {
	database *Database
}

// NewOwnerRepository constructs a memory owner repository.
func NewOwnerRepository(database *Database) (*OwnerRepository, error) {
	if database == nil {
		return nil, errors.New(
			"construct memory owner repository: database is nil",
		)
	}
	return &OwnerRepository{database: database}, nil
}

// FindByID returns one defensive aggregate copy.
func (repository *OwnerRepository) FindByID(
	ctx context.Context,
	id model.ID,
) (owner.Owner, bool, error) {
	if err := validateContext(ctx); err != nil {
		return owner.Owner{}, false, err
	}
	if repository == nil || repository.database == nil {
		return owner.Owner{}, false, errors.New(
			"find owner by ID: repository is nil",
		)
	}
	repository.database.mu.RLock()
	defer repository.database.mu.RUnlock()
	found, ok := repository.database.owners[id]
	if !ok {
		return owner.Owner{}, false, nil
	}
	return found.Clone(), true, nil
}

// FindByLastName returns a bounded stable prefix match.
func (repository *OwnerRepository) FindByLastName(
	ctx context.Context,
	prefix string,
	offset int,
	limit int,
) ([]owner.Owner, int, error) {
	if err := validateContext(ctx); err != nil {
		return nil, 0, err
	}
	if repository == nil || repository.database == nil {
		return nil, 0, errors.New(
			"find owners by last name: repository is nil",
		)
	}
	if offset < 0 {
		return nil, 0, errors.New(
			"find owners by last name: offset must not be negative",
		)
	}
	if limit < 1 || limit > 100 {
		return nil, 0, errors.New(
			"find owners by last name: limit must be between 1 and 100",
		)
	}
	normalized := strings.ToLower(strings.TrimSpace(prefix))
	repository.database.mu.RLock()
	defer repository.database.mu.RUnlock()
	result := make([]owner.Owner, 0, len(repository.database.owners))
	for _, candidate := range repository.database.owners {
		if strings.HasPrefix(
			strings.ToLower(candidate.LastName),
			normalized,
		) {
			result = append(result, candidate.Clone())
		}
	}
	slices.SortStableFunc(result, compareOwners)
	total := len(result)
	if offset >= total {
		return []owner.Owner{}, total, nil
	}
	last := min(offset+limit, total)
	return result[offset:last], total, nil
}

// Save atomically inserts or replaces one complete aggregate.
func (repository *OwnerRepository) Save(
	ctx context.Context,
	value owner.Owner,
) (owner.Owner, error) {
	if err := validateContext(ctx); err != nil {
		return owner.Owner{}, err
	}
	if repository == nil || repository.database == nil {
		return owner.Owner{}, errors.New(
			"save owner: repository is nil",
		)
	}
	if result, err := value.Validate(); err != nil {
		return owner.Owner{}, err
	} else if !result.Valid() {
		return owner.Owner{}, errors.New(
			"save owner: owner validation failed",
		)
	}
	repository.database.mu.Lock()
	defer repository.database.mu.Unlock()
	if value.New() {
		repository.database.nextOwnerID++
		value.ID = repository.database.nextOwnerID
	} else if _, found := repository.database.owners[value.ID]; !found {
		return owner.Owner{}, errors.New(
			"save owner: persisted owner was not found",
		)
	}
	assignAggregateIDs(repository.database, &value)
	repository.database.owners[value.ID] = value.Clone()
	return value.Clone(), nil
}

func assignAggregateIDs(database *Database, value *owner.Owner) {
	for petIndex := range value.Pets {
		pet := &value.Pets[petIndex]
		if pet.New() {
			database.nextPetID++
			pet.ID = database.nextPetID
		}
		for visitIndex := range pet.Visits {
			visit := &pet.Visits[visitIndex]
			if visit.New() {
				database.nextVisitID++
				visit.ID = database.nextVisitID
			}
		}
	}
}

func compareOwners(left, right owner.Owner) int {
	if compared := strings.Compare(
		strings.ToLower(left.LastName),
		strings.ToLower(right.LastName),
	); compared != 0 {
		return compared
	}
	if compared := strings.Compare(
		strings.ToLower(left.FirstName),
		strings.ToLower(right.FirstName),
	); compared != 0 {
		return compared
	}
	return compareModelID(left.ID, right.ID)
}

func compareModelID(left, right model.ID) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}
