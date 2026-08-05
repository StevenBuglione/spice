// Package postgres provides the PostgreSQL persistence target for Petclinic.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/spice-framework/spice/lifecycle"
	spicepostgres "github.com/spice-framework/spice/starter/postgres"
)

// @import { Bean } from "github.com/spice-framework/spice/annotation/core"
// @import { OnStart } from "github.com/spice-framework/spice/annotation/lifecycle"

// Database owns the Petclinic PostgreSQL pool and migration lifecycle.
type Database struct {
	native *sql.DB
}

// OpenDatabase constructs the PostgreSQL pool without hidden environment
// reads or network I/O. Spice registers the returned cleanup immediately.
//
// @Bean
func OpenDatabase(
	settings Settings,
) (*Database, lifecycle.Cleanup, error) {
	native, err := spicepostgres.Open(spicepostgres.Options{
		URL:             settings.URL,
		ApplicationName: "spice-petclinic",
		AllowInsecure:   settings.AllowInsecure,
	})
	if err != nil {
		return nil, nil, fmt.Errorf(
			"construct Petclinic PostgreSQL database: %w",
			err,
		)
	}
	database := &Database{native: native}
	return database, closeDatabase(native), nil
}

func closeDatabase(native *sql.DB) lifecycle.Cleanup {
	return func(context.Context) error {
		if err := native.Close(); err != nil {
			return fmt.Errorf(
				"close Petclinic PostgreSQL database: %w",
				err,
			)
		}
		return nil
	}
}

// Migrate verifies connectivity and applies the exact module-owned schema and
// canonical sample data before the application becomes ready.
//
// @OnStart
func (database *Database) Migrate(ctx context.Context) error {
	switch {
	case ctx == nil:
		return errors.New(
			"migrate Petclinic PostgreSQL database: context is nil",
		)
	case database == nil || database.native == nil:
		return errors.New(
			"migrate Petclinic PostgreSQL database: database is nil",
		)
	}
	if err := spicepostgres.Ping(ctx, database.native); err != nil {
		return fmt.Errorf(
			"migrate Petclinic PostgreSQL database: %w",
			err,
		)
	}
	if err := runMigrations(ctx, database.native); err != nil {
		return err
	}
	return nil
}
