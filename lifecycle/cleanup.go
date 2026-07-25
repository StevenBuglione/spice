// Package lifecycle defines explicit lifecycle metadata types used by Spice
// compiler phases and future generated application code.
package lifecycle

import "context"

// Cleanup releases one successfully constructed provider resource.
// Future generated code invokes it with a caller-owned rollback or shutdown
// context.
type Cleanup func(context.Context) error
