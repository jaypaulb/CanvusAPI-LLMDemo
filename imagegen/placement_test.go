package imagegen

import (
	"testing"
)

// TestCalculatePlacement tests the default placement calculation.
func TestCalculatePlacement(t *testing.T) {
	tests := []struct {
		name     string
		widget   SimpleWidget
		expectedX float64
		expectedY float64
	}{
		{
			name: "origin widget",
			widget: SimpleWidget{
				Location: WidgetLocation{X: 0, Y: 0},
				Size:     WidgetSize{Width: 100, Height: 100},
			},
			expectedX: DefaultOffsetX, // 0 + 300
			expectedY: DefaultOffsetY, // 0 + 50
		},
		{
			name: "offset widget",
			widget: SimpleWidget{
				Location: WidgetLocation{X: 500, Y: 200},
				Size:     WidgetSize{Width: 100, Height: 100},
			},
			expectedX: 500 + DefaultOffsetX, // 800
			expectedY: 200 + DefaultOffsetY, // 250
		},
		{
			name: "negative coordinates",
			widget: SimpleWidget{
				Location: WidgetLocation{X: -100, Y: -50},
				Size:     WidgetSize{Width: 100, Height: 100},
			},
			expectedX: -100 + DefaultOffsetX, // 200
			expectedY: -50 + DefaultOffsetY,  // 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := CalculatePlacement(tt.widget)

			if x != tt.expectedX {
				t.Errorf("CalculatePlacement() x = %v, want %v", x, tt.expectedX)
			}
			if y != tt.expectedY {
				t.Errorf("CalculatePlacement() y = %v, want %v", y, tt.expectedY)
			}
		})
	}
}

// TestCalculatePlacementWithConfig tests custom offset configuration.
func TestCalculatePlacementWithConfig(t *testing.T) {
	widget := SimpleWidget{
		Location: WidgetLocation{X: 100, Y: 100},
		Size:     WidgetSize{Width: 200, Height: 150},
	}

	tests := []struct {
		name      string
		config    PlacementConfig
		expectedX float64
		expectedY float64
	}{
		{
			name:      "default config",
			config:    DefaultPlacementConfig(),
			expectedX: 100 + DefaultOffsetX,
			expectedY: 100 + DefaultOffsetY,
		},
		{
			name:      "custom offset",
			config:    PlacementConfig{OffsetX: 50, OffsetY: 25},
			expectedX: 150, // 100 + 50
			expectedY: 125, // 100 + 25
		},
		{
			name:      "zero offset",
			config:    PlacementConfig{OffsetX: 0, OffsetY: 0},
			expectedX: 100,
			expectedY: 100,
		},
		{
			name:      "negative offset",
			config:    PlacementConfig{OffsetX: -20, OffsetY: -10},
			expectedX: 80,  // 100 - 20
			expectedY: 90,  // 100 - 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := CalculatePlacementWithConfig(widget, tt.config)

			if x != tt.expectedX {
				t.Errorf("CalculatePlacementWithConfig() x = %v, want %v", x, tt.expectedX)
			}
			if y != tt.expectedY {
				t.Errorf("CalculatePlacementWithConfig() y = %v, want %v", y, tt.expectedY)
			}
		})
	}
}

// TestCalculatePlacementWithSize tests size-aware placement.
func TestCalculatePlacementWithSize(t *testing.T) {
	tests := []struct {
		name      string
		widget    SimpleWidget
		config    PlacementConfig
		expectedX float64
		expectedY float64
	}{
		{
			name: "standard widget right edge",
			widget: SimpleWidget{
				Location: WidgetLocation{X: 100, Y: 100},
				Size:     WidgetSize{Width: 200, Height: 150},
			},
			config:    PlacementConfig{OffsetX: 20, OffsetY: 0},
			expectedX: 320, // 100 + 200 + 20
			expectedY: 100, // 100 + 0
		},
		{
			name: "large widget",
			widget: SimpleWidget{
				Location: WidgetLocation{X: 0, Y: 0},
				Size:     WidgetSize{Width: 500, Height: 400},
			},
			config:    PlacementConfig{OffsetX: 10, OffsetY: 10},
			expectedX: 510, // 0 + 500 + 10
			expectedY: 10,  // 0 + 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := CalculatePlacementWithSize(tt.widget, tt.config)

			if x != tt.expectedX {
				t.Errorf("CalculatePlacementWithSize() x = %v, want %v", x, tt.expectedX)
			}
			if y != tt.expectedY {
				t.Errorf("CalculatePlacementWithSize() y = %v, want %v", y, tt.expectedY)
			}
		})
	}
}

// TestCalculateCenteredPlacement tests centered placement below parent.
func TestCalculateCenteredPlacement(t *testing.T) {
	tests := []struct {
		name      string
		widget    SimpleWidget
		newWidth  float64
		expectedX float64
		expectedY float64
	}{
		{
			name: "same width as parent",
			widget: SimpleWidget{
				Location: WidgetLocation{X: 100, Y: 100},
				Size:     WidgetSize{Width: 200, Height: 150},
			},
			newWidth:  200,
			expectedX: 100, // 100 + (200-200)/2 = 100
			expectedY: 300, // 100 + 150 + 50
		},
		{
			name: "smaller new widget",
			widget: SimpleWidget{
				Location: WidgetLocation{X: 100, Y: 100},
				Size:     WidgetSize{Width: 200, Height: 150},
			},
			newWidth:  100,
			expectedX: 150, // 100 + (200-100)/2 = 150
			expectedY: 300, // 100 + 150 + 50
		},
		{
			name: "larger new widget",
			widget: SimpleWidget{
				Location: WidgetLocation{X: 100, Y: 100},
				Size:     WidgetSize{Width: 200, Height: 150},
			},
			newWidth:  300,
			expectedX: 50,  // 100 + (200-300)/2 = 50
			expectedY: 300, // 100 + 150 + 50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := CalculateCenteredPlacement(tt.widget, tt.newWidth)

			if x != tt.expectedX {
				t.Errorf("CalculateCenteredPlacement() x = %v, want %v", x, tt.expectedX)
			}
			if y != tt.expectedY {
				t.Errorf("CalculateCenteredPlacement() y = %v, want %v", y, tt.expectedY)
			}
		})
	}
}

// TestDefaultPlacementConfig verifies the default configuration values.
func TestDefaultPlacementConfig(t *testing.T) {
	config := DefaultPlacementConfig()

	if config.OffsetX != DefaultOffsetX {
		t.Errorf("DefaultPlacementConfig().OffsetX = %v, want %v", config.OffsetX, DefaultOffsetX)
	}
	if config.OffsetY != DefaultOffsetY {
		t.Errorf("DefaultPlacementConfig().OffsetY = %v, want %v", config.OffsetY, DefaultOffsetY)
	}
}
