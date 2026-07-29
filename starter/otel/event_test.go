package otel

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	spiceevent "github.com/StevenBuglione/spice/event"
)

func TestEventObserverEmitsModuleInteractionTraceAndMetrics(t *testing.T) {
	t.Parallel()
	exporter := tracetest.NewInMemoryExporter()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("TracerProvider.Shutdown() error = %v", err)
		}
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("MeterProvider.Shutdown() error = %v", err)
		}
	})

	observer, err := NewEventObserver(Options{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
	})
	if err != nil {
		t.Fatalf("NewEventObserver() error = %v", err)
	}
	interaction := spiceevent.Interaction{
		Event: spiceevent.Definition{
			ID:     "order.placed",
			Module: "example.com/shop/orders",
		},
		Subscriber: spiceevent.SubscriberMetadata{
			ID:     "inventory.reserve",
			Module: "example.com/shop/inventory",
			Order:  20,
		},
	}
	ctx, finish := observer.BeginEvent(context.Background(), interaction)
	if !trace.SpanFromContext(ctx).SpanContext().IsValid() {
		t.Fatal("BeginEvent() did not return a valid span")
	}
	result := spiceevent.Result{
		Interaction: interaction,
		Duration:    15 * time.Millisecond,
		Err:         errors.New("inventory unavailable"),
	}
	finish(result)
	finish(result)

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("len(GetSpans()) = %d, want 1", len(spans))
	}
	if spans[0].Name != "order.placed deliver" ||
		spans[0].SpanKind != trace.SpanKindInternal ||
		spans[0].Status.Code != codes.Error {
		t.Fatalf("span = %#v", spans[0])
	}
	assertAttribute(
		t,
		spans[0].Attributes,
		"spice.module.publisher",
		interaction.Event.Module,
	)
	assertAttribute(
		t,
		spans[0].Attributes,
		"spice.module.subscriber",
		interaction.Subscriber.Module,
	)
	assertAttribute(t, spans[0].Attributes, "spice.event.outcome", "error")

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &metrics); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	names := metricNames(metrics)
	for _, expected := range []string{
		"spice.event.active_deliveries",
		"spice.event.delivery.count",
		"spice.event.delivery.duration",
	} {
		if !slices.Contains(names, expected) {
			t.Fatalf("metric names = %v, missing %q", names, expected)
		}
	}
}

func TestEventObserverRejectsMissingProvidersAndHandlesNilReceiver(
	t *testing.T,
) {
	t.Parallel()
	if _, err := NewEventObserver(Options{}); err == nil {
		t.Fatal("NewEventObserver(no providers) error = nil")
	}
	tracerProvider := sdktrace.NewTracerProvider()
	t.Cleanup(func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("TracerProvider.Shutdown() error = %v", err)
		}
	})
	if _, err := NewEventObserver(Options{
		TracerProvider: tracerProvider,
	}); err == nil {
		t.Fatal("NewEventObserver(no meter provider) error = nil")
	}

	var observer *EventObserver
	ctx := context.Background()
	got, finish := observer.BeginEvent(ctx, spiceevent.Interaction{})
	if got != ctx || finish != nil {
		t.Fatal("nil BeginEvent() did not preserve the context")
	}
}
