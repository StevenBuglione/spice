package validation

import (
	"slices"
	"strings"
	"testing"
)

func TestErrorsPreserveImmutableOrderedViolations(t *testing.T) {
	t.Parallel()

	first, err := Field("owner.firstName", "required", "is required")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Object("owner.invalid", "owner is invalid")
	if err != nil {
		t.Fatal(err)
	}
	result, err := New(first)
	if err != nil {
		t.Fatal(err)
	}
	extended, err := result.Add(second)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(result.All(), []Violation{first}) ||
		!slices.Equal(extended.All(), []Violation{first, second}) ||
		result.Len() != 1 ||
		extended.Valid() {
		t.Fatalf("result=%#v extended=%#v", result.All(), extended.All())
	}
	exposed := extended.All()
	exposed[0].Message = "changed"
	if extended.All()[0] != first {
		t.Fatal("All() exposed mutable state")
	}
	if got := extended.ForField("owner.firstName"); !slices.Equal(
		got,
		[]Violation{first},
	) {
		t.Fatalf("ForField() = %#v", got)
	}
}

func TestErrorsRejectUnsafeOrUnboundedViolations(t *testing.T) {
	t.Parallel()

	tests := []Violation{
		{Field: "bad field", Code: "required", Message: "required"},
		{Field: "field", Code: "", Message: "required"},
		{Field: "field", Code: "bad code", Message: "required"},
		{Field: "field", Code: "required", Message: " surrounding "},
		{Field: strings.Repeat("x", maxFieldBytes+1), Code: "required", Message: "required"},
		{Field: "field", Code: strings.Repeat("x", maxCodeBytes+1), Message: "required"},
		{Field: "field", Code: "required", Message: strings.Repeat("x", maxMessageBytes+1)},
	}
	for _, violation := range tests {
		if _, err := New(violation); err == nil {
			t.Fatalf("New(%#v) error = nil", violation)
		}
	}
	tooMany := make([]Violation, maxViolations+1)
	for index := range tooMany {
		tooMany[index] = Violation{Code: "invalid", Message: "invalid"}
	}
	if _, err := New(tooMany...); err == nil {
		t.Fatal("New(too many) error = nil")
	}
}

func TestJoinPreservesInputOrder(t *testing.T) {
	t.Parallel()

	one, err := New(Violation{Code: "one", Message: "one"})
	if err != nil {
		t.Fatal(err)
	}
	two, err := New(Violation{Code: "two", Message: "two"})
	if err != nil {
		t.Fatal(err)
	}
	joined, err := Join(one, two)
	if err != nil {
		t.Fatal(err)
	}
	if got := joined.All(); got[0].Code != "one" || got[1].Code != "two" {
		t.Fatalf("Join() = %#v", got)
	}
}
