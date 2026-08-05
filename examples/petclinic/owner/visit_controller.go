package owner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spice-framework/spice/examples/petclinic/presentation"
	"github.com/spice-framework/spice/i18n"
	"github.com/spice-framework/spice/view"
	"github.com/spice-framework/spice/web"
)

// @import { Controller } from "github.com/spice-framework/spice/annotation/web"
// @import { Get, Post } from "github.com/spice-framework/spice/annotation/web"

// VisitController serves visit registration inside a pet aggregate.
//
// @Controller
type VisitController struct {
	owners   Repository
	messages *i18n.Catalog
}

// NewVisitController constructs the visit HTTP boundary.
func NewVisitController(
	owners Repository,
	messages *i18n.Catalog,
) (*VisitController, error) {
	if owners == nil {
		return nil, errors.New(
			"construct visit controller: owner repository is nil",
		)
	}
	if messages == nil {
		return nil, errors.New(
			"construct visit controller: message catalog is nil",
		)
	}
	return &VisitController{owners: owners, messages: messages}, nil
}

// NewForm renders a visit form with the pet's visit history.
//
// @Get("/owners/{ownerId}/pets/{petId}/visits/new")
func (controller *VisitController) NewForm(
	ctx context.Context,
	request NewVisitRequest,
) (view.Result, error) {
	value, pet, err := controller.ownerPet(
		ctx,
		request.OwnerID,
		request.PetID,
	)
	if err != nil {
		return view.Result{}, err
	}
	return view.Render("pets/createOrUpdateVisitForm", VisitFormModel{
		Page:  controller.page(request.Language),
		Owner: value,
		Pet:   pet,
	})
}

// Create validates and persists one visit.
//
// @Post("/owners/{ownerId}/pets/{petId}/visits/new")
func (controller *VisitController) Create(
	ctx context.Context,
	request VisitFormRequest,
	binding web.BindingResult,
) (view.Result, error) {
	value, pet, err := controller.ownerPet(
		ctx,
		request.OwnerID,
		request.PetID,
	)
	if err != nil {
		return view.Result{}, err
	}
	visit := Visit{Description: request.Description}
	parsedDate, parseErr := time.Parse(time.DateOnly, request.Date)
	if parseErr != nil {
		binding, err = rejectFieldOnce(
			binding,
			"date",
			"date.invalid",
			"must use YYYY-MM-DD",
		)
		if err != nil {
			return view.Result{}, err
		}
	} else {
		visit.Date = parsedDate
	}
	validationResult, err := visit.Validate()
	if err != nil {
		return view.Result{}, err
	}
	for _, violation := range validationResult.All() {
		binding, err = rejectFieldOnce(
			binding,
			violation.Field,
			violation.Code,
			violation.Message,
		)
		if err != nil {
			return view.Result{}, err
		}
	}
	if !binding.Valid() {
		return view.Render("pets/createOrUpdateVisitForm", VisitFormModel{
			Page:   controller.page(request.Language),
			Owner:  value,
			Pet:    pet,
			Visit:  visit,
			Errors: binding.Errors().All(),
		})
	}
	if err := value.AddVisit(pet.ID, visit); err != nil {
		return view.Result{}, fmt.Errorf("add pet visit: %w", err)
	}
	if _, err := controller.owners.Save(ctx, value); err != nil {
		return view.Result{}, fmt.Errorf("save pet visit: %w", err)
	}
	return ownerRedirect(value.ID)
}

func (controller *VisitController) page(language string) presentation.Page {
	return presentation.NewPage(controller.messages, language, "owners")
}

func (controller *VisitController) ownerPet(
	ctx context.Context,
	ownerID int,
	petID int,
) (Owner, Pet, error) {
	pets := PetController{owners: controller.owners}
	return pets.ownerPet(ctx, ownerID, petID)
}
