package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/spice-framework/spice/migration"
	spicepostgres "github.com/spice-framework/spice/starter/postgres"
)

const (
	petclinicModuleID = "github.com/spice-framework/spice/examples/petclinic"
	schemaVersion     = 202607280101
	seedVersion       = 202607280102
)

func migrationPlan() (*migration.Plan, error) {
	return migration.NewPlan([]migration.Spec{
		{
			Version: schemaVersion,
			Module:  petclinicModuleID,
			Name:    "create Petclinic schema",
			SQL:     schemaSQL,
		},
		{
			Version: seedVersion,
			Module:  petclinicModuleID,
			Name:    "load canonical Petclinic data",
			SQL:     seedSQL,
		},
	})
}

func runMigrations(ctx context.Context, native *sql.DB) error {
	plan, err := migrationPlan()
	if err != nil {
		return fmt.Errorf(
			"construct Petclinic PostgreSQL migration plan: %w",
			err,
		)
	}
	backend, err := spicepostgres.NewMigrationBackend(
		native,
		spicepostgres.MigrationOptions{},
	)
	if err != nil {
		return fmt.Errorf(
			"construct Petclinic PostgreSQL migration backend: %w",
			err,
		)
	}
	runner, err := migration.NewRunner(backend)
	if err != nil {
		return fmt.Errorf(
			"construct Petclinic PostgreSQL migration runner: %w",
			err,
		)
	}
	if _, err := runner.Run(ctx, plan); err != nil {
		return fmt.Errorf(
			"migrate Petclinic PostgreSQL database: %w",
			err,
		)
	}
	return nil
}
