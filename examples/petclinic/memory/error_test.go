package memory

import (
	"context"
	"testing"

	"github.com/StevenBuglione/spice/examples/petclinic/model"
	"github.com/StevenBuglione/spice/examples/petclinic/owner"
	"github.com/StevenBuglione/spice/examples/petclinic/vet"
)

func TestDatabaseRejectsInvalidSeedData(t *testing.T) {
	t.Parallel()

	validOwner := owner.Owner{
		Person: model.Person{
			BaseEntity: model.BaseEntity{ID: 1},
			FirstName:  "George",
			LastName:   "Franklin",
		},
	}
	tests := []struct {
		name     string
		owners   []owner.Owner
		petTypes []owner.PetType
		vets     []vet.Vet
	}{
		{name: "owner ID", owners: []owner.Owner{{}}},
		{
			name:   "duplicate owner",
			owners: []owner.Owner{validOwner, validOwner},
		},
		{
			name:   "pet ID",
			owners: []owner.Owner{withPet(validOwner, owner.Pet{})},
		},
		{
			name: "duplicate pet",
			owners: []owner.Owner{withPet(
				validOwner,
				persistedPet(1),
				persistedPet(1),
			)},
		},
		{
			name: "visit ID",
			owners: []owner.Owner{withPet(
				validOwner,
				petWithVisit(1, 0),
			)},
		},
		{name: "pet type", petTypes: []owner.PetType{{}}},
		{
			name: "duplicate pet type",
			petTypes: []owner.PetType{
				petType(1, "cat"),
				petType(2, "CAT"),
			},
		},
		{name: "vet ID", vets: []vet.Vet{{}}},
		{
			name: "vet validation",
			vets: []vet.Vet{{
				Person: model.Person{
					BaseEntity: model.BaseEntity{ID: 1},
				},
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, err := NewDatabase(
				test.owners,
				test.petTypes,
				test.vets,
			); err == nil {
				t.Fatal("NewDatabase() accepted invalid seed data")
			}
		})
	}
}

func TestRepositoryConstructorAndOperationBoundaries(t *testing.T) {
	t.Parallel()

	if _, err := NewOwnerRepository(nil); err == nil {
		t.Fatal("NewOwnerRepository(nil) succeeded")
	}
	if _, err := NewPetTypeRepository(nil); err == nil {
		t.Fatal("NewPetTypeRepository(nil) succeeded")
	}
	if _, err := NewVetRepository(nil); err == nil {
		t.Fatal("NewVetRepository(nil) succeeded")
	}

	var owners *OwnerRepository
	if _, _, err := owners.FindByID(context.Background(), 1); err == nil {
		t.Fatal("nil owner repository read succeeded")
	}
	if _, err := owners.FindByLastName(context.Background(), "", 10); err == nil {
		t.Fatal("nil owner repository search succeeded")
	}
	if _, err := owners.Save(context.Background(), owner.Owner{}); err == nil {
		t.Fatal("nil owner repository save succeeded")
	}
	var petTypes *PetTypeRepository
	if _, err := petTypes.FindAll(context.Background()); err == nil {
		t.Fatal("nil pet type repository read succeeded")
	}
	var vets *VetRepository
	if _, err := vets.FindAll(context.Background()); err == nil {
		t.Fatal("nil vet repository read succeeded")
	}

	database, err := NewDatabase(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewOwnerRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.FindByLastName(context.Background(), "", 0); err == nil {
		t.Fatal("zero search limit succeeded")
	}
	if _, err := repository.FindByLastName(context.Background(), "", 101); err == nil {
		t.Fatal("oversized search limit succeeded")
	}
	if _, err := repository.Save(context.Background(), owner.Owner{}); err == nil {
		t.Fatal("invalid owner save succeeded")
	}
	notFound := validOwnerForSave()
	notFound.ID = 99
	if _, err := repository.Save(context.Background(), notFound); err == nil {
		t.Fatal("unknown persisted owner save succeeded")
	}
	if _, found, findErr := repository.FindByID(
		context.Background(),
		99,
	); findErr != nil || found {
		t.Fatalf("missing owner = %t, %v", found, findErr)
	}
}

func TestRepositoryContextAndDefensiveReferenceData(t *testing.T) {
	t.Parallel()

	database, err := NewPetclinicDatabase()
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
	// Deliberately prove the public boundary rejects an invalid caller context.
	if _, contextErr := petTypes.FindAll(nil); contextErr == nil { //nolint:staticcheck // exercises rejection
		t.Fatal("nil context succeeded")
	}
	// Deliberately prove every reference repository preserves that boundary.
	if _, contextErr := vets.FindAll(nil); contextErr == nil { //nolint:staticcheck // exercises rejection
		t.Fatal("nil context succeeded")
	}
	types, err := petTypes.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	types[0].Name = "changed"
	again, err := petTypes.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if again[0].Name == "changed" {
		t.Fatal("pet type storage was exposed")
	}
	allVets, err := vets.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	allVets[1].Specialties[0].Name = "changed"
	againVets, err := vets.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if againVets[1].Specialties[0].Name == "changed" {
		t.Fatal("veterinarian storage was exposed")
	}
}

func withPet(value owner.Owner, pets ...owner.Pet) owner.Owner {
	value.Pets = pets
	return value
}

func persistedPet(id model.ID) owner.Pet {
	return owner.Pet{
		NamedEntity: model.NamedEntity{
			BaseEntity: model.BaseEntity{ID: id},
		},
	}
}

func petWithVisit(petID, visitID model.ID) owner.Pet {
	value := persistedPet(petID)
	value.Visits = []owner.Visit{{
		BaseEntity: model.BaseEntity{ID: visitID},
	}}
	return value
}

func petType(id model.ID, name string) owner.PetType {
	return owner.PetType{NamedEntity: model.NamedEntity{
		BaseEntity: model.BaseEntity{ID: id},
		Name:       name,
	}}
}

func validOwnerForSave() owner.Owner {
	return owner.Owner{
		Person: model.Person{
			FirstName: "George",
			LastName:  "Franklin",
		},
		Address:   "110 W. Liberty St.",
		City:      "Madison",
		Telephone: "6085551023",
	}
}
