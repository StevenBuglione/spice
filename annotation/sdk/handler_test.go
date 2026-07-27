package sdk

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestInvocationHelpersDecodeTypedArguments(t *testing.T) {
	t.Parallel()
	invocation := Invocation{
		DescriptorPackage: "example.com/annotation",
		DescriptorSymbol:  "Example",
		CanonicalName:     "fixture.Example",
		Arguments: []InvocationArgument{
			{
				Positional: true,
				Kind:       KindString,
				Value:      json.RawMessage(`"primary"`),
			},
			{
				Name:  "enabled",
				Kind:  KindBoolean,
				Value: json.RawMessage(`true`),
			},
			{
				Name:  "order",
				Kind:  KindInteger,
				Value: json.RawMessage(`7`),
			},
			{
				Name:  "values",
				Kind:  KindList,
				Value: json.RawMessage(`["one","two"]`),
			},
		},
	}
	if err := invocation.RequireDescriptor(
		"example.com/annotation",
		"Example",
	); err != nil {
		t.Fatalf("RequireDescriptor() error = %v", err)
	}
	arguments, err := BindArguments(
		invocation,
		"name",
		"name",
		"enabled",
		"order",
		"values",
	)
	if err != nil {
		t.Fatalf("BindArguments() error = %v", err)
	}
	name, nameErr := arguments.String("name", true)
	enabled, enabledErr := arguments.Boolean("enabled")
	order, orderErr := arguments.Integer("order")
	values, valuesErr := arguments.Strings("values")
	if nameErr != nil || enabledErr != nil || orderErr != nil ||
		valuesErr != nil {
		t.Fatalf(
			"decoded argument errors = %v, %v, %v, %v",
			nameErr,
			enabledErr,
			orderErr,
			valuesErr,
		)
	}
	if name != "primary" || !enabled || order != 7 ||
		len(values) != 2 || values[1] != "two" {
		t.Fatalf(
			"decoded arguments = %q, %t, %d, %q",
			name,
			enabled,
			order,
			values,
		)
	}
}

func TestInvocationHelpersFailClosed(t *testing.T) {
	t.Parallel()
	if err := (Invocation{}).RequireDescriptor(
		"example.com/annotation",
		"Example",
	); err == nil {
		t.Fatal("RequireDescriptor() error = nil")
	}
	for _, test := range []struct {
		name      string
		arguments BoundArguments
		decode    func(BoundArguments) error
		message   string
	}{
		{
			name:      "required",
			arguments: BoundArguments{},
			decode: func(values BoundArguments) error {
				_, err := values.String("name", true)
				return err
			},
			message: "required",
		},
		{
			name: "trailing JSON",
			arguments: BoundArguments{"name": {
				Kind:  KindString,
				Value: json.RawMessage(`"one" "two"`),
			}},
			decode: func(values BoundArguments) error {
				_, err := values.String("name", false)
				return err
			},
			message: "multiple JSON",
		},
		{
			name: "wrong list kind",
			arguments: BoundArguments{"values": {
				Kind:  KindString,
				Value: json.RawMessage(`"one"`),
			}},
			decode: func(values BoundArguments) error {
				_, err := values.Strings("values")
				return err
			},
			message: "must be a list",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := test.decode(test.arguments)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("decode error = %v, want %q", err, test.message)
			}
		})
	}
}

func TestOneContributionValidatesValue(t *testing.T) {
	t.Parallel()
	result, err := OneContribution(Contribution{
		Kind:        ContributionApplication,
		Application: &ApplicationContribution{},
	})
	if err != nil || len(result.Contributions) != 1 {
		t.Fatalf("OneContribution() = %+v, %v", result, err)
	}
	if _, err := OneContribution(Contribution{}); err == nil {
		t.Fatal("OneContribution(invalid) error = nil")
	}
}
