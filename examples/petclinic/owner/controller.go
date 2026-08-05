package owner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/spice-framework/spice/examples/petclinic/model"
	"github.com/spice-framework/spice/examples/petclinic/presentation"
	"github.com/spice-framework/spice/i18n"
	"github.com/spice-framework/spice/validation"
	"github.com/spice-framework/spice/view"
	"github.com/spice-framework/spice/web"
)

// @import { Controller } from "github.com/spice-framework/spice/annotation/web"
// @import { Get, Post } from "github.com/spice-framework/spice/annotation/web"

const ownerPageSize = 5

// Controller serves the Petclinic owner workflows.
//
// @Controller
type Controller struct {
	owners   Repository
	messages *i18n.Catalog
}

// NewController constructs the owner HTTP boundary.
func NewController(
	owners Repository,
	messages *i18n.Catalog,
) (*Controller, error) {
	if owners == nil {
		return nil, errors.New("construct owner controller: repository is nil")
	}
	if messages == nil {
		return nil, errors.New("construct owner controller: message catalog is nil")
	}
	return &Controller{owners: owners, messages: messages}, nil
}

// NewForm renders an empty owner form.
//
// @Get("/owners/new")
func (controller *Controller) NewForm(
	_ context.Context,
	request NewOwnerRequest,
) (view.Result, error) {
	return view.Render("owners/createOrUpdateOwnerForm", OwnerFormModel{
		Page:     controller.page(request.Language),
		Creating: true,
	})
}

// Create validates and persists a new owner.
//
// @Post("/owners/new")
func (controller *Controller) Create(
	ctx context.Context,
	request OwnerFormRequest,
	binding web.BindingResult,
) (view.Result, error) {
	value := request.Owner()
	result, err := validateOwnerBinding(value, binding)
	if err != nil {
		return view.Result{}, err
	}
	if !result.Valid() {
		return view.Render("owners/createOrUpdateOwnerForm", OwnerFormModel{
			Page:     controller.page(request.Language),
			Owner:    value,
			Errors:   result.Errors().All(),
			Creating: true,
		})
	}
	saved, err := controller.owners.Save(ctx, value)
	if err != nil {
		return view.Result{}, fmt.Errorf("create owner: %w", err)
	}
	return view.SeeOther("/owners/" + strconv.FormatInt(
		int64(saved.ID),
		10,
	))
}

// FindForm renders the owner search form.
//
// @Get("/owners/find")
func (controller *Controller) FindForm(
	_ context.Context,
	request FindOwnerFormRequest,
) (view.Result, error) {
	return view.Render("owners/findOwners", FindOwnersModel{
		Page: controller.page(request.Language),
	})
}

// Find searches owners by last-name prefix with Petclinic pagination.
//
// @Get("/owners")
func (controller *Controller) Find(
	ctx context.Context,
	request FindOwnersRequest,
) (view.Result, error) {
	page := request.Page
	if page == 0 {
		page = 1
	}
	if page < 1 {
		return view.Render("owners/findOwners", FindOwnersModel{
			Page:     controller.page(request.Language),
			LastName: request.LastName,
			Errors: []validation.Violation{{
				Field:   "page",
				Code:    "page.invalid",
				Message: "must be positive",
			}},
		})
	}
	owners, total, err := controller.owners.FindByLastName(
		ctx,
		request.LastName,
		(page-1)*ownerPageSize,
		ownerPageSize,
	)
	if err != nil {
		return view.Result{}, fmt.Errorf("find owners: %w", err)
	}
	if total == 0 {
		return view.Render("owners/findOwners", FindOwnersModel{
			Page:     controller.page(request.Language),
			LastName: request.LastName,
			Errors: []validation.Violation{{
				Field:   "lastName",
				Code:    "notFound",
				Message: "not found",
			}},
		})
	}
	if total == 1 {
		return view.SeeOther("/owners/" + strconv.FormatInt(
			int64(owners[0].ID),
			10,
		))
	}
	totalPages := (total + ownerPageSize - 1) / ownerPageSize
	return view.Render("owners/ownersList", OwnersListModel{
		Page:         controller.page(request.Language),
		Owners:       owners,
		LastName:     request.LastName,
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalItems:   total,
		PreviousPage: page - 1,
		NextPage:     page + 1,
		HasPrevious:  page > 1,
		HasNext:      page < totalPages,
	})
}

// EditForm renders one persisted owner for editing.
//
// @Get("/owners/{ownerId}/edit")
func (controller *Controller) EditForm(
	ctx context.Context,
	request OwnerIDRequest,
) (view.Result, error) {
	value, err := controller.owner(ctx, request.OwnerID)
	if err != nil {
		return view.Result{}, err
	}
	return view.Render("owners/createOrUpdateOwnerForm", OwnerFormModel{
		Page:  controller.page(request.Language),
		Owner: value,
	})
}

// Update validates and persists owner field changes without replacing pets.
//
// @Post("/owners/{ownerId}/edit")
func (controller *Controller) Update(
	ctx context.Context,
	request EditOwnerRequest,
	binding web.BindingResult,
) (view.Result, error) {
	value, err := controller.owner(ctx, request.OwnerID)
	if err != nil {
		return view.Result{}, err
	}
	request.Apply(&value)
	result, err := validateOwnerBinding(value, binding)
	if err != nil {
		return view.Result{}, err
	}
	if !result.Valid() {
		return view.Render("owners/createOrUpdateOwnerForm", OwnerFormModel{
			Page:   controller.page(request.Language),
			Owner:  value,
			Errors: result.Errors().All(),
		})
	}
	if _, err := controller.owners.Save(ctx, value); err != nil {
		return view.Result{}, fmt.Errorf("update owner %d: %w", value.ID, err)
	}
	return view.SeeOther("/owners/" + strconv.FormatInt(
		int64(value.ID),
		10,
	))
}

// Show renders one owner aggregate.
//
// @Get("/owners/{ownerId}")
func (controller *Controller) Show(
	ctx context.Context,
	request OwnerIDRequest,
) (view.Result, error) {
	value, err := controller.owner(ctx, request.OwnerID)
	if err != nil {
		return view.Result{}, err
	}
	return view.Render("owners/ownerDetails", OwnerDetailsModel{
		Page:  controller.page(request.Language),
		Owner: value,
	})
}

func (controller *Controller) page(language string) presentation.Page {
	return presentation.NewPage(controller.messages, language, "owners")
}

func (controller *Controller) owner(
	ctx context.Context,
	id int,
) (Owner, error) {
	if id < 1 {
		return Owner{}, ownerNotFound(model.ID(id))
	}
	value, found, err := controller.owners.FindByID(ctx, model.ID(id))
	if err != nil {
		return Owner{}, fmt.Errorf("find owner %d: %w", id, err)
	}
	if !found {
		return Owner{}, ownerNotFound(model.ID(id))
	}
	return value, nil
}

func ownerNotFound(id model.ID) error {
	return web.NewError(web.Problem{
		Type:   "https://spice.dev/problems/petclinic-owner-not-found",
		Title:  "Owner not found",
		Status: http.StatusNotFound,
		Detail: "The requested owner does not exist.",
	}, fmt.Errorf("owner %d: %w", id, ErrOwnerNotFound))
}

func validateOwnerBinding(
	value Owner,
	binding web.BindingResult,
) (web.BindingResult, error) {
	result, err := value.Validate()
	if err != nil {
		return web.BindingResult{}, err
	}
	for _, violation := range result.All() {
		binding, err = binding.Reject(
			violation.Field,
			violation.Code,
			violation.Message,
		)
		if err != nil {
			return web.BindingResult{}, err
		}
	}
	return binding, nil
}
