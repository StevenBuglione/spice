package observability

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/spice-framework/spice/annotation/sdk"
)

func TestObservedDefinitionAndHandler(t *testing.T) {
	t.Parallel()
	if err := Observed().Validate(); err != nil {
		t.Fatalf("Observed() definition: %v", err)
	}
	result, err := ObservedHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/spice-framework/spice/annotation/observability",
		DescriptorSymbol:  "Observed",
		CanonicalName:     "observability.Observed",
		Arguments: []sdk.InvocationArgument{{
			Name: "name", Kind: sdk.KindString, Value: json.RawMessage(`"orders.create"`),
		}},
	})
	if err != nil || len(result.Contributions) != 1 ||
		result.Contributions[0].Observation.Name != "orders.create" {
		t.Fatalf("ObservedHandler() = %#v, %v", result, err)
	}
	if _, err := ObservedHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "example.com/wrong",
		DescriptorSymbol:  "Observed",
	}); err == nil {
		t.Fatal("ObservedHandler accepted a foreign descriptor")
	}
}
