package owner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/StevenBuglione/spice/examples/petclinic/model"
	"github.com/StevenBuglione/spice/view"
	"github.com/StevenBuglione/spice/web"
)

// @import { Controller } from "github.com/StevenBuglione/spice/annotation/web"
// @import { Get, Post } from "github.com/StevenBuglione/spice/annotation/web"

// PetController serves pet creation and editing inside an owner aggregate.
//
// @Controller
type PetController struct {
	owners   Repository
	petTypes PetTypeRepository
	today    func() time.Time
}

// NewPetController constructs the pet HTTP boundary.
func NewPetController(
	owners Repository,
	petTypes PetTypeRepository,
) (*PetController, error) {
	if owners == nil {
		return nil, errors.New("construct pet controller: owner repository is nil")
	}
	if petTypes == nil {
		return nil, errors.New("construct pet controller: pet type repository is nil")
	}
	return &PetController{
		owners:   owners,
		petTypes: petTypes,
		today:    time.Now,
	}, nil
}

// NewForm renders a new-pet form for one owner.
//
// @Get("/owners/{ownerId}/pets/new")
func (controller *PetController) NewForm(
	ctx context.Context,
	request NewPetRequest,
) (view.Result, error) {
	value, err := controller.owner(ctx, request.OwnerID)
	if err != nil {
		return view.Result{}, err
	}
	types, err := controller.types(ctx)
	if err != nil {
		return view.Result{}, err
	}
	return view.Render("pets/createOrUpdatePetForm", PetFormModel{
		Owner:    value,
		PetTypes: types,
		Creating: true,
	})
}

// Create validates and adds a pet to one owner.
//
// @Post("/owners/{ownerId}/pets/new")
func (controller *PetController) Create(
	ctx context.Context,
	request PetFormRequest,
	binding web.BindingResult,
) (view.Result, error) {
	value, err := controller.owner(ctx, request.OwnerID)
	if err != nil {
		return view.Result{}, err
	}
	types, err := controller.types(ctx)
	if err != nil {
		return view.Result{}, err
	}
	pet, result, err := controller.pet(
		request.Name,
		request.BirthDate,
		request.TypeID,
		types,
		binding,
	)
	if err != nil {
		return view.Result{}, err
	}
	if _, duplicate := value.PetByName(pet.Name, false); duplicate {
		result, err = result.Reject(
			"name",
			"duplicate",
			"is already in use",
		)
		if err != nil {
			return view.Result{}, err
		}
	}
	if !result.Valid() {
		return view.Render("pets/createOrUpdatePetForm", PetFormModel{
			Owner:    value,
			Pet:      pet,
			PetTypes: types,
			Errors:   result.Errors().All(),
			Creating: true,
		})
	}
	if err := value.AddPet(pet); err != nil {
		return view.Result{}, fmt.Errorf("add owner pet: %w", err)
	}
	if _, err := controller.owners.Save(ctx, value); err != nil {
		return view.Result{}, fmt.Errorf("save owner pet: %w", err)
	}
	return ownerRedirect(value.ID)
}

// EditForm renders one existing pet for editing.
//
// @Get("/owners/{ownerId}/pets/{petId}/edit")
func (controller *PetController) EditForm(
	ctx context.Context,
	request OwnerPetRequest,
) (view.Result, error) {
	value, pet, err := controller.ownerPet(
		ctx,
		request.OwnerID,
		request.PetID,
	)
	if err != nil {
		return view.Result{}, err
	}
	types, err := controller.types(ctx)
	if err != nil {
		return view.Result{}, err
	}
	return view.Render("pets/createOrUpdatePetForm", PetFormModel{
		Owner:    value,
		Pet:      pet,
		PetTypes: types,
	})
}

// Update validates and replaces one pet while preserving its visits.
//
// @Post("/owners/{ownerId}/pets/{petId}/edit")
func (controller *PetController) Update(
	ctx context.Context,
	request EditPetRequest,
	binding web.BindingResult,
) (view.Result, error) {
	value, existing, err := controller.ownerPet(
		ctx,
		request.OwnerID,
		request.PetID,
	)
	if err != nil {
		return view.Result{}, err
	}
	types, err := controller.types(ctx)
	if err != nil {
		return view.Result{}, err
	}
	pet, result, err := controller.pet(
		request.Name,
		request.BirthDate,
		request.TypeID,
		types,
		binding,
	)
	if err != nil {
		return view.Result{}, err
	}
	pet.ID = existing.ID
	pet.Visits = existing.Visits
	if duplicate, found := value.PetByName(pet.Name, false); found &&
		duplicate.ID != existing.ID {
		result, err = result.Reject(
			"name",
			"duplicate",
			"is already in use",
		)
		if err != nil {
			return view.Result{}, err
		}
	}
	if !result.Valid() {
		return view.Render("pets/createOrUpdatePetForm", PetFormModel{
			Owner:    value,
			Pet:      pet,
			PetTypes: types,
			Errors:   result.Errors().All(),
		})
	}
	for index := range value.Pets {
		if value.Pets[index].ID == existing.ID {
			value.Pets[index] = pet
			break
		}
	}
	if _, err := controller.owners.Save(ctx, value); err != nil {
		return view.Result{}, fmt.Errorf("save edited pet: %w", err)
	}
	return ownerRedirect(value.ID)
}

func (controller *PetController) pet(
	name string,
	birthDate string,
	typeID int,
	types []PetType,
	binding web.BindingResult,
) (Pet, web.BindingResult, error) {
	pet := Pet{}
	pet.Name = name
	parsedDate, err := time.Parse(time.DateOnly, birthDate)
	if err != nil {
		binding, err = rejectFieldOnce(
			binding,
			"birthDate",
			"date.invalid",
			"must use YYYY-MM-DD",
		)
		if err != nil {
			return Pet{}, web.BindingResult{}, err
		}
	} else {
		pet.BirthDate = parsedDate
	}
	for _, candidate := range types {
		if candidate.ID == model.ID(typeID) {
			pet.Type = candidate
			break
		}
	}
	validationResult, err := pet.Validate(controller.today().UTC())
	if err != nil {
		return Pet{}, web.BindingResult{}, err
	}
	for _, violation := range validationResult.All() {
		binding, err = rejectFieldOnce(
			binding,
			violation.Field,
			violation.Code,
			violation.Message,
		)
		if err != nil {
			return Pet{}, web.BindingResult{}, err
		}
	}
	return pet, binding, nil
}

func (controller *PetController) owner(
	ctx context.Context,
	id int,
) (Owner, error) {
	if id < 1 {
		return Owner{}, ownerNotFound(model.ID(id))
	}
	value, found, err := controller.owners.FindByID(ctx, model.ID(id))
	if err != nil {
		return Owner{}, fmt.Errorf("find pet owner %d: %w", id, err)
	}
	if !found {
		return Owner{}, ownerNotFound(model.ID(id))
	}
	return value, nil
}

func (controller *PetController) ownerPet(
	ctx context.Context,
	ownerID int,
	petID int,
) (Owner, Pet, error) {
	value, err := controller.owner(ctx, ownerID)
	if err != nil {
		return Owner{}, Pet{}, err
	}
	pet, found := value.PetByID(model.ID(petID))
	if petID < 1 || !found {
		return Owner{}, Pet{}, petNotFound(
			model.ID(ownerID),
			model.ID(petID),
		)
	}
	return value, pet, nil
}

func (controller *PetController) types(
	ctx context.Context,
) ([]PetType, error) {
	types, err := controller.petTypes.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("find pet types: %w", err)
	}
	return types, nil
}

func petNotFound(ownerID, petID model.ID) error {
	return web.NewError(web.Problem{
		Type:   "https://spice.dev/problems/petclinic-pet-not-found",
		Title:  "Pet not found",
		Status: http.StatusNotFound,
		Detail: "The requested pet does not belong to this owner.",
	}, fmt.Errorf(
		"owner %d pet %d: %w",
		ownerID,
		petID,
		ErrPetNotFound,
	))
}

func ownerRedirect(id model.ID) (view.Result, error) {
	return view.SeeOther(
		"/owners/" + strconv.FormatInt(int64(id), 10),
	)
}

func rejectFieldOnce(
	binding web.BindingResult,
	field string,
	code string,
	message string,
) (web.BindingResult, error) {
	if len(binding.Errors().ForField(field)) != 0 {
		return binding, nil
	}
	return binding.Reject(
		strings.TrimSpace(field),
		code,
		message,
	)
}
