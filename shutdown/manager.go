package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go_backend/core"

	"go.uber.org/zap"
)

// Manager is the main shutdown coordination organism that composes:
//   - OperationTracker: tracks in-flight operations
//   - ShutdownRegistry: ordered cleanup functions
//   - SignalCounter: handles repeated signals for force shutdown
//
// Manager provides a unified interface for graceful shutdown management,
// coordinating context cancellation, operation tracking, cleanup execution,
// and signal handling.
//
// Usage:
//
//	logger, _ := zap.NewProduction()
//	manager := NewManager(logger)
//
//	// Register cleanup handlers (lower priority runs first)
//	manager.Register("database", 30, func(ctx context.Context) error {
//	    return db.Close()
//	})
//	manager.Register("logger", 5, func(ctx context.Context) error {
//	    return logger.Sync()
//	})
//
//	// Start signal handling
//	manager.Start()
//
//	// In request handlers:
//	err := manager.WrapOperation(ctx, "process-request", func(ctx context.Context) error {
//	    // ... handle request ...
//	    return nil
//	})
//
//	// Block until shutdown
//	<-manager.Context().Done()
//
//	// Execute shutdown sequence
//	manager.Shutdown()
type Manager struct {
	logger   *zap.Logger
	timeout  time.Duration
	mu       sync.Mutex
	started  bool
	shutdown bool

	// Internal context management
	ctx    context.Context
	cancel context.CancelFunc

	// Composed molecules
	tracker  *OperationTracker
	registry *ShutdownRegistry
	signals  *SignalCounter

	// Signal channel for cleanup
	sigChan chan os.Signal
}

// ManagerOption configures a Manager.
type ManagerOption func(*Manager)

// WithTimeout sets the shutdown timeout duration.
// Default is 60 seconds.
func WithTimeout(timeout time.Duration) ManagerOption {
	return func(m *Manager) {
		m.timeout = timeout
	}
}

// NewManager creates a new Manager ready to coordinate graceful shutdown.
// The logger is required and used for all shutdown-related logging.
//
// Default configuration:
//   - Timeout: 60 seconds
//   - Force shutdown on second signal
func NewManager(logger *zap.Logger, opts ...ManagerOption) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		logger:   logger,
		timeout:  60 * time.Second,
		ctx:      ctx,
		cancel:   cancel,
		tracker:  NewOperationTracker(),
		registry: NewShutdownRegistry(),
		sigChan:  make(chan os.Signal, 1),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	// Create signal counter with force shutdown callback
	m.signals = NewSignalCounter(2, func() {
		m.logger.Warn("Received second signal, forcing immediate shutdown")
		os.Exit(1)
	})

	return m
}

// Context returns the managed context that will be cancelled during shutdown.
// Components should use this context to detect when shutdown has been initiated.
func (m *Manager) Context() context.Context {
	return m.ctx
}

// Register adds a cleanup function to be called during shutdown.
// Lower priority values are executed first.
//
// Typical priority ranges:
//   - 0-9: Critical cleanup (flush logs, metrics)
//   - 10-19: Connection cleanup (close client connections)
//   - 20-29: Service cleanup (stop background workers)
//   - 30-39: Resource cleanup (close databases, files)
//   - 40+: Final cleanup (release locks, remove temp files)
func (m *Manager) Register(name string, priority int, fn core.ShutdownFunc) {
	m.registry.Register(name, priority, fn)
	m.logger.Debug("Registered shutdown handler",
		zap.String("name", name),
		zap.Int("priority", priority),
	)
}

// Start begins signal handling for SIGINT and SIGTERM.
// When a signal is received, the context is cancelled to initiate graceful shutdown.
// A second signal triggers immediate forced shutdown via os.Exit(1).
//
// Start must be called before shutdown will respond to OS signals.
// It is safe to call Start multiple times; subsequent calls are no-ops.
func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return
	}
	m.started = true

	signal.Notify(m.sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		for sig := range m.sigChan {
			count := m.signals.Increment()
			if count == 1 {
				m.logger.Info("Received shutdown signal, initiating graceful shutdown",
					zap.String("signal", sig.String()),
				)
				m.cancel() // Cancel context to signal components
			}
			// Force shutdown is handled by SignalCounter callback
		}
	}()

	m.logger.Info("Shutdown manager started, listening for signals")
}

// Shutdown executes the graceful shutdown sequence:
//  1. Close operation tracker to reject new operations
//  2. Wait for in-flight operations (with timeout)
//  3. Execute registered cleanup functions in priority order
//
// Shutdown returns an error if any cleanup function fails or if
// waiting for in-flight operations times out.
//
// Shutdown is idempotent; subsequent calls are no-ops and return nil.
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	if m.shutdown {
		m.mu.Unlock()
		return nil
	}
	m.shutdown = true
	m.mu.Unlock()

	startTime := time.Now()
	m.logger.Info("Initiating graceful shutdown",
		zap.Duration("timeout", m.timeout),
		zap.Int("registered_handlers", m.registry.Count()),
	)

	// Step 1: Stop accepting new operations
	m.tracker.Close()
	m.logger.Info("Closed operation tracker, rejecting new operations")

	// Step 2: Wait for in-flight operations
	activeOps := m.tracker.ActiveCount()
	if activeOps > 0 {
		m.logger.Info("Waiting for in-flight operations",
			zap.Int64("active_count", activeOps),
		)
	}

	if err := m.tracker.Wait(m.timeout); err != nil {
		m.logger.Warn("Timeout waiting for in-flight operations",
			zap.Duration("waited", time.Since(startTime)),
			zap.Int64("remaining_ops", m.tracker.ActiveCount()),
		)
	} else {
		m.logger.Info("All in-flight operations completed")
	}

	// Step 3: Execute cleanup functions with remaining timeout
	elapsed := time.Since(startTime)
	remaining := m.timeout - elapsed
	if remaining < time.Second {
		remaining = time.Second // Minimum 1 second for cleanup
	}

	ctx, cancel := context.WithTimeout(context.Background(), remaining)
	defer cancel()

	m.logger.Info("Executing cleanup functions",
		zap.Strings("handlers", m.registry.Names()),
	)

	errs := m.registry.Shutdown(ctx)

	// Log each error
	for _, err := range errs {
		m.logger.Error("Cleanup function failed", zap.Error(err))
	}

	duration := time.Since(startTime)
	if len(errs) > 0 {
		m.logger.Error("Shutdown completed with errors",
			zap.Duration("duration", duration),
			zap.Int("error_count", len(errs)),
		)
		return fmt.Errorf("shutdown had %d errors", len(errs))
	}

	m.logger.Info("Graceful shutdown completed",
		zap.Duration("duration", duration),
	)

	// Clean up signal channel
	signal.Stop(m.sigChan)
	close(m.sigChan)

	return nil
}

// Wait blocks until the managed context is cancelled.
// This is a convenience method for main goroutines.
func (m *Manager) Wait() {
	<-m.ctx.Done()
}

// WrapOperation executes a function while tracking it as an in-flight operation.
// If the system is shutting down, ErrTrackerClosed is returned and the function
// is not executed.
//
// The operation name is used for logging purposes.
//
// Example:
//
//	err := manager.WrapOperation(ctx, "process-pdf", func(ctx context.Context) error {
//	    return processPDF(ctx, pdfData)
//	})
//	if errors.Is(err, shutdown.ErrTrackerClosed) {
//	    return fmt.Errorf("system is shutting down")
//	}
func (m *Manager) WrapOperation(ctx context.Context, name string, fn func(context.Context) error) error {
	if !m.tracker.Start() {
		m.logger.Debug("Operation rejected, system shutting down",
			zap.String("operation", name),
		)
		return ErrTrackerClosed
	}
	defer m.tracker.Done()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-m.ctx.Done():
		return context.Canceled
	default:
	}

	return fn(ctx)
}

// ActiveOperations returns the count of currently in-flight operations.
func (m *Manager) ActiveOperations() int64 {
	return m.tracker.ActiveCount()
}

// IsShuttingDown returns true if shutdown has been initiated.
func (m *Manager) IsShuttingDown() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.shutdown || m.tracker.IsClosed()
}

// RegisteredHandlers returns the names of all registered cleanup handlers
// in priority order (first to execute is first in slice).
func (m *Manager) RegisteredHandlers() []string {
	return m.registry.Names()
}
