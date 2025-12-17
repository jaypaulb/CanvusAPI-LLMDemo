package handlers_test

import (
	"math"
	"testing"

	"go_backend/handlers"
)

func TestCalculateNoteSize(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		originalWidth  float64
		originalHeight float64
		originalScale  float64
		wantOriginal   bool // Should return original size?
		wantMinWidth   bool // Should respect minimum width?
	}{
		{
			name:           "short content returns original",
			content:        "Hello world",
			originalWidth:  300,
			originalHeight: 200,
			originalScale:  0.5,
			wantOriginal:   true,
		},
		{
			name:           "medium content under 150 tokens",
			content:        "This is a slightly longer text that is still under 150 tokens which is approximately 600 characters.",
			originalWidth:  400,
			originalHeight: 300,
			originalScale:  0.6,
			wantOriginal:   true,
		},
		{
			name: "long content calculates new size",
			content: `This is a much longer text that will definitely exceed 150 tokens.
It has multiple lines and quite a bit of content.
Lorem ipsum dolor sit amet, consectetur adipiscing elit.
Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.
Nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor.
In reprehenderit in voluptate velit esse cillum dolore eu fugiat.
Excepteur sint occaecat cupidatat non proident, sunt in culpa.
This adds more content to ensure we exceed the threshold easily.
More text here to make it really long and trigger the calculation.`,
			originalWidth:  300,
			originalHeight: 200,
			originalScale:  0.5,
			wantOriginal:   false,
		},
		{
			name:           "empty content returns original",
			content:        "",
			originalWidth:  500,
			originalHeight: 400,
			originalScale:  0.7,
			wantOriginal:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, scale := handlers.CalculateNoteSize(tt.content, tt.originalWidth, tt.originalHeight, tt.originalScale)

			if tt.wantOriginal {
				if size.Width != tt.originalWidth || size.Height != tt.originalHeight {
					t.Errorf("expected original size (%.0f x %.0f), got (%.0f x %.0f)",
						tt.originalWidth, tt.originalHeight, size.Width, size.Height)
				}
				if scale != tt.originalScale {
					t.Errorf("expected original scale %.2f, got %.2f", tt.originalScale, scale)
				}
			} else {
				// For long content, just verify reasonable values
				if size.Width < handlers.DefaultMinWidth {
					t.Errorf("width %.0f below minimum %.0f", size.Width, handlers.DefaultMinWidth)
				}
				if size.Height <= 0 {
					t.Errorf("height should be positive, got %.0f", size.Height)
				}
				if scale <= 0 || scale > handlers.DefaultMaxScale {
					t.Errorf("scale %.2f out of range (0, %.2f]", scale, handlers.DefaultMaxScale)
				}
			}
		})
	}
}

func TestCalculateOffsetLocation(t *testing.T) {
	tests := []struct {
		name         string
		x, y         float64
		width        float64
		height       float64
		offsetX      float64
		offsetY      float64
		wantX, wantY float64
	}{
		{
			name: "zero offset",
			x:    100, y: 200,
			width: 400, height: 300,
			offsetX: 0, offsetY: 0,
			wantX: 100, wantY: 200,
		},
		{
			name: "standard 80% offset",
			x:    100, y: 200,
			width: 400, height: 300,
			offsetX: 0.8, offsetY: 0.8,
			wantX: 420, wantY: 440, // 100 + (400*0.8), 200 + (300*0.8)
		},
		{
			name: "negative offset",
			x:    100, y: 200,
			width: 400, height: 300,
			offsetX: -0.5, offsetY: -0.5,
			wantX: -100, wantY: 50, // 100 + (400*-0.5), 200 + (300*-0.5)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := handlers.CalculateOffsetLocation(tt.x, tt.y, tt.width, tt.height, tt.offsetX, tt.offsetY)
			if math.Abs(loc.X-tt.wantX) > 0.001 || math.Abs(loc.Y-tt.wantY) > 0.001 {
				t.Errorf("got (%.0f, %.0f), want (%.0f, %.0f)", loc.X, loc.Y, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestAdjustBackgroundColorOpacity(t *testing.T) {
	tests := []struct {
		name    string
		color   string
		opacity float64
		want    string
	}{
		{
			name:    "empty color returns default",
			color:   "",
			opacity: 0.75,
			want:    "#FFFFFFBF",
		},
		{
			name:    "RRGGBB format adds alpha",
			color:   "#FF0000",
			opacity: 0.75,
			want:    "#FF0000BF",
		},
		{
			name:    "RRGGBB format full opacity",
			color:   "#00FF00",
			opacity: 1.0,
			want:    "#00FF00FF",
		},
		{
			name:    "RRGGBBAA format adjusts alpha",
			color:   "#0000FFFF",
			opacity: 0.5,
			want:    "#0000FF7F",
		},
		{
			name:    "RRGGBBAA format with half alpha",
			color:   "#00000080",
			opacity: 0.5,
			want:    "#00000040",
		},
		{
			name:    "unknown format unchanged",
			color:   "#FFF",
			opacity: 0.5,
			want:    "#FFF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handlers.AdjustBackgroundColorOpacity(tt.color, tt.opacity)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReduceBackgroundOpacity(t *testing.T) {
	tests := []struct {
		name  string
		color string
		want  string
	}{
		{
			name:  "empty color returns default",
			color: "",
			want:  "#FFFFFFBF",
		},
		{
			name:  "RRGGBB format adds 75% opacity",
			color: "#FF0000",
			want:  "#FF0000BF",
		},
		{
			name:  "RRGGBBAA format reduces alpha",
			color: "#0000FFFF",
			want:  "#0000FFDD", // 255 / 1.15 ≈ 221.74 → int(221) = 0xDD
		},
		{
			name:  "unknown format unchanged",
			color: "#FFF",
			want:  "#FFF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handlers.ReduceBackgroundOpacity(tt.color)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCalculateDepthOffset(t *testing.T) {
	tests := []struct {
		name     string
		original float64
		offset   float64
		want     float64
	}{
		{"positive offset", 100, 200, 300},
		{"negative offset", 100, -50, 50},
		{"zero offset", 100, 0, 100},
		{"negative original", -100, 200, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handlers.CalculateDepthOffset(tt.original, tt.offset)
			if got != tt.want {
				t.Errorf("got %.0f, want %.0f", got, tt.want)
			}
		})
	}
}

func TestExtractLocation(t *testing.T) {
	tests := []struct {
		name   string
		locMap map[string]interface{}
		wantX  float64
		wantY  float64
	}{
		{
			name:   "nil map",
			locMap: nil,
			wantX:  0, wantY: 0,
		},
		{
			name:   "empty map",
			locMap: map[string]interface{}{},
			wantX:  0, wantY: 0,
		},
		{
			name:   "valid coordinates",
			locMap: map[string]interface{}{"x": 100.0, "y": 200.0},
			wantX:  100, wantY: 200,
		},
		{
			name:   "missing x",
			locMap: map[string]interface{}{"y": 200.0},
			wantX:  0, wantY: 0,
		},
		{
			name:   "wrong type",
			locMap: map[string]interface{}{"x": "100", "y": 200.0},
			wantX:  0, wantY: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := handlers.ExtractLocation(tt.locMap)
			if loc.X != tt.wantX || loc.Y != tt.wantY {
				t.Errorf("got (%.0f, %.0f), want (%.0f, %.0f)", loc.X, loc.Y, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestExtractSize(t *testing.T) {
	tests := []struct {
		name    string
		sizeMap map[string]interface{}
		wantW   float64
		wantH   float64
	}{
		{
			name:    "nil map",
			sizeMap: nil,
			wantW:   0, wantH: 0,
		},
		{
			name:    "valid size",
			sizeMap: map[string]interface{}{"width": 400.0, "height": 300.0},
			wantW:   400, wantH: 300,
		},
		{
			name:    "missing width",
			sizeMap: map[string]interface{}{"height": 300.0},
			wantW:   0, wantH: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := handlers.ExtractSize(tt.sizeMap)
			if size.Width != tt.wantW || size.Height != tt.wantH {
				t.Errorf("got (%.0f x %.0f), want (%.0f x %.0f)", size.Width, size.Height, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestLocationToMap(t *testing.T) {
	loc := handlers.Location{X: 100.5, Y: 200.5}
	result := handlers.LocationToMap(loc)

	if result["x"] != 100.5 || result["y"] != 200.5 {
		t.Errorf("got %v, want {x: 100.5, y: 200.5}", result)
	}
}

func TestSizeToMap(t *testing.T) {
	size := handlers.NoteSize{Width: 400.5, Height: 300.5}
	result := handlers.SizeToMap(size)

	if result["width"] != 400.5 || result["height"] != 300.5 {
		t.Errorf("got %v, want {width: 400.5, height: 300.5}", result)
	}
}

func TestAddLocations(t *testing.T) {
	tests := []struct {
		name  string
		a, b  handlers.Location
		wantX float64
		wantY float64
	}{
		{
			name:  "simple addition",
			a:     handlers.Location{X: 100, Y: 200},
			b:     handlers.Location{X: 50, Y: 30},
			wantX: 150, wantY: 230,
		},
		{
			name:  "negative values",
			a:     handlers.Location{X: 100, Y: 200},
			b:     handlers.Location{X: -50, Y: -30},
			wantX: 50, wantY: 170,
		},
		{
			name:  "zero addition",
			a:     handlers.Location{X: 100, Y: 200},
			b:     handlers.Location{X: 0, Y: 0},
			wantX: 100, wantY: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handlers.AddLocations(tt.a, tt.b)
			if result.X != tt.wantX || result.Y != tt.wantY {
				t.Errorf("got (%.0f, %.0f), want (%.0f, %.0f)", result.X, result.Y, tt.wantX, tt.wantY)
			}
		})
	}
}
