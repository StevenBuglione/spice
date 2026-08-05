package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/spice-framework/spice/examples/petclinic/model"
	"github.com/spice-framework/spice/examples/petclinic/vet"
)

// @import { Implements, Repository } from "github.com/spice-framework/spice/annotation/core"

// VetRepository reads PostgreSQL-backed veterinarians.
//
// @Repository(constructor=NewVetRepository)
// @Implements(vet.Repository)
type VetRepository struct {
	database *Database
}

// NewVetRepository constructs a PostgreSQL veterinarian repository.
func NewVetRepository(database *Database) (*VetRepository, error) {
	if database == nil || database.native == nil {
		return nil, errors.New(
			"construct PostgreSQL vet repository: database is nil",
		)
	}
	return &VetRepository{database: database}, nil
}

// FindAll returns every veterinarian in stable display order.
func (repository *VetRepository) FindAll(
	ctx context.Context,
) ([]vet.Vet, error) {
	values, _, err := repository.find(ctx, 0, 0, false)
	return values, err
}

// FindPage returns one bounded veterinarian page and total count.
func (repository *VetRepository) FindPage(
	ctx context.Context,
	offset int,
	limit int,
) ([]vet.Vet, int, error) {
	if offset < 0 {
		return nil, 0, errors.New(
			"find veterinarian page: offset must not be negative",
		)
	}
	if limit < 1 || limit > 100 {
		return nil, 0, errors.New(
			"find veterinarian page: limit must be between 1 and 100",
		)
	}
	return repository.find(ctx, offset, limit, true)
}

func (repository *VetRepository) find(
	ctx context.Context,
	offset int,
	limit int,
	paged bool,
) (values []vet.Vet, total int, err error) {
	if contextErr := validateContext(
		ctx,
		"find veterinarians",
	); contextErr != nil {
		return nil, 0, contextErr
	}
	if repository == nil ||
		repository.database == nil ||
		repository.database.native == nil {
		return nil, 0, errors.New(
			"find veterinarians: repository is nil",
		)
	}
	native := repository.database.native
	if queryErr := native.QueryRowContext(
		ctx,
		countVetsSQL,
	).Scan(&total); queryErr != nil {
		return nil, 0, fmt.Errorf(
			"find veterinarians: count: %w",
			queryErr,
		)
	}
	statement := findAllVetsSQL
	arguments := []any{}
	if paged {
		statement = findVetPageSQL
		arguments = []any{offset, limit}
	}
	rows, err := native.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"find veterinarians: query: %w",
			err,
		)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	for rows.Next() {
		var value vet.Vet
		if err := rows.Scan(
			&value.ID,
			&value.FirstName,
			&value.LastName,
		); err != nil {
			return nil, 0, fmt.Errorf(
				"find veterinarians: scan: %w",
				err,
			)
		}
		specialties, err := repository.findSpecialties(ctx, value.ID)
		if err != nil {
			return nil, 0, err
		}
		value.Specialties = specialties
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"find veterinarians: rows: %w",
			err,
		)
	}
	if values == nil {
		values = []vet.Vet{}
	}
	return values, total, nil
}

func (repository *VetRepository) findSpecialties(
	ctx context.Context,
	vetID model.ID,
) (result []vet.Specialty, err error) {
	rows, err := repository.database.native.QueryContext(
		ctx,
		findVetSpecialtiesSQL,
		vetID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"find veterinarian specialties: query: %w",
			err,
		)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()
	for rows.Next() {
		var value vet.Specialty
		if err := rows.Scan(&value.ID, &value.Name); err != nil {
			return nil, fmt.Errorf(
				"find veterinarian specialties: scan: %w",
				err,
			)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"find veterinarian specialties: rows: %w",
			err,
		)
	}
	if result == nil {
		result = []vet.Specialty{}
	}
	return result, nil
}

const countVetsSQL = `SELECT count(*) FROM vets`

const findAllVetsSQL = `SELECT id, first_name, last_name
FROM vets
ORDER BY lower(last_name), lower(first_name), id`

const findVetPageSQL = `SELECT id, first_name, last_name
FROM vets
ORDER BY lower(last_name), lower(first_name), id
OFFSET $1
LIMIT $2`

const findVetSpecialtiesSQL = `SELECT specialties.id, specialties.name
FROM specialties
JOIN vet_specialties
	ON vet_specialties.specialty_id = specialties.id
WHERE vet_specialties.vet_id = $1
ORDER BY lower(specialties.name), specialties.id`
