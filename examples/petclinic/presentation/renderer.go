package presentation

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/StevenBuglione/spice/i18n"
	"github.com/StevenBuglione/spice/view"
	"github.com/StevenBuglione/spice/web"
)

// @import { Bean } from "github.com/StevenBuglione/spice/annotation/core"

//go:embed templates/*.html
var templates embed.FS

// NewRenderer parses all Petclinic templates once during construction.
//
// @Bean
func NewRenderer(catalog *i18n.Catalog) (*view.Renderer, error) {
	return view.Parse(
		templates,
		[]string{"templates/*.html"},
		template.FuncMap{
			"msg": catalog.Message,
		},
		view.Options{
			ErrorTemplate: "error",
			ErrorModel: func(
				request *http.Request,
				problem web.Problem,
			) any {
				return ErrorModel{
					Page: NewPage(
						catalog,
						request.Header.Get("Accept-Language"),
						"error",
					),
					Status: problem.Status,
					Title:  problem.Title,
					Detail: problem.Detail,
				}
			},
		},
	)
}
