package lifecycle

import (
	"context"
	"testing"

	"github.com/spice-framework/spice/annotation/sdk"
)

func TestLifecycleDefinitions(t *testing.T) {
	t.Parallel()
	for _, definition := range []sdk.Definition{OnStart(), OnStop()} {
		if err := definition.Validate(); err != nil {
			t.Fatalf("%s definition: %v", definition.Name, err)
		}
	}
}

func TestLifecycleHandlers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		symbol  string
		handler sdk.Handler
		phase   sdk.LifecyclePhase
	}{
		{"OnStart", OnStartHandler, sdk.LifecycleStart},
		{"OnStop", OnStopHandler, sdk.LifecycleStop},
	}
	for _, test := range tests {
		result, err := test.handler(context.Background(), sdk.Invocation{
			DescriptorPackage: "github.com/spice-framework/spice/annotation/lifecycle",
			DescriptorSymbol:  test.symbol,
			CanonicalName:     test.symbol,
		})
		if err != nil || len(result.Contributions) != 1 ||
			result.Contributions[0].Lifecycle.Phase != test.phase {
			t.Fatalf("%s handler = %#v, %v", test.symbol, result, err)
		}
		if _, err := test.handler(context.Background(), sdk.Invocation{
			DescriptorPackage: "example.com/wrong",
			DescriptorSymbol:  test.symbol,
		}); err == nil {
			t.Fatalf("%s handler accepted a foreign descriptor", test.symbol)
		}
	}
}
