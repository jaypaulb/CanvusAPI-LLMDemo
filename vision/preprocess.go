package vision

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
)

// Image preprocessing errors
var (
	ErrInvalidImage      = errors.New("vision: invalid image data")
	ErrUnsupportedFormat = errors.New("vision: unsupported image format")
	ErrInvalidDimensions = errors.New("vision: invalid dimensions")
	ErrEmptyImage        = errors.New("vision: empty image data")
)

// ResolutionSize defines common model input resolutions
type ResolutionSize int

const (
	// Resolution336 is 336x336, commonly used by vision models
	Resolution336 ResolutionSize = 336
	// Resolution448 is 448x448, used by higher resolution vision models
	Resolution448 ResolutionSize = 448
)

// DecodeImage decodes image data from common formats (PNG, JPEG, GIF).
// This is a pure function with no side effects.
func DecodeImage(data []byte) (image.Image, error) {
	if len(data) == 0 {
		return nil, ErrEmptyImage
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}

	return img, nil
}

// ResizeToSquare resizes an image to a square of the given size using high-quality scaling.
// The image is scaled to fit within the square while maintaining aspect ratio,
// then centered with padding if needed.
// This is a pure function with no side effects.
func ResizeToSquare(img image.Image, size ResolutionSize) image.Image {
	targetSize := int(size)
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate scaling factor to fit within square
	scale := float64(targetSize) / float64(max(width, height))
	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	// Create destination image
	dst := image.NewRGBA(image.Rect(0, 0, targetSize, targetSize))

	// Fill with black background
	for y := 0; y < targetSize; y++ {
		for x := 0; x < targetSize; x++ {
			dst.Set(x, y, color.Black)
		}
	}

	// Calculate centering offsets
	offsetX := (targetSize - newWidth) / 2
	offsetY := (targetSize - newHeight) / 2

	// Create scaled image
	scaled := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, bounds, draw.Over, nil)

	// Draw scaled image centered on destination
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			dst.Set(x+offsetX, y+offsetY, scaled.At(x, y))
		}
	}

	return dst
}

// ConvertToRGB converts any image to RGB format.
// If the image is already in RGB or RGBA format, pixels are extracted.
// Other formats are converted to RGB.
// This is a pure function with no side effects.
func ConvertToRGB(img image.Image) *image.RGBA {
	bounds := img.Bounds()

	// Check if already RGBA
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}

	// Convert to RGBA
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	return rgba
}

// NormalizePixels normalizes RGBA pixel values from [0,255] to [0,1] float32 range.
// Returns RGB channels only (alpha is discarded).
// Output format: width*height*3 float32 values (R,G,B,R,G,B,...)
// This is a pure function with no side effects.
func NormalizePixels(img *image.RGBA) []float32 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Allocate output array (3 channels: RGB)
	output := make([]float32, width*height*3)

	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// RGBA() returns uint32 values in [0, 65535], normalize to [0, 1]
			output[idx] = float32(r) / 65535.0
			output[idx+1] = float32(g) / 65535.0
			output[idx+2] = float32(b) / 65535.0
			idx += 3
		}
	}

	return output
}

// NormalizePixelsCentered normalizes RGBA pixel values from [0,255] to [-1,1] float32 range.
// Returns RGB channels only (alpha is discarded).
// Output format: width*height*3 float32 values (R,G,B,R,G,B,...)
// This is a pure function with no side effects.
func NormalizePixelsCentered(img *image.RGBA) []float32 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Allocate output array (3 channels: RGB)
	output := make([]float32, width*height*3)

	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// RGBA() returns uint32 values in [0, 65535]
			// Normalize to [0, 1] then shift to [-1, 1]
			output[idx] = (float32(r)/65535.0)*2.0 - 1.0
			output[idx+1] = (float32(g)/65535.0)*2.0 - 1.0
			output[idx+2] = (float32(b)/65535.0)*2.0 - 1.0
			idx += 3
		}
	}

	return output
}

// PreprocessImage performs complete image preprocessing pipeline for vision models.
// Steps: decode -> resize to square -> convert to RGB -> normalize pixels.
// This composes atomic functions into a single convenience function.
func PreprocessImage(data []byte, size ResolutionSize, centered bool) ([]float32, int, int, error) {
	// Decode
	img, err := DecodeImage(data)
	if err != nil {
		return nil, 0, 0, err
	}

	// Resize to square
	resized := ResizeToSquare(img, size)

	// Convert to RGB
	rgb := ConvertToRGB(resized)

	// Normalize pixels
	var normalized []float32
	if centered {
		normalized = NormalizePixelsCentered(rgb)
	} else {
		normalized = NormalizePixels(rgb)
	}

	bounds := rgb.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	return normalized, width, height, nil
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
