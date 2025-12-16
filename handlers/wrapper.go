// Package handlers provides request handling utilities for graceful shutdown integration.
package handlers

import (
	"context"
	"errors"
	"fmt"

	"go_backend/canvusapi"
	"go_backend/core"
	"go_backend/logging"
	"go_backend/shutdown"

	"go.uber.org/zap"
)

// ErrShuttingDown is returned when an operation is rejected because the system
// is shutting down.
var ErrShuttingDown = errors.New("operation rejected: system is shutting down")

// HandlerFunc is the signature for widget handler functions.
// The context parameter allows handlers to check for cancellation.
type HandlerFunc func(ctx context.Context, update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) error

// RequestWrapper wraps handler execution with shutdown manager integration.
// It tracks in-flight operations for graceful shutdown and rejects new
// operations when shutdown is in progress.
//
// Usage:
//
//	wrapper := handlers.NewRequestWrapper(shutdownManager, logger)
//
//	// Wrap a handler call
//	wrapper.Execute(ctx, "process-note", noteHandler, update, client, config, handlerLogger)
type RequestWrapper struct {
	manager *shutdown.Manager
	logger  *zap.Logger
}

// NewRequestWrapper creates a new RequestWrapper with the given shutdown manager.
// The logger is used for wrapper-level logging (separate from handler logging).
func NewRequestWrapper(manager *shutdown.Manager, logger *zap.Logger) *RequestWrapper {
	return &RequestWrapper{
		manager: manager,
		logger:  logger,
	}
}

// Execute runs a handler function while tracking it as an in-flight operation.
// If the system is shutting down, ErrShuttingDown is returned and the handler
// is not executed.
//
// The operation name is used for logging and debugging purposes.
//
// Example:
//
//	err := wrapper.Execute(ctx, "handle-note", func(ctx context.Context, update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) error {
//	    // Handler logic here
//	    return nil
//	}, update, client, config, logger)
func (w *RequestWrapper) Execute(
	ctx context.Context,
	operationName string,
	handler HandlerFunc,
	update map[string]interface{},
	client *canvusapi.Client,
	config *core.Config,
	logger *logging.Logger,
) error {
	// Check if shutdown is in progress before attempting to start
	if w.manager.IsShuttingDown() {
		w.logger.Debug("Operation rejected, system is shutting down",
			zap.String("operation", operationName),
		)
		return ErrShuttingDown
	}

	// Wrap the operation with shutdown tracking
	err := w.manager.WrapOperation(ctx, operationName, func(opCtx context.Context) error {
		// Execute the handler with the operation context
		return handler(opCtx, update, client, config, logger)
	})

	// Convert shutdown.ErrTrackerClosed to our ErrShuttingDown for cleaner API
	if errors.Is(err, shutdown.ErrTrackerClosed) {
		return ErrShuttingDown
	}

	return err
}

// ExecuteAsync runs a handler function asynchronously while tracking it as
// an in-flight operation. This is useful for handlers that are typically
// launched in goroutines.
//
// If the system is shutting down, the handler is not executed and no
// goroutine is spawned.
//
// Returns true if the handler was started, false if rejected due to shutdown.
//
// Example:
//
//	started := wrapper.ExecuteAsync(ctx, "handle-snapshot", snapshotHandler, update, client, config, logger)
//	if !started {
//	    log.Info("Handler not started due to shutdown")
//	}
func (w *RequestWrapper) ExecuteAsync(
	ctx context.Context,
	operationName string,
	handler HandlerFunc,
	update map[string]interface{},
	client *canvusapi.Client,
	config *core.Config,
	logger *logging.Logger,
) bool {
	// Check if shutdown is in progress before spawning goroutine
	if w.manager.IsShuttingDown() {
		w.logger.Debug("Async operation rejected, system is shutting down",
			zap.String("operation", operationName),
		)
		return false
	}

	go func() {
		err := w.Execute(ctx, operationName, handler, update, client, config, logger)
		if err != nil && !errors.Is(err, ErrShuttingDown) && !errors.Is(err, context.Canceled) {
			w.logger.Error("Async handler error",
				zap.String("operation", operationName),
				zap.Error(err),
			)
		}
	}()

	return true
}

// ActiveOperations returns the count of currently in-flight operations.
func (w *RequestWrapper) ActiveOperations() int64 {
	return w.manager.ActiveOperations()
}

// IsShuttingDown returns true if the system is shutting down.
func (w *RequestWrapper) IsShuttingDown() bool {
	return w.manager.IsShuttingDown()
}

// WrapLegacyHandler adapts a legacy handler function (without context) to
// the HandlerFunc signature. This is useful for gradual migration of existing
// handlers.
//
// Example:
//
//	// Old handler signature
//	func handleNote(update Update, client *canvusapi.Client, config *core.Config, logger *logging.Logger) {
//	    // ...
//	}
//
//	// Wrap it
//	wrapped := handlers.WrapLegacyHandler(func(ctx context.Context, update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) error {
//	    handleNote(update, client, config, logger)
//	    return nil
//	})
func WrapLegacyHandler(legacy func(update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger)) HandlerFunc {
	return func(ctx context.Context, update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) error {
		// Check context before executing legacy handler
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		legacy(update, client, config, logger)
		return nil
	}
}

// WrapLegacyHandlerWithError adapts a legacy handler function that returns an
// error to the HandlerFunc signature.
func WrapLegacyHandlerWithError(legacy func(update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) error) HandlerFunc {
	return func(ctx context.Context, update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) error {
		// Check context before executing legacy handler
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		return legacy(update, client, config, logger)
	}
}

// OperationStats provides statistics about handler operations.
type OperationStats struct {
	ActiveCount   int64
	IsShuttingDown bool
}

// Stats returns current operation statistics.
func (w *RequestWrapper) Stats() OperationStats {
	return OperationStats{
		ActiveCount:    w.manager.ActiveOperations(),
		IsShuttingDown: w.manager.IsShuttingDown(),
	}
}

// String returns a string representation of the wrapper status for logging.
func (w *RequestWrapper) String() string {
	stats := w.Stats()
	return fmt.Sprintf("RequestWrapper{active=%d, shutting_down=%v}",
		stats.ActiveCount, stats.IsShuttingDown)
}
