package management

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/StevenBuglione/spice/lifecycle"
	"github.com/StevenBuglione/spice/web"
)

func TestManagerReportsDeterministicallyWithoutLeakingErrors(t *testing.T) {
	secret := errors.New("database password is secret")
	var called []string
	manager, err := New(
		Check{
			Name:   "zeta",
			Module: "example.com/orders",
			Groups: []Group{GroupHealth, GroupReadiness},
			Probe: func(context.Context) error {
				called = append(called, "zeta")
				return secret
			},
		},
		Check{
			Name:   "alpha",
			Module: "example.com/inventory",
			Groups: []Group{GroupHealth},
			Probe: func(context.Context) error {
				called = append(called, "alpha")
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	report, err := manager.Report(context.Background(), GroupHealth)
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if report.Status != StatusDown ||
		!slices.Equal(called, []string{"alpha", "zeta"}) ||
		len(report.Components) != 2 ||
		report.Components[0].Name != "alpha" ||
		report.Components[0].Status != StatusUp ||
		report.Components[1].Status != StatusDown {
		t.Fatalf("Report() = %#v, called=%v", report, called)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secret.Error()) {
		t.Fatalf("report leaked probe error: %s", encoded)
	}

	readiness, err := manager.Report(context.Background(), GroupReadiness)
	if err != nil || readiness.Status != StatusDown || len(readiness.Components) != 1 {
		t.Fatalf("readiness = %#v, %v", readiness, err)
	}
	liveness, err := manager.Report(context.Background(), GroupLiveness)
	if err != nil || liveness.Status != StatusUp || len(liveness.Components) != 0 {
		t.Fatalf("liveness = %#v, %v", liveness, err)
	}
}

func TestManagerValidationAndCancellation(t *testing.T) {
	valid := Check{
		Name:   "database",
		Groups: []Group{GroupReadiness},
		Probe:  func(context.Context) error { return nil },
	}
	tests := []struct {
		name   string
		checks []Check
		want   string
	}{
		{name: "invalid name", checks: []Check{{Name: "not valid", Groups: valid.Groups, Probe: valid.Probe}}, want: "name"},
		{name: "nil probe", checks: []Check{{Name: "nil", Groups: valid.Groups}}, want: "probe is nil"},
		{name: "no groups", checks: []Check{{Name: "none", Probe: valid.Probe}}, want: "no groups"},
		{name: "bad group", checks: []Check{{Name: "bad", Groups: []Group{"other"}, Probe: valid.Probe}}, want: "unsupported group"},
		{name: "duplicate", checks: []Check{valid, valid}, want: "duplicate check"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := New(test.checks...); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("New() error = %v, want %q", err, test.want)
			}
		})
	}

	manager, err := New(valid)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report, err := manager.Report(ctx, GroupReadiness)
	if err != nil || report.Status != StatusDown {
		t.Fatalf("canceled report = %#v, %v", report, err)
	}
	if _, err := manager.Report(context.Background(), "other"); err == nil {
		t.Fatal("Report(unknown group) error = nil")
	}
	if _, err := (*Manager)(nil).Report(context.Background(), GroupHealth); err == nil {
		t.Fatal("nil Manager.Report() error = nil")
	}
	if _, err := manager.Report(nil, GroupHealth); err == nil { //nolint:staticcheck // Verify the defensive public API.
		t.Fatal("Report(nil context) error = nil")
	}
}

func TestLifecycleChecksMapApplicationState(t *testing.T) {
	state := lifecycle.StateConstructed
	checks, err := LifecycleChecks("application", "example.com/app", func() lifecycle.State {
		return state
	})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := New(checks...)
	if err != nil {
		t.Fatal(err)
	}
	assertGroupStatus(t, manager, GroupHealth, StatusUp)
	assertGroupStatus(t, manager, GroupLiveness, StatusUp)
	assertGroupStatus(t, manager, GroupReadiness, StatusDown)

	state = lifecycle.StateReady
	assertGroupStatus(t, manager, GroupReadiness, StatusUp)
	state = lifecycle.StateFailed
	assertGroupStatus(t, manager, GroupHealth, StatusDown)
	assertGroupStatus(t, manager, GroupLiveness, StatusDown)
	if _, err := LifecycleChecks("application", "", nil); err == nil {
		t.Fatal("LifecycleChecks(nil) error = nil")
	}
}

func TestHandlerServesIsolatedManagementEndpoints(t *testing.T) {
	manager, err := New(
		Check{
			Name:   "live",
			Groups: []Group{GroupHealth, GroupLiveness},
			Probe:  func(context.Context) error { return nil },
		},
		Check{
			Name:   "database",
			Groups: []Group{GroupReadiness},
			Probe:  func(context.Context) error { return errors.New("secret DSN") },
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	info := map[string]string{"name": "commerce", "version": "1.0.0"}
	handler, err := NewHandler(HandlerOptions{Manager: manager, Info: info})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	info["version"] = "changed"
	if handler.Pattern() != "/actuator/" {
		t.Fatalf("Pattern() = %q", handler.Pattern())
	}

	response := serve(handler, http.MethodGet, "/actuator/health")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"UP"`) {
		t.Fatalf("health response = %d %s", response.Code, response.Body.String())
	}
	response = serve(handler, http.MethodGet, "/actuator/health/readiness")
	if response.Code != http.StatusServiceUnavailable ||
		strings.Contains(response.Body.String(), "secret DSN") {
		t.Fatalf("readiness response = %d %s", response.Code, response.Body.String())
	}
	response = serve(handler, http.MethodGet, "/actuator/info")
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"version":"1.0.0"`) ||
		strings.Contains(response.Body.String(), "changed") {
		t.Fatalf("info response = %d %s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("info headers = %#v", response.Header())
	}
	response = serve(handler, http.MethodPost, "/actuator/health")
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d", response.Code)
	}
	response = serve(handler, http.MethodGet, "/other")
	if response.Code != http.StatusNotFound {
		t.Fatalf("unowned path status = %d", response.Code)
	}
	response = serve(handler, http.MethodGet, "/actuator/metrics")
	if response.Code != http.StatusNotFound {
		t.Fatalf("disabled metrics status = %d", response.Code)
	}

	root := http.NewServeMux()
	root.Handle(handler.Pattern(), handler)
	response = serve(root, http.MethodGet, "/actuator/health/liveness")
	if response.Code != http.StatusOK {
		t.Fatalf("mounted liveness status = %d", response.Code)
	}

	metrics := NewHTTPMetrics()
	route := web.RouteMetadata{ID: "route", Module: "example.com/app", Method: http.MethodGet, Pattern: "/items"}
	_, finish := metrics.BeginHTTP(context.Background(), route)
	finish(web.HTTPResult{Status: http.StatusOK, Bytes: 7})
	handler, err = NewHandler(HandlerOptions{Manager: manager, Metrics: metrics})
	if err != nil {
		t.Fatal(err)
	}
	response = serve(handler, http.MethodGet, "/actuator/metrics")
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"requests":1`) ||
		!strings.Contains(response.Body.String(), `"example.com/app"`) {
		t.Fatalf("metrics response = %d %s", response.Code, response.Body.String())
	}
}

func TestHandlerValidationAndNilReceiver(t *testing.T) {
	manager, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, basePath := range []string{"/", "relative", "/bad/", "/bad/{value}", "/bad path"} {
		if _, err := NewHandler(HandlerOptions{BasePath: basePath, Manager: manager}); err == nil {
			t.Fatalf("NewHandler(%q) error = nil", basePath)
		}
	}
	if _, err := NewHandler(HandlerOptions{}); err == nil {
		t.Fatal("NewHandler(nil manager) error = nil")
	}
	var handler *Handler
	if handler.Pattern() != "" {
		t.Fatalf("nil Pattern() = %q", handler.Pattern())
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil handler status = %d", response.Code)
	}
}

func assertGroupStatus(t *testing.T, manager *Manager, group Group, want Status) {
	t.Helper()
	report, err := manager.Report(context.Background(), group)
	if err != nil || report.Status != want {
		t.Fatalf("Report(%s) = %#v, %v, want %s", group, report, err, want)
	}
}

func serve(handler http.Handler, method, target string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
