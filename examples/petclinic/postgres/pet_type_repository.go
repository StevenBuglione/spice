package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/spice-framework/spice/examples/petclinic/owner"
)

// @import { Implements, Repository } from "github.com/spice-framework/spice/annotation/core"

// PetTypeRepository reads PostgreSQL-backed pet type reference data.
//
// @Repository(constructor=NewPetTypeRepository)
// @Implements(owner.PetTypeRepository)
type PetTypeRepository struct {
	database *Database
}

// NewPetTypeRepository constructs a PostgreSQL pet type repository.
func NewPetTypeRepository(
	database *Database,
) (*PetTypeRepository, error) {
	if database == nil || database.native == nil {
		return nil, errors.New(
			"construct PostgreSQL pet type repository: database is nil",
		)
	}
	return &PetTypeRepository{database: database}, nil
}

// FindAll returns pet types in stable display order.
func (repository *PetTypeRepository) FindAll(
	ctx context.Context,
) (result []owner.PetType, err error) {
	if contextErr := validateContext(
		ctx,
		"find pet types",
	); contextErr != nil {
		return nil, contextErr
	}
	if repository == nil ||
		repository.database == nil ||
		repository.database.native == nil {
		return nil, errors.New("find pet types: repository is nil")
	}
	rows, err := repository.database.native.QueryContext(
		ctx,
		findPetTypesSQL,
	)
	if err != nil {
		return nil, fmt.Errorf("find pet types: query: %w", err)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	for rows.Next() {
		var value owner.PetType
		if err := rows.Scan(&value.ID, &value.Name); err != nil {
			return nil, fmt.Errorf(
				"find pet types: scan: %w",
				err,
			)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find pet types: rows: %w", err)
	}
	if result == nil {
		result = []owner.PetType{}
	}
	return result, nil
}

const findPetTypesSQL = `SELECT id, name
FROM types
ORDER BY lower(name), id`
