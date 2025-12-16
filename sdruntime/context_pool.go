// Package sdruntime provides Stable Diffusion image generation capabilities.
//
// context_pool.go implements a thread-safe context pool for managing SDContext instances.
// This is a molecule that composes atoms from cgo_bindings.go and errors.go.
package sdruntime

import (
	"context"
	"sync"
)

// PooledContext wraps an SDContext with pool management metadata.
// This provides tracking for pool membership while preserving the underlying context.
type PooledContext struct {
	// SDContext is the underlying context from cgo_bindings
	*SDContext
	// poolID identifies which pool this context belongs to
	poolID int
	// inUse tracks whether this context is currently acquired
	inUse bool
}

// ContextPool manages a pool of SDContext instances for efficient reuse.
// It provides thread-safe acquisition and release of contexts, with support
// for context deadline handling during acquisition.
//
// This molecule composes:
//   - LoadModel (atom from cgo_bindings) for context creation
//   - FreeContext (atom from cgo_bindings) for context cleanup
//   - ErrContextPoolClosed, ErrAcquireTimeout (atoms from errors.go)
type ContextPool struct {
	mu        sync.Mutex
	contexts  chan *PooledContext
	maxSize   int
	modelPath string
	closed    bool
	created   int // tracks number of contexts created
	nextID    int // next pool ID to assign
}

// NewContextPool creates a new context pool with the specified maximum size.
// The modelPath is used for lazy initialization of contexts on first Acquire.
//
// Parameters:
//   - maxSize: maximum number of contexts the pool can hold (must be > 0)
//   - modelPath: path to the SD model file for context creation
//
// Returns an error if maxSize is invalid.
func NewContextPool(maxSize int, modelPath string) (*ContextPool, error) {
	if maxSize <= 0 {
		return nil, ErrInvalidParams
	}

	return &ContextPool{
		contexts:  make(chan *PooledContext, maxSize),
		maxSize:   maxSize,
		modelPath: modelPath,
		closed:    false,
		created:   0,
		nextID:    1,
	}, nil
}

// Acquire retrieves a context from the pool, respecting the provided context's deadline.
// If no context is available and the pool has capacity, a new context is lazily created.
//
// Returns:
//   - *PooledContext: a usable context from the pool
//   - error: ErrContextPoolClosed if pool is closed, ErrAcquireTimeout if context deadline exceeded
func (p *ContextPool) Acquire(ctx context.Context) (*PooledContext, error) {
	// Check if pool is closed first
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrContextPoolClosed
	}

	// Try to get from channel without blocking first
	select {
	case pc := <-p.contexts:
		pc.inUse = true
		p.mu.Unlock()
		return pc, nil
	default:
		// No context available immediately
	}

	// Check if we can create a new context
	if p.created < p.maxSize {
		// Create new context while holding lock to update created count
		poolID := p.nextID
		p.nextID++
		p.created++
		p.mu.Unlock()

		sdCtx, err := LoadModel(p.modelPath)
		if err != nil {
			// Failed to create context, decrement created count
			p.mu.Lock()
			p.created--
			p.mu.Unlock()
			return nil, err
		}

		return &PooledContext{
			SDContext: sdCtx,
			poolID:    poolID,
			inUse:     true,
		}, nil
	}
	p.mu.Unlock()

	// Pool at capacity, wait for a context to be released or context cancellation
	select {
	case pc := <-p.contexts:
		if pc == nil {
			// Channel closed
			return nil, ErrContextPoolClosed
		}
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			// Pool closed while waiting, free the context
			FreeContext(pc.SDContext)
			return nil, ErrContextPoolClosed
		}
		pc.inUse = true
		p.mu.Unlock()
		return pc, nil

	case <-ctx.Done():
		// Context deadline or cancellation
		return nil, ErrAcquireTimeout
	}
}

// Release returns a context to the pool for reuse.
// If the pool is closed, the context is freed instead.
// Passing nil is a safe no-op.
func (p *ContextPool) Release(pc *PooledContext) {
	if pc == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	pc.inUse = false

	if p.closed {
		// Pool is closed, free the context instead of returning it
		FreeContext(pc.SDContext)
		p.created--
		return
	}

	// Non-blocking send to return context to pool
	select {
	case p.contexts <- pc:
		// Successfully returned to pool
	default:
		// Pool is full (shouldn't happen with proper usage), free context
		FreeContext(pc.SDContext)
		p.created--
	}
}

// Close shuts down the pool and frees all contexts.
// After Close is called, Acquire will return ErrContextPoolClosed.
// Close is safe to call multiple times.
func (p *ContextPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil // Already closed
	}

	p.closed = true
	close(p.contexts)

	// Drain and free all contexts in the channel
	for pc := range p.contexts {
		if pc != nil && pc.SDContext != nil {
			FreeContext(pc.SDContext)
			p.created--
		}
	}

	return nil
}

// Size returns the number of contexts currently available in the pool.
// This does not include contexts that are currently acquired.
func (p *ContextPool) Size() int {
	return len(p.contexts)
}

// Created returns the total number of contexts that have been created by this pool.
// This includes both available and in-use contexts.
func (p *ContextPool) Created() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.created
}

// MaxSize returns the maximum capacity of the pool.
func (p *ContextPool) MaxSize() int {
	return p.maxSize
}

// IsClosed returns whether the pool has been closed.
func (p *ContextPool) IsClosed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}

// ModelPath returns the model path used by this pool.
func (p *ContextPool) ModelPath() string {
	return p.modelPath
}
