package lifecycle

import (
	"testing"

	"github.com/StevenBuglione/spice/annotation/sdk"
)

func TestLifecycleDefinitions(t *testing.T) {
	t.Parallel()
	for _, definition := range []sdk.Definition{OnStart(), OnStop()} {
		if err := definition.Validate(); err != nil {
			t.Fatalf("%s definition: %v", definition.Name, err)
		}
	}
}
