package memory

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/StevenBuglione/spice/examples/petclinic/model"
	"github.com/StevenBuglione/spice/examples/petclinic/owner"
)

func TestReferenceDatabaseMatchesPetclinicData(t *testing.T) {
	t.Parallel()

	database, err := NewPetclinicDatabase()
	if err != nil {
		t.Fatal(err)
	}
	owners, err := NewOwnerRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	petTypes, err := NewPetTypeRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	vets, err := NewVetRepository(database)
	if err != nil {
		t.Fatal(err)
	}

	davis, err := owners.FindByLastName(
		context.Background(),
		"Dav",
		10,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(davis) != 2 ||
		davis[0].FirstName != "Betty" ||
		davis[1].FirstName != "Harold" {
		t.Fatalf("Davis owners = %#v", davis)
	}
	types, err := petTypes.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 6 || types[0].Name != "bird" {
		t.Fatalf("pet types = %#v", types)
	}
	allVets, err := vets.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(allVets) != 6 ||
		allVets[0].LastName != "Carter" ||
		len(allVets[1].Specialties) != 2 {
		t.Fatalf("vets = %#v", allVets)
	}

	jean, found, err := owners.FindByID(
		context.Background(),
		6,
	)
	if err != nil || !found {
		t.Fatalf("FindByID() = %#v, %t, %v", jean, found, err)
	}
	if len(jean.Pets) != 2 ||
		len(jean.Pets[0].Visits)+len(jean.Pets[1].Visits) != 4 {
		t.Fatalf("Jean aggregate = %#v", jean)
	}
}

func TestOwnerRepositorySaveIsAtomicAndDefensive(t *testing.T) {
	t.Parallel()

	database, err := NewDatabase(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewOwnerRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	value := owner.Owner{
		Person: model.Person{
			FirstName: "New",
			LastName:  "Owner",
		},
		Address:   "One Main Street",
		City:      "Madison",
		Telephone: "6085550000",
		Pets: []owner.Pet{{
			NamedEntity: model.NamedEntity{Name: "Ada"},
		}},
	}
	saved, err := repository.Save(context.Background(), value)
	if err != nil {
		t.Fatal(err)
	}
	if !saved.ID.Valid() || !saved.Pets[0].ID.Valid() {
		t.Fatalf("saved owner = %#v", saved)
	}
	saved.FirstName = "mutated"
	saved.Pets[0].Name = "mutated"
	found, ok, err := repository.FindByID(
		context.Background(),
		saved.ID,
	)
	if err != nil || !ok {
		t.Fatalf("FindByID() = %#v, %t, %v", found, ok, err)
	}
	if found.FirstName != "New" || found.Pets[0].Name != "Ada" {
		t.Fatalf("stored owner was mutated: %#v", found)
	}
}

func TestMemoryRepositoriesHonorCancellationAndConcurrency(t *testing.T) {
	t.Parallel()

	database, err := NewPetclinicDatabase()
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewOwnerRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repository.FindByLastName(ctx, "", 10); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("canceled FindByLastName() error = %v", err)
	}

	var wait sync.WaitGroup
	for index := range 32 {
		wait.Add(1)
		go func(id int) {
			defer wait.Done()
			results, findErr := repository.FindByLastName(
				context.Background(),
				"",
				100,
			)
			if findErr != nil || len(results) != 10 {
				t.Errorf(
					"reader %d = %d results, %v",
					id,
					len(results),
					findErr,
				)
			}
		}(index)
	}
	wait.Wait()
}
