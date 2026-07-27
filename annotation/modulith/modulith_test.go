package modulith

import (
	"testing"

	"github.com/StevenBuglione/spice/annotation/sdk"
)

func TestModulithDefinitions(t *testing.T) {
	t.Parallel()
	for _, definition := range []sdk.Definition{Module(), NamedInterface()} {
		if err := definition.Validate(); err != nil {
			t.Fatalf("%s definition: %v", definition.Name, err)
		}
	}
}
