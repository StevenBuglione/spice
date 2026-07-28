package vet

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/StevenBuglione/spice/examples/petclinic/model"
)

var errVetRepository = errors.New("vet repository failed")

type controllerRepository struct {
	findAll  func(context.Context) ([]Vet, error)
	findPage func(context.Context, int, int) ([]Vet, int, error)
}

func (repository controllerRepository) FindAll(
	ctx context.Context,
) ([]Vet, error) {
	return repository.findAll(ctx)
}

func (repository controllerRepository) FindPage(
	ctx context.Context,
	offset int,
	limit int,
) ([]Vet, int, error) {
	return repository.findPage(ctx, offset, limit)
}

func TestControllerRepresentationsAndPagination(t *testing.T) {
	t.Parallel()

	if _, err := NewController(nil); err == nil {
		t.Fatal("NewController(nil) succeeded")
	}
	controller, err := NewController(successfulControllerRepository())
	if err != nil {
		t.Fatal(err)
	}
	for _, page := range []int{0, 2} {
		result, listErr := controller.ListHTML(
			context.Background(),
			ListRequest{Page: page},
		)
		if listErr != nil {
			t.Fatal(listErr)
		}
		if validationErr := result.Validate(); validationErr != nil {
			t.Fatal(validationErr)
		}
	}
	for _, page := range []int{-1, 3} {
		if _, listErr := controller.ListHTML(
			context.Background(),
			ListRequest{Page: page},
		); listErr == nil {
			t.Fatalf("ListHTML(page=%d) succeeded", page)
		}
	}
	response, err := controller.ListJSON(
		context.Background(),
		AllRequest{},
	)
	if err != nil {
		t.Fatal(err)
	}
	content, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"firstName":"James"`) ||
		!strings.Contains(string(content), `"specialties"`) {
		t.Fatalf("JSON = %s", content)
	}
}

func TestControllerRepositoryFailures(t *testing.T) {
	t.Parallel()

	repository := successfulControllerRepository()
	repository.findPage = func(
		context.Context,
		int,
		int,
	) ([]Vet, int, error) {
		return nil, 0, errVetRepository
	}
	controller, err := NewController(repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, listErr := controller.ListHTML(
		context.Background(),
		ListRequest{},
	); !errors.Is(listErr, errVetRepository) {
		t.Fatalf("ListHTML() error = %v", listErr)
	}

	repository = successfulControllerRepository()
	repository.findAll = func(context.Context) ([]Vet, error) {
		return nil, errVetRepository
	}
	controller, err = NewController(repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, listErr := controller.ListJSON(
		context.Background(),
		AllRequest{},
	); !errors.Is(listErr, errVetRepository) {
		t.Fatalf("ListJSON() error = %v", listErr)
	}
}

func successfulControllerRepository() controllerRepository {
	values := []Vet{
		vetFixture(1, "James", "Carter"),
		vetFixture(2, "Linda", "Douglas"),
		vetFixture(3, "Sharon", "Jenkins"),
		vetFixture(4, "Helen", "Leary"),
		vetFixture(5, "Rafael", "Ortega"),
		vetFixture(6, "Henry", "Stevens"),
	}
	return controllerRepository{
		findAll: func(context.Context) ([]Vet, error) {
			return values, nil
		},
		findPage: func(
			_ context.Context,
			offset int,
			limit int,
		) ([]Vet, int, error) {
			if offset >= len(values) {
				return []Vet{}, len(values), nil
			}
			return values[offset:min(offset+limit, len(values))],
				len(values),
				nil
		},
	}
}

func vetFixture(
	id model.ID,
	firstName string,
	lastName string,
) Vet {
	return Vet{Person: model.Person{
		BaseEntity: model.BaseEntity{ID: id},
		FirstName:  firstName,
		LastName:   lastName,
	}}
}
