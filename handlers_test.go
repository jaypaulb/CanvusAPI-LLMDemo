package main

import (
	"testing"

	"go_backend/handlers"
	"go_backend/logging"
)

// TestGenerateCorrelationID tests the correlation ID generation
func TestGenerateCorrelationID(t *testing.T) {
	id1 := generateCorrelationID()
	id2 := generateCorrelationID()

	// Check that IDs are 8 characters
	if len(id1) != 8 {
		t.Errorf("Expected correlation ID length 8, got %d", len(id1))
	}

	// Check that two IDs are different (uniqueness)
	if id1 == id2 {
		t.Errorf("Expected unique correlation IDs, got identical: %s", id1)
	}
}

// TestTruncateText tests the text truncation helper
func TestTruncateText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		length   int
		expected string
	}{
		{
			name:     "short text stays unchanged",
			input:    "hello",
			length:   10,
			expected: "hello",
		},
		{
			name:     "long text gets truncated",
			input:    "hello world this is a long string",
			length:   10,
			expected: "hello worl",
		},
		{
			name:     "exact length stays unchanged",
			input:    "hello",
			length:   5,
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			length:   5,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateText(tt.input, tt.length)
			if result != tt.expected {
				t.Errorf("truncateText(%q, %d) = %q, want %q", tt.input, tt.length, result, tt.expected)
			}
		})
	}
}

// TestValidateUpdate tests the update validation
func TestValidateUpdate(t *testing.T) {
	tests := []struct {
		name    string
		update  Update
		wantErr bool
	}{
		{
			name: "valid update",
			update: Update{
				"id":          "test-id",
				"widget_type": "Note",
				"location":    map[string]interface{}{"x": 100.0, "y": 200.0},
				"size":        map[string]interface{}{"width": 300.0, "height": 400.0},
			},
			wantErr: false,
		},
		{
			name: "missing id",
			update: Update{
				"widget_type": "Note",
				"location":    map[string]interface{}{"x": 100.0, "y": 200.0},
				"size":        map[string]interface{}{"width": 300.0, "height": 400.0},
			},
			wantErr: true,
		},
		{
			name: "empty id",
			update: Update{
				"id":          "",
				"widget_type": "Note",
				"location":    map[string]interface{}{"x": 100.0, "y": 200.0},
				"size":        map[string]interface{}{"width": 300.0, "height": 400.0},
			},
			wantErr: true,
		},
		{
			name: "missing widget_type",
			update: Update{
				"id":       "test-id",
				"location": map[string]interface{}{"x": 100.0, "y": 200.0},
				"size":     map[string]interface{}{"width": 300.0, "height": 400.0},
			},
			wantErr: true,
		},
		{
			name: "missing location",
			update: Update{
				"id":          "test-id",
				"widget_type": "Note",
				"size":        map[string]interface{}{"width": 300.0, "height": 400.0},
			},
			wantErr: true,
		},
		{
			name: "missing size",
			update: Update{
				"id":          "test-id",
				"widget_type": "Note",
				"location":    map[string]interface{}{"x": 100.0, "y": 200.0},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handlers.ValidateUpdate(tt.update)
			if (err != nil) != tt.wantErr {
				t.Errorf("handlers.ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestEstimateTokenCount tests the token estimation
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimateTokenCount(tt.input)
			if result != tt.expected {
				t.Errorf("estimateTokenCount(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSplitIntoChunks tests the chunk splitting
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := splitIntoChunks(tt.text, tt.maxChunkSize)
			if len(chunks) < tt.minChunks || len(chunks) > tt.maxChunks {
				t.Errorf("splitIntoChunks() returned %d chunks, expected between %d and %d", len(chunks), tt.minChunks, tt.maxChunks)
			}
		})
	}
}

// TestIsAzureOpenAIEndpoint tests Azure endpoint detection
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAzureOpenAIEndpoint(tt.endpoint)
			if result != tt.expected {
				t.Errorf("isAzureOpenAIEndpoint(%q) = %v, want %v", tt.endpoint, result, tt.expected)
			}
		})
	}
}

// TestLoggerWithContext tests that logger context is properly applied
func TestLoggerWithContext(t *testing.T) {
	// Create a test logger (using development mode which writes to console)
	logger, err := logging.NewLogger(true, "/tmp/test_handlers.log")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Test that With() returns a new logger with context
	correlationID := generateCorrelationID()
	contextLogger := logger.With(
		// zap fields would be added here in real usage
	)

	if contextLogger == nil {
		t.Error("Expected non-nil logger from With()")
	}

	// Test that logger methods don't panic
	logger.Info("test info message")
	logger.Debug("test debug message")
	logger.Warn("test warn message")

	t.Logf("Correlation ID generated: %s", correlationID)
}
