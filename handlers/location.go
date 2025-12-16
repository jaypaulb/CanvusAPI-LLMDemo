// Package handlers provides widget geometry calculation atoms.
package handlers

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// NoteSize represents the dimensions of a note widget.
type NoteSize struct {
	Width  float64
	Height float64
}

// Location represents a 2D coordinate.
type Location struct {
	X float64
	Y float64
}

// Default sizing constants for note content fitting.
const (
	DefaultTargetWidth  = 830.0 // Target width that works well for notes
	DefaultCharsPerLine = 100.0 // Approximate characters that fit in target width
	DefaultLinesPerUnit = 40.0  // Lines that fit in 1200 height units
	DefaultBaseScale    = 0.37  // Base scale for full-size notes
	DefaultMinWidth     = 300.0 // Minimum width for very short content
	DefaultHeightUnit   = 1200.0
	DefaultMaxScale     = 1.0
	ShortContentTokens  = 150.0 // Below this, use original size
)

// CalculateNoteSize computes the optimal note size based on content.
// For short content (< 150 tokens), returns the original size unchanged.
// For longer content, calculates width and height based on content structure.
//
// This is a pure function (atom) with no external dependencies.
//
// Example:
//
//	size, scale := handlers.CalculateNoteSize(content, origWidth, origHeight, origScale)
func CalculateNoteSize(content string, originalWidth, originalHeight, originalScale float64) (NoteSize, float64) {
	// Calculate content length in tokens (rough approximation: 1 token = 4 characters)
	contentTokens := float64(len(content)) / 4.0

	// For short content, use original size
	if contentTokens < ShortContentTokens {
		return NoteSize{Width: originalWidth, Height: originalHeight}, originalScale
	}

	// Calculate content requirements
	contentLines := float64(strings.Count(content, "\n") + 1)
	maxLineLength := 0.0
	totalChars := 0.0

	// Calculate max and average line lengths
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		lineLen := float64(len(line))
		if lineLen > maxLineLength {
			maxLineLength = lineLen
		}
		totalChars += lineLen
	}
	averageLineLength := totalChars / contentLines

	// Determine if text is formatted (narrow lines with many breaks)
	isFormattedText := averageLineLength < (DefaultCharsPerLine*0.5) && contentLines > 5

	// Calculate width
	var width float64
	if isFormattedText {
		width = math.Max(DefaultMinWidth, (maxLineLength/DefaultCharsPerLine)*DefaultTargetWidth)
	} else {
		width = DefaultTargetWidth
	}

	// Calculate height
	totalLines := contentLines
	if !isFormattedText {
		totalLines += (maxLineLength / DefaultCharsPerLine)
	}
	height := (totalLines / DefaultLinesPerUnit) * DefaultHeightUnit

	// Calculate scale based on content ratio
	contentRatio := math.Min(1.0, math.Max(width/DefaultTargetWidth, height/DefaultHeightUnit))
	scale := DefaultBaseScale * (1.0 + (1.0 - contentRatio))
	scale = math.Min(DefaultMaxScale, scale*2)

	return NoteSize{Width: width, Height: height}, scale
}

// CalculateOffsetLocation computes a new location offset from the original.
// Useful for placing response notes near trigger notes.
//
// Example:
//
//	newLoc := handlers.CalculateOffsetLocation(origX, origY, width, height, 0.8, 0.8)
func CalculateOffsetLocation(x, y, width, height, offsetXRatio, offsetYRatio float64) Location {
	return Location{
		X: x + (width * offsetXRatio),
		Y: y + (height * offsetYRatio),
	}
}

// AdjustBackgroundColorOpacity modifies a hex color's opacity.
// Accepts colors in #RRGGBB or #RRGGBBAA format.
// Returns the color with adjusted opacity.
//
// Example:
//
//	color := handlers.AdjustBackgroundColorOpacity("#FF0000", 0.75) // "#FF0000BF"
func AdjustBackgroundColorOpacity(color string, opacityFactor float64) string {
	if color == "" {
		return "#FFFFFFBF" // Default white with 75% opacity
	}

	if len(color) == 9 { // #RRGGBBAA format
		baseColor := color[:7]
		alpha, err := strconv.ParseInt(color[7:], 16, 0)
		if err != nil {
			return color
		}
		newAlpha := int(float64(alpha) * opacityFactor)
		if newAlpha > 255 {
			newAlpha = 255
		}
		return fmt.Sprintf("%s%02X", baseColor, newAlpha)
	}

	if len(color) == 7 { // #RRGGBB format
		alphaHex := fmt.Sprintf("%02X", int(255*opacityFactor))
		return color + alphaHex
	}

	return color // Keep original if format unknown
}

// ReduceBackgroundOpacity reduces the alpha value of a color by ~15%.
// This is commonly used to make response notes slightly transparent.
//
// Example:
//
//	newColor := handlers.ReduceBackgroundOpacity("#FF0000FF") // Reduces alpha by ~15%
func ReduceBackgroundOpacity(color string) string {
	if color == "" {
		return "#FFFFFFBF" // Default white with 75% opacity
	}

	if len(color) == 9 { // #RRGGBBAA format
		baseColor := color[:7]
		alpha, err := strconv.ParseInt(color[7:], 16, 0)
		if err != nil {
			return color
		}
		// Reduce by ~15% (divide by 1.15)
		newAlpha := int(float64(alpha) / 1.15)
		if newAlpha > 255 {
			newAlpha = 255
		}
		return fmt.Sprintf("%s%02X", baseColor, newAlpha)
	}

	if len(color) == 7 { // #RRGGBB format
		return color + "BF" // Add 75% opacity
	}

	return color // Keep original if format unknown
}

// CalculateDepthOffset returns a new depth value offset from the original.
// Used to place new notes above or below existing ones in z-order.
//
// Example:
//
//	newDepth := handlers.CalculateDepthOffset(originalDepth, 200) // 200 units above
func CalculateDepthOffset(originalDepth, offset float64) float64 {
	return originalDepth + offset
}

// ExtractLocation extracts x, y coordinates from a location map.
// Returns Location{0, 0} if the map is nil or missing required fields.
//
// Example:
//
//	loc := handlers.ExtractLocation(update["location"].(map[string]interface{}))
func ExtractLocation(locMap map[string]interface{}) Location {
	if locMap == nil {
		return Location{X: 0, Y: 0}
	}
	x, okX := locMap["x"].(float64)
	y, okY := locMap["y"].(float64)
	if !okX || !okY {
		return Location{X: 0, Y: 0}
	}
	return Location{X: x, Y: y}
}

// ExtractSize extracts width, height from a size map.
// Returns NoteSize{0, 0} if the map is nil or missing required fields.
//
// Example:
//
//	size := handlers.ExtractSize(update["size"].(map[string]interface{}))
func ExtractSize(sizeMap map[string]interface{}) NoteSize {
	if sizeMap == nil {
		return NoteSize{Width: 0, Height: 0}
	}
	width, okW := sizeMap["width"].(float64)
	height, okH := sizeMap["height"].(float64)
	if !okW || !okH {
		return NoteSize{Width: 0, Height: 0}
	}
	return NoteSize{Width: width, Height: height}
}

// LocationToMap converts a Location to a map for API payloads.
//
// Example:
//
//	payload["location"] = handlers.LocationToMap(loc)
func LocationToMap(loc Location) map[string]float64 {
	return map[string]float64{
		"x": loc.X,
		"y": loc.Y,
	}
}

// SizeToMap converts a NoteSize to a map for API payloads.
//
// Example:
//
//	payload["size"] = handlers.SizeToMap(size)
func SizeToMap(size NoteSize) map[string]interface{} {
	return map[string]interface{}{
		"width":  size.Width,
		"height": size.Height,
	}
}

// AddLocations adds two locations together (vector addition).
//
// Example:
//
//	absoluteLoc := handlers.AddLocations(parentLoc, relativeLoc)
func AddLocations(a, b Location) Location {
	return Location{
		X: a.X + b.X,
		Y: a.Y + b.Y,
	}
}
