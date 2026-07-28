package postgres

import (
	"context"
	"errors"
)

func validateContext(ctx context.Context, operation string) error {
	if ctx == nil {
		return errors.New(operation + ": context is nil")
	}
	if cause := context.Cause(ctx); cause != nil {
		return errors.Join(errors.New(operation+": context ended"), cause)
	}
	return nil
}
