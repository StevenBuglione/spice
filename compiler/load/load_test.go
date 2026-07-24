package load_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	spiceload "github.com/StevenBuglione/spice/compiler/load"
)

func TestLoadMultiPackageProgram(t *testing.T) {
	root := fixture(t, map[string]string{
		"go.mod": "module example.com/shop\n\ngo 1.23.0\n",
		"contract/service.go": `package contract

type Service interface { Name() string }
`,
		"app/app.go": `package app

import "example.com/shop/contract"

func New(service contract.Service) *Application { return &Application{Service: service} }
type Application struct { Service contract.Service }
`,
	})

	program, err := spiceload.Load(context.Background(), spiceload.Config{Dir: root}, "./...")
	if err != nil {
		t.Fatalf("Load() error = %v; diagnostics = %#v", err, program.Diagnostics())
	}

	packages := program.Packages()
	paths := packagePaths(packages)
	wantPaths := []string{"example.com/shop/app", "example.com/shop/contract"}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("package paths = %v, want %v", paths, wantPaths)
	}
	for _, pkg := range packages {
		if pkg.ModulePath() != "example.com/shop" {
			t.Errorf("package %s module path = %q", pkg.Path(), pkg.ModulePath())
		}
		if pkg.Dir() == "" || !filepath.IsAbs(pkg.Dir()) {
			t.Errorf("package %s directory = %q, want absolute", pkg.Path(), pkg.Dir())
		}
		files := pkg.CompiledGoFiles()
		if !sort.StringsAreSorted(files) {
			t.Errorf("package %s compiled files not sorted: %v", pkg.Path(), files)
		}
	}

	newSymbol := symbolByID(t, program.Symbols(), "example.com/shop/app.New")
	if newSymbol.Kind != spiceload.SymbolFunction || newSymbol.Signature == nil {
		t.Fatalf("New symbol = %#v", newSymbol)
	}
	parameter := newSymbol.Signature.Params().At(0).Type()
	named, ok := parameter.(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "example.com/shop/contract" {
		t.Fatalf("New parameter = %s, want imported contract.Service", parameter)
	}
}

func TestLoadDeclarationKindsAndReceiverNormalization(t *testing.T) {
	root := fixture(t, map[string]string{
		"go.mod": "module example.com/symbols\n\ngo 1.23.0\n",
		"symbols.go": `package symbols

const Answer = 42
var Current int
type Item struct{}
type Generic[T any] struct{}
func Build() Item { return Item{} }
func (Item) Value() {}
func (*Item) Pointer() {}
func (Generic[T]) GenericMethod() {}
`,
	})

	program, err := spiceload.Load(context.Background(), spiceload.Config{Dir: root}, ".")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := map[spiceload.SymbolID]spiceload.SymbolKind{
		"example.com/symbols":                       spiceload.SymbolPackage,
		"example.com/symbols.Answer":                spiceload.SymbolConstant,
		"example.com/symbols.Current":               spiceload.SymbolVariable,
		"example.com/symbols.Item":                  spiceload.SymbolType,
		"example.com/symbols.Generic":               spiceload.SymbolType,
		"example.com/symbols.Build":                 spiceload.SymbolFunction,
		"example.com/symbols.Item.Value":            spiceload.SymbolMethod,
		"example.com/symbols.Item.Pointer":          spiceload.SymbolMethod,
		"example.com/symbols.Generic.GenericMethod": spiceload.SymbolMethod,
	}
	got := make(map[spiceload.SymbolID]spiceload.SymbolKind)
	for _, symbol := range program.Symbols() {
		got[symbol.ID] = symbol.Kind
		if strings.Contains(string(symbol.ID), root) {
			t.Errorf("stable ID contains fixture path: %s", symbol.ID)
		}
	}
	if !reflect.DeepEqual(got, want) {
		gotJSON, _ := json.MarshalIndent(got, "", "  ")
		wantJSON, _ := json.MarshalIndent(want, "", "  ")
		t.Fatalf("symbols = %s\nwant = %s", gotJSON, wantJSON)
	}
}

func TestLoadBuildConstraints(t *testing.T) {
	root := fixture(t, map[string]string{
		"go.mod": "module example.com/tags\n\ngo 1.23.0\n",
		"base.go": `package tags
const Base = true
`,
		"enabled.go": `//go:build spice_active
package tags
const Enabled = true
`,
		"excluded.go": `//go:build !spice_active
package tags
const Excluded = true
`,
	})

	program, err := spiceload.Load(context.Background(), spiceload.Config{
		Dir:        root,
		BuildFlags: []string{"-tags=spice_active"},
	}, ".")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	packages := program.Packages()
	if len(packages) != 1 {
		t.Fatalf("packages = %d, want 1", len(packages))
	}
	files := baseNames(packages[0].CompiledGoFiles())
	wantFiles := []string{"base.go", "enabled.go"}
	if !reflect.DeepEqual(files, wantFiles) {
		t.Fatalf("compiled files = %v, want %v", files, wantFiles)
	}
	ids := symbolIDs(program.Symbols())
	if contains(ids, "example.com/tags.Excluded") || !contains(ids, "example.com/tags.Enabled") {
		t.Fatalf("symbol IDs = %v", ids)
	}
}

func TestLoadDeterministic(t *testing.T) {
	root := fixture(t, map[string]string{
		"go.mod": "module example.com/deterministic\n\ngo 1.23.0\n",
		"b/b.go": "package b\nvar B int\n",
		"a/a.go": "package a\nvar A int\n",
	})
	var first []byte
	for iteration := 0; iteration < 3; iteration++ {
		program, err := spiceload.Load(context.Background(), spiceload.Config{Dir: root}, "./b", "./a")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		encoded, err := json.Marshal(program.Summary())
		if err != nil {
			t.Fatal(err)
		}
		if iteration == 0 {
			first = encoded
			continue
		}
		if !bytes.Equal(encoded, first) {
			t.Fatalf("summary changed:\nfirst %s\nnext  %s", first, encoded)
		}
	}
}

func TestLoadTypeErrors(t *testing.T) {
	root := fixture(t, map[string]string{
		"go.mod": "module example.com/broken\n\ngo 1.23.0\n",
		"broken.go": `package broken
var Number int = "wrong"
`,
	})
	program, err := spiceload.Load(context.Background(), spiceload.Config{Dir: root}, ".")
	if err == nil {
		t.Fatal("Load() error = nil, want type error")
	}
	packages := program.Packages()
	if len(packages) != 1 || !packages[0].IllTyped() {
		t.Fatalf("packages = %#v, want one ill-typed root", packages)
	}
	diagnostics := program.Diagnostics()
	if len(diagnostics) == 0 || diagnostics[0].Kind != spiceload.DiagnosticType {
		t.Fatalf("diagnostics = %#v, want type diagnostic", diagnostics)
	}
	if diagnostics[0].Position.Line != 2 || !strings.Contains(diagnostics[0].Message, "cannot use") {
		t.Fatalf("diagnostic = %#v", diagnostics[0])
	}
}

func TestLoadSyntaxErrors(t *testing.T) {
	root := fixture(t, map[string]string{
		"go.mod":    "module example.com/syntax\n\ngo 1.23.0\n",
		"broken.go": "package syntax\nfunc Broken( {\n",
	})
	program, err := spiceload.Load(context.Background(), spiceload.Config{Dir: root}, ".")
	if err == nil {
		t.Fatal("Load() error = nil, want syntax error")
	}
	found := false
	for _, diagnostic := range program.Diagnostics() {
		if diagnostic.Kind == spiceload.DiagnosticParse {
			found = true
		}
	}
	if !found {
		t.Fatalf("diagnostics = %#v, want parse diagnostic", program.Diagnostics())
	}
}

func TestLoadMissingPattern(t *testing.T) {
	root := fixture(t, map[string]string{"go.mod": "module example.com/missing\n\ngo 1.23.0\n"})
	program, err := spiceload.Load(context.Background(), spiceload.Config{Dir: root}, "./does-not-exist")
	if err == nil {
		t.Fatal("Load() error = nil, want missing-pattern failure")
	}
	if len(program.Diagnostics()) == 0 {
		t.Fatal("diagnostics empty")
	}
	if !strings.Contains(strings.ToLower(program.Diagnostics()[0].Message), "does-not-exist") &&
		!strings.Contains(strings.ToLower(program.Diagnostics()[0].Message), "matched no packages") {
		t.Fatalf("diagnostic = %q", program.Diagnostics()[0].Message)
	}
}

func TestLoadCancelledContext(t *testing.T) {
	root := fixture(t, map[string]string{
		"go.mod":  "module example.com/cancelled\n\ngo 1.23.0\n",
		"main.go": "package cancelled\n",
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	_, err := spiceload.Load(ctx, spiceload.Config{Dir: root}, ".")
	if !errors.Is(err, context.Canceled) || !spiceload.IsContextError(err) {
		t.Fatalf("Load() error = %v, want context.Canceled", err)
	}
	if time.Since(started) > time.Second {
		t.Fatalf("cancelled load took %s", time.Since(started))
	}
}

func TestLoadDoesNotReturnTestVariants(t *testing.T) {
	root := fixture(t, map[string]string{
		"go.mod":           "module example.com/notests\n\ngo 1.23.0\n",
		"value.go":         "package notests\nvar Value int\n",
		"value_test.go":    "package notests\nfunc helper() {}\n",
		"external_test.go": "package notests_test\nfunc external() {}\n",
	})
	program, err := spiceload.Load(context.Background(), spiceload.Config{Dir: root}, "./...")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	packages := program.Packages()
	if len(packages) != 1 || packages[0].Path() != "example.com/notests" {
		t.Fatalf("packages = %v", packagePaths(packages))
	}
	for _, file := range packages[0].CompiledGoFiles() {
		if strings.HasSuffix(file, "_test.go") {
			t.Fatalf("test file returned: %s", file)
		}
	}
}

func TestLoadOverlay(t *testing.T) {
	root := fixture(t, map[string]string{
		"go.mod":   "module example.com/overlay\n\ngo 1.23.0\n",
		"value.go": "package overlay\nvar Value int\n",
	})
	path := filepath.Join(root, "value.go")
	program, err := spiceload.Load(context.Background(), spiceload.Config{
		Dir: root,
		Overlay: map[string][]byte{
			path: []byte("package overlay\nvar Replacement string\n"),
		},
	}, ".")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	ids := symbolIDs(program.Symbols())
	if !contains(ids, "example.com/overlay.Replacement") || contains(ids, "example.com/overlay.Value") {
		t.Fatalf("symbols = %v", ids)
	}
}

func TestLoadLibraryIsQuiet(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-wide pipe replacement is flaky on Windows")
	}
	root := fixture(t, map[string]string{
		"go.mod":   "module example.com/quiet\n\ngo 1.23.0\n",
		"value.go": "package quiet\n",
	})
	stdout, stderr := captureProcessOutput(t, func() {
		_, err := spiceload.Load(context.Background(), spiceload.Config{
			Dir: root,
			Env: append(os.Environ(), "GOPACKAGESDEBUG=1"),
		}, ".")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
	})
	if stdout != "" || stderr != "" {
		t.Fatalf("library output stdout=%q stderr=%q", stdout, stderr)
	}
}

func fixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for path, contents := range files {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func packagePaths(packages []spiceload.Package) []string {
	result := make([]string, len(packages))
	for index, pkg := range packages {
		result[index] = pkg.Path()
	}
	return result
}

func symbolIDs(symbols []spiceload.Symbol) []string {
	result := make([]string, len(symbols))
	for index, symbol := range symbols {
		result[index] = string(symbol.ID)
	}
	return result
}

func symbolByID(t *testing.T, symbols []spiceload.Symbol, id spiceload.SymbolID) spiceload.Symbol {
	t.Helper()
	for _, symbol := range symbols {
		if symbol.ID == id {
			return symbol
		}
	}
	t.Fatalf("symbol %s not found in %v", id, symbolIDs(symbols))
	return spiceload.Symbol{}
}

func baseNames(paths []string) []string {
	result := make([]string, len(paths))
	for index, path := range paths {
		result[index] = filepath.Base(path)
	}
	sort.Strings(result)
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func captureProcessOutput(t *testing.T, run func()) (string, string) {
	t.Helper()
	oldStdout, oldStderr := os.Stdout, os.Stderr
	stdoutRead, stdoutWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderrRead, stderrWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = stdoutWrite, stderrWrite
	defer func() { os.Stdout, os.Stderr = oldStdout, oldStderr }()

	run()
	_ = stdoutWrite.Close()
	_ = stderrWrite.Close()
	var stdoutBuffer, stderrBuffer bytes.Buffer
	_, _ = stdoutBuffer.ReadFrom(stdoutRead)
	_, _ = stderrBuffer.ReadFrom(stderrRead)
	_ = stdoutRead.Close()
	_ = stderrRead.Close()
	return stdoutBuffer.String(), stderrBuffer.String()
}
