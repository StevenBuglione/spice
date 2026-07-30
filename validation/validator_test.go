package validation

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

type order struct {
	ID     string
	Amount int
}

func TestValidateCombinesValidatorsInOrder(t *testing.T) {
	t.Parallel()

	identifier := ValidatorFunc[order](func(
		_ context.Context,
		order order,
	) (Errors, error) {
		if order.ID != "" {
			return Errors{}, nil
		}
		violation, err := Field("id", "required", "ID is required")
		if err != nil {
			return Errors{}, err
		}
		return New(violation)
	})
	amount := ValidatorFunc[order](func(
		_ context.Context,
		order order,
	) (Errors, error) {
		if order.Amount > 0 {
			return Errors{}, nil
		}
		violation, err := Field(
			"amount",
			"positive",
			"Amount must be positive",
		)
		if err != nil {
			return Errors{}, err
		}
		return New(violation)
	})
	result, err := Validate(
		context.Background(),
		order{},
		identifier,
		amount,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.All(); len(got) != 2 ||
		got[0].Field != "id" ||
		got[1].Field != "amount" {
		t.Fatalf("Validate() = %+v", got)
	}
}

func TestValidateStopsOnOperationalFailure(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("catalog unavailable")
	called := false
	first := ValidatorFunc[order](func(
		context.Context,
		order,
	) (Errors, error) {
		return Errors{}, sentinel
	})
	second := ValidatorFunc[order](func(
		context.Context,
		order,
	) (Errors, error) {
		called = true
		return Errors{}, nil
	})
	if _, err := Validate(
		context.Background(),
		order{},
		first,
		second,
	); !errors.Is(err, sentinel) {
		t.Fatalf("Validate() error = %v", err)
	}
	if called {
		t.Fatal("Validate() continued after an operational failure")
	}
}

func TestValidateCancellationAndInvalidValidators(t *testing.T) {
	t.Parallel()

	//nolint:staticcheck // The public API promises a fail-closed nil boundary.
	if _, err := Validate[order](nil, order{}); err == nil {
		t.Fatal("Validate() accepted a nil context")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Validate[order](
		cancelled,
		order{},
		ValidatorFunc[order](func(
			context.Context,
			order,
		) (Errors, error) {
			return Errors{}, nil
		}),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("Validate(cancelled) error = %v", err)
	}
	var nilValidator Validator[order]
	if _, err := Validate(
		context.Background(),
		order{},
		nilValidator,
	); err == nil {
		t.Fatal("Validate() accepted a nil validator")
	}
	var nilFunction ValidatorFunc[order]
	if _, err := nilFunction.Validate(
		context.Background(),
		order{},
	); err == nil {
		t.Fatal("nil ValidatorFunc.Validate() succeeded")
	}
	//nolint:staticcheck // The adapter itself validates callers at this boundary.
	if _, err := nilFunction.Validate(nil, order{}); err == nil {
		t.Fatal("ValidatorFunc.Validate() accepted a nil context")
	}

	during, cancelDuring := context.WithCancel(context.Background())
	if _, err := Validate(
		during,
		order{},
		validatorStub[order](func(
			context.Context,
			order,
		) (Errors, error) {
			cancelDuring()
			return Errors{}, nil
		}),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("Validate(cancel during validator) error = %v", err)
	}
}

func TestValidateEnforcesViolationLimit(t *testing.T) {
	t.Parallel()

	violations := make([]Violation, maxViolations)
	for index := range violations {
		violations[index] = Violation{
			Field:   fmt.Sprintf("field%d", index),
			Code:    "invalid",
			Message: "Invalid",
		}
	}
	full, err := New(violations...)
	if err != nil {
		t.Fatal(err)
	}
	one, err := New(Violation{
		Field:   "overflow",
		Code:    "invalid",
		Message: "Invalid",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Validate(
		context.Background(),
		order{},
		ValidatorFunc[order](func(
			context.Context,
			order,
		) (Errors, error) {
			return full, nil
		}),
		ValidatorFunc[order](func(
			context.Context,
			order,
		) (Errors, error) {
			return one, nil
		}),
	); err == nil {
		t.Fatal("Validate() accepted too many violations")
	}
}

func ExampleValidate() {
	positive := ValidatorFunc[int](func(
		_ context.Context,
		value int,
	) (Errors, error) {
		if value > 0 {
			return Errors{}, nil
		}
		violation, err := Object("positive", "Value must be positive")
		if err != nil {
			return Errors{}, err
		}
		return New(violation)
	})
	result, err := Validate(context.Background(), 0, positive)
	if err != nil {
		fmt.Println("validation failed")
		return
	}
	fmt.Println(result.All()[0].Code)
	// Output: positive
}

type validatorStub[T any] func(context.Context, T) (Errors, error)

func (validate validatorStub[T]) Validate(
	ctx context.Context,
	value T,
) (Errors, error) {
	return validate(ctx, value)
}
