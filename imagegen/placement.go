// Package imagegen provides image generation utilities for the Canvus canvas.
package imagegen

// DefaultOffsetX is the horizontal offset from parent widget for image placement.
// Places image to the right of the prompt widget.
const DefaultOffsetX = 300.0

// DefaultOffsetY is the vertical offset from parent widget for image placement.
// Slightly below to create visual hierarchy.
const DefaultOffsetY = 50.0

// WidgetLocation represents a 2D position on the canvas.
type WidgetLocation struct {
	X float64
	Y float64
}

// WidgetSize represents dimensions of a widget.
type WidgetSize struct {
	Width  float64
	Height float64
}

// Widget represents the minimal widget data needed for placement calculations.
// This interface allows flexibility in what widget types can be passed.
type Widget interface {
	GetLocation() WidgetLocation
	GetSize() WidgetSize
}

// SimpleWidget is a concrete implementation of Widget for testing and simple use cases.
type SimpleWidget struct {
	Location WidgetLocation
	Size     WidgetSize
}

// GetLocation returns the widget's location.
func (w SimpleWidget) GetLocation() WidgetLocation {
	return w.Location
}

// GetSize returns the widget's size.
func (w SimpleWidget) GetSize() WidgetSize {
	return w.Size
}

// PlacementConfig holds configuration for image placement calculations.
type PlacementConfig struct {
	OffsetX float64
	OffsetY float64
}

// DefaultPlacementConfig returns the default placement configuration.
func DefaultPlacementConfig() PlacementConfig {
	return PlacementConfig{
		OffsetX: DefaultOffsetX,
		OffsetY: DefaultOffsetY,
	}
}

// CalculatePlacement computes the x, y coordinates for placing an image
// relative to a parent widget. By default, places the image to the right
// of the parent widget (x+300, y+50).
//
// This molecule composes:
// - Widget location extraction (atom)
// - Offset addition (atom)
//
// The calculation respects the Canvus coordinate system where widget
// locations are relative to their parent.
func CalculatePlacement(parentWidget Widget) (x, y float64) {
	return CalculatePlacementWithConfig(parentWidget, DefaultPlacementConfig())
}

// CalculatePlacementWithConfig computes placement with custom offsets.
// This allows callers to customize the placement behavior while
// reusing the core coordinate math.
func CalculatePlacementWithConfig(parentWidget Widget, config PlacementConfig) (x, y float64) {
	loc := parentWidget.GetLocation()
	return addOffset(loc.X, loc.Y, config.OffsetX, config.OffsetY)
}

// CalculatePlacementWithSize computes placement accounting for parent widget size.
// Places the image to the right edge of the parent plus offset.
// Useful when you want the image adjacent to the parent without overlap.
func CalculatePlacementWithSize(parentWidget Widget, config PlacementConfig) (x, y float64) {
	loc := parentWidget.GetLocation()
	size := parentWidget.GetSize()

	// Position at right edge of parent, plus configured offset
	x = loc.X + size.Width + config.OffsetX
	y = loc.Y + config.OffsetY
	return x, y
}

// addOffset is an atom-level pure function for offset calculation.
// It adds offset values to base coordinates.
func addOffset(baseX, baseY, offsetX, offsetY float64) (float64, float64) {
	return baseX + offsetX, baseY + offsetY
}

// CalculateCenteredPlacement places the new widget centered below the parent.
// Useful for creating visual hierarchies where responses appear below prompts.
func CalculateCenteredPlacement(parentWidget Widget, newWidth float64) (x, y float64) {
	loc := parentWidget.GetLocation()
	size := parentWidget.GetSize()

	// Center horizontally relative to parent
	x = loc.X + (size.Width-newWidth)/2
	// Place below parent with standard offset
	y = loc.Y + size.Height + DefaultOffsetY

	return x, y
}
