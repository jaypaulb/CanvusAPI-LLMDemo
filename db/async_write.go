// Package db provides database utilities including async write functionality.
package db

import (
	"context"
	"sync"
	"time"
)

// DefaultChannelCapacity is the default buffer size for async write channels.
const DefaultChannelCapacity = 100

// DefaultDrainTimeout is the maximum time to wait for pending writes during shutdown.
const DefaultDrainTimeout = 30 * time.Second

// WriteOperation represents a database write operation to be processed asynchronously.
type WriteOperation struct {
	// Data holds the write payload
	Data interface{}
	// Timestamp when the operation was queued
	Timestamp time.Time
}

// WriteHandler is a function type that processes write operations.
// Implementations should handle their own error logging/recovery.
type WriteHandler func(op WriteOperation) error

// AsyncWriter provides non-blocking database write functionality
// using a buffered channel and background goroutine processing.
//
// This molecule composes:
// - Channel send/receive (atoms)
// - Context cancellation (atom)
// - Graceful shutdown with drain (composition)
type AsyncWriter struct {
	writeChan chan WriteOperation
	handler   WriteHandler
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	started   bool
	mu        sync.Mutex
}

// AsyncWriterConfig holds configuration for the async writer.
type AsyncWriterConfig struct {
	// ChannelCapacity is the buffer size for pending writes
	ChannelCapacity int
	// DrainTimeout is the maximum wait time during shutdown
	DrainTimeout time.Duration
}

// DefaultAsyncWriterConfig returns the default configuration.
func DefaultAsyncWriterConfig() AsyncWriterConfig {
	return AsyncWriterConfig{
		ChannelCapacity: DefaultChannelCapacity,
		DrainTimeout:    DefaultDrainTimeout,
	}
}

// NewAsyncWriter creates a new async writer with default configuration.
// The handler function will be called for each write operation.
func NewAsyncWriter(handler WriteHandler) *AsyncWriter {
	return NewAsyncWriterWithConfig(handler, DefaultAsyncWriterConfig())
}

// NewAsyncWriterWithConfig creates a new async writer with custom configuration.
func NewAsyncWriterWithConfig(handler WriteHandler, config AsyncWriterConfig) *AsyncWriter {
	ctx, cancel := context.WithCancel(context.Background())

	return &AsyncWriter{
		writeChan: make(chan WriteOperation, config.ChannelCapacity),
		handler:   handler,
		ctx:       ctx,
		cancel:    cancel,
		started:   false,
	}
}

// Start begins the background processing goroutine.
// This must be called before Write operations can be processed.
// Returns immediately; processing happens in background.
func (w *AsyncWriter) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.started {
		return // Already started
	}

	w.started = true
	w.wg.Add(1)
	go w.processWrites()
}

// processWrites is the background goroutine that handles write operations.
func (w *AsyncWriter) processWrites() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			// Context cancelled, drain remaining operations
			w.drainChannel()
			return
		case op, ok := <-w.writeChan:
			if !ok {
				// Channel closed
				return
			}
			// Process the write operation
			// Handler is responsible for its own error handling
			_ = w.handler(op)
		}
	}
}

// drainChannel processes any remaining operations in the buffer.
func (w *AsyncWriter) drainChannel() {
	for {
		select {
		case op, ok := <-w.writeChan:
			if !ok {
				return // Channel closed
			}
			_ = w.handler(op)
		default:
			return // No more pending operations
		}
	}
}

// Write queues a write operation for async processing.
// Returns true if the operation was queued, false if the channel is full or closed.
// This is a non-blocking operation.
func (w *AsyncWriter) Write(data interface{}) bool {
	op := WriteOperation{
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case w.writeChan <- op:
		return true
	default:
		// Channel full or closed
		return false
	}
}

// WriteWithTimeout queues a write operation with a timeout.
// Returns true if queued within timeout, false otherwise.
func (w *AsyncWriter) WriteWithTimeout(data interface{}, timeout time.Duration) bool {
	op := WriteOperation{
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case w.writeChan <- op:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Pending returns the number of operations waiting in the buffer.
func (w *AsyncWriter) Pending() int {
	return len(w.writeChan)
}

// Stop signals the background goroutine to stop and waits for
// graceful drain of pending operations.
func (w *AsyncWriter) Stop() {
	w.cancel()
	w.wg.Wait()
}

// StopWithTimeout stops the writer with a maximum wait time.
// Returns true if stopped gracefully, false if timed out.
func (w *AsyncWriter) StopWithTimeout(timeout time.Duration) bool {
	w.cancel()

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Close stops the writer and closes the channel.
// After Close, no more writes can be queued.
func (w *AsyncWriter) Close() {
	w.Stop()
	close(w.writeChan)
}

// IsStarted returns whether the background processor is running.
func (w *AsyncWriter) IsStarted() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.started
}
