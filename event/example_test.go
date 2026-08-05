package event_test

import (
	"context"
	"fmt"

	"github.com/spice-framework/spice/event"
)

type orderPlaced struct {
	ID string
}

func ExampleTopic() {
	topic, err := event.NewTopic(
		event.Definition{
			ID:     "orders.OrderPlaced",
			Module: "example.com/shop/orders",
		},
		[]event.Subscriber[orderPlaced]{
			{
				ID:     "inventory.Reserve",
				Module: "example.com/shop/inventory",
				Handle: func(_ context.Context, placed orderPlaced) error {
					fmt.Printf("inventory reserved %s\n", placed.ID)
					return nil
				},
			},
		},
	)
	if err != nil {
		fmt.Printf("construct topic: %v\n", err)
		return
	}
	if err := topic.Publish(context.Background(), orderPlaced{ID: "order-1"}); err != nil {
		fmt.Printf("publish: %v\n", err)
	}
	// Output:
	// inventory reserved order-1
}
