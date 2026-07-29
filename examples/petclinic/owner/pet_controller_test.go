package owner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/StevenBuglione/spice/examples/petclinic/model"
	"github.com/StevenBuglione/spice/web"
)

type controllerPetTypeRepository struct {
	findAll func(context.Context) ([]PetType, error)
}

func (repository controllerPetTypeRepository) FindAll(
	ctx context.Context,
) ([]PetType, error) {
	return repository.findAll(ctx)
}

func TestPetControllerConstruction(t *testing.T) {
	t.Parallel()

	types := successfulPetTypeRepository()
	if _, err := NewPetController(nil, types, testCatalog(t)); err == nil {
		t.Fatal("nil owner repository succeeded")
	}
	if _, err := NewPetController(
		successfulPetControllerRepository(),
		nil,
		testCatalog(t),
	); err == nil {
		t.Fatal("nil pet type repository succeeded")
	}
	if _, err := NewPetController(
		successfulPetControllerRepository(),
		types,
		testCatalog(t),
	); err != nil {
		t.Fatal(err)
	}
}

func TestPetControllerCreateAndForms(t *testing.T) {
	t.Parallel()

	controller := newTestPetController(t)
	binding, err := web.NewBindingResult()
	if err != nil {
		t.Fatal(err)
	}
	result, err := controller.NewForm(
		context.Background(),
		NewPetRequest{OwnerID: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	if validationErr := result.Validate(); validationErr != nil {
		t.Fatal(validationErr)
	}
	result, err = controller.Create(
		context.Background(),
		PetFormRequest{
			OwnerID:   1,
			Name:      "Comet",
			BirthDate: "2018-02-03",
			TypeID:    2,
		},
		binding,
	)
	if err != nil {
		t.Fatal(err)
	}
	if validationErr := result.Validate(); validationErr != nil {
		t.Fatal(validationErr)
	}
	for _, request := range []PetFormRequest{
		{OwnerID: 1, Name: "Leo", BirthDate: "2018-02-03", TypeID: 2},
		{OwnerID: 1, Name: "Comet", BirthDate: "invalid", TypeID: 99},
		{OwnerID: 1, Name: "Comet", BirthDate: "2027-01-01", TypeID: 2},
	} {
		result, err = controller.Create(
			context.Background(),
			request,
			binding,
		)
		if err != nil {
			t.Fatal(err)
		}
		if validationErr := result.Validate(); validationErr != nil {
			t.Fatal(validationErr)
		}
	}
}

func TestPetControllerEditAndFailureBoundaries(t *testing.T) {
	t.Parallel()

	controller := newTestPetController(t)
	binding, err := web.NewBindingResult()
	if err != nil {
		t.Fatal(err)
	}
	result, err := controller.EditForm(
		context.Background(),
		OwnerPetRequest{OwnerID: 1, PetID: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	if validationErr := result.Validate(); validationErr != nil {
		t.Fatal(validationErr)
	}
	result, err = controller.Update(
		context.Background(),
		EditPetRequest{
			OwnerID:   1,
			PetID:     1,
			Name:      "Leo II",
			BirthDate: "2018-02-03",
			TypeID:    1,
		},
		binding,
	)
	if err != nil {
		t.Fatal(err)
	}
	if validationErr := result.Validate(); validationErr != nil {
		t.Fatal(validationErr)
	}
	for _, request := range []OwnerPetRequest{
		{OwnerID: 0, PetID: 1},
		{OwnerID: 1, PetID: 0},
		{OwnerID: 1, PetID: 99},
	} {
		if _, formErr := controller.EditForm(
			context.Background(),
			request,
		); formErr == nil {
			t.Fatalf("EditForm(%#v) succeeded", request)
		}
	}

	failingTypes := successfulPetTypeRepository()
	failingTypes.findAll = func(context.Context) ([]PetType, error) {
		return nil, errRepository
	}
	controller, err = NewPetController(
		successfulPetControllerRepository(),
		failingTypes,
		testCatalog(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, formErr := controller.NewForm(
		context.Background(),
		NewPetRequest{OwnerID: 1},
	); !errors.Is(formErr, errRepository) {
		t.Fatalf("NewForm() error = %v", formErr)
	}
	if _, formErr := controller.EditForm(
		context.Background(),
		OwnerPetRequest{OwnerID: 1, PetID: 1},
	); !errors.Is(formErr, errRepository) {
		t.Fatalf("EditForm() type error = %v", formErr)
	}
	if _, createErr := controller.Create(
		context.Background(),
		PetFormRequest{
			OwnerID:   1,
			Name:      "Comet",
			BirthDate: "2018-02-03",
			TypeID:    2,
		},
		binding,
	); !errors.Is(createErr, errRepository) {
		t.Fatalf("Create() type error = %v", createErr)
	}

	failingOwners := successfulPetControllerRepository()
	failingOwners.findByID = func(
		context.Context,
		model.ID,
	) (Owner, bool, error) {
		return Owner{}, false, errRepository
	}
	controller, err = NewPetController(
		failingOwners,
		successfulPetTypeRepository(),
		testCatalog(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, formErr := controller.NewForm(
		context.Background(),
		NewPetRequest{OwnerID: 1},
	); !errors.Is(formErr, errRepository) {
		t.Fatalf("NewForm() owner error = %v", formErr)
	}

	failingOwners = successfulPetControllerRepository()
	failingOwners.save = func(context.Context, Owner) (Owner, error) {
		return Owner{}, errRepository
	}
	controller, err = NewPetController(
		failingOwners,
		successfulPetTypeRepository(),
		testCatalog(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, createErr := controller.Create(
		context.Background(),
		PetFormRequest{
			OwnerID:   1,
			Name:      "Comet",
			BirthDate: "2018-02-03",
			TypeID:    2,
		},
		binding,
	); !errors.Is(createErr, errRepository) {
		t.Fatalf("Create() save error = %v", createErr)
	}
	if _, updateErr := controller.Update(
		context.Background(),
		EditPetRequest{
			OwnerID:   1,
			PetID:     1,
			Name:      "Leo II",
			BirthDate: "2018-02-03",
			TypeID:    1,
		},
		binding,
	); !errors.Is(updateErr, errRepository) {
		t.Fatalf("Update() save error = %v", updateErr)
	}

	duplicateOwners := successfulPetControllerRepository()
	originalFind := duplicateOwners.findByID
	duplicateOwners.findByID = func(
		ctx context.Context,
		id model.ID,
	) (Owner, bool, error) {
		value, found, findErr := originalFind(ctx, id)
		if findErr != nil || !found {
			return value, found, findErr
		}
		value.Pets = append(value.Pets, Pet{
			NamedEntity: model.NamedEntity{
				BaseEntity: model.BaseEntity{ID: 2},
				Name:       "Comet",
			},
			BirthDate: time.Date(
				2018,
				2,
				3,
				0,
				0,
				0,
				0,
				time.UTC,
			),
			Type: petTypeFixture(2, "dog"),
		})
		return value, true, nil
	}
	controller, err = NewPetController(
		duplicateOwners,
		successfulPetTypeRepository(),
		testCatalog(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	controller.today = func() time.Time {
		return time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	}
	result, updateErr := controller.Update(
		context.Background(),
		EditPetRequest{
			OwnerID:   1,
			PetID:     1,
			Name:      "Comet",
			BirthDate: "2018-02-03",
			TypeID:    1,
		},
		binding,
	)
	if updateErr != nil {
		t.Fatal(updateErr)
	}
	if validationErr := result.Validate(); validationErr != nil {
		t.Fatal(validationErr)
	}
}

func TestVisitControllerWorkflow(t *testing.T) {
	t.Parallel()

	if _, err := NewVisitController(nil, testCatalog(t)); err == nil {
		t.Fatal("NewVisitController(nil) succeeded")
	}
	controller, err := NewVisitController(
		successfulPetControllerRepository(),
		testCatalog(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := web.NewBindingResult()
	if err != nil {
		t.Fatal(err)
	}
	result, err := controller.NewForm(
		context.Background(),
		NewVisitRequest{OwnerID: 1, PetID: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	if validationErr := result.Validate(); validationErr != nil {
		t.Fatal(validationErr)
	}
	for _, request := range []VisitFormRequest{
		{
			OwnerID:     1,
			PetID:       1,
			Date:        "2026-07-28",
			Description: "annual wellness examination",
		},
		{
			OwnerID: 1,
			PetID:   1,
			Date:    "invalid",
		},
	} {
		result, err = controller.Create(
			context.Background(),
			request,
			binding,
		)
		if err != nil {
			t.Fatal(err)
		}
		if validationErr := result.Validate(); validationErr != nil {
			t.Fatal(validationErr)
		}
	}

	failing := successfulPetControllerRepository()
	failing.save = func(context.Context, Owner) (Owner, error) {
		return Owner{}, errRepository
	}
	controller, err = NewVisitController(failing, testCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, createErr := controller.Create(
		context.Background(),
		VisitFormRequest{
			OwnerID:     1,
			PetID:       1,
			Date:        "2026-07-28",
			Description: "annual wellness examination",
		},
		binding,
	); !errors.Is(createErr, errRepository) {
		t.Fatalf("Create() save error = %v", createErr)
	}
	if _, formErr := controller.NewForm(
		context.Background(),
		NewVisitRequest{OwnerID: 1, PetID: 99},
	); !errors.Is(formErr, ErrPetNotFound) {
		t.Fatalf("NewForm() missing pet error = %v", formErr)
	}
	failing.findByID = func(
		context.Context,
		model.ID,
	) (Owner, bool, error) {
		return Owner{}, false, errRepository
	}
	controller, err = NewVisitController(failing, testCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, createErr := controller.Create(
		context.Background(),
		VisitFormRequest{OwnerID: 1, PetID: 1},
		binding,
	); !errors.Is(createErr, errRepository) {
		t.Fatalf("Create() owner error = %v", createErr)
	}
}

func newTestPetController(t *testing.T) *PetController {
	t.Helper()

	controller, err := NewPetController(
		successfulPetControllerRepository(),
		successfulPetTypeRepository(),
		testCatalog(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	controller.today = func() time.Time {
		return time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	}
	return controller
}

func successfulPetControllerRepository() controllerRepository {
	value := ownerFixture(1, "Franklin")
	value.Pets = []Pet{{
		NamedEntity: model.NamedEntity{
			BaseEntity: model.BaseEntity{ID: 1},
			Name:       "Leo",
		},
		BirthDate: time.Date(2010, 9, 7, 0, 0, 0, 0, time.UTC),
		Type:      petTypeFixture(1, "cat"),
	}}
	return controllerRepository{
		findByID: func(
			_ context.Context,
			id model.ID,
		) (Owner, bool, error) {
			if id != value.ID {
				return Owner{}, false, nil
			}
			return value.Clone(), true, nil
		},
		findByLastName: func(
			context.Context,
			string,
			int,
			int,
		) ([]Owner, int, error) {
			return []Owner{value.Clone()}, 1, nil
		},
		save: func(_ context.Context, saved Owner) (Owner, error) {
			return saved, nil
		},
	}
}

func successfulPetTypeRepository() controllerPetTypeRepository {
	return controllerPetTypeRepository{
		findAll: func(context.Context) ([]PetType, error) {
			return []PetType{
				petTypeFixture(1, "cat"),
				petTypeFixture(2, "dog"),
			}, nil
		},
	}
}

func petTypeFixture(id model.ID, name string) PetType {
	return PetType{NamedEntity: model.NamedEntity{
		BaseEntity: model.BaseEntity{ID: id},
		Name:       name,
	}}
}
