package async

import (
	"context"
	"testing"

	"github.com/StevenBuglione/spice/annotation/sdk"
)

func TestExecuteDefinition(t *testing.T) {
	t.Parallel()
	if err := Execute().Validate(); err != nil {
		t.Fatalf("Execute() definition: %v", err)
	}
}

func TestAsyncExecuteHandler(t *testing.T) {
	t.Parallel()
	result, err := AsyncExecuteHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/StevenBuglione/spice/annotation/async",
		DescriptorSymbol:  "Execute",
		CanonicalName:     "async.Execute",
	})
	if err != nil || len(result.Contributions) != 1 ||
		result.Contributions[0].Kind != sdk.ContributionAsync {
		t.Fatalf("AsyncExecuteHandler() = %#v, %v", result, err)
	}
	if _, err := AsyncExecuteHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "example.com/wrong",
		DescriptorSymbol:  "Execute",
	}); err == nil {
		t.Fatal("AsyncExecuteHandler accepted a foreign descriptor")
	}
}
