// Package system owns application-wide Petclinic web behavior.
package system

import (
	"context"

	"github.com/StevenBuglione/spice/view"
)

// @import { Controller } from "github.com/StevenBuglione/spice/annotation/web"
// @import { Get } from "github.com/StevenBuglione/spice/annotation/web"

// WelcomeController serves the recognizable Petclinic landing page.
//
// @Controller
type WelcomeController struct{}

// NewWelcomeController constructs the stateless landing-page controller.
func NewWelcomeController() *WelcomeController {
	return &WelcomeController{}
}

// Show renders the Petclinic welcome page.
//
// @Get("/")
func (*WelcomeController) Show(
	_ context.Context,
	_ WelcomeRequest,
) (view.Result, error) {
	return view.Render("welcome", struct{}{})
}
