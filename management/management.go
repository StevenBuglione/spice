// Package management provides deterministic, opt-in production management
// probes and HTTP endpoints without a global registry.
package management

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/StevenBuglione/spice/lifecycle"
	"github.com/StevenBuglione/spice/web"
)

// Group identifies one independently queryable health concern.
type Group string

const (
	// GroupHealth contains broad application health checks.
	GroupHealth Group = "health"
	// GroupLiveness contains checks that decide whether a process is alive.
	GroupLiveness Group = "liveness"
	// GroupReadiness contains checks that decide whether traffic is safe.
	GroupReadiness Group = "readiness"
)

// Status is the stable client-facing result of a probe or report.
type Status string

const (
	// StatusUp means every selected check passed.
	StatusUp Status = "UP"
	// StatusDown means at least one selected check failed.
	StatusDown Status = "DOWN"
)

// Probe is one caller-owned, context-aware health check.
type Probe func(context.Context) error

// Check declares one named probe in one or more management groups. Module is
// optional ownership metadata and is safe to expose.
type Check struct {
	Name   string
	Module string
	Groups []Group
	Probe  Probe
}

// Component is one safe client-facing check result. Probe errors are
// intentionally excluded.
type Component struct {
	Name   string `json:"name"`
	Module string `json:"module,omitempty"`
	Status Status `json:"status"`
}

// Report is a deterministic group result.
type Report struct {
	Group      Group       `json:"group"`
	Status     Status      `json:"status"`
	Components []Component `json:"components"`
}

// Manager is an immutable collection of validated health checks.
type Manager struct {
	checks map[Group][]Check
}

// New validates, copies, and deterministically orders health checks.
func New(checks ...Check) (*Manager, error) {
	grouped := map[Group][]Check{
		GroupHealth:    {},
		GroupLiveness:  {},
		GroupReadiness: {},
	}
	seen := make(map[string]struct{})
	for checkIndex, check := range checks {
		if !validName(check.Name) {
			return nil, fmt.Errorf(
				"management check %d name %q must contain only letters, digits, '.', '_', or '-'",
				checkIndex,
				check.Name,
			)
		}
		if check.Probe == nil {
			return nil, fmt.Errorf("management check %q probe is nil", check.Name)
		}
		if len(check.Groups) == 0 {
			return nil, fmt.Errorf("management check %q declares no groups", check.Name)
		}
		for _, group := range check.Groups {
			if !validGroup(group) {
				return nil, fmt.Errorf("management check %q has unsupported group %q", check.Name, group)
			}
			key := string(group) + "\x00" + check.Name
			if _, duplicate := seen[key]; duplicate {
				return nil, fmt.Errorf("management group %q contains duplicate check %q", group, check.Name)
			}
			seen[key] = struct{}{}
			grouped[group] = append(grouped[group], Check{
				Name:   check.Name,
				Module: check.Module,
				Groups: []Group{group},
				Probe:  check.Probe,
			})
		}
	}
	for group := range grouped {
		slices.SortFunc(grouped[group], func(left, right Check) int {
			if compared := strings.Compare(left.Name, right.Name); compared != 0 {
				return compared
			}
			return strings.Compare(left.Module, right.Module)
		})
	}
	return &Manager{checks: grouped}, nil
}

// Report runs one group in deterministic check order. Probe failures are
// represented as DOWN and are never exposed as response details.
func (manager *Manager) Report(ctx context.Context, group Group) (Report, error) {
	if manager == nil {
		return Report{}, errors.New("management report: manager is nil")
	}
	if ctx == nil {
		return Report{}, errors.New("management report: context is nil")
	}
	checks, found := manager.checks[group]
	if !found {
		return Report{}, fmt.Errorf("management report: unsupported group %q", group)
	}
	report := Report{
		Group:      group,
		Status:     StatusUp,
		Components: make([]Component, 0, len(checks)),
	}
	for _, check := range checks {
		status := StatusUp
		if ctx.Err() != nil || check.Probe(ctx) != nil {
			status = StatusDown
			report.Status = StatusDown
		}
		report.Components = append(report.Components, Component{
			Name:   check.Name,
			Module: check.Module,
			Status: status,
		})
	}
	return report, nil
}

// LifecycleChecks adapts one generated application's observable state into
// health, liveness, and readiness checks.
func LifecycleChecks(
	name string,
	module string,
	state func() lifecycle.State,
) ([]Check, error) {
	if state == nil {
		return nil, errors.New("management lifecycle checks: state function is nil")
	}
	live := func(context.Context) error {
		switch current := state(); current {
		case lifecycle.StateConstructed,
			lifecycle.StateStarting,
			lifecycle.StateReady,
			lifecycle.StateStopping:
			return nil
		case lifecycle.StateInvalid,
			lifecycle.StateStopped,
			lifecycle.StateFailed:
			return fmt.Errorf("application lifecycle state is %s", current)
		default:
			return fmt.Errorf("application lifecycle state %q is unsupported", current)
		}
	}
	ready := func(context.Context) error {
		if current := state(); current != lifecycle.StateReady {
			return fmt.Errorf("application lifecycle state is %s", current)
		}
		return nil
	}
	return []Check{
		{Name: name, Module: module, Groups: []Group{GroupHealth, GroupLiveness}, Probe: live},
		{Name: name, Module: module, Groups: []Group{GroupReadiness}, Probe: ready},
	}, nil
}

// HandlerOptions configures one isolated management HTTP handler.
type HandlerOptions struct {
	BasePath string
	Manager  *Manager
	Info     map[string]string
}

// Handler serves one isolated set of management endpoints.
type Handler struct {
	basePath string
	manager  *Manager
	info     map[string]string
	mux      *http.ServeMux
}

// NewHandler constructs health, liveness, readiness, and info endpoints.
func NewHandler(options HandlerOptions) (*Handler, error) {
	if options.Manager == nil {
		return nil, errors.New("construct management handler: manager is nil")
	}
	basePath := options.BasePath
	if basePath == "" {
		basePath = "/actuator"
	}
	if !validBasePath(basePath) {
		return nil, fmt.Errorf("construct management handler: base path %q must be a clean absolute path below root", basePath)
	}
	handler := &Handler{
		basePath: basePath,
		manager:  options.Manager,
		info:     cloneInfo(options.Info),
		mux:      http.NewServeMux(),
	}
	handler.mux.HandleFunc("GET "+basePath+"/health", handler.serveReport(GroupHealth))
	handler.mux.HandleFunc("GET "+basePath+"/health/liveness", handler.serveReport(GroupLiveness))
	handler.mux.HandleFunc("GET "+basePath+"/health/readiness", handler.serveReport(GroupReadiness))
	handler.mux.HandleFunc("GET "+basePath+"/info", handler.serveInfo)
	return handler, nil
}

// Pattern returns the ServeMux subtree pattern used to mount this handler.
func (handler *Handler) Pattern() string {
	if handler == nil {
		return ""
	}
	return handler.basePath + "/"
}

// ServeHTTP dispatches management requests without exposing other routes.
func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if handler == nil || handler.mux == nil {
		http.Error(writer, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	handler.mux.ServeHTTP(writer, request)
}

func (handler *Handler) serveReport(group Group) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		report, err := handler.manager.Report(request.Context(), group)
		if err != nil {
			if writeErr := web.WriteError(writer, request, err, nil); writeErr != nil {
				return
			}
			return
		}
		status := http.StatusOK
		if report.Status == StatusDown {
			status = http.StatusServiceUnavailable
		}
		if writeErr := web.WriteJSON(writer, status, report); writeErr != nil {
			return
		}
	}
}

func (handler *Handler) serveInfo(writer http.ResponseWriter, _ *http.Request) {
	if writeErr := web.WriteJSON(writer, http.StatusOK, handler.info); writeErr != nil {
		return
	}
}

func validName(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			character == '.' || character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func validGroup(group Group) bool {
	return group == GroupHealth || group == GroupLiveness || group == GroupReadiness
}

func validBasePath(value string) bool {
	return strings.HasPrefix(value, "/") &&
		value != "/" &&
		path.Clean(value) == value &&
		!strings.ContainsAny(value, "{}* \t\r\n")
}

func cloneInfo(info map[string]string) map[string]string {
	result := maps.Clone(info)
	if result == nil {
		return map[string]string{}
	}
	return result
}
