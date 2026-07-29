package otel

import (
	"context"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	spiceevent "github.com/StevenBuglione/spice/event"
	"github.com/StevenBuglione/spice/web"
)

func TestObserverComposesHTTPAndEventAdapters(t *testing.T) {
	t.Parallel()
	tracerProvider := sdktrace.NewTracerProvider()
	meterProvider := sdkmetric.NewMeterProvider()
	t.Cleanup(func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("TracerProvider.Shutdown() error = %v", err)
		}
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("MeterProvider.Shutdown() error = %v", err)
		}
	})
	observer, err := NewObserver(Options{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
	})
	if err != nil {
		t.Fatalf("NewObserver() error = %v", err)
	}
	if _, finish := observer.BeginHTTP(
		context.Background(),
		web.RouteMetadata{ID: "route", Method: "GET", Pattern: "/"},
	); finish == nil {
		t.Fatal("BeginHTTP() finish = nil")
	}
	if _, finish := observer.BeginEvent(
		context.Background(),
		spiceevent.Interaction{
			Event: spiceevent.Definition{ID: "event", Module: "publisher"},
			Subscriber: spiceevent.SubscriberMetadata{
				ID:     "subscriber",
				Module: "consumer",
			},
		},
	); finish == nil {
		t.Fatal("BeginEvent() finish = nil")
	}

	var nilObserver *Observer
	ctx := context.Background()
	if got, finish := nilObserver.BeginHTTP(ctx, web.RouteMetadata{}); got != ctx ||
		finish != nil {
		t.Fatal("nil BeginHTTP() did not preserve the context")
	}
	if got, finish := nilObserver.BeginEvent(
		ctx,
		spiceevent.Interaction{},
	); got != ctx || finish != nil {
		t.Fatal("nil BeginEvent() did not preserve the context")
	}
}
