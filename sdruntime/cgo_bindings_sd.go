//go:build sd && cgo && !stub
// +build sd,cgo,!stub

// Real CGo implementation of stable-diffusion.cpp bindings.
// Build with: CGO_ENABLED=1 go build -tags sd
//
// Prerequisites:
//  1. stable-diffusion.cpp must be compiled as a shared library
//  2. Library and headers in deps/stable-diffusion.cpp/
//  3. Compiled library in lib/
//
// Build stable-diffusion.cpp first:
//
//	cd deps/stable-diffusion.cpp
//	./build-linux.sh  # or build-windows.ps1
//
// Then build Go application:
//
//	CGO_ENABLED=1 go build -tags sd
package sdruntime

/*
#cgo CFLAGS: -I${SRCDIR}/../deps/stable-diffusion.cpp/include
#cgo linux LDFLAGS: -L${SRCDIR}/../lib -lstable-diffusion -Wl,-rpath,${SRCDIR}/../lib
#cgo windows LDFLAGS: -L${SRCDIR}/../lib -lstable-diffusion
#cgo darwin LDFLAGS: -L${SRCDIR}/../lib -lstable-diffusion -Wl,-rpath,${SRCDIR}/../lib

#include <stable-diffusion.h>
#include <stdlib.h>
#include <stdbool.h>
*/
import "C"

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// sdContextCounter generates unique IDs for contexts
var sdContextCounter uint64

// cgoContext holds the C context pointer alongside Go metadata
type cgoContext struct {
	cCtx *C.sd_ctx_t
}

// contextMap stores the mapping from SDContext.id to cgoContext
// Thread-safe map for concurrent access from multiple goroutines
var contextMap sync.Map

// loadModelImpl is the real CGo implementation of LoadModel.
func loadModelImpl(modelPath string) (*SDContext, error) {
	// Validate file exists first
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrModelNotFound, modelPath)
	} else if err != nil {
		return nil, fmt.Errorf("%w: unable to access %s: %v", ErrModelLoadFailed, modelPath, err)
	}

	// Convert Go string to C string
	cModelPath := C.CString(modelPath)
	defer C.free(unsafe.Pointer(cModelPath))

	// Determine optimal thread count
	numThreads := runtime.NumCPU()

	// Call C library to create context
	// sd_ctx_create parameters:
	//   - model_path: path to model file
	//   - vae_path: NULL (use built-in VAE)
	//   - taesd_path: NULL (no TAESD for fast preview)
	//   - lora_model_dir: NULL (no LoRA models)
	//   - vae_decode_only: true (txt2img only, no img2img)
	//   - n_threads: CPU threads for non-CUDA ops
	//   - vae_tiling: false (disable for performance)
	//   - free_params_immediately: false (keep params loaded)
	cCtx := C.sd_ctx_create(
		cModelPath,
		nil,                // vae_path (NULL = use built-in)
		nil,                // taesd_path (NULL = no fast preview)
		nil,                // lora_model_dir (NULL = no LoRA)
		C.bool(true),       // vae_decode_only (txt2img only)
		C.int(numThreads),  // n_threads
		C.bool(false),      // vae_tiling (disable for performance)
		C.bool(false),      // free_params_immediately (keep loaded)
	)

	if cCtx == nil {
		return nil, fmt.Errorf("%w: C library returned null context", ErrModelLoadFailed)
	}

	// Generate unique ID and store in thread-safe map
	id := atomic.AddUint64(&sdContextCounter, 1)
	contextMap.Store(id, &cgoContext{cCtx: cCtx})

	return &SDContext{
		id:        id,
		modelPath: modelPath,
		valid:     true,
	}, nil
}

// generateImageImpl is the real CGo implementation of GenerateImage.
func generateImageImpl(ctx *SDContext, params GenerateParams) (*GenerateResult, error) {
	if ctx == nil || !ctx.valid {
		return nil, fmt.Errorf("%w: context is nil or invalid", ErrGenerationFailed)
	}

	// Get C context from thread-safe map
	val, ok := contextMap.Load(ctx.id)
	if !ok {
		return nil, fmt.Errorf("%w: no valid C context found", ErrGenerationFailed)
	}

	cgoCtx, ok := val.(*cgoContext)
	if !ok || cgoCtx == nil || cgoCtx.cCtx == nil {
		return nil, fmt.Errorf("%w: invalid C context type", ErrGenerationFailed)
	}

	// Convert Go strings to C strings
	cPrompt := C.CString(params.Prompt)
	defer C.free(unsafe.Pointer(cPrompt))

	cNegPrompt := C.CString(params.NegativePrompt)
	defer C.free(unsafe.Pointer(cNegPrompt))

	// Resolve seed if random requested
	seed := params.Seed
	if seed < 0 {
		seed = RandomSeed()
	}

	// Call txt2img with full parameter set
	// txt2img parameters:
	//   - ctx: SD context
	//   - prompt: text description
	//   - negative_prompt: what to avoid
	//   - clip_skip: -1 (use default)
	//   - cfg_scale: guidance scale
	//   - width: image width (must be multiple of 8)
	//   - height: image height (must be multiple of 8)
	//   - sample_method: SD_SAMPLE_DPMPP_2M (recommended)
	//   - sample_steps: inference steps
	//   - seed: random seed
	//   - batch_count: 1 (single image)
	imgPtr := C.txt2img(
		cgoCtx.cCtx,
		cPrompt,
		cNegPrompt,
		C.int(-1),                       // clip_skip (-1 = default)
		C.float(params.CFGScale),
		C.int(params.Width),
		C.int(params.Height),
		C.SD_SAMPLE_DPMPP_2M,            // sample_method (recommended)
		C.int(params.Steps),
		C.int64_t(seed),
		C.int(1),                        // batch_count (single image)
	)

	if imgPtr == nil {
		return nil, fmt.Errorf("%w: txt2img returned null", ErrGenerationFailed)
	}
	defer C.sd_free_image(imgPtr)

	// Extract image data from C struct
	// The C API returns sd_image_t with RGBA data
	width := int(imgPtr.width)
	height := int(imgPtr.height)
	channels := int(imgPtr.channels)

	// Calculate image size (RGBA: 4 bytes per pixel)
	imgSize := width * height * channels

	// Copy C memory to Go slice before freeing
	imgData := C.GoBytes(unsafe.Pointer(imgPtr.data), C.int(imgSize))

	// Convert RGBA bytes to PNG format using atom function
	pngData, err := EncodeToPNG(imgData, width, height)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to encode PNG: %v", ErrGenerationFailed, err)
	}

	return &GenerateResult{
		ImageData: pngData,
		Width:     width,
		Height:    height,
		Seed:      seed,
	}, nil
}

// freeContextImpl is the real CGo implementation of FreeContext.
func freeContextImpl(ctx *SDContext) {
	if ctx == nil {
		return
	}

	// Get and remove C context from thread-safe map
	val, ok := contextMap.LoadAndDelete(ctx.id)
	if ok {
		if cgoCtx, ok := val.(*cgoContext); ok && cgoCtx != nil && cgoCtx.cCtx != nil {
			C.sd_ctx_free(cgoCtx.cCtx)
		}
	}

	ctx.valid = false
}

// getBackendInfoImpl returns backend info from the C library.
func getBackendInfoImpl() string {
	cInfo := C.sd_get_backend_info()
	if cInfo != nil {
		return C.GoString(cInfo)
	}
	return "sd (CGo bindings - unknown backend)"
}

// IsCUDAAvailable checks if CUDA is available via the C library.
func IsCUDAAvailable() bool {
	return bool(C.sd_cuda_available())
}

// Ensure atomic is used to avoid unused import error
var _ = atomic.AddUint64
