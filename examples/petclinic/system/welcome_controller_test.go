package system

import (
	"testing"

	"github.com/StevenBuglione/spice/examples/petclinic/presentation"
)

func TestWelcomeControllerRendersCanonicalPage(t *testing.T) {
	t.Parallel()

	catalog, err := presentation.NewCatalog()
	if err != nil {
		t.Fatal(err)
	}
	controller, err := NewWelcomeController(catalog)
	if err != nil {
		t.Fatal(err)
	}
	result, err := controller.Show(t.Context(), WelcomeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("result validation: %v", err)
	}
}
