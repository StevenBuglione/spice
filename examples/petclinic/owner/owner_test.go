package owner

import (
	"slices"
	"testing"
	"time"

	"github.com/StevenBuglione/spice/examples/petclinic/model"
)

func TestOwnerValidationMatchesReferenceConstraints(t *testing.T) {
	t.Parallel()

	value := Owner{
		Person: model.Person{
			FirstName: "",
			LastName:  "0123456789012345678901234567890",
		},
		Address:   " ",
		City:      "",
		Telephone: "608-555",
	}
	result, err := value.Validate()
	if err != nil {
		t.Fatal(err)
	}
	var fields []string
	for _, violation := range result.All() {
		fields = append(fields, violation.Field)
	}
	if want := []string{
		"firstName",
		"lastName",
		"address",
		"city",
		"telephone",
	}; !slices.Equal(fields, want) {
		t.Fatalf("violation fields = %v, want %v", fields, want)
	}
}

func TestOwnerAggregateLookupMutationAndCloning(t *testing.T) {
	t.Parallel()

	value := Owner{
		Person: model.Person{
			BaseEntity: model.BaseEntity{ID: 1},
			FirstName:  "George",
			LastName:   "Franklin",
		},
		Address:   "address",
		City:      "city",
		Telephone: "0123456789",
		Pets: []Pet{{
			NamedEntity: model.NamedEntity{
				BaseEntity: model.BaseEntity{ID: 7},
				Name:       "Samantha",
			},
			Visits: []Visit{{
				BaseEntity:  model.BaseEntity{ID: 1},
				Date:        time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
				Description: "checkup",
			}},
		}},
	}
	if pet, found := value.PetByName("sAmAnThA", true); !found ||
		pet.ID != 7 {
		t.Fatalf("PetByName() = %#v, %t", pet, found)
	}
	if _, found := value.PetByID(0); found {
		t.Fatal("PetByID(0) found a new pet")
	}
	if err := value.AddVisit(7, Visit{
		Date:        time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		Description: "earlier",
	}); err != nil {
		t.Fatal(err)
	}
	if got := value.Pets[0].Visits[0].Description; got != "earlier" {
		t.Fatalf("first visit = %q", got)
	}
	cloned := value.Clone()
	cloned.Pets[0].Name = "changed"
	cloned.Pets[0].Visits[0].Description = "changed"
	if value.Pets[0].Name == "changed" ||
		value.Pets[0].Visits[0].Description == "changed" {
		t.Fatal("Clone() exposed aggregate storage")
	}
}

func TestPetAndVisitValidation(t *testing.T) {
	t.Parallel()

	today := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	pet := Pet{
		BirthDate: today.AddDate(0, 0, 1),
	}
	petResult, err := pet.Validate(today)
	if err != nil {
		t.Fatal(err)
	}
	if petResult.Len() != 3 {
		t.Fatalf("pet violations = %#v", petResult.All())
	}
	visitResult, err := (Visit{}).Validate()
	if err != nil {
		t.Fatal(err)
	}
	if visitResult.Len() != 2 {
		t.Fatalf("visit violations = %#v", visitResult.All())
	}
}

func TestOwnerAggregateFailureBoundaries(t *testing.T) {
	t.Parallel()

	var missing *Owner
	if clone := missing.Clone(); clone.ID != 0 || clone.Pets != nil {
		t.Fatalf("nil clone = %#v", clone)
	}
	if result, err := missing.Validate(); err != nil || result.Len() != 1 {
		t.Fatalf("nil validation = %#v, %v", result.All(), err)
	}
	if _, found := missing.PetByID(1); found {
		t.Fatal("nil owner returned a pet")
	}
	if _, found := missing.PetByName("Leo", false); found {
		t.Fatal("nil owner returned a pet by name")
	}
	if err := missing.AddPet(Pet{}); err == nil {
		t.Fatal("nil owner accepted a pet")
	}
	if err := missing.AddVisit(1, Visit{}); err == nil {
		t.Fatal("nil owner accepted a visit")
	}

	value := Owner{}
	if err := value.AddPet(Pet{
		NamedEntity: model.NamedEntity{
			BaseEntity: model.BaseEntity{ID: 1},
		},
	}); err == nil {
		t.Fatal("persisted pet was accepted as new")
	}
	if err := value.AddVisit(1, Visit{}); err == nil {
		t.Fatal("missing pet accepted a visit")
	}

	first := Pet{NamedEntity: model.NamedEntity{Name: "zeta"}}
	second := Pet{NamedEntity: model.NamedEntity{Name: "Alpha"}}
	if err := value.AddPet(first); err != nil {
		t.Fatal(err)
	}
	if err := value.AddPet(second); err != nil {
		t.Fatal(err)
	}
	if value.Pets[0].Name != "Alpha" {
		t.Fatalf("pets = %#v", value.Pets)
	}
	if _, found := value.PetByName("Alpha", true); found {
		t.Fatal("new pet was not ignored")
	}
}

func TestPetNilAndVisitOrderingBoundaries(t *testing.T) {
	t.Parallel()

	var missing *Pet
	if clone := missing.Clone(); clone.ID != 0 || clone.Visits != nil {
		t.Fatalf("nil clone = %#v", clone)
	}
	if result, err := missing.Validate(time.Now()); err != nil ||
		result.Len() != 1 {
		t.Fatalf("nil validation = %#v, %v", result.All(), err)
	}
	if err := missing.AddVisit(Visit{}); err == nil {
		t.Fatal("nil pet accepted a visit")
	}

	value := Pet{}
	later := Visit{
		BaseEntity: model.BaseEntity{ID: 2},
		Date:       time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	earlier := Visit{
		BaseEntity: model.BaseEntity{ID: 3},
		Date:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	sameDateLowerID := Visit{
		BaseEntity: model.BaseEntity{ID: 1},
		Date:       later.Date,
	}
	for _, visit := range []Visit{later, earlier, sameDateLowerID} {
		if err := value.AddVisit(visit); err != nil {
			t.Fatal(err)
		}
	}
	if got := []model.ID{
		value.Visits[0].ID,
		value.Visits[1].ID,
		value.Visits[2].ID,
	}; !slices.Equal(got, []model.ID{3, 1, 2}) {
		t.Fatalf("visit order = %v", got)
	}
}
