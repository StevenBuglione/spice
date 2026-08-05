package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/spice-framework/spice/examples/petclinic/model"
	"github.com/spice-framework/spice/examples/petclinic/owner"
)

// @import { Implements, Repository } from "github.com/spice-framework/spice/annotation/core"

// OwnerRepository persists complete owner aggregates in PostgreSQL.
//
// @Repository(constructor=NewOwnerRepository)
// @Implements(owner.Repository)
type OwnerRepository struct {
	database *Database
}

// NewOwnerRepository constructs a PostgreSQL owner repository.
func NewOwnerRepository(
	database *Database,
) (*OwnerRepository, error) {
	if database == nil || database.native == nil {
		return nil, errors.New(
			"construct PostgreSQL owner repository: database is nil",
		)
	}
	return &OwnerRepository{database: database}, nil
}

// FindByID returns one complete owner aggregate.
func (repository *OwnerRepository) FindByID(
	ctx context.Context,
	id model.ID,
) (owner.Owner, bool, error) {
	if err := validateContext(ctx, "find owner by ID"); err != nil {
		return owner.Owner{}, false, err
	}
	if repository == nil ||
		repository.database == nil ||
		repository.database.native == nil {
		return owner.Owner{}, false, errors.New(
			"find owner by ID: repository is nil",
		)
	}
	if !id.Valid() {
		return owner.Owner{}, false, errors.New(
			"find owner by ID: ID must be positive",
		)
	}
	value, found, err := loadOwner(
		ctx,
		repository.database.native,
		id,
	)
	if err != nil {
		return owner.Owner{}, false, fmt.Errorf(
			"find owner by ID: %w",
			err,
		)
	}
	return value, found, nil
}

// FindByLastName returns one bounded stable prefix page.
func (repository *OwnerRepository) FindByLastName(
	ctx context.Context,
	prefix string,
	offset int,
	limit int,
) ([]owner.Owner, int, error) {
	if err := validateContext(
		ctx,
		"find owners by last name",
	); err != nil {
		return nil, 0, err
	}
	if repository == nil ||
		repository.database == nil ||
		repository.database.native == nil {
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
	native := repository.database.native
	var total int
	if err := native.QueryRowContext(
		ctx,
		countOwnersSQL,
		prefix,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf(
			"find owners by last name: count: %w",
			err,
		)
	}
	identities, err := findOwnerIDs(
		ctx,
		native,
		prefix,
		offset,
		limit,
	)
	if err != nil {
		return nil, 0, err
	}
	result := make([]owner.Owner, 0, len(identities))
	for _, id := range identities {
		value, found, err := loadOwner(ctx, native, id)
		if err != nil {
			return nil, 0, fmt.Errorf(
				"find owners by last name: load owner %d: %w",
				id,
				err,
			)
		}
		if !found {
			return nil, 0, fmt.Errorf(
				"find owners by last name: owner %d disappeared",
				id,
			)
		}
		result = append(result, value)
	}
	return result, total, nil
}

func findOwnerIDs(
	ctx context.Context,
	native *sql.DB,
	prefix string,
	offset int,
	limit int,
) (identities []model.ID, err error) {
	rows, err := native.QueryContext(
		ctx,
		findOwnerIDsSQL,
		prefix,
		offset,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"find owners by last name: query page: %w",
			err,
		)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()
	identities = make([]model.ID, 0, limit)
	for rows.Next() {
		var id model.ID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf(
				"find owners by last name: scan identity: %w",
				err,
			)
		}
		identities = append(identities, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"find owners by last name: read identities: %w",
			err,
		)
	}
	return identities, nil
}

// Save atomically inserts or updates one complete owner aggregate.
func (repository *OwnerRepository) Save(
	ctx context.Context,
	value owner.Owner,
) (saved owner.Owner, err error) {
	if contextErr := validateContext(ctx, "save owner"); contextErr != nil {
		return owner.Owner{}, contextErr
	}
	if repository == nil ||
		repository.database == nil ||
		repository.database.native == nil {
		return owner.Owner{}, errors.New("save owner: repository is nil")
	}
	result, err := value.Validate()
	if err != nil {
		return owner.Owner{}, fmt.Errorf("save owner: validate: %w", err)
	}
	if !result.Valid() {
		return owner.Owner{}, errors.New(
			"save owner: owner validation failed",
		)
	}
	transaction, err := repository.database.native.BeginTx(ctx, nil)
	if err != nil {
		return owner.Owner{}, fmt.Errorf(
			"save owner: begin transaction: %w",
			err,
		)
	}
	committed := false
	defer func() {
		if !committed {
			rollbackErr := transaction.Rollback()
			if errors.Is(rollbackErr, sql.ErrTxDone) {
				rollbackErr = nil
			}
			err = errors.Join(err, rollbackErr)
		}
	}()
	if err := saveOwnerRow(ctx, transaction, &value); err != nil {
		return owner.Owner{}, err
	}
	for index := range value.Pets {
		if err := savePet(
			ctx,
			transaction,
			value.ID,
			&value.Pets[index],
		); err != nil {
			return owner.Owner{}, err
		}
	}
	if err := transaction.Commit(); err != nil {
		return owner.Owner{}, fmt.Errorf(
			"save owner: commit transaction: %w",
			err,
		)
	}
	committed = true
	return value.Clone(), nil
}

func saveOwnerRow(
	ctx context.Context,
	transaction *sql.Tx,
	value *owner.Owner,
) error {
	if value.New() {
		if err := transaction.QueryRowContext(
			ctx,
			insertOwnerSQL,
			value.FirstName,
			value.LastName,
			value.Address,
			value.City,
			value.Telephone,
		).Scan(&value.ID); err != nil {
			return fmt.Errorf("save owner: insert: %w", err)
		}
		return nil
	}
	result, err := transaction.ExecContext(
		ctx,
		updateOwnerSQL,
		value.FirstName,
		value.LastName,
		value.Address,
		value.City,
		value.Telephone,
		value.ID,
	)
	if err != nil {
		return fmt.Errorf("save owner: update: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("save owner: update result: %w", err)
	}
	if affected != 1 {
		return errors.New("save owner: persisted owner was not found")
	}
	return nil
}

func savePet(
	ctx context.Context,
	transaction *sql.Tx,
	ownerID model.ID,
	value *owner.Pet,
) error {
	if value.New() {
		if err := transaction.QueryRowContext(
			ctx,
			insertPetSQL,
			value.Name,
			value.BirthDate,
			value.Type.ID,
			ownerID,
		).Scan(&value.ID); err != nil {
			return fmt.Errorf("save owner: insert pet: %w", err)
		}
	} else {
		result, err := transaction.ExecContext(
			ctx,
			updatePetSQL,
			value.Name,
			value.BirthDate,
			value.Type.ID,
			value.ID,
			ownerID,
		)
		if err != nil {
			return fmt.Errorf("save owner: update pet: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf(
				"save owner: update pet result: %w",
				err,
			)
		}
		if affected != 1 {
			return errors.New(
				"save owner: persisted pet was not found",
			)
		}
	}
	for index := range value.Visits {
		if err := saveVisit(
			ctx,
			transaction,
			value.ID,
			&value.Visits[index],
		); err != nil {
			return err
		}
	}
	return nil
}

func saveVisit(
	ctx context.Context,
	transaction *sql.Tx,
	petID model.ID,
	value *owner.Visit,
) error {
	if value.New() {
		if err := transaction.QueryRowContext(
			ctx,
			insertVisitSQL,
			petID,
			value.Date,
			value.Description,
		).Scan(&value.ID); err != nil {
			return fmt.Errorf("save owner: insert visit: %w", err)
		}
		return nil
	}
	result, err := transaction.ExecContext(
		ctx,
		updateVisitSQL,
		value.Date,
		value.Description,
		value.ID,
		petID,
	)
	if err != nil {
		return fmt.Errorf("save owner: update visit: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"save owner: update visit result: %w",
			err,
		)
	}
	if affected != 1 {
		return errors.New("save owner: persisted visit was not found")
	}
	return nil
}
