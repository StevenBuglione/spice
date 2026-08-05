package starter_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spice-framework/spice/starter"
)

func TestManifestCompatibilitySurface(t *testing.T) {
	t.Parallel()
	spec := starter.Spec{
		Schema:    starter.Schema,
		ID:        "example.com/acme/starter/search",
		Module:    "example.com/acme/starter",
		Version:   "1.2.3",
		SpiceAPI:  starter.APIVersion,
		MinimumGo: "1.26",
		License:   "Apache-2.0",
		Review:    "docs/dependency-review.md",
		Capabilities: []string{
			"search.client",
		},
		Activation: starter.Activation{
			Mode: starter.ActivationExplicitConstructor,
			EntryPoints: []starter.EntryPoint{{
				Package: "example.com/acme/starter/search",
				Symbol:  "New",
			}},
		},
	}
	manifest, err := starter.New(spec)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	content, err := manifest.JSON()
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	parsed, err := starter.Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if parsed.Spec().ID != spec.ID {
		t.Fatalf("parsed ID = %q, want %q", parsed.Spec().ID, spec.ID)
	}
	if got := starter.Must(spec).Spec().ID; got != spec.ID {
		t.Fatalf("Must().Spec().ID = %q, want %q", got, spec.ID)
	}
}

func TestManifestCompatibilitySurfaceRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	if _, err := starter.New(starter.Spec{}); err == nil {
		t.Fatal("New() accepted an empty specification")
	}
	if _, err := starter.Parse([]byte(`{"schema":"unknown"}`)); err == nil {
		t.Fatal("Parse() accepted an unsupported schema")
	}

	deferred := false
	func() {
		defer func() {
			value := recover()
			deferred = value != nil && strings.Contains(fmt.Sprint(value), "manifest")
		}()
		starter.Must(starter.Spec{})
	}()
	if !deferred {
		t.Fatal("Must() did not panic with a manifest validation error")
	}
}
