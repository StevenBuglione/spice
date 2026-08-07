package retry

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/spice-framework/spice/annotation/sdk"
)

func TestRetryableDefinitionAndHandler(t *testing.T) {
	t.Parallel()
	if err := Retryable().Validate(); err != nil {
		t.Fatalf("Retryable() definition: %v", err)
	}
	result, err := RetryableHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/spice-framework/spice/annotation/retry",
		DescriptorSymbol:  "Retryable",
		CanonicalName:     "retry.Retryable",
		Arguments: []sdk.InvocationArgument{
			{Name: "maxAttempts", Kind: sdk.KindInteger, Value: json.RawMessage(`5`)},
			{Name: "initialBackoff", Kind: sdk.KindString, Value: json.RawMessage(`"10ms"`)},
			{Name: "maxBackoff", Kind: sdk.KindString, Value: json.RawMessage(`"250ms"`)},
			{Name: "multiplier", Kind: sdk.KindInteger, Value: json.RawMessage(`3`)},
			{Name: "classifier", Kind: sdk.KindIdentifier, Value: json.RawMessage(`"IsTransient"`)},
		},
	})
	if err != nil || len(result.Contributions) != 1 {
		t.Fatalf("RetryableHandler() = %#v, %v", result, err)
	}
	retry := result.Contributions[0].Retry
	if retry == nil || retry.MaxAttempts != 5 || retry.InitialBackoff != "10ms" ||
		retry.MaxBackoff != "250ms" || retry.Multiplier != 3 || retry.Classifier != "IsTransient" {
		t.Fatalf("RetryableHandler() contribution = %#v", retry)
	}
}

func TestRetryableHandlerDefaultsAndRejectsForeignDescriptor(t *testing.T) {
	t.Parallel()
	result, err := RetryableHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/spice-framework/spice/annotation/retry",
		DescriptorSymbol:  "Retryable",
		CanonicalName:     "retry.Retryable",
	})
	if err != nil {
		t.Fatal(err)
	}
	retry := result.Contributions[0].Retry
	if retry.MaxAttempts != 3 || retry.InitialBackoff != "100ms" ||
		retry.MaxBackoff != "1s" || retry.Multiplier != 2 {
		t.Fatalf("RetryableHandler() defaults = %#v", retry)
	}
	if _, err := RetryableHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "example.com/wrong",
		DescriptorSymbol:  "Retryable",
	}); err == nil {
		t.Fatal("RetryableHandler accepted a foreign descriptor")
	}
}
