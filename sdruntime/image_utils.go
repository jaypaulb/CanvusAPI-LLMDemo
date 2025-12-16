package sdruntime

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
)

// PNG magic bytes for file identification
var pngMagic = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// Image validation errors
var (
	ErrImageEmpty       = errors.New("sdruntime: image data is empty")
	ErrImageNotPNG      = errors.New("sdruntime: image data is not a valid PNG")
	ErrImageTooSmall    = errors.New("sdruntime: image data too small to be valid")
	ErrImageDecodeFail  = errors.New("sdruntime: failed to decode image")
	ErrImageInvalidSize = errors.New("sdruntime: invalid image dimensions")
)

// IsPNG checks if the given data starts with PNG magic bytes.
// This is a pure function with no side effects.
func IsPNG(data []byte) bool {
	if len(data) < len(pngMagic) {
		return false
	}
	return bytes.Equal(data[:len(pngMagic)], pngMagic)
}

// ValidateImageData validates that data is a valid PNG image.
// Returns nil if valid, error otherwise.
// This is a pure function with no side effects.
func ValidateImageData(data []byte) error {
	if len(data) == 0 {
		return ErrImageEmpty
	}

	// Minimum PNG file size (header + IHDR + IEND chunks)
	// 8 (signature) + 25 (IHDR) + 12 (IEND) = 45 bytes minimum
	if len(data) < 45 {
		return ErrImageTooSmall
	}

	if !IsPNG(data) {
		return ErrImageNotPNG
	}

	// Attempt to decode to validate structure
	_, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrImageDecodeFail, err)
	}

	return nil
}

// EncodeToPNG encodes raw RGBA pixels to PNG format.
// pixels must be in RGBA format (4 bytes per pixel).
// Returns the encoded PNG data or an error.
// This is a pure function with no side effects.
func EncodeToPNG(pixels []byte, width, height int) ([]byte, error) {
	// Validate dimensions
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("%w: width=%d height=%d", ErrImageInvalidSize, width, height)
	}

	// Validate pixel data length (4 bytes per pixel for RGBA)
	expectedLen := width * height * 4
	if len(pixels) != expectedLen {
		return nil, fmt.Errorf("%w: expected %d bytes for %dx%d RGBA, got %d",
			ErrImageInvalidSize, expectedLen, width, height, len(pixels))
	}

	// Create RGBA image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(img.Pix, pixels)

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrImageDecodeFail, err)
	}

	return buf.Bytes(), nil
}

// ImageDataSize calculates the byte size needed for RGBA image data.
// This is a pure helper function.
func ImageDataSize(width, height int) int {
	return width * height * 4
}
