package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestMigrationPlanOwnsCanonicalSchemaAndSeed(t *testing.T) {
	t.Parallel()

	plan, err := migrationPlan()
	if err != nil {
		t.Fatalf("migrationPlan() error = %v", err)
	}
	migrations := plan.Migrations()
	if len(migrations) != 2 {
		t.Fatalf("migration count = %d", len(migrations))
	}
	if migrations[0].Version() != schemaVersion ||
		migrations[1].Version() != seedVersion ||
		migrations[0].Module() != petclinicModuleID ||
		migrations[1].Module() != petclinicModuleID {
		t.Fatalf("unexpected migration metadata: %#v", migrations)
	}
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS owners",
		"CREATE TABLE IF NOT EXISTS pets",
		"CREATE TABLE IF NOT EXISTS visits",
		"unique_owner_pet_name",
	} {
		if !strings.Contains(migrations[0].SQL(), required) {
			t.Fatalf("schema does not contain %q", required)
		}
	}
	for _, required := range []string{
		"'George', 'Franklin'",
		"'Samantha'",
		"'rabies shot'",
	} {
		if !strings.Contains(migrations[1].SQL(), required) {
			t.Fatalf("seed does not contain %q", required)
		}
	}
}

func TestOpenDatabaseRejectsInvalidSettingsWithoutLeakingSecret(
	t *testing.T,
) {
	t.Parallel()

	const secret = "do-not-expose"
	_, cleanup, err := OpenDatabase(Settings{
		URL: "postgres://petclinic:" + secret + "@missing-port/petclinic",
	})
	if err == nil {
		t.Fatal("OpenDatabase() error = nil")
	}
	if cleanup != nil {
		t.Fatal("OpenDatabase() cleanup is non-nil after failure")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("OpenDatabase() exposed secret: %v", err)
	}
}

func TestRepositoriesRejectNilDependencies(t *testing.T) {
	t.Parallel()

	if _, err := NewOwnerRepository(nil); err == nil {
		t.Fatal("NewOwnerRepository(nil) error = nil")
	}
	if _, err := NewPetTypeRepository(nil); err == nil {
		t.Fatal("NewPetTypeRepository(nil) error = nil")
	}
	if _, err := NewVetRepository(nil); err == nil {
		t.Fatal("NewVetRepository(nil) error = nil")
	}
}

func TestRepositoryBoundariesValidateBeforeDatabaseUse(t *testing.T) {
	t.Parallel()

	ownerRepository := &OwnerRepository{}
	//nolint:staticcheck // Nil context is an explicit public boundary contract.
	if _, _, err := ownerRepository.FindByID(nil, 1); err == nil {
		t.Fatal("FindByID(nil) error = nil")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := ownerRepository.FindByLastName(
		cancelled,
		"",
		0,
		10,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("FindByLastName(cancelled) error = %v", err)
	}
	if _, _, err := ownerRepository.FindByLastName(
		context.Background(),
		"",
		-1,
		10,
	); err == nil {
		t.Fatal("FindByLastName(negative offset) error = nil")
	}

	vetRepository := &VetRepository{}
	if _, _, err := vetRepository.FindPage(
		context.Background(),
		0,
		0,
	); err == nil {
		t.Fatal("FindPage(invalid limit) error = nil")
	}
}

func TestDatabaseMigrateRejectsNilInputs(t *testing.T) {
	t.Parallel()

	var database *Database
	if err := database.Migrate(context.Background()); err == nil {
		t.Fatal("nil Database.Migrate() error = nil")
	}
	database = &Database{}
	//nolint:staticcheck // Nil context is an explicit public boundary contract.
	if err := database.Migrate(nil); err == nil {
		t.Fatal("Database.Migrate(nil) error = nil")
	}
}
