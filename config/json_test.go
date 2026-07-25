package config

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func FuzzDecodeJSONObject(f *testing.F) {
	for _, seed := range []string{
		`{}`,
		`{"server":{"port":8080,"enabled":true}}`,
		`{"database.password":"secret"}`,
		`{"duplicate":1,"duplicate":2}`,
		`[]`,
		`{"array":[]}`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, content []byte) {
		if len(content) > 64<<10 {
			t.Skip()
		}
		first, firstErr := decodeJSONObject(content)
		second, secondErr := decodeJSONObject(content)
		if (firstErr == nil) != (secondErr == nil) {
			t.Fatalf("decode error stability = %v, %v", firstErr, secondErr)
		}
		if firstErr == nil && !reflect.DeepEqual(first, second) {
			t.Fatalf("decode output changed: %v, %v", first, second)
		}
	})
}

func TestJSONSourceLoadsBaseAndProfilesInOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeJSONFile(t, root, "application.json", `{
  "server": {
    "host": "127.0.0.1",
    "port": 8080,
    "tls": false
  },
  "timeout": "5s"
}`)
	writeJSONFile(t, root, "application-dev.json", `{
  "server": {"port": 8081, "tls": true}
}`)
	writeJSONFile(t, root, "application-local.json", `{
  "server": {"port": 9090}
}`)
	source, err := NewJSONSource(
		"files",
		root,
		"application",
		JSONOptions{Required: true},
	)
	if err != nil {
		t.Fatalf("NewJSONSource() error = %v", err)
	}
	schema := MustSchema(
		Property{Key: "server.host", Kind: KindString, Required: true},
		Property{Key: "server.port", Kind: KindInteger, Required: true},
		Property{Key: "server.tls", Kind: KindBoolean, Required: true},
		Property{Key: "timeout", Kind: KindDuration, Required: true},
	)
	snapshot, err := Resolve(
		context.Background(),
		schema,
		Options{Profiles: []string{"dev", "local"}},
		source,
	)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got, want := snapshot.Keys(), []string{
		"server.host",
		"server.port",
		"server.tls",
		"timeout",
	}; !slices.Equal(got, want) {
		t.Fatalf("Keys() = %v, want %v", got, want)
	}
	if port, err := snapshot.Integer("server.port"); err != nil || port != 9090 {
		t.Fatalf("Integer(server.port) = %d, %v", port, err)
	}
	if tls, err := snapshot.Boolean("server.tls"); err != nil || !tls {
		t.Fatalf("Boolean(server.tls) = %t, %v", tls, err)
	}
	if entry, ok := snapshot.Entry("server.port"); !ok || entry.Origin.Source != "files" {
		t.Fatalf("server.port entry = %#v, %t", entry, ok)
	}
}

func TestJSONSourceHandlesRequiredAndOptionalFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	optional, err := NewJSONSource("files", root, "application", JSONOptions{})
	if err != nil {
		t.Fatal(err)
	}
	values, err := optional.Load(
		context.Background(),
		Request{profiles: []string{"missing"}},
	)
	if err != nil || len(values) != 0 {
		t.Fatalf("optional.Load() = %v, %v", values, err)
	}
	required, err := NewJSONSource(
		"files",
		root,
		"application",
		JSONOptions{Required: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := required.Load(context.Background(), Request{}); err == nil ||
		!strings.Contains(err.Error(), "application.json") {
		t.Fatalf("required.Load() error = %v", err)
	}
}

func TestJSONSourceRejectsInvalidConstructionAndBoundedReads(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, directory, base string
		options               JSONOptions
	}{
		{name: "", directory: ".", base: "application"},
		{name: "files", directory: "", base: "application"},
		{name: "files", directory: ".", base: "../application"},
		{name: "files", directory: ".", base: "application", options: JSONOptions{MaxBytes: -1}},
	}
	for _, test := range tests {
		if _, err := NewJSONSource(
			test.name,
			test.directory,
			test.base,
			test.options,
		); err == nil {
			t.Fatalf("NewJSONSource(%q, %q, %q) error = nil", test.name, test.directory, test.base)
		}
	}
	root := t.TempDir()
	writeJSONFile(t, root, "application.json", `{"server.port":8080}`)
	source, err := NewJSONSource(
		"files",
		root,
		"application",
		JSONOptions{Required: true, MaxBytes: 8},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.Load(context.Background(), Request{}); err == nil ||
		!strings.Contains(err.Error(), "exceeds 8 byte limit") {
		t.Fatalf("Load(oversized) error = %v", err)
	}
	if _, err := NewJSONSource(
		"files",
		filepath.Join(root, "missing"),
		"application",
		JSONOptions{},
	); err != nil {
		t.Fatalf("NewJSONSource(missing directory) error = %v", err)
	}
	missingRoot := JSONSource{
		name:      "files",
		directory: filepath.Join(root, "missing"),
		baseName:  "application",
		maxBytes:  defaultJSONMaxBytes,
	}
	if _, err := missingRoot.Load(context.Background(), Request{}); err == nil ||
		!strings.Contains(err.Error(), "open configuration root") {
		t.Fatalf("Load(missing root) error = %v", err)
	}
}

func TestJSONSourceRejectsAmbiguousOrUnsupportedJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, content, expected string
	}{
		{name: "root-array", content: `[]`, expected: "root must be an object"},
		{name: "duplicate", content: `{"port":1,"port":2}`, expected: `duplicate key "port"`},
		{
			name:     "flattened-collision",
			content:  `{"server":{"port":1},"server.port":2}`,
			expected: `flattened configuration key "server.port" is declared more than once`,
		},
		{name: "array", content: `{"ports":[8080]}`, expected: "must not contain an array"},
		{name: "null", content: `{"password":null}`, expected: "null values are not supported"},
		{name: "invalid-key", content: `{"Bad":true}`, expected: "must match"},
		{name: "trailing", content: `{} {}`, expected: "trailing value"},
		{name: "unclosed", content: `{"port":8080`, expected: "close configuration object"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeJSONFile(t, root, "application.json", test.content)
			source, err := NewJSONSource(
				"files",
				root,
				"application",
				JSONOptions{Required: true},
			)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := source.Load(context.Background(), Request{}); err == nil ||
				!strings.Contains(err.Error(), test.expected) {
				t.Fatalf("Load() error = %v, want %q", err, test.expected)
			}
		})
	}
}

func TestJSONSourceHonorsCancellation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeJSONFile(t, root, "application.json", `{}`)
	source, err := NewJSONSource("files", root, "application", JSONOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := source.Load(ctx, Request{}); err == nil {
		t.Fatal("Load(canceled) error = nil")
	}
}

func TestDecodeJSONObjectScalarForms(t *testing.T) {
	t.Parallel()
	values, err := decodeJSONObject([]byte(`{"string":"value","boolean":true,"integer":42,"decimal":1.5}`))
	if err != nil {
		t.Fatalf("decodeJSONObject() error = %v", err)
	}
	want := map[string]string{
		"string":  "value",
		"boolean": "true",
		"integer": "42",
		"decimal": "1.5",
	}
	if len(values) != len(want) {
		t.Fatalf("values = %v, want %v", values, want)
	}
	for key, expected := range want {
		if values[key] != expected {
			t.Fatalf("values[%q] = %q, want %q", key, values[key], expected)
		}
	}
	if _, err := decodeJSONObject(nil); err == nil {
		t.Fatal("decodeJSONObject(empty) error = nil")
	}
}

func writeJSONFile(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
