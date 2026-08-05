// Package otel provides opt-in OpenTelemetry adapters for generated Spice HTTP
// routes and typed module-event interactions. It uses caller-owned providers
// and never installs global state.
package otel

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/spice-framework/spice/web"
)

const instrumentationName = "github.com/spice-framework/spice/starter/otel"

// Options supplies the application-owned OpenTelemetry providers.
type Options struct {
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

// HTTPObserver creates module-aware server spans and bounded route metrics.
type HTTPObserver struct {
	tracer        trace.Tracer
	requests      metric.Int64Counter
	active        metric.Int64UpDownCounter
	duration      metric.Float64Histogram
	responseBytes metric.Int64Histogram
}

// NewHTTPObserver constructs the OpenTelemetry HTTP adapter. Both providers
// are required so enabling the starter cannot silently emit partial telemetry.
func NewHTTPObserver(options Options) (*HTTPObserver, error) {
	if nilProvider(options.TracerProvider) {
		return nil, errors.New("construct OpenTelemetry HTTP observer: tracer provider is nil")
	}
	if nilProvider(options.MeterProvider) {
		return nil, errors.New("construct OpenTelemetry HTTP observer: meter provider is nil")
	}
	meter := options.MeterProvider.Meter(instrumentationName)
	requests, err := meter.Int64Counter(
		"http.server.request.count",
		metric.WithUnit("{request}"),
		metric.WithDescription("Completed generated HTTP requests."),
	)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request counter: %w", err)
	}
	active, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithUnit("{request}"),
		metric.WithDescription("Generated HTTP requests currently executing."),
	)
	if err != nil {
		return nil, fmt.Errorf("create active HTTP request counter: %w", err)
	}
	duration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Generated HTTP request duration."),
	)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request duration histogram: %w", err)
	}
	responseBytes, err := meter.Int64Histogram(
		"http.server.response.body.size",
		metric.WithUnit("By"),
		metric.WithDescription("Generated HTTP response body size."),
	)
	if err != nil {
		return nil, fmt.Errorf("create HTTP response body histogram: %w", err)
	}
	return &HTTPObserver{
		tracer:        options.TracerProvider.Tracer(instrumentationName),
		requests:      requests,
		active:        active,
		duration:      duration,
		responseBytes: responseBytes,
	}, nil
}

func nilProvider(provider any) bool {
	if provider == nil {
		return true
	}
	value := reflect.ValueOf(provider)
	return (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) && value.IsNil()
}

// BeginHTTP implements web.HTTPObserver.
func (observer *HTTPObserver) BeginHTTP(
	ctx context.Context,
	route web.RouteMetadata,
) (context.Context, func(web.HTTPResult)) {
	if observer == nil {
		return ctx, nil
	}
	routeAttributes := []attribute.KeyValue{
		attribute.String("spice.route.id", route.ID),
		attribute.String("spice.module", route.Module),
		attribute.String("http.request.method", route.Method),
		attribute.String("http.route", route.Pattern),
	}
	measurement := metric.WithAttributeSet(attribute.NewSet(routeAttributes...))
	spanContext, span := observer.tracer.Start(
		ctx,
		route.Method+" "+route.Pattern,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(routeAttributes...),
	)
	observer.active.Add(spanContext, 1, measurement)

	var once sync.Once
	return spanContext, func(result web.HTTPResult) {
		once.Do(func() {
			resultAttributes := append(
				append([]attribute.KeyValue(nil), routeAttributes...),
				attribute.Int("http.response.status_code", result.Status),
			)
			if result.Panicked {
				resultAttributes = append(resultAttributes, attribute.String("error.type", "panic"))
			}
			resultMeasurement := metric.WithAttributeSet(attribute.NewSet(resultAttributes...))
			observer.requests.Add(spanContext, 1, resultMeasurement)
			observer.active.Add(spanContext, -1, measurement)
			observer.duration.Record(spanContext, result.Duration.Seconds(), resultMeasurement)
			observer.responseBytes.Record(spanContext, result.Bytes, resultMeasurement)
			span.SetAttributes(resultAttributes...)
			if result.Panicked || result.Status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, "HTTP request failed")
			}
			span.End()
		})
	}
}

var _ web.HTTPObserver = (*HTTPObserver)(nil)
