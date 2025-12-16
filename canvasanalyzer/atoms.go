// Package canvasanalyzer provides canvas analysis functionality for CanvusLocalLLM.
//
// This package extracts canvas analysis functionality from handlers.go,
// providing a clean interface for fetching canvas widgets and generating
// AI-powered analysis of canvas content.
//
// Architecture (Atomic Design):
//   - atoms.go: Pure utility functions (filtering, formatting)
//   - fetcher.go: Fetcher molecule for retrieving widgets with retry logic
//   - processor.go: Processor molecule for AI-powered analysis generation
//   - analyzer.go: Analyzer organism that orchestrates the complete analysis pipeline
package canvasanalyzer

import (
	"encoding/json"
	"strings"
)

// Widget represents a canvas widget with common properties.
// This provides type-safe access to widget data from the Canvus API.
type Widget map[string]interface{}

// GetID returns the widget's unique identifier.
func (w Widget) GetID() string {
	if id, ok := w["id"].(string); ok {
		return id
	}
	return ""
}

// GetType returns the widget's type (note, image, pdf, etc.).
func (w Widget) GetType() string {
	if t, ok := w["type"].(string); ok {
		return t
	}
	return ""
}

// GetTitle returns the widget's title if present.
func (w Widget) GetTitle() string {
	if title, ok := w["title"].(string); ok {
		return title
	}
	return ""
}

// GetText returns the widget's text content if present.
func (w Widget) GetText() string {
	if text, ok := w["text"].(string); ok {
		return text
	}
	return ""
}

// FilterWidgets returns a new slice excluding widgets with any of the specified IDs.
//
// Example:
//
//	filtered := FilterWidgets(widgets, "trigger-id-1", "trigger-id-2")
func FilterWidgets(widgets []Widget, excludeIDs ...string) []Widget {
	if len(excludeIDs) == 0 {
		return widgets
	}

	excludeMap := make(map[string]bool, len(excludeIDs))
	for _, id := range excludeIDs {
		excludeMap[id] = true
	}

	result := make([]Widget, 0, len(widgets))
	for _, w := range widgets {
		if !excludeMap[w.GetID()] {
			result = append(result, w)
		}
	}
	return result
}

// FilterWidgetsByType returns widgets matching any of the specified types.
//
// Example:
//
//	notes := FilterWidgetsByType(widgets, "note")
//	media := FilterWidgetsByType(widgets, "image", "pdf", "video")
func FilterWidgetsByType(widgets []Widget, types ...string) []Widget {
	if len(types) == 0 {
		return widgets
	}

	typeMap := make(map[string]bool, len(types))
	for _, t := range types {
		typeMap[strings.ToLower(t)] = true
	}

	result := make([]Widget, 0)
	for _, w := range widgets {
		if typeMap[strings.ToLower(w.GetType())] {
			result = append(result, w)
		}
	}
	return result
}

// WidgetsToJSON marshals widgets to JSON for AI processing.
// Returns an error if marshaling fails.
func WidgetsToJSON(widgets []Widget) (string, error) {
	data, err := json.Marshal(widgets)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// CountWidgetsByType returns a map of widget type to count.
//
// Example:
//
//	counts := CountWidgetsByType(widgets)
//	// counts["note"] = 5, counts["image"] = 3
func CountWidgetsByType(widgets []Widget) map[string]int {
	counts := make(map[string]int)
	for _, w := range widgets {
		t := strings.ToLower(w.GetType())
		if t != "" {
			counts[t]++
		}
	}
	return counts
}

// SummarizeWidgets returns a human-readable summary of widget types and counts.
//
// Example:
//
//	summary := SummarizeWidgets(widgets)
//	// "5 notes, 3 images, 2 PDFs"
func SummarizeWidgets(widgets []Widget) string {
	counts := CountWidgetsByType(widgets)
	if len(counts) == 0 {
		return "no widgets"
	}

	parts := make([]string, 0, len(counts))
	for t, count := range counts {
		plural := "s"
		if count == 1 {
			plural = ""
		}
		parts = append(parts, strings.ToLower(t)+plural)
	}

	// Build summary string
	var sb strings.Builder
	for i, part := range parts {
		if i > 0 {
			if i == len(parts)-1 {
				sb.WriteString(" and ")
			} else {
				sb.WriteString(", ")
			}
		}
		// Find the count for this part
		for t, count := range counts {
			if strings.HasPrefix(strings.ToLower(part), strings.ToLower(t)) {
				sb.WriteString(strings.ToLower(part))
				sb.WriteString(" (")
				sb.WriteString(string(rune('0' + count%10))) // Simple int to string
				sb.WriteString(")")
				break
			}
		}
	}

	// Simpler approach: just list counts
	result := make([]string, 0, len(counts))
	for t, count := range counts {
		plural := "s"
		if count == 1 {
			plural = ""
		}
		result = append(result, formatCount(count)+" "+t+plural)
	}

	return strings.Join(result, ", ")
}

// formatCount converts an integer to a string.
func formatCount(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + formatCount(-n)
	}

	// Build digits in reverse
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	return string(digits)
}

// ExtractWidgetContent extracts meaningful content from a widget for analysis.
// Returns title and text content combined, or empty string if no content.
func ExtractWidgetContent(w Widget) string {
	var parts []string

	if title := w.GetTitle(); title != "" {
		parts = append(parts, title)
	}
	if text := w.GetText(); text != "" {
		parts = append(parts, text)
	}

	return strings.Join(parts, ": ")
}
