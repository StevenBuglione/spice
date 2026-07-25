package management

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/StevenBuglione/spice/web"
)

func TestHTTPMetricsSnapshotsConcurrentGeneratedRoutes(t *testing.T) {
	metrics := NewHTTPMetrics()
	route := web.RouteMetadata{
		ID:      "orders-get",
		Module:  "example.com/commerce/orders",
		Method:  http.MethodGet,
		Pattern: "/orders/{id}",
	}
	ctx := context.WithValue(context.Background(), metricsContextKey{}, "preserved")
	observedContext, finish := metrics.BeginHTTP(ctx, route)
	if observedContext.Value(metricsContextKey{}) != "preserved" {
		t.Fatal("BeginHTTP() did not preserve context")
	}
	snapshot := metrics.Snapshot()
	if len(snapshot.Routes) != 1 ||
		snapshot.Routes[0].Requests != 1 ||
		snapshot.Routes[0].InFlight != 1 {
		t.Fatalf("in-flight snapshot = %#v", snapshot)
	}
	finish(web.HTTPResult{
		Status:   http.StatusOK,
		Bytes:    12,
		Duration: 3 * time.Millisecond,
	})
	finish(web.HTTPResult{Status: http.StatusInternalServerError, Panicked: true})

	const concurrentRequests = 40
	var wait sync.WaitGroup
	for index := range concurrentRequests {
		wait.Go(func() {
			_, complete := metrics.BeginHTTP(context.Background(), route)
			status := http.StatusOK
			panicked := false
			if index%2 != 0 {
				status = http.StatusInternalServerError
				panicked = true
			}
			complete(web.HTTPResult{
				Status:   status,
				Bytes:    5,
				Duration: time.Millisecond,
				Panicked: panicked,
			})
		})
	}
	wait.Wait()

	value := metrics.Snapshot().Routes[0]
	if value.Requests != concurrentRequests+1 ||
		value.InFlight != 0 ||
		value.Bytes != 12+concurrentRequests*5 ||
		value.TotalDurationNanos != int64(3*time.Millisecond)+concurrentRequests*int64(time.Millisecond) ||
		value.MaxDurationNanos != int64(3*time.Millisecond) ||
		value.Panics != concurrentRequests/2 ||
		len(value.Responses) != 2 ||
		value.Responses[0] != (StatusCount{Status: http.StatusOK, Count: concurrentRequests/2 + 1}) ||
		value.Responses[1] != (StatusCount{Status: http.StatusInternalServerError, Count: concurrentRequests / 2}) {
		t.Fatalf("completed metrics = %#v", value)
	}
}

func TestHTTPMetricsSortsRoutesAndSupportsZeroValues(t *testing.T) {
	var zero HTTPMetrics
	routes := []web.RouteMetadata{
		{ID: "second", Module: "example.com/b", Method: http.MethodGet, Pattern: "/same"},
		{ID: "post", Module: "example.com/a", Method: http.MethodPost, Pattern: "/same"},
		{ID: "first", Module: "example.com/a", Method: http.MethodGet, Pattern: "/same"},
	}
	for _, route := range routes {
		_, finish := zero.BeginHTTP(context.Background(), route)
		finish(web.HTTPResult{Status: http.StatusNoContent, Bytes: -1, Duration: -1})
	}
	snapshot := zero.Snapshot()
	if len(snapshot.Routes) != 3 ||
		snapshot.Routes[0].Route.ID != "first" ||
		snapshot.Routes[1].Route.ID != "post" ||
		snapshot.Routes[2].Route.ID != "second" ||
		snapshot.Routes[0].Bytes != 0 ||
		snapshot.Routes[0].TotalDurationNanos != 0 {
		t.Fatalf("sorted snapshot = %#v", snapshot)
	}

	var nilMetrics *HTTPMetrics
	ctx := context.Background()
	observed, finish := nilMetrics.BeginHTTP(ctx, routes[0])
	if observed != ctx || finish != nil || len(nilMetrics.Snapshot().Routes) != 0 {
		t.Fatalf(
			"nil metrics = context %v, finishNil=%t, snapshot %#v",
			observed,
			finish == nil,
			nilMetrics.Snapshot(),
		)
	}

	limited := &HTTPMetrics{
		routes:    make(map[web.RouteMetadata]*httpRouteMetric),
		maxRoutes: 1,
	}
	_, finish = limited.BeginHTTP(ctx, routes[0])
	finish(web.HTTPResult{Status: http.StatusOK})
	_, finish = limited.BeginHTTP(ctx, routes[1])
	if finish != nil {
		t.Fatal("cardinality-limited observation returned a finisher")
	}
	snapshot = limited.Snapshot()
	if len(snapshot.Routes) != 1 || snapshot.DroppedObservations != 1 {
		t.Fatalf("cardinality-limited snapshot = %#v", snapshot)
	}
}

type metricsContextKey struct{}
