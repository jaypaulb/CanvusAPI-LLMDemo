// Package sdruntime provides a CGo wrapper for stable-diffusion.cpp image generation.
//
// This package enables text-to-image generation using Stable Diffusion models
// with CUDA acceleration on NVIDIA GPUs. It follows atomic design principles:
//
//   - Atoms: Pure functions (ValidateParams, ValidatePrompt, RandomSeed, etc.)
//   - Molecules: Simple compositions (ContextPool, GenerateImage)
//   - Organism: This complete package exposing a unified API
//
// # Public API
//
// The primary public API consists of three functions on ContextPool:
//
//   - NewContextPool(maxSize int, modelPath string) (*ContextPool, error)
//   - (*ContextPool) Generate(ctx context.Context, params GenerateParams) ([]byte, error)
//   - (*ContextPool) Close() error
//
// # Quick Start
//
// Basic usage:
//
//	// Create a context pool with max 2 concurrent generations
//	pool, err := sdruntime.NewContextPool(2, "/path/to/sd-v1-5.safetensors")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer pool.Close()
//
//	// Generate an image
//	params := sdruntime.DefaultParams()
//	params.Prompt = "a sunset over mountains"
//
//	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
//	defer cancel()
//
//	imageData, err := pool.Generate(ctx, params)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// imageData is PNG bytes, write to file or upload to canvas
//	os.WriteFile("output.png", imageData, 0644)
//
// # Alternative API (Generator)
//
// For convenience, a higher-level Generator type is also available:
//
//	gen, err := sdruntime.NewGenerator(2, "/path/to/model.safetensors")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer gen.Close()
//
//	imageData, err := gen.Generate(ctx, params)
//
// # Configuration
//
// Use LoadSDConfig() to load configuration from environment variables:
//
//	SD_IMAGE_SIZE=512         # Default image size (512, 768, 1024)
//	SD_INFERENCE_STEPS=20     # Denoising steps (1-100)
//	SD_GUIDANCE_SCALE=7.5     # CFG scale (1.0-30.0)
//	SD_TIMEOUT_SECONDS=120    # Generation timeout
//	SD_MAX_CONCURRENT=2       # Max concurrent generations
//	SD_MODEL_PATH=/path/to/model.safetensors
//
// # Build Tags
//
// The package supports two build modes:
//
//   - Stub mode (default): go build
//     Returns errors for generation but allows testing pool logic
//
//   - Real mode: CGO_ENABLED=1 go build -tags sd
//     Requires stable-diffusion.cpp library to be built and available
//
// # Error Handling
//
// The package defines domain-specific errors:
//
//   - ErrModelNotFound: Model file does not exist
//   - ErrModelLoadFailed: Failed to load model into GPU memory
//   - ErrModelCorrupted: Model checksum mismatch
//   - ErrGenerationFailed: Image generation failed
//   - ErrGenerationTimeout: Generation took too long
//   - ErrInvalidPrompt: Empty or too long prompt
//   - ErrInvalidParams: Parameter validation failed
//   - ErrCUDANotAvailable: NVIDIA GPU required
//   - ErrOutOfVRAM: GPU memory exhausted
//   - ErrContextPoolClosed: Pool has been shut down
//   - ErrAcquireTimeout: Timeout waiting for context
//
// Use errors.Is() for error checking:
//
//	_, err := pool.Generate(ctx, params)
//	if errors.Is(err, sdruntime.ErrOutOfVRAM) {
//	    // Reduce image size or max concurrent
//	}
//
// # Thread Safety
//
// ContextPool is safe for concurrent use. Multiple goroutines can call
// Generate() simultaneously, and the pool will manage context acquisition
// and release automatically.
//
// # Model Verification
//
// Use VerifyModelChecksum() to validate model integrity:
//
//	if err := sdruntime.VerifyModelChecksum("/path/to/model.safetensors"); err != nil {
//	    if errors.Is(err, sdruntime.ErrModelCorrupted) {
//	        log.Fatal("Model file corrupted, please re-download")
//	    }
//	}
package sdruntime
