// Package handlers provides request handling utilities including JSON parsing atoms.
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// JSON parsing errors
var (
	// ErrNoJSONFound is returned when no JSON object is found in the text.
	ErrNoJSONFound = errors.New("no JSON object found in text")
	// ErrInvalidJSON is returned when JSON parsing fails.
	ErrInvalidJSON = errors.New("invalid JSON")
	// ErrMissingContentField is returned when the required "content" field is missing.
	ErrMissingContentField = errors.New("missing 'content' field")
	// ErrMissingTypeField is returned when the required "type" field is missing.
	ErrMissingTypeField = errors.New("missing 'type' field")
)

// AIResponse represents the expected structure of an AI response.
// It contains a type (text or image) and content fields.
type AIResponse struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// ExtractJSONFromText extracts the first JSON object from a text string.
// It finds the first '{' and last '}' and extracts the text between them.
// Returns the extracted JSON string or an error if no valid JSON boundaries are found.
//
// This is a pure function (atom) with no external dependencies.
//
// Example:
//
//	json, err := handlers.ExtractJSONFromText("Some text {\"key\": \"value\"} more text")
//	if err != nil {
//	    log.Printf("No JSON found: %v", err)
//	    return
//	}
//	// json == `{"key": "value"}`
func ExtractJSONFromText(text string) (string, error) {
	startIdx := strings.Index(text, "{")
	endIdx := strings.LastIndex(text, "}")

	if startIdx == -1 || endIdx == -1 || startIdx > endIdx {
		return "", ErrNoJSONFound
	}

	return text[startIdx : endIdx+1], nil
}

// ParseJSONToMap parses a JSON string into a map[string]interface{}.
// Returns the parsed map or an error if parsing fails.
//
// This is a pure function (atom) with no external dependencies.
//
// Example:
//
//	data, err := handlers.ParseJSONToMap(`{"type": "text", "content": "hello"}`)
//	if err != nil {
//	    log.Printf("Parse error: %v", err)
//	    return
//	}
//	// data["type"] == "text"
func ParseJSONToMap(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}
	return result, nil
}

// ValidateAIResponseFields checks that a parsed JSON map has the required
// "type" and "content" fields as strings.
// Returns nil if valid, or an error describing the missing/invalid field.
//
// This is a pure function (atom) with no external dependencies.
//
// Example:
//
//	err := handlers.ValidateAIResponseFields(data)
//	if err != nil {
//	    log.Printf("Invalid AI response: %v", err)
//	    return
//	}
func ValidateAIResponseFields(data map[string]interface{}) error {
	if _, ok := data["type"].(string); !ok {
		return ErrMissingTypeField
	}
	if _, ok := data["content"].(string); !ok {
		return ErrMissingContentField
	}
	return nil
}

// ValidateContentField checks that a parsed JSON map has the required
// "content" field as a string. Use this for responses that don't require "type".
// Returns nil if valid, or an error if the field is missing/invalid.
//
// This is a pure function (atom) with no external dependencies.
//
// Example:
//
//	err := handlers.ValidateContentField(data)
//	if err != nil {
//	    log.Printf("Missing content: %v", err)
//	    return
//	}
func ValidateContentField(data map[string]interface{}) error {
	if _, ok := data["content"].(string); !ok {
		return ErrMissingContentField
	}
	return nil
}

// ExtractAndParseAIResponse extracts JSON from text and parses it as an AI response.
// This is a composition of ExtractJSONFromText, ParseJSONToMap, and ValidateAIResponseFields.
// Returns the AIResponse struct or an error at any stage.
//
// This is a pure function (atom) with no external dependencies.
//
// Example:
//
//	resp, err := handlers.ExtractAndParseAIResponse(rawText)
//	if err != nil {
//	    log.Printf("Failed to parse AI response: %v", err)
//	    return
//	}
//	if resp.Type == "image" {
//	    generateImage(resp.Content)
//	}
func ExtractAndParseAIResponse(text string) (*AIResponse, error) {
	jsonStr, err := ExtractJSONFromText(text)
	if err != nil {
		return nil, err
	}

	data, err := ParseJSONToMap(jsonStr)
	if err != nil {
		return nil, err
	}

	if err := ValidateAIResponseFields(data); err != nil {
		return nil, err
	}

	return &AIResponse{
		Type:    data["type"].(string),
		Content: data["content"].(string),
	}, nil
}

// ExtractAndParseContent extracts JSON from text and returns only the content field.
// Use this for responses that only need the content value, not the type.
// Returns the content string or an error.
//
// This is a pure function (atom) with no external dependencies.
//
// Example:
//
//	content, err := handlers.ExtractAndParseContent(rawText)
//	if err != nil {
//	    log.Printf("Failed to parse content: %v", err)
//	    return
//	}
//	displayContent(content)
func ExtractAndParseContent(text string) (string, error) {
	jsonStr, err := ExtractJSONFromText(text)
	if err != nil {
		return "", err
	}

	data, err := ParseJSONToMap(jsonStr)
	if err != nil {
		return "", err
	}

	if err := ValidateContentField(data); err != nil {
		return "", err
	}

	return data["content"].(string), nil
}

// GetStringFieldFromJSON parses JSON and returns a specific string field.
// Returns an error if the JSON is invalid or the field doesn't exist/isn't a string.
//
// This is a pure function (atom) with no external dependencies.
//
// Example:
//
//	value, err := handlers.GetStringFieldFromJSON(`{"name": "test"}`, "name")
//	if err != nil {
//	    log.Printf("Failed to get field: %v", err)
//	    return
//	}
//	// value == "test"
func GetStringFieldFromJSON(jsonStr string, fieldName string) (string, error) {
	data, err := ParseJSONToMap(jsonStr)
	if err != nil {
		return "", err
	}

	value, ok := data[fieldName].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid '%s' field", fieldName)
	}

	return value, nil
}

// NormalizeNewlines converts escaped newlines (\n as literal backslash-n) to actual newlines.
// This is commonly needed when parsing AI responses where newlines are escaped.
//
// This is a pure function (atom) with no external dependencies.
//
// Example:
//
//	text := handlers.NormalizeNewlines("Hello\\nWorld")
//	// text == "Hello\nWorld"
func NormalizeNewlines(text string) string {
	return strings.ReplaceAll(text, "\\n", "\n")
}
