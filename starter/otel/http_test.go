package otel

import (
	"context"
	"slices"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/spice-framework/spice/web"
)

func TestHTTPObserverEmitsModuleAwareTraceAndMetrics(t *testing.T) {
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

	observer, err := NewHTTPObserver(Options{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
	})
	if err != nil {
		t.Fatalf("NewHTTPObserver() error = %v", err)
	}
	route := web.RouteMetadata{
		ID:      "route-1",
		Module:  "example.com/shop/orders",
		Method:  "POST",
		Pattern: "/orders",
	}
	ctx, finish := observer.BeginHTTP(context.Background(), route)
	if !trace.SpanFromContext(ctx).SpanContext().IsValid() {
		t.Fatal("BeginHTTP() did not return a valid server span")
	}
	result := web.HTTPResult{
		Status:   503,
		Bytes:    128,
		Duration: 25 * time.Millisecond,
	}
	finish(result)
	finish(result)

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("len(GetSpans()) = %d, want 1", len(spans))
	}
	if spans[0].Name != "POST /orders" ||
		spans[0].SpanKind != trace.SpanKindServer ||
		spans[0].Status.Code != codes.Error {
		t.Fatalf("span = %#v", spans[0])
	}
	assertAttribute(t, spans[0].Attributes, "spice.module", route.Module)
	assertAttribute(t, spans[0].Attributes, "http.route", route.Pattern)
	assertAttribute(t, spans[0].Attributes, "http.response.status_code", int64(503))

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &metrics); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	names := metricNames(metrics)
	for _, expected := range []string{
		"http.server.active_requests",
		"http.server.request.count",
		"http.server.request.duration",
		"http.server.response.body.size",
	} {
		if !slices.Contains(names, expected) {
			t.Fatalf("metric names = %v, missing %q", names, expected)
		}
	}
}

func TestHTTPObserverRejectsMissingProviders(t *testing.T) {
	t.Parallel()
	if _, err := NewHTTPObserver(Options{}); err == nil {
		t.Fatal("NewHTTPObserver(no providers) error = nil")
	}
	tracerProvider := sdktrace.NewTracerProvider()
	t.Cleanup(func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("TracerProvider.Shutdown() error = %v", err)
		}
	})
	if _, err := NewHTTPObserver(Options{TracerProvider: tracerProvider}); err == nil {
		t.Fatal("NewHTTPObserver(no meter provider) error = nil")
	}
	var typedNil *sdktrace.TracerProvider
	if _, err := NewHTTPObserver(Options{
		TracerProvider: typedNil,
		MeterProvider:  sdkmetric.NewMeterProvider(),
	}); err == nil {
		t.Fatal("NewHTTPObserver(typed-nil tracer provider) error = nil")
	}
}

func assertAttribute(
	t *testing.T,
	attributes []attribute.KeyValue,
	key string,
	want any,
) {
	t.Helper()
	for _, item := range attributes {
		if string(item.Key) != key {
			continue
		}
		if got := item.Value.AsInterface(); got != want {
			t.Fatalf("attribute %q = %#v, want %#v", key, got, want)
		}
		return
	}
	t.Fatalf("attributes = %#v, missing %q", attributes, key)
}

func metricNames(metrics metricdata.ResourceMetrics) []string {
	var names []string
	for _, scope := range metrics.ScopeMetrics {
		for _, item := range scope.Metrics {
			names = append(names, item.Name)
		}
	}
	slices.Sort(names)
	return names
}
