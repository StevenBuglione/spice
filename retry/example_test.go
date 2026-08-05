package retry_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/spice-framework/spice/retry"
)

func ExampleRun() {
	transient := errors.New("service unavailable")
	calls := 0
	err := retry.Run(
		context.Background(),
		retry.Policy{
			ID:          "inventory.Refresh",
			Module:      "example.com/shop/inventory",
			MaxAttempts: 3,
			Retryable: func(err error) bool {
				return errors.Is(err, transient)
			},
		},
		func(_ context.Context, _ retry.Attempt) error {
			calls++
			if calls < 3 {
				return transient
			}
			return nil
		},
	)
	fmt.Printf("calls=%d err=%v\n", calls, err)
	// Output:
	// calls=3 err=<nil>
}
