package sdk

import (
	"strings"
	"testing"
)

func TestDefinitionValidate(t *testing.T) {
	definition := validDefinition()
	if err := definition.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestDefinitionValidateRejectsInvalidMetadata(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*Definition)
		message string
	}{
		{
			name:    "name",
			change:  func(value *Definition) { value.Name = "not valid" },
			message: "qualified Go identifier",
		},
		{
			name:    "summary",
			change:  func(value *Definition) { value.Summary = "" },
			message: "requires a summary",
		},
		{
			name:    "targets",
			change:  func(value *Definition) { value.Targets = nil },
			message: "at least one target",
		},
		{
			name: "argument docs",
			change: func(value *Definition) {
				value.Arguments[0].Description = ""
			},
			message: "requires documentation",
		},
		{
			name: "list kinds",
			change: func(value *Definition) {
				value.Arguments[0].ListElementKinds = []Kind{KindString}
			},
			message: "without accepting lists",
		},
		{
			name: "protocol",
			change: func(value *Definition) {
				value.Implementation.Protocol = "future"
			},
			message: "unsupported protocol",
		},
		{
			name: "compatibility",
			change: func(value *Definition) {
				value.Compatibility.MinimumSpice = ""
			},
			message: "compatibility versions",
		},
		{
			name: "examples",
			change: func(value *Definition) {
				value.Examples = nil
			},
			message: "documented example",
		},
		{
			name: "source",
			change: func(value *Definition) {
				value.Implementation.Source.Package = "."
			},
			message: "implementation source",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := validDefinition()
			test.change(&definition)
			err := definition.Validate()
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("Validate() error = %v, want %q", err, test.message)
			}
		})
	}
}

func validDefinition() Definition {
	return Definition{
		Name:    "web.Controller",
		Summary: "Marks an HTTP controller.",
		Targets: []Target{TargetType},
		Arguments: []Argument{{
			Name:        "prefix",
			Kinds:       []Kind{KindString},
			Description: "Optional route prefix.",
		}},
		Examples: []Example{{
			Title: "Controller",
			Code:  "// @Controller",
		}},
		Compatibility: Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: Implementation{
			Tool:     "example.com/plugin/cmd/annotations",
			Handler:  "web/controller",
			Protocol: ProtocolV1Alpha1,
			Source: Symbol{
				Package: "example.com/plugin/internal/web",
				Name:    "ControllerHandler",
			},
		},
	}
}
