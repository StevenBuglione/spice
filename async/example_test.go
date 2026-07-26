package async_test

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/StevenBuglione/spice/async"
)

func ExampleExecutor() {
	executor, err := async.NewExecutor(context.Background(), 4)
	if err != nil {
		fmt.Printf("construct: %v\n", err)
		return
	}
	var completed atomic.Bool
	err = executor.Submit(
		context.Background(),
		async.Definition{ID: "orders.Notify", Module: "example.com/shop/orders"},
		func(context.Context) error {
			completed.Store(true)
			return nil
		},
	)
	if err != nil {
		fmt.Printf("submit: %v\n", err)
		return
	}
	err = executor.Shutdown(context.Background())
	fmt.Printf("completed=%v err=%v\n", completed.Load(), err)
	// Output:
	// completed=true err=<nil>
}
