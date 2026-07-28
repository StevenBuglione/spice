package vet

import (
	"testing"

	"github.com/StevenBuglione/spice/examples/petclinic/model"
)

func TestVetSpecialtiesAreUniqueOrderedAndDefensive(t *testing.T) {
	t.Parallel()

	value := Vet{
		Person: model.Person{FirstName: "Linda", LastName: "Douglas"},
	}
	value.AddSpecialty(specialty(2, "surgery"))
	value.AddSpecialty(specialty(1, "dentistry"))
	value.AddSpecialty(specialty(1, "duplicate"))
	if len(value.Specialties) != 2 ||
		value.Specialties[0].Name != "dentistry" {
		t.Fatalf("specialties = %#v", value.Specialties)
	}
	cloned := value.Clone()
	cloned.Specialties[0].Name = "changed"
	if value.Specialties[0].Name == "changed" {
		t.Fatal("Clone() exposed specialty storage")
	}
}

func TestVetNilAndValidationBehavior(t *testing.T) {
	t.Parallel()

	var missing *Vet
	missing.AddSpecialty(Specialty{})
	if clone := missing.Clone(); clone.ID != 0 || clone.Specialties != nil {
		t.Fatalf("nil clone = %#v", clone)
	}
	result, err := missing.Validate()
	if err != nil || result.Len() != 1 {
		t.Fatalf("nil validation = %#v, %v", result.All(), err)
	}
	invalid := Vet{}
	result, err = invalid.Validate()
	if err != nil || result.Len() != 2 {
		t.Fatalf("invalid validation = %#v, %v", result.All(), err)
	}
}

func specialty(id model.ID, name string) Specialty {
	return Specialty{NamedEntity: model.NamedEntity{
		BaseEntity: model.BaseEntity{ID: id},
		Name:       name,
	}}
}
