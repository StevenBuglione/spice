package cache

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/spice-framework/spice/annotation/sdk"
)

func TestCacheableDefinition(t *testing.T) {
	t.Parallel()
	if err := Cacheable().Validate(); err != nil {
		t.Fatalf("Cacheable() definition: %v", err)
	}
}

func TestCacheableHandler(t *testing.T) {
	t.Parallel()
	result, err := CacheableHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/spice-framework/spice/annotation/cache",
		DescriptorSymbol:  "Cacheable",
		CanonicalName:     "cache.Cacheable",
		Arguments: []sdk.InvocationArgument{{
			Name:  "name",
			Kind:  sdk.KindString,
			Value: json.RawMessage(`"orders.by-id"`),
		}},
	})
	if err != nil || len(result.Contributions) != 1 ||
		result.Contributions[0].Cache.Name != "orders.by-id" {
		t.Fatalf("CacheableHandler() = %#v, %v", result, err)
	}
	if _, err := CacheableHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "example.com/wrong",
		DescriptorSymbol:  "Cacheable",
	}); err == nil {
		t.Fatal("CacheableHandler accepted a foreign descriptor")
	}
}
