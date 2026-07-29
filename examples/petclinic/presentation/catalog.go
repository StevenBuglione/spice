package presentation

import (
	"embed"

	"github.com/StevenBuglione/spice/i18n"
)

// @import { Bean } from "github.com/StevenBuglione/spice/annotation/core"

//go:embed messages/*.properties
var messageFiles embed.FS

// NewCatalog constructs the immutable Petclinic message catalog.
//
// @Bean
func NewCatalog() (*i18n.Catalog, error) {
	return i18n.ParseProperties(messageFiles, "messages/*.properties", "en")
}
