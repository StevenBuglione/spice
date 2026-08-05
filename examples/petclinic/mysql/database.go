// Package mysql provides the MySQL persistence target for Petclinic.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/spice-framework/spice/lifecycle"
	spicemysql "github.com/spice-framework/spice/starter/mysql"
)

// @import { Bean } from "github.com/spice-framework/spice/annotation/core"
// @import { OnStart } from "github.com/spice-framework/spice/annotation/lifecycle"

// Database owns the Petclinic MySQL pool and migration lifecycle.
type Database struct {
	native *sql.DB
}

// OpenDatabase constructs the MySQL pool without hidden environment reads or
// network I/O. Spice registers the returned cleanup immediately.
//
// @Bean
func OpenDatabase(
	settings Settings,
) (*Database, lifecycle.Cleanup, error) {
	native, err := spicemysql.Open(spicemysql.Options{
		URL:             settings.URL,
		ApplicationName: "spice-petclinic",
		AllowInsecure:   settings.AllowInsecure,
	})
	if err != nil {
		return nil, nil, fmt.Errorf(
			"construct Petclinic MySQL database: %w",
			err,
		)
	}
	database := &Database{native: native}
	return database, closeDatabase(native), nil
}

func closeDatabase(native *sql.DB) lifecycle.Cleanup {
	return func(context.Context) error {
		if err := native.Close(); err != nil {
			return fmt.Errorf("close Petclinic MySQL database: %w", err)
		}
		return nil
	}
}

// Migrate verifies connectivity and applies the locked, idempotent Petclinic
// schema before the application becomes ready.
//
// MySQL DDL implicitly commits, so this profile is deliberately resumable
// rather than falsely claiming cross-statement transactional DDL.
//
// @OnStart
func (database *Database) Migrate(ctx context.Context) error {
	switch {
	case ctx == nil:
		return errors.New("migrate Petclinic MySQL database: context is nil")
	case database == nil || database.native == nil:
		return errors.New("migrate Petclinic MySQL database: database is nil")
	}
	if err := spicemysql.Ping(ctx, database.native); err != nil {
		return fmt.Errorf("migrate Petclinic MySQL database: %w", err)
	}
	if err := runMigrations(ctx, database.native); err != nil {
		return err
	}
	return nil
}
