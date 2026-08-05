package schedule_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/spice-framework/spice/schedule"
)

func ExampleScheduler() {
	lifetime, cancel := context.WithCancel(context.Background())
	var runs atomic.Uint64
	scheduler, err := schedule.New(lifetime, []schedule.Job{{
		Definition: schedule.Definition{
			ID:     "inventory.Refresh",
			Module: "example.com/shop/inventory",
		},
		Delay: time.Minute,
		Run: func(context.Context) error {
			runs.Add(1)
			cancel()
			return nil
		},
	}}, nil)
	if err != nil {
		fmt.Printf("construct: %v\n", err)
		return
	}
	err = scheduler.Start(context.Background())
	if err == nil {
		<-scheduler.Done()
		err = scheduler.Shutdown(context.Background())
	}
	fmt.Printf("runs=%d err=%v\n", runs.Load(), err)
	// Output:
	// runs=1 err=<nil>
}
