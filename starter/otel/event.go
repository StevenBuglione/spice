package otel

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	spiceevent "github.com/StevenBuglione/spice/event"
)

// EventObserver creates one bounded span and metric set for each generated
// publisher-to-subscriber interaction. It records identities owned by the
// compiler and never records event payloads or error text.
type EventObserver struct {
	tracer     trace.Tracer
	deliveries metric.Int64Counter
	active     metric.Int64UpDownCounter
	duration   metric.Float64Histogram
}

// NewEventObserver constructs the OpenTelemetry event adapter. Both providers
// are required so enabling it cannot silently emit partial telemetry.
func NewEventObserver(options Options) (*EventObserver, error) {
	if nilProvider(options.TracerProvider) {
		return nil, fmt.Errorf("construct OpenTelemetry event observer: tracer provider is nil")
	}
	if nilProvider(options.MeterProvider) {
		return nil, fmt.Errorf("construct OpenTelemetry event observer: meter provider is nil")
	}
	meter := options.MeterProvider.Meter(instrumentationName)
	deliveries, err := meter.Int64Counter(
		"spice.event.delivery.count",
		metric.WithUnit("{delivery}"),
		metric.WithDescription("Completed generated event deliveries."),
	)
	if err != nil {
		return nil, fmt.Errorf("create event delivery counter: %w", err)
	}
	active, err := meter.Int64UpDownCounter(
		"spice.event.active_deliveries",
		metric.WithUnit("{delivery}"),
		metric.WithDescription("Generated event deliveries currently executing."),
	)
	if err != nil {
		return nil, fmt.Errorf("create active event delivery counter: %w", err)
	}
	duration, err := meter.Float64Histogram(
		"spice.event.delivery.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Generated event delivery duration."),
	)
	if err != nil {
		return nil, fmt.Errorf("create event delivery duration histogram: %w", err)
	}
	return &EventObserver{
		tracer:     options.TracerProvider.Tracer(instrumentationName),
		deliveries: deliveries,
		active:     active,
		duration:   duration,
	}, nil
}

// BeginEvent implements event.Observer.
func (observer *EventObserver) BeginEvent(
	ctx context.Context,
	interaction spiceevent.Interaction,
) (context.Context, func(spiceevent.Result)) {
	if observer == nil {
		return ctx, nil
	}
	interactionAttributes := eventInteractionAttributes(interaction)
	measurement := metric.WithAttributeSet(
		attribute.NewSet(interactionAttributes...),
	)
	spanContext, span := observer.tracer.Start(
		ctx,
		interaction.Event.ID+" deliver",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(interactionAttributes...),
	)
	observer.active.Add(spanContext, 1, measurement)

	var once sync.Once
	return spanContext, func(result spiceevent.Result) {
		once.Do(func() {
			resultAttributes := append(
				[]attribute.KeyValue(nil),
				interactionAttributes...,
			)
			outcome := "success"
			if result.Panicked {
				outcome = "panic"
			} else if result.Err != nil {
				outcome = "error"
			}
			resultAttributes = append(
				resultAttributes,
				attribute.String("spice.event.outcome", outcome),
			)
			resultMeasurement := metric.WithAttributeSet(
				attribute.NewSet(resultAttributes...),
			)
			observer.deliveries.Add(spanContext, 1, resultMeasurement)
			observer.active.Add(spanContext, -1, measurement)
			observer.duration.Record(
				spanContext,
				result.Duration.Seconds(),
				resultMeasurement,
			)
			span.SetAttributes(resultAttributes...)
			if result.Err != nil || result.Panicked {
				span.SetStatus(codes.Error, "event delivery failed")
			}
			span.End()
		})
	}
}

func eventInteractionAttributes(
	interaction spiceevent.Interaction,
) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("spice.event.id", interaction.Event.ID),
		attribute.String(
			"spice.module.publisher",
			interaction.Event.Module,
		),
		attribute.String(
			"spice.module.subscriber",
			interaction.Subscriber.Module,
		),
		attribute.String(
			"spice.event.subscriber.id",
			interaction.Subscriber.ID,
		),
		attribute.Int(
			"spice.event.subscriber.order",
			interaction.Subscriber.Order,
		),
	}
}

var _ spiceevent.Observer = (*EventObserver)(nil)
