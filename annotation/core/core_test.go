package core

import (
	"testing"

	"github.com/StevenBuglione/spice/annotation/sdk"
)

func TestCoreDefinitions(t *testing.T) {
	t.Parallel()
	for _, definition := range []sdk.Definition{
		Application(),
		Bean(),
		Configuration(),
		Service(),
	} {
		if err := definition.Validate(); err != nil {
			t.Fatalf("%s definition: %v", definition.Name, err)
		}
	}
}
