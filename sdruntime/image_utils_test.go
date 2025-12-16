package sdruntime

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"testing"
)

func TestIsPNG_ValidPNG(t *testing.T) {
	// Create a valid PNG in memory
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	var buf bytes.Buffer
	png.Encode(&buf, img)

	if !IsPNG(buf.Bytes()) {
		t.Error("expected IsPNG to return true for valid PNG")
	}
}

func TestIsPNG_InvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too short", []byte{0x89, 0x50}},
		{"wrong magic", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{"jpeg magic", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsPNG(tt.data) {
				t.Errorf("expected IsPNG to return false for %s", tt.name)
			}
		})
	}
}

func TestValidateImageData_Valid(t *testing.T) {
	// Create a valid PNG in memory
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	var buf bytes.Buffer
	png.Encode(&buf, img)

	err := ValidateImageData(buf.Bytes())
	if err != nil {
		t.Errorf("expected no error for valid PNG, got: %v", err)
	}
}

func TestValidateImageData_Empty(t *testing.T) {
	err := ValidateImageData([]byte{})
	if err == nil {
		t.Error("expected error for empty data")
	}
	if !errors.Is(err, ErrImageEmpty) {
		t.Errorf("expected ErrImageEmpty, got: %v", err)
	}
}

func TestValidateImageData_TooSmall(t *testing.T) {
	err := ValidateImageData([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	if err == nil {
		t.Error("expected error for data too small")
	}
	if !errors.Is(err, ErrImageTooSmall) {
		t.Errorf("expected ErrImageTooSmall, got: %v", err)
	}
}

func TestValidateImageData_NotPNG(t *testing.T) {
	// Create data with wrong magic but sufficient length
	data := make([]byte, 100)
	data[0] = 0xFF // JPEG magic start

	err := ValidateImageData(data)
	if err == nil {
		t.Error("expected error for non-PNG data")
	}
	if !errors.Is(err, ErrImageNotPNG) {
		t.Errorf("expected ErrImageNotPNG, got: %v", err)
	}
}

func TestEncodeToPNG_Valid(t *testing.T) {
	width, height := 2, 2
	// Create RGBA pixel data (4 bytes per pixel)
	pixels := []byte{
		255, 0, 0, 255, // red
		0, 255, 0, 255, // green
		0, 0, 255, 255, // blue
		255, 255, 255, 255, // white
	}

	data, err := EncodeToPNG(pixels, width, height)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify result is valid PNG
	if !IsPNG(data) {
		t.Error("result should be valid PNG")
	}

	// Decode and verify dimensions
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != width || bounds.Dy() != height {
		t.Errorf("expected %dx%d, got %dx%d", width, height, bounds.Dx(), bounds.Dy())
	}
}

func TestEncodeToPNG_InvalidDimensions(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"zero width", 0, 10},
		{"zero height", 10, 0},
		{"negative width", -1, 10},
		{"negative height", 10, -1},
	}

	pixels := make([]byte, 400) // 10x10x4

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncodeToPNG(pixels, tt.width, tt.height)
			if err == nil {
				t.Error("expected error for invalid dimensions")
			}
		})
	}
}

func TestEncodeToPNG_WrongPixelDataLength(t *testing.T) {
	pixels := make([]byte, 10) // Wrong length for any reasonable image

	_, err := EncodeToPNG(pixels, 10, 10)
	if err == nil {
		t.Error("expected error for wrong pixel data length")
	}
	if !errors.Is(err, ErrImageInvalidSize) {
		t.Errorf("expected ErrImageInvalidSize, got: %v", err)
	}
}

func TestImageDataSize(t *testing.T) {
	tests := []struct {
		width    int
		height   int
		expected int
	}{
		{1, 1, 4},
		{10, 10, 400},
		{512, 512, 1048576},
	}

	for _, tt := range tests {
		result := ImageDataSize(tt.width, tt.height)
		if result != tt.expected {
			t.Errorf("ImageDataSize(%d, %d) = %d, expected %d",
				tt.width, tt.height, result, tt.expected)
		}
	}
}
