package modulith

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/spice-framework/spice/annotation/sdk"
)

func TestModulithDefinitions(t *testing.T) {
	t.Parallel()
	for _, definition := range []sdk.Definition{Module(), NamedInterface()} {
		if err := definition.Validate(); err != nil {
			t.Fatalf("%s definition: %v", definition.Name, err)
		}
	}
}

func TestModulithHandlers(t *testing.T) {
	t.Parallel()
	module, err := ModuleHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/spice-framework/spice/annotation/modulith",
		DescriptorSymbol:  "Module",
		CanonicalName:     "Module",
		Arguments: []sdk.InvocationArgument{{
			Name:  "allowedDependencies",
			Kind:  sdk.KindList,
			Value: json.RawMessage(`["inventory::api"]`),
		}},
	})
	if err != nil || len(module.Contributions) != 1 ||
		len(module.Contributions[0].Module.AllowedDependencies) != 1 {
		t.Fatalf("ModuleHandler() = %#v, %v", module, err)
	}
	named, err := NamedInterfaceHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/spice-framework/spice/annotation/modulith",
		DescriptorSymbol:  "NamedInterface",
		CanonicalName:     "NamedInterface",
		Arguments: []sdk.InvocationArgument{{
			Kind:       sdk.KindString,
			Positional: true,
			Value:      json.RawMessage(`"api"`),
		}},
	})
	if err != nil || len(named.Contributions) != 1 ||
		named.Contributions[0].NamedInterface.Name != "api" {
		t.Fatalf("NamedInterfaceHandler() = %#v, %v", named, err)
	}
	for symbol, handler := range map[string]sdk.Handler{
		"Module":         ModuleHandler,
		"NamedInterface": NamedInterfaceHandler,
	} {
		if _, err := handler(context.Background(), sdk.Invocation{
			DescriptorPackage: "example.com/wrong",
			DescriptorSymbol:  symbol,
		}); err == nil {
			t.Fatalf("%s handler accepted a foreign descriptor", symbol)
		}
	}
}
