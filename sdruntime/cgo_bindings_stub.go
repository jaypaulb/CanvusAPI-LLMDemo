//go:build !sd || stub

// Stub implementation of CGo bindings for when stable-diffusion.cpp is not available.
// Build with: go build -tags stub
// Or simply build without the "sd" tag: go build

package sdruntime

import (
	"fmt"
	"os"
	"sync/atomic"
)

// stubContextCounter generates unique IDs for stub contexts
var stubContextCounter uint64

// loadModelImpl is the stub implementation of LoadModel.
// It validates the model path exists but does not actually load a model.
func loadModelImpl(modelPath string) (*SDContext, error) {
	// Check if file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrModelNotFound, modelPath)
	} else if err != nil {
		return nil, fmt.Errorf("%w: unable to access %s: %v", ErrModelLoadFailed, modelPath, err)
	}

	// Create stub context
	ctx := &SDContext{
		id:        atomic.AddUint64(&stubContextCounter, 1),
		modelPath: modelPath,
		valid:     true,
	}

	return ctx, nil
}

// generateImageImpl is the stub implementation of GenerateImage.
// It returns an error indicating the real library is not available.
func generateImageImpl(ctx *SDContext, params GenerateParams) (*GenerateResult, error) {
	if ctx == nil || !ctx.valid {
		return nil, fmt.Errorf("%w: context is nil or invalid", ErrGenerationFailed)
	}

	// Stub mode cannot actually generate images
	return nil, fmt.Errorf("%w: stable-diffusion.cpp library not available (stub mode). "+
		"Build with CGO and the 'sd' tag to enable image generation", ErrGenerationFailed)
}

// freeContextImpl is the stub implementation of FreeContext.
// It marks the context as invalid.
func freeContextImpl(ctx *SDContext) {
	if ctx == nil {
		return
	}
	ctx.valid = false
}

// getBackendInfoImpl returns backend info for stub mode.
func getBackendInfoImpl() string {
	return "stub (no stable-diffusion.cpp library linked)"
}
