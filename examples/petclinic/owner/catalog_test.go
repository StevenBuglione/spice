package owner

import (
	"testing"

	"github.com/spice-framework/spice/examples/petclinic/presentation"
	"github.com/spice-framework/spice/i18n"
)

func testCatalog(t *testing.T) *i18n.Catalog {
	t.Helper()
	catalog, err := presentation.NewCatalog()
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}
