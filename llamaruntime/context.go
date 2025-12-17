// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
// This file contains the ContextPool molecule for managing reusable inference contexts.
//
// The ContextPool provides efficient management of llama.cpp inference contexts,
// enabling concurrent inference operations while ensuring thread safety. It uses
// a channel-based pool pattern for lock-free context acquisition and release.
//
// Architecture:
// - ContextPool owns the model and creates a fixed number of contexts at initialization
// - Contexts are acquired from the pool for inference operations
// - After use, contexts are released back to the pool for reuse
// - Pool closure properly cleans up all resources
//
// Thread Safety:
// - Pool operations are thread-safe via channels and mutexes
// - Individual contexts are NOT thread-safe and must not be shared
// - Each goroutine should acquire its own context for inference
package llamaruntime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ContextPoolConfig contains configuration for the context pool.
type ContextPoolConfig struct {
	// ModelPath is the path to the GGUF model file.
	// Required - no default.
	ModelPath string

	// NumContexts is the number of inference contexts to create.
	// This determines the maximum concurrent inference capacity.
	// Defaults to 5 if not specified.
	NumContexts int

	// ContextSize is the context window size in tokens.
	// Defaults to DefaultContextSize.
	ContextSize int

	// BatchSize is the batch size for inference.
	// Defaults to DefaultBatchSize.
	BatchSize int

	// NumGPULayers is the number of model layers to offload to GPU.
	// -1 means all layers. Defaults to DefaultNumGPULayers.
	NumGPULayers int

	// NumThreads is the number of CPU threads for inference.
	// Defaults to DefaultNumThreads.
	NumThreads int

	// UseMMap enables memory-mapped model loading.
	// Defaults to true.
	UseMMap bool

	// UseMlock enables memory locking to prevent swapping.
	// Defaults to false.
	UseMlock bool

	// AcquireTimeout is the maximum time to wait for a context.
	// Defaults to 30 seconds.
	AcquireTimeout time.Duration
}

// DefaultContextPoolConfig returns a ContextPoolConfig with sensible defaults.
func DefaultContextPoolConfig() ContextPoolConfig {
	return ContextPoolConfig{
		NumContexts:    5,
		ContextSize:    DefaultContextSize,
		BatchSize:      DefaultBatchSize,
		NumGPULayers:   DefaultNumGPULayers,
		NumThreads:     DefaultNumThreads,
		UseMMap:        true,
		UseMlock:       false,
		AcquireTimeout: 30 * time.Second,
	}
}

// ContextPool manages a pool of reusable inference contexts.
// It provides thread-safe access to a fixed number of llama.cpp contexts
// for concurrent inference operations.
//
// Example usage:
//
//	config := llamaruntime.DefaultContextPoolConfig()
//	config.ModelPath = "models/bunny-v1.1.gguf"
//	config.NumContexts = 3
//
//	pool, err := llamaruntime.NewContextPool(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer pool.Close()
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	llamaCtx, err := pool.Acquire(ctx)
//	if err != nil {
//	    log.Printf("Failed to acquire context: %v", err)
//	    return
//	}
//	defer pool.Release(llamaCtx)
//
//	// Use llamaCtx for inference...
type ContextPool struct {
	model    *llamaModel
	config   ContextPoolConfig
	contexts chan *llamaContext
	mu       sync.RWMutex
	closed   bool

	// Metrics
	totalAcquires   int64
	totalReleases   int64
	acquireTimeouts int64
	acquireErrors   int64
	createdAt       time.Time
}

// NewContextPool creates a new context pool with the given configuration.
// It loads the model and pre-creates all inference contexts.
//
// The pool will have NumContexts available contexts. Each concurrent
// inference operation requires one context, so NumContexts determines
// the maximum parallelism.
//
// Returns an error if:
// - ModelPath is empty or file doesn't exist
// - Model loading fails
// - Any context creation fails (e.g., insufficient GPU memory)
func NewContextPool(config ContextPoolConfig) (*ContextPool, error) {
	// Validate and apply defaults
	if config.ModelPath == "" {
		return nil, &LlamaError{
			Op:      "NewContextPool",
			Code:    -1,
			Message: "ModelPath is required",
		}
	}

	if config.NumContexts <= 0 {
		config.NumContexts = 5
	}
	if config.ContextSize <= 0 {
		config.ContextSize = DefaultContextSize
	}
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultBatchSize
	}
	if config.NumGPULayers == 0 {
		config.NumGPULayers = DefaultNumGPULayers
	}
	if config.NumThreads <= 0 {
		config.NumThreads = DefaultNumThreads
	}
	if config.AcquireTimeout <= 0 {
		config.AcquireTimeout = 30 * time.Second
	}

	// Initialize llama backend
	llamaInit()

	// Load model
	model, err := loadModel(config.ModelPath, config.NumGPULayers, config.UseMMap, config.UseMlock)
	if err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	// Create pool
	pool := &ContextPool{
		model:     model,
		config:    config,
		contexts:  make(chan *llamaContext, config.NumContexts),
		createdAt: time.Now(),
	}

	// Pre-create all contexts
	for i := 0; i < config.NumContexts; i++ {
		ctx, err := createContext(model, config.ContextSize, config.BatchSize, config.NumThreads)
		if err != nil {
			// Clean up already created contexts and model
			pool.Close()
			return nil, &LlamaError{
				Op:      "NewContextPool",
				Code:    -1,
				Message: fmt.Sprintf("failed to create context %d of %d", i+1, config.NumContexts),
				Err:     err,
			}
		}
		pool.contexts <- ctx
	}

	return pool, nil
}

// NewContextPoolWithModel creates a context pool using an existing model.
// This allows sharing a model across multiple pools or for custom model management.
//
// Note: The pool does NOT own the model and will NOT free it on Close().
// The caller is responsible for freeing the model after all pools using it are closed.
func NewContextPoolWithModel(model *llamaModel, config ContextPoolConfig) (*ContextPool, error) {
	if model == nil {
		return nil, &LlamaError{
			Op:      "NewContextPoolWithModel",
			Code:    -1,
			Message: "model is required",
		}
	}

	// Apply defaults
	if config.NumContexts <= 0 {
		config.NumContexts = 5
	}
	if config.ContextSize <= 0 {
		config.ContextSize = DefaultContextSize
	}
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultBatchSize
	}
	if config.NumThreads <= 0 {
		config.NumThreads = DefaultNumThreads
	}
	if config.AcquireTimeout <= 0 {
		config.AcquireTimeout = 30 * time.Second
	}

	// Create pool (note: model is nil so Close() won't free it)
	pool := &ContextPool{
		model:     nil, // Don't own the model
		config:    config,
		contexts:  make(chan *llamaContext, config.NumContexts),
		createdAt: time.Now(),
	}

	// Pre-create all contexts
	for i := 0; i < config.NumContexts; i++ {
		ctx, err := createContext(model, config.ContextSize, config.BatchSize, config.NumThreads)
		if err != nil {
			pool.Close()
			return nil, &LlamaError{
				Op:      "NewContextPoolWithModel",
				Code:    -1,
				Message: fmt.Sprintf("failed to create context %d of %d", i+1, config.NumContexts),
				Err:     err,
			}
		}
		pool.contexts <- ctx
	}

	return pool, nil
}

// Acquire obtains an inference context from the pool.
// It blocks until a context is available or the given context is cancelled/times out.
//
// The returned context must be released back to the pool using Release() when done.
// Failure to release contexts will eventually exhaust the pool.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	llamaCtx, err := pool.Acquire(ctx)
//	if err != nil {
//	    if errors.Is(err, context.DeadlineExceeded) {
//	        log.Println("Timeout waiting for context")
//	    }
//	    return
//	}
//	defer pool.Release(llamaCtx)
func (p *ContextPool) Acquire(ctx context.Context) (*llamaContext, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, &LlamaError{
			Op:      "Acquire",
			Code:    -1,
			Message: "pool is closed",
		}
	}
	p.mu.RUnlock()

	// Use config timeout if context doesn't have a deadline
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		ctx, cancel = context.WithTimeout(ctx, p.config.AcquireTimeout)
		defer cancel()
	}

	select {
	case llamaCtx := <-p.contexts:
		atomic.AddInt64(&p.totalAcquires, 1)
		return llamaCtx, nil
	case <-ctx.Done():
		atomic.AddInt64(&p.acquireTimeouts, 1)
		return nil, ctx.Err()
	}
}

// TryAcquire attempts to obtain a context without blocking.
// Returns nil, false if no context is immediately available.
//
// This is useful for implementing non-blocking or load-shedding patterns.
//
// Example:
//
//	llamaCtx, ok := pool.TryAcquire()
//	if !ok {
//	    return errors.New("server busy, try again later")
//	}
//	defer pool.Release(llamaCtx)
func (p *ContextPool) TryAcquire() (*llamaContext, bool) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, false
	}
	p.mu.RUnlock()

	select {
	case llamaCtx := <-p.contexts:
		atomic.AddInt64(&p.totalAcquires, 1)
		return llamaCtx, true
	default:
		return nil, false
	}
}

// Release returns a context to the pool for reuse.
// This should be called after each Acquire() when the context is no longer needed.
//
// It is safe to call Release with a nil context (no-op).
// Calling Release after Close() will free the context instead of returning it to pool.
//
// The context's KV cache is cleared before returning to the pool to ensure
// each inference operation starts fresh.
func (p *ContextPool) Release(llamaCtx *llamaContext) {
	if llamaCtx == nil {
		return
	}

	// Clear KV cache for next user
	llamaCtx.ClearKVCache()

	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		// Pool is closed, free the context
		freeContext(llamaCtx)
		return
	}

	select {
	case p.contexts <- llamaCtx:
		atomic.AddInt64(&p.totalReleases, 1)
	default:
		// Pool is full (shouldn't happen with proper usage)
		// Free the context to prevent memory leak
		freeContext(llamaCtx)
	}
}

// Close releases all resources held by the pool.
// This includes all contexts and the model (if owned by the pool).
//
// After Close, any calls to Acquire will return an error.
// Contexts already acquired can still be used but should be released
// (they will be freed instead of returned to pool).
//
// Close is safe to call multiple times (idempotent).
func (p *ContextPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	// Close the channel to prevent new contexts from being added
	close(p.contexts)

	// Free all contexts in the pool
	for llamaCtx := range p.contexts {
		freeContext(llamaCtx)
	}

	// Free the model if we own it
	if p.model != nil {
		freeModel(p.model)
		p.model = nil
	}

	return nil
}

// Stats returns pool statistics for monitoring.
type ContextPoolStats struct {
	// Configuration
	NumContexts int
	ContextSize int
	BatchSize   int

	// Availability
	Available int // Contexts currently in pool (not acquired)
	InUse     int // Contexts currently acquired

	// Metrics
	TotalAcquires   int64
	TotalReleases   int64
	AcquireTimeouts int64
	AcquireErrors   int64
	Uptime          time.Duration

	// Status
	Closed bool
}

// Stats returns current pool statistics.
// Useful for monitoring and debugging.
func (p *ContextPool) Stats() ContextPoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	available := len(p.contexts)
	return ContextPoolStats{
		NumContexts:     p.config.NumContexts,
		ContextSize:     p.config.ContextSize,
		BatchSize:       p.config.BatchSize,
		Available:       available,
		InUse:           p.config.NumContexts - available,
		TotalAcquires:   atomic.LoadInt64(&p.totalAcquires),
		TotalReleases:   atomic.LoadInt64(&p.totalReleases),
		AcquireTimeouts: atomic.LoadInt64(&p.acquireTimeouts),
		AcquireErrors:   atomic.LoadInt64(&p.acquireErrors),
		Uptime:          time.Since(p.createdAt),
		Closed:          p.closed,
	}
}

// Model returns the underlying model.
// Returns nil if the pool was created with NewContextPoolWithModel
// (i.e., the pool doesn't own the model).
func (p *ContextPool) Model() *llamaModel {
	return p.model
}

// Config returns the pool configuration.
func (p *ContextPool) Config() ContextPoolConfig {
	return p.config
}

// IsClosed returns whether the pool is closed.
func (p *ContextPool) IsClosed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}

// WaitForAvailable blocks until at least one context is available
// or the context is cancelled.
//
// This is useful for implementing backpressure or queue management.
func (p *ContextPool) WaitForAvailable(ctx context.Context) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return errors.New("pool is closed")
	}
	p.mu.RUnlock()

	// Poll until available or cancelled
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if len(p.contexts) > 0 {
				return nil
			}
		}
	}
}
