package cache_test

import (
	"context"
	"fmt"
	"time"

	"github.com/spice-framework/spice/cache"
)

func ExampleMemory() {
	memory, err := cache.NewMemory[string, string](
		cache.Definition{ID: "products.by-id", Module: "example.com/shop/products"},
		100,
		nil,
	)
	if err != nil {
		fmt.Printf("construct: %v\n", err)
		return
	}
	err = memory.Put(context.Background(), "sku-1", "coffee", time.Minute)
	if err != nil {
		fmt.Printf("put: %v\n", err)
		return
	}
	value, found, err := memory.Get(context.Background(), "sku-1")
	fmt.Printf("value=%s found=%v err=%v\n", value, found, err)
	// Output:
	// value=coffee found=true err=<nil>
}
