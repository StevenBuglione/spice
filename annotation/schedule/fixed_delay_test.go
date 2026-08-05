package schedule

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/spice-framework/spice/annotation/sdk"
)

func TestFixedDelayDefinition(t *testing.T) {
	t.Parallel()
	if err := FixedDelay().Validate(); err != nil {
		t.Fatalf("FixedDelay() definition: %v", err)
	}
}

func TestFixedDelayHandler(t *testing.T) {
	t.Parallel()
	result, err := FixedDelayHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/spice-framework/spice/annotation/schedule",
		DescriptorSymbol:  "FixedDelay",
		CanonicalName:     "schedule.FixedDelay",
		Arguments: []sdk.InvocationArgument{
			{
				Name:  "delay",
				Kind:  sdk.KindString,
				Value: json.RawMessage(`"5s"`),
			},
			{
				Name:  "initialDelay",
				Kind:  sdk.KindString,
				Value: json.RawMessage(`"1s"`),
			},
			{
				Name:  "continueOnError",
				Kind:  sdk.KindBoolean,
				Value: json.RawMessage(`true`),
			},
		},
	})
	if err != nil || len(result.Contributions) != 1 ||
		result.Contributions[0].Schedule.Delay != "5s" ||
		result.Contributions[0].Schedule.InitialDelay != "1s" ||
		!result.Contributions[0].Schedule.ContinueOnError {
		t.Fatalf("FixedDelayHandler() = %#v, %v", result, err)
	}
	if _, err := FixedDelayHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "example.com/wrong",
		DescriptorSymbol:  "FixedDelay",
	}); err == nil {
		t.Fatal("FixedDelayHandler accepted a foreign descriptor")
	}
}
