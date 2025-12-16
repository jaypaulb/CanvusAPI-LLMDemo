// Package handlers provides request handling utilities including validation atoms.
package handlers

import (
	"errors"
	"fmt"
)

// Validation errors
var (
	// ErrMissingID is returned when an update is missing a required ID field.
	ErrMissingID = errors.New("missing or empty ID")
	// ErrMissingType is returned when an update is missing a required widget type.
	ErrMissingType = errors.New("missing widget type")
	// ErrMissingLocation is returned when an update is missing location data.
	ErrMissingLocation = errors.New("missing location")
	// ErrMissingSize is returned when an update is missing size data.
	ErrMissingSize = errors.New("missing size")
)

// Update is a type alias for widget update data.
// Using map[string]interface{} to match the JSON structure from Canvus API.
type Update = map[string]interface{}

// ValidateUpdate checks if an update contains the required fields for processing.
// Required fields: id (string), widget_type (string), location (map), size (map).
//
// This is a pure function (atom) with no external dependencies.
//
// Example:
//
//	err := handlers.ValidateUpdate(update)
//	if err != nil {
//	    log.Printf("Invalid update: %v", err)
//	    return
//	}
func ValidateUpdate(update Update) error {
	id, hasID := update["id"].(string)
	if !hasID || id == "" {
		return ErrMissingID
	}
	widgetType, hasType := update["widget_type"].(string)
	if !hasType || widgetType == "" {
		return ErrMissingType
	}
	if _, hasLocation := update["location"].(map[string]interface{}); !hasLocation {
		return ErrMissingLocation
	}
	if _, hasSize := update["size"].(map[string]interface{}); !hasSize {
		return ErrMissingSize
	}
	return nil
}

// ValidateUpdateField checks if a specific field exists and is non-empty in the update.
// Returns the field value if valid, or an error if the field is missing or empty.
//
// This is useful for checking individual fields without full validation.
//
// Example:
//
//	text, err := handlers.ValidateUpdateField[string](update, "text")
//	if err != nil {
//	    log.Printf("No text field: %v", err)
//	}
func ValidateUpdateField[T any](update Update, fieldName string) (T, error) {
	var zero T
	value, exists := update[fieldName]
	if !exists {
		return zero, fmt.Errorf("missing field: %s", fieldName)
	}
	typed, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("invalid type for field %s: expected %T, got %T", fieldName, zero, value)
	}
	return typed, nil
}

// ValidateNonEmptyString checks that a string field exists and is non-empty.
//
// Example:
//
//	text, err := handlers.ValidateNonEmptyString(update, "text")
func ValidateNonEmptyString(update Update, fieldName string) (string, error) {
	value, err := ValidateUpdateField[string](update, fieldName)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("empty field: %s", fieldName)
	}
	return value, nil
}

// HasField checks if a field exists in the update (may be nil or empty).
//
// Example:
//
//	if handlers.HasField(update, "parent_id") {
//	    // Process with parent
//	}
func HasField(update Update, fieldName string) bool {
	_, exists := update[fieldName]
	return exists
}

// GetStringField returns a string field value with a default if not found.
// Unlike ValidateNonEmptyString, this does not treat empty strings as errors.
//
// Example:
//
//	text := handlers.GetStringField(update, "text", "")
func GetStringField(update Update, fieldName string, defaultValue string) string {
	if value, ok := update[fieldName].(string); ok {
		return value
	}
	return defaultValue
}

// GetMapField returns a map field value or nil if not found/wrong type.
//
// Example:
//
//	location := handlers.GetMapField(update, "location")
//	if location != nil {
//	    x := location["x"].(float64)
//	}
func GetMapField(update Update, fieldName string) map[string]interface{} {
	if value, ok := update[fieldName].(map[string]interface{}); ok {
		return value
	}
	return nil
}
