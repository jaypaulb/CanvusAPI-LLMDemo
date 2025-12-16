package core

import (
	"context"
)

// ShutdownFunc is the function signature for cleanup handlers during graceful shutdown.
// Each shutdown function receives a context that may have a deadline for cleanup,
// and returns an error if cleanup fails.
//
// Implementations should:
//   - Respect context cancellation/deadline
//   - Return nil on success
//   - Return an error describing any failure
//   - Be idempotent (safe to call multiple times)
//
// Example usage:
//
//	var dbShutdown ShutdownFunc = func(ctx context.Context) error {
//	    return db.Close()
//	}
type ShutdownFunc func(ctx context.Context) error
