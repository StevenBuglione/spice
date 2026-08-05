package starter

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/spice-framework/spice/annotation"
)

func TestManifestNormalizesAndDefensivelyCopiesMetadata(t *testing.T) {
	t.Parallel()
	manifest, err := New(validAnnotationSpec())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	spec := manifest.Spec()
	if !slices.Equal(spec.Capabilities, []string{"http.management", "observability.logging"}) {
		t.Fatalf("Capabilities = %#v", spec.Capabilities)
	}
	if got := spec.Activation.EntryPoints; got[0].Symbol != "New" || got[1].Symbol != "Start" {
		t.Fatalf("EntryPoints = %#v", got)
	}
	if got := spec.Dependencies; got[0].Module != "example.com/alpha" ||
		got[1].Module != "example.com/zeta" {
		t.Fatalf("Dependencies = %#v", got)
	}
	if got := spec.Annotations[0].Arguments; got[0].Name != "expose" || got[1].Name != "mode" {
		t.Fatalf("Arguments = %#v", got)
	}
	if got := spec.ApplicationFeatures[0].Options[0].AllowedStrings; !slices.Equal(
		got,
		[]string{"health", "info"},
	) {
		t.Fatalf("AllowedStrings = %#v", got)
	}
	if got := spec.ApplicationFeatures[0].EntryPoints; got[0].Symbol != "New" ||
		got[1].Symbol != "Start" {
		t.Fatalf("feature EntryPoints = %#v", got)
	}

	definitions := manifest.Definitions()
	if len(definitions) != 1 ||
		definitions[0].Name != "acme.Enable" ||
		!definitions[0].Targets.Contains(annotation.TargetFunction) {
		t.Fatalf("Definitions = %#v", definitions)
	}
	spec.Capabilities[0] = "mutated"
	spec.Annotations[0].Arguments[0].Kinds[0] = annotation.KindBoolean
	spec.ApplicationFeatures[0].EntryPoints[0].Symbol = "Mutated"
	definitions[0].Arguments[0].Kinds[0] = annotation.KindBoolean
	second := manifest.Spec()
	secondDefinitions := manifest.Definitions()
	if second.Capabilities[0] != "http.management" ||
		second.Annotations[0].Arguments[0].Kinds[0] != annotation.KindList ||
		second.ApplicationFeatures[0].EntryPoints[0].Symbol != "New" ||
		secondDefinitions[0].Arguments[0].Kinds[0] != annotation.KindList {
		t.Fatal("manifest accessors returned mutable storage")
	}
}

func TestManifestJSONIsCanonicalAndStrictlyRoundTrips(t *testing.T) {
	t.Parallel()
	manifest := Must(validAnnotationSpec())
	first, err := manifest.JSON()
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	second, err := manifest.JSON()
	if err != nil {
		t.Fatalf("second JSON() error = %v", err)
	}
	if !bytes.Equal(first, second) || len(first) == 0 || first[len(first)-1] != '\n' {
		t.Fatalf("JSON() is not canonical:\n%s", first)
	}
	parsed, err := Parse(first)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !reflect.DeepEqual(parsed.Spec(), manifest.Spec()) {
		t.Fatalf("round trip = %#v, want %#v", parsed.Spec(), manifest.Spec())
	}

	compact, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	decoded, err := Parse(compact)
	if err != nil {
		t.Fatalf("Parse(compact) error = %v", err)
	}
	if !reflect.DeepEqual(decoded.Spec(), manifest.Spec()) {
		t.Fatalf("JSON interfaces round trip = %#v", decoded.Spec())
	}
}

func TestManifestCompatibility(t *testing.T) {
	t.Parallel()
	manifest := Must(validAnnotationSpec())
	for _, goVersion := range []string{"1.26", "go1.26.5", "1.27.0"} {
		if err := manifest.Compatible(APIVersion, goVersion); err != nil {
			t.Fatalf("Compatible(%q) error = %v", goVersion, err)
		}
	}
	for _, test := range []struct {
		api       string
		goVersion string
	}{
		{api: "v2alpha1", goVersion: "1.26"},
		{api: APIVersion, goVersion: "1.25.9"},
		{api: APIVersion, goVersion: "future"},
	} {
		if err := manifest.Compatible(test.api, test.goVersion); err == nil {
			t.Fatalf("Compatible(%q, %q) error = nil", test.api, test.goVersion)
		}
	}
}

func TestConstructorActivationManifest(t *testing.T) {
	t.Parallel()
	spec := validAnnotationSpec()
	spec.Activation.Mode = ActivationExplicitConstructor
	spec.Annotations = nil
	spec.ApplicationFeatures = nil
	manifest, err := New(spec)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if len(manifest.Definitions()) != 0 {
		t.Fatalf("Definitions() = %#v", manifest.Definitions())
	}
}

func TestNewRejectsInvalidSpecs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*Spec)
	}{
		{name: "schema", mutate: func(spec *Spec) { spec.Schema = "other" }},
		{name: "id", mutate: func(spec *Spec) { spec.ID = "../starter" }},
		{name: "module", mutate: func(spec *Spec) { spec.Module = "example.com/other" }},
		{name: "version", mutate: func(spec *Spec) { spec.Version = "latest" }},
		{name: "api", mutate: func(spec *Spec) { spec.SpiceAPI = "alpha" }},
		{name: "minimum-go", mutate: func(spec *Spec) { spec.MinimumGo = "1" }},
		{name: "license", mutate: func(spec *Spec) { spec.License = "Apache 2" }},
		{name: "review", mutate: func(spec *Spec) { spec.Review = " docs/review.md" }},
		{name: "capabilities-empty", mutate: func(spec *Spec) { spec.Capabilities = nil }},
		{
			name: "capability-invalid",
			mutate: func(spec *Spec) {
				spec.Capabilities[0] = "HTTP Management"
			},
		},
		{
			name: "capability-duplicate",
			mutate: func(spec *Spec) {
				spec.Capabilities = []string{"http.management", "http.management"}
			},
		},
		{
			name: "entrypoints-empty",
			mutate: func(spec *Spec) {
				spec.Activation.EntryPoints = nil
			},
		},
		{
			name: "entrypoint-package",
			mutate: func(spec *Spec) {
				spec.Activation.EntryPoints[0].Package = "../package"
			},
		},
		{
			name: "entrypoint-module-ownership",
			mutate: func(spec *Spec) {
				spec.Activation.EntryPoints[0].Package = "example.com/other/starter"
			},
		},
		{
			name: "entrypoint-symbol",
			mutate: func(spec *Spec) {
				spec.Activation.EntryPoints[0].Symbol = "new"
			},
		},
		{
			name: "entrypoint-duplicate",
			mutate: func(spec *Spec) {
				spec.Activation.EntryPoints[1] = spec.Activation.EntryPoints[0]
			},
		},
		{
			name: "activation-mode",
			mutate: func(spec *Spec) {
				spec.Activation.Mode = "classpath"
			},
		},
		{
			name: "constructor-annotations",
			mutate: func(spec *Spec) {
				spec.Activation.Mode = ActivationExplicitConstructor
			},
		},
		{
			name: "annotation-metadata-missing",
			mutate: func(spec *Spec) {
				spec.Annotations = nil
			},
		},
		{
			name: "dependency-module",
			mutate: func(spec *Spec) {
				spec.Dependencies[0].Module = "../dependency"
			},
		},
		{
			name: "dependency-version",
			mutate: func(spec *Spec) {
				spec.Dependencies[0].Version = "main"
			},
		},
		{
			name: "dependency-license",
			mutate: func(spec *Spec) {
				spec.Dependencies[0].License = "unknown license"
			},
		},
		{
			name: "dependency-duplicate",
			mutate: func(spec *Spec) {
				spec.Dependencies[1].Module = spec.Dependencies[0].Module
			},
		},
		{
			name: "annotation-unqualified",
			mutate: func(spec *Spec) {
				spec.Annotations[0].Name = "Enable"
			},
		},
		{
			name: "annotation-target-duplicate",
			mutate: func(spec *Spec) {
				spec.Annotations[0].Targets = []annotation.Target{
					annotation.TargetFunction,
					annotation.TargetFunction,
				}
			},
		},
		{
			name: "annotation-kind",
			mutate: func(spec *Spec) {
				spec.Annotations[0].Arguments[0].Kinds[0] = "object"
			},
		},
		{
			name: "annotation-kind-duplicate",
			mutate: func(spec *Spec) {
				spec.Annotations[0].Arguments[0].Kinds = []annotation.Kind{
					annotation.KindList,
					annotation.KindList,
				}
			},
		},
		{
			name: "feature-annotation",
			mutate: func(spec *Spec) {
				spec.ApplicationFeatures[0].Annotation = "acme.Missing"
			},
		},
		{
			name: "feature-capability",
			mutate: func(spec *Spec) {
				spec.ApplicationFeatures[0].Capability = "missing"
			},
		},
		{
			name: "feature-entrypoints-empty",
			mutate: func(spec *Spec) {
				spec.ApplicationFeatures[0].EntryPoints = nil
			},
		},
		{
			name: "feature-entrypoint-undeclared",
			mutate: func(spec *Spec) {
				spec.ApplicationFeatures[0].EntryPoints[0].Symbol = "Missing"
			},
		},
		{
			name: "feature-entrypoint-duplicate",
			mutate: func(spec *Spec) {
				spec.ApplicationFeatures[0].EntryPoints[1] = spec.ApplicationFeatures[0].EntryPoints[0]
			},
		},
		{
			name: "activation-entrypoint-unreferenced",
			mutate: func(spec *Spec) {
				spec.ApplicationFeatures[0].EntryPoints = spec.ApplicationFeatures[0].EntryPoints[:1]
			},
		},
		{
			name: "feature-option",
			mutate: func(spec *Spec) {
				spec.ApplicationFeatures[0].Options[0].Name = "missing"
			},
		},
		{
			name: "feature-option-kind",
			mutate: func(spec *Spec) {
				spec.ApplicationFeatures[0].Options[0].Kind = annotation.KindString
			},
		},
		{
			name: "feature-list-kind",
			mutate: func(spec *Spec) {
				spec.ApplicationFeatures[0].Options[0].ListItemKinds = []annotation.Kind{
					annotation.KindInteger,
				}
			},
		},
		{
			name: "feature-minimum",
			mutate: func(spec *Spec) {
				spec.ApplicationFeatures[0].Options[0].MinimumItems = -1
			},
		},
		{
			name: "feature-allowed-duplicate",
			mutate: func(spec *Spec) {
				spec.ApplicationFeatures[0].Options[0].AllowedStrings = []string{"health", "health"}
			},
		},
		{
			name: "feature-requirement",
			mutate: func(spec *Spec) {
				spec.ApplicationFeatures[0].Requirements = []string{"HTTP ServeMux"}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			spec := validAnnotationSpec()
			test.mutate(&spec)
			if _, err := New(spec); err == nil {
				t.Fatal("New() error = nil")
			}
		})
	}
}

func TestParseRejectsMalformedUnknownAndTrailingJSON(t *testing.T) {
	t.Parallel()
	valid, err := Must(validAnnotationSpec()).JSON()
	if err != nil {
		t.Fatal(err)
	}
	unknown := bytes.Replace(valid, []byte(`"schema":`), []byte(`"unknown":true,"schema":`), 1)
	for _, content := range [][]byte{
		[]byte(`{`),
		unknown,
		append(append([]byte(nil), valid...), []byte(`{}`)...),
	} {
		if _, err := Parse(content); err == nil {
			t.Fatalf("Parse(%q) error = nil", content)
		}
	}
}

func TestMustPanicsForInvalidPackageMetadata(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Fatal("Must() did not panic")
		}
	}()
	Must(Spec{})
}

func TestZeroManifestCannotEncodeOrReportCompatibility(t *testing.T) {
	t.Parallel()
	var manifest Manifest
	if _, err := manifest.JSON(); err == nil {
		t.Fatal("zero Manifest.JSON() error = nil")
	}
	if _, err := json.Marshal(manifest); err == nil {
		t.Fatal("json.Marshal(zero Manifest) error = nil")
	}
	if err := manifest.Compatible(APIVersion, "1.26"); err == nil {
		t.Fatal("zero Manifest.Compatible() error = nil")
	}
}

func FuzzParseManifest(f *testing.F) {
	content, err := Must(validAnnotationSpec()).JSON()
	if err != nil {
		f.Fatal(err)
	}
	f.Add(content)
	f.Add([]byte(`{}`))
	f.Add([]byte(`not-json`))
	f.Fuzz(func(t *testing.T, candidate []byte) {
		manifest, err := Parse(candidate)
		if err != nil {
			return
		}
		first, err := manifest.JSON()
		if err != nil {
			t.Fatal(err)
		}
		reparsed, err := Parse(first)
		if err != nil {
			t.Fatal(err)
		}
		second, err := reparsed.JSON()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("manifest JSON is unstable:\n%s\n%s", first, second)
		}
	})
}

func validAnnotationSpec() Spec {
	return Spec{
		Schema:    Schema,
		ID:        "example.com/acme/starter/http",
		Version:   "1.2.3-beta.1",
		Module:    "example.com/acme",
		SpiceAPI:  APIVersion,
		MinimumGo: "1.26",
		License:   "Apache-2.0",
		Review:    "docs/dependency-review.md",
		Activation: Activation{
			Mode: ActivationExplicitAnnotation,
			EntryPoints: []EntryPoint{
				{Package: "example.com/acme/starter/http", Symbol: "Start"},
				{Package: "example.com/acme/starter/http", Symbol: "New"},
			},
		},
		Capabilities: []string{"observability.logging", "http.management"},
		Dependencies: []Dependency{
			{Module: "example.com/zeta", Version: "v2.0.0", License: "MIT"},
			{Module: "example.com/alpha", Version: "v1.4.0", License: "BSD-3-Clause"},
		},
		Annotations: []AnnotationSpec{
			{
				Name:    "acme.Enable",
				Targets: []annotation.Target{annotation.TargetFunction},
				Arguments: []ArgumentSpec{
					{
						Name:     "mode",
						Kinds:    []annotation.Kind{annotation.KindString},
						Required: true,
					},
					{
						Name:             "expose",
						Kinds:            []annotation.Kind{annotation.KindList},
						ListElementKinds: []annotation.Kind{annotation.KindString},
						Required:         true,
					},
				},
			},
		},
		ApplicationFeatures: []FeatureSpec{
			{
				Annotation: "acme.Enable",
				Capability: "http.management",
				EntryPoints: []EntryPoint{
					{Package: "example.com/acme/starter/http", Symbol: "Start"},
					{Package: "example.com/acme/starter/http", Symbol: "New"},
				},
				Options: []OptionSpec{
					{
						Name:           "expose",
						Kind:           annotation.KindList,
						ListItemKinds:  []annotation.Kind{annotation.KindString},
						AllowedStrings: []string{"info", "health"},
						Required:       true,
						UniqueItems:    true,
						MinimumItems:   1,
						SortItems:      true,
					},
				},
				Requirements: []string{"http.serve-mux"},
			},
		},
	}
}

func TestErrorMessagesDoNotContainManifestJSON(t *testing.T) {
	t.Parallel()
	content := []byte(`{"schema":"spice.starter/v1","secret":"raw-token"}`)
	_, err := Parse(content)
	if err == nil {
		t.Fatal("Parse() error = nil")
	}
	if strings.Contains(err.Error(), "raw-token") {
		t.Fatalf("Parse() leaked input value: %v", err)
	}
}
