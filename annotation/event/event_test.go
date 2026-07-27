package event

import (
	"testing"

	"github.com/StevenBuglione/spice/annotation/sdk"
)

func TestEventDefinitions(t *testing.T) {
	t.Parallel()
	for _, definition := range []sdk.Definition{Listener(), Topic()} {
		if err := definition.Validate(); err != nil {
			t.Fatalf("%s definition: %v", definition.Name, err)
		}
	}
}
