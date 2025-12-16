// Package sdruntime provides Stable Diffusion image generation capabilities.
//
// generate.go implements the high-level Generate API for image generation.
// This is a molecule that composes atoms from context_pool.go, seed.go,
// types.go, cgo_bindings.go, and image_utils.go.
package sdruntime

import (
	"context"
	"fmt"
)

// Generator provides high-level image generation using a context pool.
// It manages the lifecycle of SD contexts and provides a simple Generate API.
//
// This molecule composes:
//   - ContextPool: for thread-safe context management
//   - ValidateParams: for parameter validation (atom)
//   - RandomSeed: for seed generation (atom)
//   - GenerateImage: for actual generation (CGo binding)
//   - ValidateImageData: for output validation (atom)
type Generator struct {
	pool *ContextPool
}

// NewGenerator creates a Generator with a context pool of the specified size.
//
// Parameters:
//   - poolSize: number of contexts to maintain in the pool (must be > 0)
//   - modelPath: path to the SD model file
//
// Returns an error if poolSize is invalid or model cannot be validated.
func NewGenerator(poolSize int, modelPath string) (*Generator, error) {
	pool, err := NewContextPool(poolSize, modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create context pool: %w", err)
	}

	return &Generator{
		pool: pool,
	}, nil
}

// Generate creates an image from the given parameters.
// It acquires a context from the pool, generates the image, and releases the context.
//
// The ctx parameter controls cancellation and timeout. If ctx is cancelled or
// times out while waiting for a pool context, ErrAcquireTimeout is returned.
//
// Parameters:
//   - ctx: context for cancellation/timeout
//   - params: generation parameters (prompt, dimensions, etc.)
//
// Returns PNG image data as []byte, or an error.
//
// Error cases:
//   - ErrInvalidParams: parameters fail validation
//   - ErrAcquireTimeout: context.Done() before acquiring pool context
//   - ErrContextPoolClosed: generator has been closed
//   - ErrGenerationFailed: SD generation failed
//   - ErrOutOfVRAM: GPU memory exhausted
func (g *Generator) Generate(ctx context.Context, params GenerateParams) ([]byte, error) {
	// Step 1: Validate parameters (atom)
	if err := ValidateParams(params); err != nil {
		return nil, err
	}

	// Step 2: Handle seed (-1 means random)
	if params.Seed < 0 {
		params.Seed = RandomSeed()
	}

	// Step 3: Acquire context from pool (molecule)
	pooledCtx, err := g.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer g.pool.Release(pooledCtx)

	// Step 4: Generate image using CGo binding
	result, err := GenerateImage(pooledCtx.SDContext, params)
	if err != nil {
		return nil, err
	}

	// Step 5: Validate output image (atom)
	if err := ValidateImageData(result.ImageData); err != nil {
		return nil, fmt.Errorf("generated image validation failed: %w", err)
	}

	return result.ImageData, nil
}

// GenerateWithResult creates an image and returns full result metadata.
// This variant returns the seed used and dimensions alongside the image data.
//
// This is useful when params.Seed is -1 and the caller needs to know the
// actual seed used for reproduction.
func (g *Generator) GenerateWithResult(ctx context.Context, params GenerateParams) (*GenerateResult, error) {
	// Step 1: Validate parameters (atom)
	if err := ValidateParams(params); err != nil {
		return nil, err
	}

	// Step 2: Handle seed (-1 means random)
	actualSeed := params.Seed
	if actualSeed < 0 {
		actualSeed = RandomSeed()
		params.Seed = actualSeed
	}

	// Step 3: Acquire context from pool (molecule)
	pooledCtx, err := g.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer g.pool.Release(pooledCtx)

	// Step 4: Generate image using CGo binding
	result, err := GenerateImage(pooledCtx.SDContext, params)
	if err != nil {
		return nil, err
	}

	// Step 5: Validate output image (atom)
	if err := ValidateImageData(result.ImageData); err != nil {
		return nil, fmt.Errorf("generated image validation failed: %w", err)
	}

	// Ensure result has actual seed
	result.Seed = actualSeed

	return result, nil
}

// Close shuts down the generator and releases all pooled contexts.
// After Close is called, Generate will return ErrContextPoolClosed.
// Close is safe to call multiple times.
func (g *Generator) Close() error {
	return g.pool.Close()
}

// PoolSize returns the maximum number of contexts in the pool.
func (g *Generator) PoolSize() int {
	return g.pool.MaxSize()
}

// PoolAvailable returns the number of contexts currently available.
func (g *Generator) PoolAvailable() int {
	return g.pool.Size()
}

// IsClosed returns whether the generator has been closed.
func (g *Generator) IsClosed() bool {
	return g.pool.IsClosed()
}

// DefaultParams returns sensible default parameters for image generation.
// The caller should at minimum set the Prompt field.
//
// Default values:
//   - Width: 512
//   - Height: 512
//   - Steps: 20
//   - CFGScale: 7.0
//   - Seed: -1 (random)
func DefaultParams() GenerateParams {
	return GenerateParams{
		Prompt:         "",
		NegativePrompt: "",
		Width:          512,
		Height:         512,
		Steps:          20,
		CFGScale:       7.0,
		Seed:           -1,
	}
}

// QuickGenerate is a convenience function for simple one-off generation.
// It creates a temporary generator, generates the image, and cleans up.
//
// This is less efficient for multiple generations as it loads the model
// each time. For batch generation, use NewGenerator and reuse it.
//
// Parameters:
//   - ctx: context for cancellation/timeout
//   - modelPath: path to the SD model file
//   - prompt: text description of the image
//
// Returns PNG image data as []byte, or an error.
func QuickGenerate(ctx context.Context, modelPath string, prompt string) ([]byte, error) {
	gen, err := NewGenerator(1, modelPath)
	if err != nil {
		return nil, err
	}
	defer gen.Close()

	params := DefaultParams()
	params.Prompt = prompt

	return gen.Generate(ctx, params)
}
