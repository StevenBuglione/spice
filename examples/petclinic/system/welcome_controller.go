// Package system owns application-wide Petclinic web behavior.
package system

import (
	"context"
	"errors"

	"github.com/StevenBuglione/spice/examples/petclinic/presentation"
	"github.com/StevenBuglione/spice/i18n"
	"github.com/StevenBuglione/spice/view"
)

// @import { Controller } from "github.com/StevenBuglione/spice/annotation/web"
// @import { Get } from "github.com/StevenBuglione/spice/annotation/web"

// WelcomeController serves the recognizable Petclinic landing page.
//
// @Controller
type WelcomeController struct {
	messages *i18n.Catalog
}

// NewWelcomeController constructs the stateless landing-page controller.
func NewWelcomeController(messages *i18n.Catalog) (*WelcomeController, error) {
	if messages == nil {
		return nil, errors.New(
			"construct welcome controller: message catalog is nil",
		)
	}
	return &WelcomeController{messages: messages}, nil
}

// Show renders the Petclinic welcome page.
//
// @Get("/")
func (controller *WelcomeController) Show(
	_ context.Context,
	request WelcomeRequest,
) (view.Result, error) {
	return view.Render("welcome", WelcomeModel{
		Page: presentation.NewPage(controller.messages, request.Language, "home"),
	})
}
