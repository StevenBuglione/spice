//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/StevenBuglione/spice/examples/petclinic/model"
	"github.com/StevenBuglione/spice/examples/petclinic/owner"
)

func TestPostgreSQLPetclinicWorkflow(t *testing.T) {
	connectionURL := os.Getenv("SPICE_POSTGRES_TEST_URL")
	if connectionURL == "" {
		t.Fatal("SPICE_POSTGRES_TEST_URL is required")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	database, cleanup, err := OpenDatabase(Settings{
		URL:           connectionURL,
		AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("OpenDatabase() error = %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := cleanup(t.Context()); cleanupErr != nil {
			t.Errorf("database cleanup error = %v", cleanupErr)
		}
	})
	if err := database.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	owners, err := NewOwnerRepository(database)
	if err != nil {
		t.Fatalf("NewOwnerRepository() error = %v", err)
	}
	petTypes, err := NewPetTypeRepository(database)
	if err != nil {
		t.Fatalf("NewPetTypeRepository() error = %v", err)
	}
	vets, err := NewVetRepository(database)
	if err != nil {
		t.Fatalf("NewVetRepository() error = %v", err)
	}

	types, err := petTypes.FindAll(ctx)
	if err != nil || len(types) != 6 || types[0].Name != "bird" {
		t.Fatalf("FindAll() pet types = %#v, %v", types, err)
	}
	found, present, err := owners.FindByID(ctx, 6)
	if err != nil || !present || found.LastName != "Coleman" ||
		len(found.Pets) != 2 ||
		len(found.Pets[0].Visits)+len(found.Pets[1].Visits) != 4 {
		t.Fatalf("FindByID() = %#v, %t, %v", found, present, err)
	}
	page, total, err := owners.FindByLastName(ctx, "Da", 0, 10)
	if err != nil || total != 2 || len(page) != 2 ||
		page[0].FirstName != "Betty" ||
		page[1].FirstName != "Harold" {
		t.Fatalf(
			"FindByLastName() = %#v, %d, %v",
			page,
			total,
			err,
		)
	}
	allVets, err := vets.FindAll(ctx)
	if err != nil || len(allVets) != 6 ||
		allVets[0].LastName != "Carter" {
		t.Fatalf("FindAll() vets = %#v, %v", allVets, err)
	}
	vetPage, vetTotal, err := vets.FindPage(ctx, 1, 2)
	if err != nil || vetTotal != 6 || len(vetPage) != 2 {
		t.Fatalf(
			"FindPage() = %#v, %d, %v",
			vetPage,
			vetTotal,
			err,
		)
	}

	created, err := owners.Save(ctx, owner.Owner{
		Person: model.Person{
			FirstName: "Spice",
			LastName:  "Developer",
		},
		Address:   "1 Generated Way",
		City:      "Madison",
		Telephone: "6085550101",
		Pets: []owner.Pet{{
			NamedEntity: model.NamedEntity{Name: "Gopher"},
			BirthDate:   time.Date(2020, time.January, 2, 0, 0, 0, 0, time.UTC),
			Type:        types[1],
			Visits: []owner.Visit{{
				Date:        time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC),
				Description: "annual checkup",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("Save(create) error = %v", err)
	}
	if !created.ID.Valid() ||
		!created.Pets[0].ID.Valid() ||
		!created.Pets[0].Visits[0].ID.Valid() {
		t.Fatalf("Save(create) identities = %#v", created)
	}
	created.City = "Middleton"
	updated, err := owners.Save(ctx, created)
	if err != nil || updated.City != "Middleton" {
		t.Fatalf("Save(update) = %#v, %v", updated, err)
	}
	reloaded, present, err := owners.FindByID(ctx, created.ID)
	if err != nil || !present || reloaded.City != "Middleton" ||
		len(reloaded.Pets) != 1 ||
		len(reloaded.Pets[0].Visits) != 1 {
		t.Fatalf("FindByID(created) = %#v, %t, %v", reloaded, present, err)
	}
}
