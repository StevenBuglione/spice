package web

import (
	"testing"

	"github.com/StevenBuglione/spice/annotation/sdk"
)

func TestWebDefinitions(t *testing.T) {
	t.Parallel()
	for _, definition := range []sdk.Definition{
		Controller(),
		Get(),
		Post(),
	} {
		if err := definition.Validate(); err != nil {
			t.Fatalf("%s definition: %v", definition.Name, err)
		}
	}
}
