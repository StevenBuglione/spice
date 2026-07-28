package model

import (
	"strings"
	"testing"
)

func TestEntityIdentity(t *testing.T) {
	t.Parallel()

	if !(BaseEntity{}).New() {
		t.Fatal("zero-value entity should be new")
	}
	if (BaseEntity{ID: 1}).New() {
		t.Fatal("positive identity should be persisted")
	}
	if ID(-1).Valid() || !ID(1).Valid() {
		t.Fatal("ID validity did not enforce positive values")
	}
}

func TestPersonValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value Person
		count int
	}{
		{
			name:  "valid",
			value: Person{FirstName: "James", LastName: "Carter"},
		},
		{
			name:  "blank",
			value: Person{FirstName: " ", LastName: ""},
			count: 2,
		},
		{
			name: "too long",
			value: Person{
				FirstName: strings.Repeat("a", 31),
				LastName:  strings.Repeat("b", 31),
			},
			count: 2,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result, err := test.value.Validate()
			if err != nil {
				t.Fatal(err)
			}
			if result.Len() != test.count {
				t.Fatalf("violations = %#v", result.All())
			}
		})
	}
}
