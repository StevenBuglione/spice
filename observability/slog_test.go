package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/spice-framework/spice/lifecycle"
	"github.com/spice-framework/spice/web"
)

func TestSlogHTTPObserverWritesBoundedRouteMetadata(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	logger := testLogger(&output)
	observer, err := NewSlogHTTPObserver(logger)
	if err != nil {
		t.Fatalf("NewSlogHTTPObserver() error = %v", err)
	}
	route := web.RouteMetadata{
		ID:      "route-1",
		Module:  "example.com/shop/orders",
		Method:  "POST",
		Pattern: "/orders/{id}",
	}
	ctx, finish := observer.BeginHTTP(context.Background(), route)
	if ctx == nil || finish == nil {
		t.Fatal("BeginHTTP() returned nil context or finisher")
	}
	finish(web.HTTPResult{
		Status:   409,
		Bytes:    42,
		Duration: 3 * time.Millisecond,
	})

	record := decodeRecord(t, output.Bytes())
	assertRecordValue(t, record, "level", "WARN")
	assertRecordValue(t, record, "event", "http.server.request")
	assertRecordValue(t, record, "module", route.Module)
	assertRecordValue(t, record, "http_route", route.Pattern)
	assertRecordValue(t, record, "http_status", float64(409))
	if bytes.Contains(output.Bytes(), []byte("customer-secret")) {
		t.Fatal("HTTP log unexpectedly contains client data")
	}
}

func TestSlogLifecycleObserverWritesFailureContext(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	observer, err := NewSlogLifecycleObserver(testLogger(&output))
	if err != nil {
		t.Fatalf("NewSlogLifecycleObserver() error = %v", err)
	}
	observer(context.Background(), lifecycle.Observation{
		Module:    "example.com/shop/orders",
		Component: "orders.Service",
		Operation: lifecycle.OperationStart,
		Phase:     lifecycle.PhaseEnd,
		Err:       errors.New("database unavailable"),
	})

	record := decodeRecord(t, output.Bytes())
	assertRecordValue(t, record, "level", "ERROR")
	assertRecordValue(t, record, "event", "application.lifecycle")
	assertRecordValue(t, record, "operation", "start")
	assertRecordValue(t, record, "error", "database unavailable")
}

func TestSlogObserversRejectNilLogger(t *testing.T) {
	t.Parallel()
	if _, err := NewSlogHTTPObserver(nil); err == nil {
		t.Fatal("NewSlogHTTPObserver(nil) error = nil")
	}
	if _, err := NewSlogLifecycleObserver(nil); err == nil {
		t.Fatal("NewSlogLifecycleObserver(nil) error = nil")
	}
}

func testLogger(output *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func decodeRecord(t *testing.T, content []byte) map[string]any {
	t.Helper()
	var record map[string]any
	if err := json.Unmarshal(content, &record); err != nil {
		t.Fatalf("Unmarshal(log) error = %v; content = %q", err, content)
	}
	return record
}

func assertRecordValue(t *testing.T, record map[string]any, key string, want any) {
	t.Helper()
	if got := record[key]; got != want {
		t.Fatalf("record[%q] = %#v, want %#v; record = %#v", key, got, want, record)
	}
}
