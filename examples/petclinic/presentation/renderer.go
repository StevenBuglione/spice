package presentation

import (
	"embed"

	"github.com/StevenBuglione/spice/view"
)

// @import { Bean } from "github.com/StevenBuglione/spice/annotation/core"

//go:embed templates/*.html
var templates embed.FS

// NewRenderer parses all Petclinic templates once during construction.
//
// @Bean
func NewRenderer() (*view.Renderer, error) {
	return view.Parse(
		templates,
		[]string{"templates/*.html"},
		nil,
		view.Options{},
	)
}
