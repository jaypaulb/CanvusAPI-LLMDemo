package vision

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// createTestImage creates a simple test image with known pixel values
func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a gradient pattern for testing
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(128)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	return img
}

// encodePNG encodes an image to PNG bytes
func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestDecodeImage(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		errType error
	}{
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
			errType: ErrEmptyImage,
		},
		{
			name:    "invalid data",
			data:    []byte{0x00, 0x01, 0x02},
			wantErr: true,
			errType: ErrInvalidImage,
		},
		{
			name:    "valid PNG",
			data:    encodePNG(createTestImage(10, 10)),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, err := DecodeImage(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Errorf("DecodeImage() expected error but got nil")
					return
				}
				return
			}

			if err != nil {
				t.Errorf("DecodeImage() unexpected error: %v", err)
				return
			}

			if img == nil {
				t.Errorf("DecodeImage() returned nil image")
			}
		})
	}
}

func TestResizeToSquare(t *testing.T) {
	tests := []struct {
		name       string
		inputW     int
		inputH     int
		targetSize ResolutionSize
		wantW      int
		wantH      int
	}{
		{
			name:       "square to 336",
			inputW:     100,
			inputH:     100,
			targetSize: Resolution336,
			wantW:      336,
			wantH:      336,
		},
		{
			name:       "landscape to 336",
			inputW:     200,
			inputH:     100,
			targetSize: Resolution336,
			wantW:      336,
			wantH:      336,
		},
		{
			name:       "portrait to 448",
			inputW:     100,
			inputH:     200,
			targetSize: Resolution448,
			wantW:      448,
			wantH:      448,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestImage(tt.inputW, tt.inputH)
			result := ResizeToSquare(input, tt.targetSize)

			bounds := result.Bounds()
			gotW := bounds.Dx()
			gotH := bounds.Dy()

			if gotW != tt.wantW {
				t.Errorf("ResizeToSquare() width = %d, want %d", gotW, tt.wantW)
			}
			if gotH != tt.wantH {
				t.Errorf("ResizeToSquare() height = %d, want %d", gotH, tt.wantH)
			}
		})
	}
}

func TestConvertToRGB(t *testing.T) {
	tests := []struct {
		name  string
		input image.Image
	}{
		{
			name:  "RGBA image",
			input: createTestImage(10, 10),
		},
		{
			name:  "Gray image",
			input: image.NewGray(image.Rect(0, 0, 10, 10)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToRGB(tt.input)

			if result == nil {
				t.Errorf("ConvertToRGB() returned nil")
				return
			}

			bounds := result.Bounds()
			inputBounds := tt.input.Bounds()

			if bounds != inputBounds {
				t.Errorf("ConvertToRGB() bounds = %v, want %v", bounds, inputBounds)
			}
		})
	}
}

func TestNormalizePixels(t *testing.T) {
	// Create a simple 2x2 image with known values
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{0, 0, 0, 255})       // Black
	img.Set(1, 0, color.RGBA{255, 0, 0, 255})     // Red
	img.Set(0, 1, color.RGBA{0, 255, 0, 255})     // Green
	img.Set(1, 1, color.RGBA{255, 255, 255, 255}) // White

	result := NormalizePixels(img)

	// Should have 2*2*3 = 12 values
	expectedLen := 12
	if len(result) != expectedLen {
		t.Errorf("NormalizePixels() length = %d, want %d", len(result), expectedLen)
	}

	// Check normalization range [0, 1]
	for i, val := range result {
		if val < 0.0 || val > 1.0 {
			t.Errorf("NormalizePixels()[%d] = %f, want value in [0, 1]", i, val)
		}
	}

	// Check specific values
	// Pixel (0,0) = black -> R=0, G=0, B=0
	if result[0] != 0.0 || result[1] != 0.0 || result[2] != 0.0 {
		t.Errorf("NormalizePixels() black pixel incorrect: R=%f, G=%f, B=%f", result[0], result[1], result[2])
	}

	// Pixel (1,0) = red -> R=1, G=0, B=0
	if result[3] < 0.99 || result[4] != 0.0 || result[5] != 0.0 {
		t.Errorf("NormalizePixels() red pixel incorrect: R=%f, G=%f, B=%f", result[3], result[4], result[5])
	}
}

func TestNormalizePixelsCentered(t *testing.T) {
	// Create a simple 2x2 image with known values
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{0, 0, 0, 255})       // Black -> -1
	img.Set(1, 0, color.RGBA{255, 255, 255, 255}) // White -> 1
	img.Set(0, 1, color.RGBA{128, 128, 128, 255}) // Gray -> ~0
	img.Set(1, 1, color.RGBA{255, 0, 0, 255})     // Red

	result := NormalizePixelsCentered(img)

	// Should have 2*2*3 = 12 values
	expectedLen := 12
	if len(result) != expectedLen {
		t.Errorf("NormalizePixelsCentered() length = %d, want %d", len(result), expectedLen)
	}

	// Check normalization range [-1, 1]
	for i, val := range result {
		if val < -1.0 || val > 1.0 {
			t.Errorf("NormalizePixelsCentered()[%d] = %f, want value in [-1, 1]", i, val)
		}
	}

	// Check black pixel (0,0) -> should be close to -1
	if result[0] > -0.99 || result[1] > -0.99 || result[2] > -0.99 {
		t.Errorf("NormalizePixelsCentered() black pixel incorrect: R=%f, G=%f, B=%f (expected ~-1)", result[0], result[1], result[2])
	}

	// Check white pixel (1,0) -> should be close to 1
	if result[3] < 0.99 || result[4] < 0.99 || result[5] < 0.99 {
		t.Errorf("NormalizePixelsCentered() white pixel incorrect: R=%f, G=%f, B=%f (expected ~1)", result[3], result[4], result[5])
	}
}

func TestPreprocessImage(t *testing.T) {
	tests := []struct {
		name       string
		size       ResolutionSize
		centered   bool
		wantWidth  int
		wantHeight int
		wantLen    int
	}{
		{
			name:       "336x336 normalized [0,1]",
			size:       Resolution336,
			centered:   false,
			wantWidth:  336,
			wantHeight: 336,
			wantLen:    336 * 336 * 3,
		},
		{
			name:       "448x448 normalized [-1,1]",
			size:       Resolution448,
			centered:   true,
			wantWidth:  448,
			wantHeight: 448,
			wantLen:    448 * 448 * 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image and encode as PNG
			testImg := createTestImage(100, 150)
			data := encodePNG(testImg)

			// Preprocess
			pixels, width, height, err := PreprocessImage(data, tt.size, tt.centered)
			if err != nil {
				t.Errorf("PreprocessImage() unexpected error: %v", err)
				return
			}

			// Verify dimensions
			if width != tt.wantWidth {
				t.Errorf("PreprocessImage() width = %d, want %d", width, tt.wantWidth)
			}
			if height != tt.wantHeight {
				t.Errorf("PreprocessImage() height = %d, want %d", height, tt.wantHeight)
			}

			// Verify pixel array length
			if len(pixels) != tt.wantLen {
				t.Errorf("PreprocessImage() pixels length = %d, want %d", len(pixels), tt.wantLen)
			}

			// Verify normalization range
			var minVal, maxVal float32
			if tt.centered {
				minVal, maxVal = -1.0, 1.0
			} else {
				minVal, maxVal = 0.0, 1.0
			}

			for i, val := range pixels {
				if val < minVal || val > maxVal {
					t.Errorf("PreprocessImage()[%d] = %f, want value in [%f, %f]", i, val, minVal, maxVal)
					break
				}
			}
		})
	}
}

func TestPreprocessImage_InvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "invalid image",
			data:    []byte{0x00, 0x01, 0x02},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := PreprocessImage(tt.data, Resolution336, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("PreprocessImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkPreprocessImage336(b *testing.B) {
	testImg := createTestImage(800, 600)
	data := encodePNG(testImg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, err := PreprocessImage(data, Resolution336, false)
		if err != nil {
			b.Fatalf("PreprocessImage() error: %v", err)
		}
	}
}

func BenchmarkPreprocessImage448(b *testing.B) {
	testImg := createTestImage(1024, 768)
	data := encodePNG(testImg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, err := PreprocessImage(data, Resolution448, false)
		if err != nil {
			b.Fatalf("PreprocessImage() error: %v", err)
		}
	}
}
