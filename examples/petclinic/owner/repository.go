package owner

import (
	"context"

	"github.com/spice-framework/spice/examples/petclinic/model"
)

// Repository persists complete owner aggregates.
type Repository interface {
	FindByID(context.Context, model.ID) (Owner, bool, error)
	FindByLastName(
		context.Context,
		string,
		int,
		int,
	) ([]Owner, int, error)
	Save(context.Context, Owner) (Owner, error)
}
