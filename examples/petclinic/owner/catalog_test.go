package owner

import (
	"testing"

	"github.com/StevenBuglione/spice/examples/petclinic/presentation"
	"github.com/StevenBuglione/spice/i18n"
)

func testCatalog(t *testing.T) *i18n.Catalog {
	t.Helper()
	catalog, err := presentation.NewCatalog()
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}
