// Package shutdown provides graceful shutdown infrastructure molecules.
// This package composes atoms from core (ShutdownFunc, exit codes) into
// higher-level shutdown coordination components.
package shutdown

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ErrTrackerClosed is returned when trying to start an operation on a closed tracker.
var ErrTrackerClosed = errors.New("operation tracker is closed")

// ErrWaitTimeout is returned when Wait times out before all operations complete.
var ErrWaitTimeout = errors.New("wait timeout: operations did not complete in time")

// OperationTracker tracks in-flight operations and provides a mechanism
// to wait for them to complete during graceful shutdown.
//
// This is a molecule that composes sync.WaitGroup with atomic counters
// and a closed state to provide a safe way to track operations during shutdown.
//
// Usage:
//
//	tracker := NewOperationTracker()
//
//	// In request handler:
//	if !tracker.Start() {
//	    return // shutting down, reject request
//	}
//	defer tracker.Done()
//	// ... handle request ...
//
//	// During shutdown:
//	tracker.Close()
//	if err := tracker.Wait(30 * time.Second); err != nil {
//	    log.Println("timeout waiting for operations")
//	}
type OperationTracker struct {
	wg     sync.WaitGroup
	mu     sync.RWMutex
	active int64
	closed bool
}

// NewOperationTracker creates a new OperationTracker ready to track operations.
func NewOperationTracker() *OperationTracker {
	return &OperationTracker{}
}

// Start attempts to start tracking a new operation.
// Returns true if the operation was started, false if the tracker is closed.
//
// If Start returns true, the caller MUST call Done when the operation completes.
// If Start returns false, the caller should reject the operation as the system
// is shutting down.
func (t *OperationTracker) Start() bool {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return false
	}
	t.mu.RUnlock()

	// Double-check with write lock to avoid race
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return false
	}
	t.wg.Add(1)
	atomic.AddInt64(&t.active, 1)
	t.mu.Unlock()
	return true
}

// Done marks an operation as complete.
// Must be called exactly once for each successful Start call.
func (t *OperationTracker) Done() {
	atomic.AddInt64(&t.active, -1)
	t.wg.Done()
}

// Wait blocks until all tracked operations complete or the timeout is reached.
// Returns nil if all operations completed, or ErrWaitTimeout if the timeout was reached.
func (t *OperationTracker) Wait(timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return ErrWaitTimeout
	}
}

// Close marks the tracker as closed, preventing new operations from starting.
// Operations already in progress will continue until they call Done.
func (t *OperationTracker) Close() {
	t.mu.Lock()
	t.closed = true
	t.mu.Unlock()
}

// ActiveCount returns the current number of active operations.
func (t *OperationTracker) ActiveCount() int64 {
	return atomic.LoadInt64(&t.active)
}

// IsClosed returns true if the tracker has been closed.
func (t *OperationTracker) IsClosed() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.closed
}
