package shutdown

import (
	"context"
	"sort"
	"sync"

	"go_backend/core"
)

// shutdownEntry holds a registered shutdown function with metadata.
type shutdownEntry struct {
	name     string
	fn       core.ShutdownFunc
	priority int // lower = earlier execution
}

// ShutdownRegistry maintains an ordered collection of shutdown functions.
//
// This is a molecule that composes core.ShutdownFunc with priority ordering
// and thread-safe registration to coordinate cleanup during graceful shutdown.
//
// Usage:
//
//	registry := NewShutdownRegistry()
//
//	// Register handlers (lower priority runs first)
//	registry.Register("connections", 10, func(ctx context.Context) error {
//	    return closeConnections()
//	})
//	registry.Register("database", 20, func(ctx context.Context) error {
//	    return db.Close()
//	})
//
//	// During shutdown:
//	errs := registry.Shutdown(ctx)
//	for _, err := range errs {
//	    log.Printf("shutdown error: %v", err)
//	}
type ShutdownRegistry struct {
	mu      sync.Mutex
	entries []shutdownEntry
	closed  bool
}

// NewShutdownRegistry creates a new ShutdownRegistry ready to accept registrations.
func NewShutdownRegistry() *ShutdownRegistry {
	return &ShutdownRegistry{
		entries: make([]shutdownEntry, 0),
	}
}

// Register adds a shutdown function with a name and priority.
// Lower priority values execute earlier during shutdown.
// Registration after Shutdown has been called is a no-op.
//
// Typical priority ranges:
//   - 0-9: Critical cleanup (flush logs, metrics)
//   - 10-19: Connection cleanup (close client connections)
//   - 20-29: Service cleanup (stop background workers)
//   - 30-39: Resource cleanup (close databases, files)
//   - 40+: Final cleanup (release locks, remove temp files)
func (r *ShutdownRegistry) Register(name string, priority int, fn core.ShutdownFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return
	}

	r.entries = append(r.entries, shutdownEntry{
		name:     name,
		fn:       fn,
		priority: priority,
	})
}

// Shutdown executes all registered shutdown functions in priority order.
// Returns a slice of errors from functions that failed (nil entries omitted).
// Each function receives the provided context for cancellation/timeout.
//
// All functions are called even if some fail. Errors are collected and returned.
// After Shutdown completes, the registry is marked closed.
func (r *ShutdownRegistry) Shutdown(ctx context.Context) []error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true

	// Copy and sort entries by priority
	sorted := make([]shutdownEntry, len(r.entries))
	copy(sorted, r.entries)
	r.mu.Unlock()

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].priority < sorted[j].priority
	})

	var errs []error
	for _, entry := range sorted {
		if err := entry.fn(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

// Names returns the names of all registered shutdown functions in priority order.
func (r *ShutdownRegistry) Names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Copy and sort entries by priority
	sorted := make([]shutdownEntry, len(r.entries))
	copy(sorted, r.entries)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].priority < sorted[j].priority
	})

	names := make([]string, len(sorted))
	for i, entry := range sorted {
		names[i] = entry.name
	}
	return names
}

// Count returns the number of registered shutdown functions.
func (r *ShutdownRegistry) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.entries)
}

// IsClosed returns true if Shutdown has been called.
func (r *ShutdownRegistry) IsClosed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed
}
