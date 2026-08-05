package owner

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spice-framework/spice/examples/petclinic/model"
	"github.com/spice-framework/spice/web"
)

var errRepository = errors.New("repository failed")

type controllerRepository struct {
	findByID       func(context.Context, model.ID) (Owner, bool, error)
	findByLastName func(context.Context, string, int, int) ([]Owner, int, error)
	save           func(context.Context, Owner) (Owner, error)
}

func (repository controllerRepository) FindByID(
	ctx context.Context,
	id model.ID,
) (Owner, bool, error) {
	return repository.findByID(ctx, id)
}

func (repository controllerRepository) FindByLastName(
	ctx context.Context,
	lastName string,
	offset int,
	limit int,
) ([]Owner, int, error) {
	return repository.findByLastName(ctx, lastName, offset, limit)
}

func (repository controllerRepository) Save(
	ctx context.Context,
	value Owner,
) (Owner, error) {
	return repository.save(ctx, value)
}

func TestControllerConstructionAndStaticForms(t *testing.T) {
	t.Parallel()

	if _, err := NewController(nil, testCatalog(t)); err == nil {
		t.Fatal("NewController(nil) succeeded")
	}
	controller, err := NewController(successfulControllerRepository(), testCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	result, err := controller.NewForm(context.Background(), NewOwnerRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if validationErr := result.Validate(); validationErr != nil {
		t.Fatal(validationErr)
	}
	result, err = controller.FindForm(
		context.Background(),
		FindOwnerFormRequest{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if validationErr := result.Validate(); validationErr != nil {
		t.Fatal(validationErr)
	}
}

func TestControllerCreate(t *testing.T) {
	t.Parallel()

	binding, err := web.NewBindingResult()
	if err != nil {
		t.Fatal(err)
	}
	controller, err := NewController(successfulControllerRepository(), testCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	invalid, err := controller.Create(
		context.Background(),
		OwnerFormRequest{},
		binding,
	)
	if err != nil {
		t.Fatal(err)
	}
	if validationErr := invalid.Validate(); validationErr != nil {
		t.Fatal(validationErr)
	}
	created, err := controller.Create(
		context.Background(),
		validOwnerFormRequest("Ada", "Lovelace"),
		binding,
	)
	if err != nil {
		t.Fatal(err)
	}
	if validationErr := created.Validate(); validationErr != nil {
		t.Fatal(validationErr)
	}

	failing := successfulControllerRepository()
	failing.save = func(context.Context, Owner) (Owner, error) {
		return Owner{}, errRepository
	}
	controller, err = NewController(failing, testCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controller.Create(
		context.Background(),
		validOwnerFormRequest("Ada", "Lovelace"),
		binding,
	); !errors.Is(err, errRepository) {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestControllerFindBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		request    FindOwnersRequest
		repository controllerRepository
		wantError  bool
	}{
		{
			name:       "invalid page",
			request:    FindOwnersRequest{Page: -1},
			repository: successfulControllerRepository(),
		},
		{
			name:    "no owners",
			request: FindOwnersRequest{LastName: "missing"},
			repository: repositoryWithOwners(
				nil,
				0,
			),
		},
		{
			name:    "single owner",
			request: FindOwnersRequest{LastName: "Franklin"},
			repository: repositoryWithOwners(
				[]Owner{ownerFixture(1, "Franklin")},
				1,
			),
		},
		{
			name:    "multiple owners and default page",
			request: FindOwnersRequest{LastName: "Davis"},
			repository: repositoryWithOwners(
				[]Owner{
					ownerFixture(2, "Davis"),
					ownerFixture(3, "Davis"),
				},
				7,
			),
		},
		{
			name:    "last page",
			request: FindOwnersRequest{LastName: "Davis", Page: 2},
			repository: repositoryWithOwners(
				[]Owner{ownerFixture(3, "Davis")},
				6,
			),
		},
		{
			name:    "repository failure",
			request: FindOwnersRequest{LastName: "Davis"},
			repository: controllerRepository{
				findByLastName: func(
					context.Context,
					string,
					int,
					int,
				) ([]Owner, int, error) {
					return nil, 0, errRepository
				},
			},
			wantError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			controller, err := NewController(test.repository, testCatalog(t))
			if err != nil {
				t.Fatal(err)
			}
			result, err := controller.Find(
				context.Background(),
				test.request,
			)
			if test.wantError {
				if !errors.Is(err, errRepository) {
					t.Fatalf("Find() error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := result.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestControllerShowEditAndUpdate(t *testing.T) {
	t.Parallel()

	binding, err := web.NewBindingResult()
	if err != nil {
		t.Fatal(err)
	}
	controller, err := NewController(successfulControllerRepository(), testCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []struct {
		name string
		run  func() error
	}{
		{
			name: "show",
			run: func() error {
				result, runErr := controller.Show(
					context.Background(),
					OwnerIDRequest{OwnerID: 1},
				)
				if runErr != nil {
					return runErr
				}
				return result.Validate()
			},
		},
		{
			name: "edit form",
			run: func() error {
				result, runErr := controller.EditForm(
					context.Background(),
					OwnerIDRequest{OwnerID: 1},
				)
				if runErr != nil {
					return runErr
				}
				return result.Validate()
			},
		},
		{
			name: "invalid update",
			run: func() error {
				result, runErr := controller.Update(
					context.Background(),
					EditOwnerRequest{OwnerID: 1},
					binding,
				)
				if runErr != nil {
					return runErr
				}
				return result.Validate()
			},
		},
		{
			name: "valid update",
			run: func() error {
				request := validEditOwnerRequest(1, "Georgina", "Franklin")
				result, runErr := controller.Update(
					context.Background(),
					request,
					binding,
				)
				if runErr != nil {
					return runErr
				}
				return result.Validate()
			},
		},
	} {
		t.Run(operation.name, func(t *testing.T) {
			t.Parallel()
			if err := operation.run(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestControllerLookupAndUpdateFailures(t *testing.T) {
	t.Parallel()

	binding, err := web.NewBindingResult()
	if err != nil {
		t.Fatal(err)
	}
	notFound := repositoryWithOwners(nil, 0)
	controller, err := NewController(notFound, testCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{0, 99} {
		if _, showErr := controller.Show(
			context.Background(),
			OwnerIDRequest{OwnerID: id},
		); !errors.Is(showErr, ErrOwnerNotFound) {
			t.Fatalf("Show(%d) error = %v", id, showErr)
		}
	}

	failing := successfulControllerRepository()
	failing.findByID = func(
		context.Context,
		model.ID,
	) (Owner, bool, error) {
		return Owner{}, false, errRepository
	}
	controller, err = NewController(failing, testCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, editErr := controller.EditForm(
		context.Background(),
		OwnerIDRequest{OwnerID: 1},
	); !errors.Is(editErr, errRepository) {
		t.Fatalf("EditForm() error = %v", editErr)
	}

	failing = successfulControllerRepository()
	failing.save = func(context.Context, Owner) (Owner, error) {
		return Owner{}, errRepository
	}
	controller, err = NewController(failing, testCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controller.Update(
		context.Background(),
		validEditOwnerRequest(1, "George", "Franklin"),
		binding,
	); !errors.Is(err, errRepository) {
		t.Fatalf("Update() error = %v", err)
	}
}

func TestEditOwnerRequestNilTarget(t *testing.T) {
	t.Parallel()

	validEditOwnerRequest(1, "George", "Franklin").Apply(nil)
}

func successfulControllerRepository() controllerRepository {
	return controllerRepository{
		findByID: func(
			_ context.Context,
			id model.ID,
		) (Owner, bool, error) {
			if id != 1 {
				return Owner{}, false, nil
			}
			return ownerFixture(id, "Franklin"), true, nil
		},
		findByLastName: func(
			_ context.Context,
			_ string,
			_ int,
			_ int,
		) ([]Owner, int, error) {
			return []Owner{ownerFixture(1, "Franklin")}, 1, nil
		},
		save: func(
			_ context.Context,
			value Owner,
		) (Owner, error) {
			if !value.ID.Valid() {
				value.ID = 11
			}
			return value, nil
		},
	}
}

func repositoryWithOwners(
	values []Owner,
	total int,
) controllerRepository {
	repository := successfulControllerRepository()
	repository.findByID = func(
		_ context.Context,
		_ model.ID,
	) (Owner, bool, error) {
		return Owner{}, false, nil
	}
	repository.findByLastName = func(
		_ context.Context,
		_ string,
		_ int,
		_ int,
	) ([]Owner, int, error) {
		return values, total, nil
	}
	return repository
}

func ownerFixture(id model.ID, lastName string) Owner {
	return Owner{
		Person: model.Person{
			BaseEntity: model.BaseEntity{ID: id},
			FirstName:  "George",
			LastName:   lastName,
		},
		Address:   "110 W. Liberty St.",
		City:      "Madison",
		Telephone: "6085551023",
	}
}

func validOwnerFormRequest(
	firstName string,
	lastName string,
) OwnerFormRequest {
	return OwnerFormRequest{
		FirstName: firstName,
		LastName:  lastName,
		Address:   "110 W. Liberty St.",
		City:      "Madison",
		Telephone: strings.Repeat("1", 10),
	}
}

func validEditOwnerRequest(
	id int,
	firstName string,
	lastName string,
) EditOwnerRequest {
	request := validOwnerFormRequest(firstName, lastName)
	return EditOwnerRequest{
		OwnerID:   id,
		FirstName: request.FirstName,
		LastName:  request.LastName,
		Address:   request.Address,
		City:      request.City,
		Telephone: request.Telephone,
	}
}
