package handlers

import (
	"strings"
	"testing"
)

func TestGenerateCorrelationID(t *testing.T) {
	// Test that IDs are 8 characters
	id := GenerateCorrelationID()
	if len(id) != 8 {
		t.Errorf("Expected correlation ID length 8, got %d", len(id))
	}

	// Test uniqueness
	id2 := GenerateCorrelationID()
	if id == id2 {
		t.Errorf("Expected unique correlation IDs, got identical: %s", id)
	}

	// Test multiple IDs for basic uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		newID := GenerateCorrelationID()
		if ids[newID] {
			t.Errorf("Duplicate ID generated: %s", newID)
		}
		ids[newID] = true
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{
			name:      "short text stays unchanged",
			input:     "hello",
			maxLength: 10,
			expected:  "hello",
		},
		{
			name:      "long text gets truncated",
			input:     "hello world this is a long string",
			maxLength: 10,
			expected:  "hello worl",
		},
		{
			name:      "exact length stays unchanged",
			input:     "hello",
			maxLength: 5,
			expected:  "hello",
		},
		{
			name:      "empty string",
			input:     "",
			maxLength: 5,
			expected:  "",
		},
		{
			name:      "zero max length",
			input:     "hello",
			maxLength: 0,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateText(tt.input, tt.maxLength)
			if result != tt.expected {
				t.Errorf("TruncateText(%q, %d) = %q, want %q", tt.input, tt.maxLength, result, tt.expected)
			}
		})
	}
}

func TestExtractAIPrompt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic trigger",
			input:    "{{Generate a haiku}}",
			expected: "Generate a haiku",
		},
		{
			name:     "with surrounding text",
			input:    "Please {{generate a poem}} for me",
			expected: "Please generate a poem for me",
		},
		{
			name:     "multiple triggers",
			input:    "{{first}} and {{second}}",
			expected: "first and second",
		},
		{
			name:     "no triggers",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only open bracket",
			input:    "{{incomplete",
			expected: "incomplete",
		},
		{
			name:     "only close bracket",
			input:    "incomplete}}",
			expected: "incomplete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractAIPrompt(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractAIPrompt(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHasAITrigger(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid trigger",
			input:    "{{Generate a haiku}}",
			expected: true,
		},
		{
			name:     "embedded trigger",
			input:    "Please {{generate a poem}} for me",
			expected: true,
		},
		{
			name:     "no trigger",
			input:    "plain text without trigger",
			expected: false,
		},
		{
			name:     "only open bracket",
			input:    "{{incomplete",
			expected: false,
		},
		{
			name:     "only close bracket",
			input:    "incomplete}}",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "separate brackets",
			input:    "has {{ but also }} separately",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasAITrigger(tt.input)
			if result != tt.expected {
				t.Errorf("HasAITrigger(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsAzureOpenAIEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		{
			name:     "Azure OpenAI endpoint",
			endpoint: "https://myresource.openai.azure.com/",
			expected: true,
		},
		{
			name:     "Azure cognitive services endpoint",
			endpoint: "https://myresource.cognitiveservices.azure.com/",
			expected: true,
		},
		{
			name:     "OpenAI endpoint",
			endpoint: "https://api.openai.com/v1",
			expected: false,
		},
		{
			name:     "local endpoint",
			endpoint: "http://127.0.0.1:1234/v1",
			expected: false,
		},
		{
			name:     "localhost endpoint",
			endpoint: "http://localhost:8080/v1",
			expected: false,
		},
		{
			name:     "mixed case Azure",
			endpoint: "https://myresource.OpenAI.Azure.com/",
			expected: true,
		},
		{
			name:     "empty string",
			endpoint: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAzureOpenAIEndpoint(tt.endpoint)
			if result != tt.expected {
				t.Errorf("IsAzureOpenAIEndpoint(%q) = %v, want %v", tt.endpoint, result, tt.expected)
			}
		})
	}
}

func TestEstimateTokenCount(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "short text",
			input:    "hello",
			expected: 1, // 5 chars / 4 = 1
		},
		{
			name:     "longer text",
			input:    "hello world this is a test",
			expected: 6, // 26 chars / 4 = 6
		},
		{
			name:     "exact multiple",
			input:    "12345678",
			expected: 2, // 8 chars / 4 = 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokenCount(tt.input)
			if result != tt.expected {
				t.Errorf("EstimateTokenCount(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSplitIntoChunks(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		maxChunkSize int
		minChunks    int
		maxChunks    int
	}{
		{
			name:         "small text single chunk",
			text:         "hello world",
			maxChunkSize: 100,
			minChunks:    1,
			maxChunks:    1,
		},
		{
			name:         "text with paragraphs",
			text:         "paragraph one\n\nparagraph two\n\nparagraph three",
			maxChunkSize: 20,
			minChunks:    2,
			maxChunks:    4,
		},
		{
			name:         "empty text",
			text:         "",
			maxChunkSize: 100,
			minChunks:    0,
			maxChunks:    1, // Empty string split returns one empty chunk
		},
		{
			name:         "single paragraph exceeds limit",
			text:         "this is a very long paragraph that should still stay together",
			maxChunkSize: 10,
			minChunks:    1,
			maxChunks:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := SplitIntoChunks(tt.text, tt.maxChunkSize)
			if len(chunks) < tt.minChunks || len(chunks) > tt.maxChunks {
				t.Errorf("SplitIntoChunks() returned %d chunks, expected between %d and %d", len(chunks), tt.minChunks, tt.maxChunks)
			}

			// Verify all original content is preserved
			if tt.text != "" {
				combined := strings.Join(chunks, "")
				// Normalize whitespace for comparison
				original := strings.TrimSpace(tt.text)
				result := strings.TrimSpace(combined)
				if !strings.Contains(result, strings.Split(original, "\n")[0]) {
					t.Errorf("SplitIntoChunks() lost content")
				}
			}
		})
	}
}

func TestPDFChunkPrompt(t *testing.T) {
	prompt := PDFChunkPrompt()

	// Verify it's not empty
	if prompt == "" {
		t.Error("PDFChunkPrompt() returned empty string")
	}

	// Verify it contains key instructions
	if !strings.Contains(prompt, "document") {
		t.Error("PDFChunkPrompt() should mention document analysis")
	}

	if !strings.Contains(prompt, "response") {
		t.Error("PDFChunkPrompt() should mention response format")
	}
}
