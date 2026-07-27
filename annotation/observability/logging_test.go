package observability

import (
	"context"
	"testing"

	"github.com/StevenBuglione/spice/annotation/sdk"
)

func TestLoggingDefinition(t *testing.T) {
	t.Parallel()
	if err := Logging().Validate(); err != nil {
		t.Fatalf("Logging() definition: %v", err)
	}
}

func TestObservabilityLoggingHandler(t *testing.T) {
	t.Parallel()
	result, err := ObservabilityLoggingHandler(
		context.Background(),
		sdk.Invocation{
			DescriptorPackage: "github.com/StevenBuglione/spice/annotation/observability",
			DescriptorSymbol:  "Logging",
			CanonicalName:     "observability.Logging",
		},
	)
	if err != nil || len(result.Contributions) != 1 ||
		result.Contributions[0].Bootstrap.Capability !=
			"observability.logging" {
		t.Fatalf("ObservabilityLoggingHandler() = %#v, %v", result, err)
	}
	if _, err := ObservabilityLoggingHandler(
		context.Background(),
		sdk.Invocation{
			DescriptorPackage: "example.com/wrong",
			DescriptorSymbol:  "Logging",
		},
	); err == nil {
		t.Fatal("ObservabilityLoggingHandler accepted a foreign descriptor")
	}
}
