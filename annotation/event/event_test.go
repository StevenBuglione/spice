package event

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/StevenBuglione/spice/annotation/sdk"
)

func TestEventDefinitions(t *testing.T) {
	t.Parallel()
	for _, definition := range []sdk.Definition{Listener(), Topic()} {
		if err := definition.Validate(); err != nil {
			t.Fatalf("%s definition: %v", definition.Name, err)
		}
	}
}

func TestEventHandlers(t *testing.T) {
	t.Parallel()
	topic, err := EventTopicHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/StevenBuglione/spice/annotation/event",
		DescriptorSymbol:  "Topic",
		CanonicalName:     "event.Topic",
	})
	if err != nil || len(topic.Contributions) != 1 ||
		topic.Contributions[0].Kind != sdk.ContributionEventTopic {
		t.Fatalf("EventTopicHandler() = %#v, %v", topic, err)
	}
	listener, err := EventListenerHandler(context.Background(), sdk.Invocation{
		DescriptorPackage: "github.com/StevenBuglione/spice/annotation/event",
		DescriptorSymbol:  "Listener",
		CanonicalName:     "event.Listener",
		Arguments: []sdk.InvocationArgument{{
			Name:  "order",
			Kind:  sdk.KindInteger,
			Value: json.RawMessage(`20`),
		}},
	})
	if err != nil || len(listener.Contributions) != 1 ||
		listener.Contributions[0].EventListener.Order != 20 {
		t.Fatalf("EventListenerHandler() = %#v, %v", listener, err)
	}
	for symbol, handler := range map[string]sdk.Handler{
		"Listener": EventListenerHandler,
		"Topic":    EventTopicHandler,
	} {
		if _, err := handler(context.Background(), sdk.Invocation{
			DescriptorPackage: "example.com/wrong",
			DescriptorSymbol:  symbol,
		}); err == nil {
			t.Fatalf("%s handler accepted a foreign descriptor", symbol)
		}
	}
}
