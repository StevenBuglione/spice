package batch_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spice-framework/spice/batch"
)

func ExampleMemoryStore_restart() {
	store, err := batch.NewMemoryStore(100)
	if err != nil {
		panic(err)
	}
	loadAttempts := 0
	job, err := batch.NewJob(
		batch.Definition{
			ID:     "orders.import",
			Module: "example.com/shop/orders",
		},
		[]batch.StepSpec{
			{
				ID: "extract",
				Run: func(context.Context) error {
					fmt.Println("extract")
					return nil
				},
			},
			{
				ID: "load",
				Run: func(context.Context) error {
					loadAttempts++
					fmt.Println("load")
					if loadAttempts == 1 {
						return errors.New("database unavailable")
					}
					return nil
				},
			},
		},
	)
	if err != nil {
		panic(err)
	}
	failureContext := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), time.Second)
	}
	runner, err := batch.NewRunner(store, failureContext)
	if err != nil {
		panic(err)
	}

	first, firstErr := runner.Run(context.Background(), job, "2026-07-26")
	second, secondErr := runner.Run(context.Background(), job, "2026-07-26")
	third, thirdErr := runner.Run(context.Background(), job, "2026-07-26")
	fmt.Println(first.Attempt, first.StepsCompleted, firstErr != nil)
	fmt.Println(second.Attempt, second.StepsSkipped, secondErr)
	fmt.Println(third.Attempt, third.AlreadyComplete, thirdErr)

	// Output:
	// extract
	// load
	// load
	// 1 1 true
	// 2 1 <nil>
	// 2 true <nil>
}
