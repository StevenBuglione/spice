package management

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func TestModuleReportIsDeterministicAndHandlerOwned(t *testing.T) {
	t.Parallel()
	definitions, edges, unassigned := validModuleReportInput()
	report, err := NewModuleReport(definitions, edges, unassigned)
	if err != nil {
		t.Fatal(err)
	}
	if report.Schema != "spice.modules/v1" ||
		len(report.Modules) != 2 ||
		report.Modules[0].ID != "example.com/inventory" ||
		!slices.Equal(
			report.Modules[1].ObservedDependencies,
			[]string{"example.com/inventory::stock"},
		) ||
		len(report.Edges) != 1 ||
		!slices.Equal(
			report.UnassignedPackages,
			[]string{"example.com/bootstrap"},
		) {
		t.Fatalf("module report = %#v", report)
	}

	manager, err := New()
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(HandlerOptions{
		Manager: manager,
		Modules: &report,
		Expose:  []Endpoint{EndpointModules},
	})
	if err != nil {
		t.Fatal(err)
	}
	report.Modules[0].ID = "mutated"
	report.Modules[0].Packages[0] = "mutated"
	report.Edges[0].API = "mutated"
	report.UnassignedPackages[0] = "mutated"
	response := serve(handler, http.MethodGet, "/actuator/modules")
	if response.Code != http.StatusOK ||
		strings.Contains(response.Body.String(), "mutated") {
		t.Fatalf(
			"modules response = %d %s",
			response.Code,
			response.Body.String(),
		)
	}
	var decoded ModuleReport
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Schema != "spice.modules/v1" ||
		decoded.Modules[0].ID != "example.com/inventory" ||
		decoded.Edges[0].API != "stock" {
		t.Fatalf("decoded module report = %#v", decoded)
	}
	if response := serve(
		handler,
		http.MethodGet,
		"/actuator/health",
	); response.Code != http.StatusNotFound {
		t.Fatalf("unexposed health status = %d", response.Code)
	}
}

func TestModuleReportRejectsInconsistentMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func([]ModuleDefinition, []ModuleEdge, []string) (
			[]ModuleDefinition,
			[]ModuleEdge,
			[]string,
		)
		want string
	}{
		{
			name: "duplicate module",
			mutate: func(
				definitions []ModuleDefinition,
				edges []ModuleEdge,
				unassigned []string,
			) ([]ModuleDefinition, []ModuleEdge, []string) {
				definitions[1].ID = definitions[0].ID
				return definitions, edges, unassigned
			},
			want: "duplicated",
		},
		{
			name: "missing root",
			mutate: func(
				definitions []ModuleDefinition,
				edges []ModuleEdge,
				unassigned []string,
			) ([]ModuleDefinition, []ModuleEdge, []string) {
				definitions[0].Packages = []string{"example.com/inventory/internal"}
				return definitions, edges, unassigned
			},
			want: "omit root",
		},
		{
			name: "unknown allowed dependency",
			mutate: func(
				definitions []ModuleDefinition,
				edges []ModuleEdge,
				unassigned []string,
			) ([]ModuleDefinition, []ModuleEdge, []string) {
				definitions[1].AllowedDependencies = []string{"example.com/missing"}
				return definitions, edges, unassigned
			},
			want: "unknown dependency",
		},
		{
			name: "disallowed edge",
			mutate: func(
				definitions []ModuleDefinition,
				edges []ModuleEdge,
				unassigned []string,
			) ([]ModuleDefinition, []ModuleEdge, []string) {
				definitions[0].AllowedDependencies = nil
				return definitions, edges, unassigned
			},
			want: "is not allowed",
		},
		{
			name: "unknown edge module",
			mutate: func(
				definitions []ModuleDefinition,
				edges []ModuleEdge,
				unassigned []string,
			) ([]ModuleDefinition, []ModuleEdge, []string) {
				edges[0].ToModule = "example.com/missing"
				return definitions, edges, unassigned
			},
			want: "unknown target module",
		},
		{
			name: "assigned and unassigned",
			mutate: func(
				definitions []ModuleDefinition,
				edges []ModuleEdge,
				_ []string,
			) ([]ModuleDefinition, []ModuleEdge, []string) {
				return definitions, edges, []string{"example.com/orders"}
			},
			want: "both unassigned and owned",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			definitions, edges, unassigned := validModuleReportInput()
			definitions, edges, unassigned = test.mutate(
				definitions,
				edges,
				unassigned,
			)
			if _, err := NewModuleReport(
				definitions,
				edges,
				unassigned,
			); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("NewModuleReport() error = %v, want %q", err, test.want)
			}
		})
	}
}

func validModuleReportInput() (
	[]ModuleDefinition,
	[]ModuleEdge,
	[]string,
) {
	return []ModuleDefinition{
			{
				ID:          "example.com/orders",
				RootPackage: "example.com/orders",
				Packages:    []string{"example.com/orders"},
				AllowedDependencies: []string{
					"example.com/inventory::stock",
				},
			},
			{
				ID:          "example.com/inventory",
				RootPackage: "example.com/inventory",
				Packages: []string{
					"example.com/inventory/internal",
					"example.com/inventory",
				},
				NamedInterfaces: []NamedInterface{{
					Name:        "stock",
					PackagePath: "example.com/inventory/internal",
				}},
			},
		},
		[]ModuleEdge{{
			FromModule:  "example.com/orders",
			ToModule:    "example.com/inventory",
			API:         "stock",
			FromPackage: "example.com/orders",
			ToPackage:   "example.com/inventory/internal",
		}},
		[]string{"example.com/bootstrap"}
}
