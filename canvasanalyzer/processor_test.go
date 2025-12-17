package canvasanalyzer

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDefaultProcessorConfig(t *testing.T) {
	config := DefaultProcessorConfig()

	if config.Model != "gpt-4" {
		t.Errorf("Model = %v, want gpt-4", config.Model)
	}
	if config.SystemPrompt == "" {
		t.Error("SystemPrompt should not be empty")
	}
	if config.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %v, want 4096", config.MaxTokens)
	}
	if config.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", config.Temperature)
	}
	if config.Timeout != 2*time.Minute {
		t.Errorf("Timeout = %v, want 2m", config.Timeout)
	}
}

func TestNewProcessor(t *testing.T) {
	logger := newTestLogger()

	t.Run("with valid config", func(t *testing.T) {
		config := DefaultProcessorConfig()
		processor := NewProcessor(config, nil, logger)
		if processor == nil {
			t.Error("NewProcessor returned nil")
		}
	})

	t.Run("with empty model defaults to gpt-4", func(t *testing.T) {
		config := ProcessorConfig{Model: ""}
		processor := NewProcessor(config, nil, logger)
		if processor.config.Model != "gpt-4" {
			t.Errorf("Model = %v, want gpt-4", processor.config.Model)
		}
	})

	t.Run("with empty system prompt uses default", func(t *testing.T) {
		config := ProcessorConfig{SystemPrompt: ""}
		processor := NewProcessor(config, nil, logger)
		if processor.config.SystemPrompt == "" {
			t.Error("SystemPrompt should not be empty")
		}
	})

	t.Run("with invalid max tokens defaults to 4096", func(t *testing.T) {
		config := ProcessorConfig{MaxTokens: 0}
		processor := NewProcessor(config, nil, logger)
		if processor.config.MaxTokens != 4096 {
			t.Errorf("MaxTokens = %v, want 4096", processor.config.MaxTokens)
		}
	})

	t.Run("with invalid temperature defaults to 0.7", func(t *testing.T) {
		config := ProcessorConfig{Temperature: -1}
		processor := NewProcessor(config, nil, logger)
		if processor.config.Temperature != 0.7 {
			t.Errorf("Temperature = %v, want 0.7", processor.config.Temperature)
		}
	})

	t.Run("with invalid timeout defaults to 2m", func(t *testing.T) {
		config := ProcessorConfig{Timeout: 0}
		processor := NewProcessor(config, nil, logger)
		if processor.config.Timeout != 2*time.Minute {
			t.Errorf("Timeout = %v, want 2m", processor.config.Timeout)
		}
	})
}

func TestProcessor_GetConfig(t *testing.T) {
	config := ProcessorConfig{
		Model:        "gpt-3.5-turbo",
		SystemPrompt: "Custom prompt",
		MaxTokens:    2048,
		Temperature:  0.5,
		Timeout:      1 * time.Minute,
	}
	processor := NewProcessor(config, nil, newTestLogger())
	got := processor.GetConfig()

	if got.Model != "gpt-3.5-turbo" {
		t.Errorf("Model = %v, want gpt-3.5-turbo", got.Model)
	}
	if got.SystemPrompt != "Custom prompt" {
		t.Errorf("SystemPrompt = %v, want Custom prompt", got.SystemPrompt)
	}
	if got.MaxTokens != 2048 {
		t.Errorf("MaxTokens = %v, want 2048", got.MaxTokens)
	}
}

func TestProcessor_SetModel(t *testing.T) {
	processor := NewProcessor(DefaultProcessorConfig(), nil, newTestLogger())
	processor.SetModel("gpt-3.5-turbo")

	if processor.config.Model != "gpt-3.5-turbo" {
		t.Errorf("Model = %v, want gpt-3.5-turbo", processor.config.Model)
	}
}

func TestProcessor_SetSystemPrompt(t *testing.T) {
	processor := NewProcessor(DefaultProcessorConfig(), nil, newTestLogger())
	processor.SetSystemPrompt("New prompt")

	if processor.config.SystemPrompt != "New prompt" {
		t.Errorf("SystemPrompt = %v, want New prompt", processor.config.SystemPrompt)
	}
}

func TestProcessor_SetMaxTokens(t *testing.T) {
	processor := NewProcessor(DefaultProcessorConfig(), nil, newTestLogger())

	t.Run("valid value", func(t *testing.T) {
		processor.SetMaxTokens(8192)
		if processor.config.MaxTokens != 8192 {
			t.Errorf("MaxTokens = %v, want 8192", processor.config.MaxTokens)
		}
	})

	t.Run("invalid value ignored", func(t *testing.T) {
		processor.SetMaxTokens(0)
		if processor.config.MaxTokens != 8192 {
			t.Errorf("MaxTokens changed to %v, should remain 8192", processor.config.MaxTokens)
		}
	})
}

func TestProcessor_SetTemperature(t *testing.T) {
	processor := NewProcessor(DefaultProcessorConfig(), nil, newTestLogger())

	t.Run("valid value", func(t *testing.T) {
		processor.SetTemperature(0.5)
		if processor.config.Temperature != 0.5 {
			t.Errorf("Temperature = %v, want 0.5", processor.config.Temperature)
		}
	})

	t.Run("invalid low value ignored", func(t *testing.T) {
		processor.SetTemperature(-1)
		if processor.config.Temperature != 0.5 {
			t.Errorf("Temperature changed to %v, should remain 0.5", processor.config.Temperature)
		}
	})

	t.Run("invalid high value ignored", func(t *testing.T) {
		processor.SetTemperature(3)
		if processor.config.Temperature != 0.5 {
			t.Errorf("Temperature changed to %v, should remain 0.5", processor.config.Temperature)
		}
	})
}

func TestExtractJSONContent(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
	}{
		{
			name:     "valid JSON with content field",
			response: `{"content": "This is the analysis content"}`,
			want:     "This is the analysis content",
		},
		{
			name:     "JSON with text field",
			response: `{"text": "Analysis text here"}`,
			want:     "Analysis text here",
		},
		{
			name:     "JSON with response field",
			response: `{"response": "Response content"}`,
			want:     "Response content",
		},
		{
			name:     "JSON with analysis field",
			response: `{"analysis": "Analysis here"}`,
			want:     "Analysis here",
		},
		{
			name:     "JSON embedded in text",
			response: `Some text before {"content": "Extracted content"} some text after`,
			want:     "Extracted content",
		},
		{
			name:     "plain text (no JSON)",
			response: "This is plain text analysis",
			want:     "",
		},
		{
			name:     "malformed JSON",
			response: `{"content": "unclosed`,
			want:     "",
		},
		{
			name:     "JSON without content fields",
			response: `{"id": "123", "type": "note"}`,
			want:     "",
		},
		{
			name:     "empty JSON object",
			response: `{}`,
			want:     "",
		},
		{
			name:     "empty string",
			response: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONContent(tt.response)
			if got != tt.want {
				t.Errorf("extractJSONContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text string
		want int
	}{
		{"", 0},
		{"abcd", 1},
		{"hello world", 2},                       // 11 chars / 4 = 2
		{"this is a longer text for testing", 8}, // 34 chars / 4 = 8
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := estimateTokens(tt.text)
			if got != tt.want {
				t.Errorf("estimateTokens(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestProcessor_extractContent(t *testing.T) {
	processor := NewProcessor(DefaultProcessorConfig(), nil, newTestLogger())

	tests := []struct {
		name        string
		rawResponse string
		wantContent string
	}{
		{
			name:        "JSON response with content",
			rawResponse: `{"content": "Extracted from JSON"}`,
			wantContent: "Extracted from JSON",
		},
		{
			name:        "plain text response",
			rawResponse: "This is plain text analysis",
			wantContent: "This is plain text analysis",
		},
		{
			name:        "response with escaped newlines",
			rawResponse: "Line 1\\nLine 2\\nLine 3",
			wantContent: "Line 1\nLine 2\nLine 3",
		},
		{
			name:        "response with leading/trailing whitespace",
			rawResponse: "  Content with spaces  ",
			wantContent: "Content with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processor.extractContent(tt.rawResponse)
			if got != tt.wantContent {
				t.Errorf("extractContent() = %q, want %q", got, tt.wantContent)
			}
		})
	}
}

func TestAnalysisResult(t *testing.T) {
	result := &AnalysisResult{
		Content:          "Test analysis",
		RawResponse:      `{"content": "Test analysis"}`,
		PromptTokens:     100,
		CompletionTokens: 50,
		Duration:         1 * time.Second,
		WidgetCount:      5,
		Model:            "gpt-4",
	}

	if result.Content != "Test analysis" {
		t.Errorf("Content = %v, want Test analysis", result.Content)
	}
	if result.PromptTokens != 100 {
		t.Errorf("PromptTokens = %v, want 100", result.PromptTokens)
	}
	if result.CompletionTokens != 50 {
		t.Errorf("CompletionTokens = %v, want 50", result.CompletionTokens)
	}
	if result.WidgetCount != 5 {
		t.Errorf("WidgetCount = %v, want 5", result.WidgetCount)
	}
	if result.Model != "gpt-4" {
		t.Errorf("Model = %v, want gpt-4", result.Model)
	}
}

// TestProcessor_Analyze_NilClient verifies behavior with nil client.
// Real API tests are in integration_test.go.
func TestProcessor_Analyze_NilClient(t *testing.T) {
	processor := NewProcessor(DefaultProcessorConfig(), nil, newTestLogger())

	widgets := []Widget{
		{"id": "1", "type": "note", "text": "Test note"},
	}

	// This should panic or error gracefully
	// We test that it handles nil client appropriately
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Recovered from panic as expected with nil client: %v", r)
		}
	}()

	_, err := processor.Analyze(context.Background(), widgets)
	if err == nil {
		t.Error("Expected error with nil client")
	}
}

// newTestLogger is defined in fetcher_test.go, but we need it here too
func init() {
	// Logger is shared across test files
	_ = zap.NewNop()
}
