package memory

import (
	"context"
	"errors"
	"fmt"
)

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("memory repository context is nil")
	}
	if cause := context.Cause(ctx); cause != nil {
		return fmt.Errorf("memory repository context: %w", cause)
	}
	return nil
}
