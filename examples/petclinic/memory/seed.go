package memory

import (
	"time"

	"github.com/StevenBuglione/spice/examples/petclinic/model"
	"github.com/StevenBuglione/spice/examples/petclinic/owner"
	"github.com/StevenBuglione/spice/examples/petclinic/vet"
)

// @import { Bean } from "github.com/StevenBuglione/spice/annotation/core"

// NewPetclinicDatabase returns the recognizable Spring Petclinic reference
// data without filesystem, environment, or network access.
//
// @Bean
func NewPetclinicDatabase() (*Database, error) {
	petTypes := referencePetTypes()
	owners := referenceOwners(petTypes)
	vets := referenceVets()
	return NewDatabase(owners, petTypes, vets)
}

func referencePetTypes() []owner.PetType {
	names := []string{"cat", "dog", "lizard", "snake", "bird", "hamster"}
	result := make([]owner.PetType, len(names))
	for index, name := range names {
		result[index] = owner.PetType{
			NamedEntity: model.NamedEntity{
				BaseEntity: model.BaseEntity{
					ID: model.ID(index + 1),
				},
				Name: name,
			},
		}
	}
	return result
}

func referenceOwners(types []owner.PetType) []owner.Owner {
	owners := []owner.Owner{
		newOwner(1, "George", "Franklin", "110 W. Liberty St.", "Madison", "6085551023"),
		newOwner(2, "Betty", "Davis", "638 Cardinal Ave.", "Sun Prairie", "6085551749"),
		newOwner(3, "Eduardo", "Rodriquez", "2693 Commerce St.", "McFarland", "6085558763"),
		newOwner(4, "Harold", "Davis", "563 Friendly St.", "Windsor", "6085553198"),
		newOwner(5, "Peter", "McTavish", "2387 S. Fair Way", "Madison", "6085552765"),
		newOwner(6, "Jean", "Coleman", "105 N. Lake St.", "Monona", "6085552654"),
		newOwner(7, "Jeff", "Black", "1450 Oak Blvd.", "Monona", "6085555387"),
		newOwner(8, "Maria", "Escobito", "345 Maple St.", "Madison", "6085557683"),
		newOwner(9, "David", "Schroeder", "2749 Blackhawk Trail", "Madison", "6085559435"),
		newOwner(10, "Carlos", "Estaban", "2335 Independence La.", "Waunakee", "6085555487"),
	}
	pets := []struct {
		id      model.ID
		name    string
		date    time.Time
		typeID  int
		ownerID int
		visits  []owner.Visit
	}{
		{1, "Leo", referenceDate(2010, 9, 7), 1, 1, nil},
		{2, "Basil", referenceDate(2012, 8, 6), 6, 2, nil},
		{3, "Rosy", referenceDate(2011, 4, 17), 2, 3, nil},
		{4, "Jewel", referenceDate(2010, 3, 7), 2, 3, nil},
		{5, "Iggy", referenceDate(2010, 11, 30), 3, 4, nil},
		{6, "George", referenceDate(2010, 1, 20), 4, 5, nil},
		{7, "Samantha", referenceDate(2012, 9, 4), 1, 6, []owner.Visit{
			newVisit(1, 1, "rabies shot"),
			newVisit(4, 4, "spayed"),
		}},
		{8, "Max", referenceDate(2012, 9, 4), 1, 6, []owner.Visit{
			newVisit(2, 2, "rabies shot"),
			newVisit(3, 3, "neutered"),
		}},
		{9, "Lucky", referenceDate(2011, 8, 6), 5, 7, nil},
		{10, "Mulligan", referenceDate(2007, 2, 24), 2, 8, nil},
		{11, "Freddy", referenceDate(2010, 3, 9), 5, 9, nil},
		{12, "Lucky", referenceDate(2010, 6, 24), 2, 10, nil},
		{13, "Sly", referenceDate(2012, 6, 8), 1, 10, nil},
	}
	for _, value := range pets {
		owners[value.ownerID-1].Pets = append(
			owners[value.ownerID-1].Pets,
			owner.Pet{
				NamedEntity: model.NamedEntity{
					BaseEntity: model.BaseEntity{ID: value.id},
					Name:       value.name,
				},
				BirthDate: value.date,
				Type:      types[value.typeID-1],
				Visits:    value.visits,
			},
		)
	}
	return owners
}

func newOwner(
	id model.ID,
	firstName string,
	lastName string,
	address string,
	city string,
	telephone string,
) owner.Owner {
	return owner.Owner{
		Person: model.Person{
			BaseEntity: model.BaseEntity{ID: id},
			FirstName:  firstName,
			LastName:   lastName,
		},
		Address:   address,
		City:      city,
		Telephone: telephone,
	}
}

func newVisit(
	id model.ID,
	day int,
	description string,
) owner.Visit {
	return owner.Visit{
		BaseEntity:  model.BaseEntity{ID: id},
		Date:        referenceDate(2013, time.January, day),
		Description: description,
	}
}

func referenceVets() []vet.Vet {
	radiology := newSpecialty(1, "radiology")
	surgery := newSpecialty(2, "surgery")
	dentistry := newSpecialty(3, "dentistry")
	return []vet.Vet{
		newVet(1, "James", "Carter"),
		newVet(2, "Helen", "Leary", radiology),
		newVet(3, "Linda", "Douglas", surgery, dentistry),
		newVet(4, "Rafael", "Ortega", surgery),
		newVet(5, "Henry", "Stevens", radiology),
		newVet(6, "Sharon", "Jenkins"),
	}
}

func newSpecialty(id model.ID, name string) vet.Specialty {
	return vet.Specialty{NamedEntity: model.NamedEntity{
		BaseEntity: model.BaseEntity{ID: id},
		Name:       name,
	}}
}

func newVet(
	id model.ID,
	firstName string,
	lastName string,
	specialties ...vet.Specialty,
) vet.Vet {
	return vet.Vet{
		Person: model.Person{
			BaseEntity: model.BaseEntity{ID: id},
			FirstName:  firstName,
			LastName:   lastName,
		},
		Specialties: specialties,
	}
}

func referenceDate(
	year int,
	month time.Month,
	day int,
) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
