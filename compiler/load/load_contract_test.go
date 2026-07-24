package load

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLoadHonorsEnvironment(t *testing.T) {
	dir := writeModule(t, map[string]string{
		"go.mod": "module example.com/environment\n\ngo 1.23.0\n",
		"selected/default.go": `//go:build !spice_env

package selected

var Default int
`,
		"selected/environment.go": `//go:build spice_env

package selected

var FromEnvironment int
`,
	})

	program, err := Load(context.Background(), Options{
		Dir: dir,
		Env: replaceEnvironmentValue(os.Environ(), "GOFLAGS", "-tags=spice_env"),
	}, "./...")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if symbolByID(program.Symbols(), "example.com/environment/selected.FromEnvironment") == nil {
		t.Fatalf("environment-selected declaration missing: %v", symbolIDs(program.Symbols()))
	}
	if symbolByID(program.Symbols(), "example.com/environment/selected.Default") != nil {
		t.Fatalf("default declaration present despite GOFLAGS environment: %v", symbolIDs(program.Symbols()))
	}
}

func TestLoadReturnsStableSymbolOrder(t *testing.T) {
	dir := writeModule(t, map[string]string{
		"go.mod": "module example.com/order\n\ngo 1.23.0\n",
		"order/order.go": `package order

var Zulu int
const Alpha = 1
type Middle struct{}
`,
	})

	program, err := Load(context.Background(), Options{Dir: dir}, "./order")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	got := symbolIDs(program.Symbols())
	want := []string{
		"example.com/order/order",
		"example.com/order/order.Alpha",
		"example.com/order/order.Middle",
		"example.com/order/order.Zulu",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("symbol IDs = %v, want exact stable order %v", got, want)
	}
}

func TestLoadReturnsStableDiagnosticOrder(t *testing.T) {
	dir := writeModule(t, map[string]string{
		"go.mod": "module example.com/diagnostics\n\ngo 1.23.0\n",
		"zbroken/z.go": `package zbroken

var Value string = 1
`,
		"abroken/a.go": `package abroken

var Value string = 1
`,
	})

	program, err := Load(context.Background(), Options{Dir: dir}, "./zbroken", "./abroken")
	if err == nil {
		t.Fatal("Load() error = nil, want type errors")
	}
	diagnostics := program.Diagnostics()
	got := make([]string, len(diagnostics))
	for i, diagnostic := range diagnostics {
		got[i] = diagnostic.PackagePath + "|" + diagnostic.Kind
	}
	want := []string{
		"example.com/diagnostics/abroken|list",
		"example.com/diagnostics/abroken|type",
		"example.com/diagnostics/zbroken|list",
		"example.com/diagnostics/zbroken|type",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diagnostics = %v, want exact stable order %v", got, want)
	}
}

func TestLoadOmitsNonAddressableDeclarationsAndReturnsUniqueSymbolIDs(t *testing.T) {
	dir := writeModule(t, map[string]string{
		"go.mod": "module example.com/identity\n\ngo 1.23.0\n",
		"catalog/first.go": `package catalog

func init() {}

type _ int
func _() {}
var _ = 1
const _ = 2

type Service struct{}
func (Service) _() {}
func (Service) Start() {}

var First int
const FirstConstant = 3
`,
		"catalog/second.go": `package catalog

func init() {}

type _ string
func _() {}
var _ = 4
const _ = 5

func (Service) _() {}

func Build() *Service { return &Service{} }
const Second = 6
`,
	})

	program, err := Load(context.Background(), Options{Dir: dir}, "./...")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	symbols := program.Symbols()
	assertUniqueSymbolIDs(t, symbols)
	if got, want := symbolIDs(symbols), []string{
		"example.com/identity/catalog",
		"example.com/identity/catalog.Build",
		"example.com/identity/catalog.First",
		"example.com/identity/catalog.FirstConstant",
		"example.com/identity/catalog.Second",
		"example.com/identity/catalog.Service",
		"example.com/identity/catalog.Service.Start",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("symbol IDs = %v, want every non-addressable declaration omitted as %v", got, want)
	}
	for _, symbol := range symbols {
		if symbol.Name == "_" || symbol.Name == "init" {
			t.Fatalf("non-addressable %s declaration %q entered symbol catalog: %v", symbol.Kind, symbol.ID, symbolIDs(symbols))
		}
	}
}

func replaceEnvironmentValue(environment []string, name, value string) []string {
	prefix := name + "="
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if strings.HasPrefix(entry, prefix) {
			continue
		}
		result = append(result, entry)
	}
	return append(result, prefix+value)
}
