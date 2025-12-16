package main

import (
	"os"
	"testing"

	"go_backend/canvusapi"
	"go_backend/core"
	"go_backend/imagegen"
	"go_backend/logging"
)

// createTestLogger creates a logger for testing that writes to a temp file.
func createTestLogger(t *testing.T) *logging.Logger {
	t.Helper()
	// Create temp file for log output
	tmpFile, err := os.CreateTemp("", "test_*.log")
	if err != nil {
		t.Fatalf("failed to create temp log file: %v", err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	logger, err := logging.NewLogger(true, tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}

// TestParseImagePrompt tests the parseImagePrompt function with various prompt formats.
func TestParseImagePrompt(t *testing.T) {
	logger := createTestLogger(t)
	m := &Monitor{logger: logger}

	tests := []struct {
		name          string
		text          string
		expectPrompt  string
		expectMatched bool
	}{
		{
			name:          "basic image prompt",
			text:          "{{image: a beautiful sunset over mountains}}",
			expectPrompt:  "a beautiful sunset over mountains",
			expectMatched: true,
		},
		{
			name:          "image prompt without space after colon",
			text:          "{{image:a cat sitting on a windowsill}}",
			expectPrompt:  "a cat sitting on a windowsill",
			expectMatched: true,
		},
		{
			name:          "image prompt with extra spaces",
			text:          "{{  image:   a futuristic city  }}",
			expectPrompt:  "a futuristic city",
			expectMatched: true,
		},
		{
			name:          "uppercase IMAGE prefix",
			text:          "{{IMAGE: robot dancing}}",
			expectPrompt:  "robot dancing",
			expectMatched: true,
		},
		{
			name:          "mixed case Image prefix",
			text:          "{{Image: abstract art}}",
			expectPrompt:  "abstract art",
			expectMatched: true,
		},
		{
			name:          "text with prefix before trigger",
			text:          "Please generate: {{image: a dragon}}",
			expectPrompt:  "a dragon",
			expectMatched: true,
		},
		{
			name:          "text with suffix after trigger",
			text:          "{{image: a forest}} - my prompt",
			expectPrompt:  "a forest",
			expectMatched: true,
		},
		{
			name:          "regular text trigger not image",
			text:          "{{explain quantum physics}}",
			expectPrompt:  "",
			expectMatched: false,
		},
		{
			name:          "no trigger at all",
			text:          "just plain text",
			expectPrompt:  "",
			expectMatched: false,
		},
		{
			name:          "empty text",
			text:          "",
			expectPrompt:  "",
			expectMatched: false,
		},
		{
			name:          "unclosed trigger",
			text:          "{{image: unclosed",
			expectPrompt:  "",
			expectMatched: false,
		},
		{
			name:          "empty image prompt",
			text:          "{{image:}}",
			expectPrompt:  "",
			expectMatched: false,
		},
		{
			name:          "image prompt with only whitespace",
			text:          "{{image:   }}",
			expectPrompt:  "",
			expectMatched: false,
		},
		{
			name:          "nested braces",
			text:          "{{image: {special} prompt}}",
			expectPrompt:  "{special} prompt",
			expectMatched: true,
		},
		{
			name:          "multiline prompt",
			text:          "{{image: a landscape\nwith mountains}}",
			expectPrompt:  "a landscape\nwith mountains",
			expectMatched: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := Update{"text": tt.text}
			prompt, matched := m.parseImagePrompt(update)

			if matched != tt.expectMatched {
				t.Errorf("parseImagePrompt() matched = %v, want %v", matched, tt.expectMatched)
			}
			if prompt != tt.expectPrompt {
				t.Errorf("parseImagePrompt() prompt = %q, want %q", prompt, tt.expectPrompt)
			}
		})
	}
}

// TestParseImagePromptMissingText tests parseImagePrompt with missing text field.
func TestParseImagePromptMissingText(t *testing.T) {
	logger := createTestLogger(t)
	m := &Monitor{logger: logger}

	tests := []struct {
		name   string
		update Update
	}{
		{
			name:   "nil text",
			update: Update{"text": nil},
		},
		{
			name:   "missing text field",
			update: Update{"other_field": "value"},
		},
		{
			name:   "wrong type for text",
			update: Update{"text": 123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, matched := m.parseImagePrompt(tt.update)
			if matched {
				t.Errorf("parseImagePrompt() should not match for %s", tt.name)
			}
			if prompt != "" {
				t.Errorf("parseImagePrompt() prompt should be empty for %s, got %q", tt.name, prompt)
			}
		})
	}
}

// TestCreateParentWidget tests the createParentWidget function.
func TestCreateParentWidget(t *testing.T) {
	logger := createTestLogger(t)
	m := &Monitor{logger: logger}

	t.Run("valid update creates parent widget", func(t *testing.T) {
		update := Update{
			"id": "widget-123",
			"location": map[string]interface{}{
				"x": 100.0,
				"y": 200.0,
			},
			"size": map[string]interface{}{
				"width":  300.0,
				"height": 400.0,
			},
			"scale": 0.5,
			"depth": 10.0,
		}

		widget, err := m.createParentWidget(update)
		if err != nil {
			t.Fatalf("createParentWidget() error = %v", err)
		}

		// Type assert to CanvasWidget to check values
		cw, ok := widget.(imagegen.CanvasWidget)
		if !ok {
			t.Fatalf("expected CanvasWidget, got %T", widget)
		}

		if cw.GetID() != "widget-123" {
			t.Errorf("ID = %q, want %q", cw.GetID(), "widget-123")
		}
		if cw.GetLocation().X != 100.0 {
			t.Errorf("Location.X = %v, want %v", cw.GetLocation().X, 100.0)
		}
		if cw.GetLocation().Y != 200.0 {
			t.Errorf("Location.Y = %v, want %v", cw.GetLocation().Y, 200.0)
		}
		if cw.GetSize().Width != 300.0 {
			t.Errorf("Size.Width = %v, want %v", cw.GetSize().Width, 300.0)
		}
		if cw.GetSize().Height != 400.0 {
			t.Errorf("Size.Height = %v, want %v", cw.GetSize().Height, 400.0)
		}
		if cw.GetScale() != 0.5 {
			t.Errorf("Scale = %v, want %v", cw.GetScale(), 0.5)
		}
		if cw.GetDepth() != 10.0 {
			t.Errorf("Depth = %v, want %v", cw.GetDepth(), 10.0)
		}
	})

	t.Run("defaults for missing scale and depth", func(t *testing.T) {
		update := Update{
			"id": "widget-456",
			"location": map[string]interface{}{
				"x": 50.0,
				"y": 60.0,
			},
			"size": map[string]interface{}{
				"width":  100.0,
				"height": 100.0,
			},
		}

		widget, err := m.createParentWidget(update)
		if err != nil {
			t.Fatalf("createParentWidget() error = %v", err)
		}

		cw, ok := widget.(imagegen.CanvasWidget)
		if !ok {
			t.Fatalf("expected CanvasWidget, got %T", widget)
		}

		if cw.GetScale() != 1.0 {
			t.Errorf("Scale = %v, want default %v", cw.GetScale(), 1.0)
		}
		if cw.GetDepth() != 0.0 {
			t.Errorf("Depth = %v, want default %v", cw.GetDepth(), 0.0)
		}
	})

	t.Run("missing id returns error", func(t *testing.T) {
		update := Update{
			"location": map[string]interface{}{"x": 0.0, "y": 0.0},
			"size":     map[string]interface{}{"width": 100.0, "height": 100.0},
		}

		_, err := m.createParentWidget(update)
		if err == nil {
			t.Error("createParentWidget() should return error for missing id")
		}
	})

	t.Run("missing location returns error", func(t *testing.T) {
		update := Update{
			"id":   "widget-789",
			"size": map[string]interface{}{"width": 100.0, "height": 100.0},
		}

		_, err := m.createParentWidget(update)
		if err == nil {
			t.Error("createParentWidget() should return error for missing location")
		}
	})

	t.Run("missing size returns error", func(t *testing.T) {
		update := Update{
			"id":       "widget-abc",
			"location": map[string]interface{}{"x": 0.0, "y": 0.0},
		}

		_, err := m.createParentWidget(update)
		if err == nil {
			t.Error("createParentWidget() should return error for missing size")
		}
	})
}

// TestTruncatePrompt tests the truncatePrompt helper function.
func TestTruncatePrompt(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxLen   int
		expected string
	}{
		{
			name:     "short text unchanged",
			text:     "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length unchanged",
			text:     "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long text truncated with ellipsis",
			text:     "hello world this is a long prompt",
			maxLen:   15,
			expected: "hello world ...",
		},
		{
			name:     "very short maxLen",
			text:     "hello",
			maxLen:   3,
			expected: "hel",
		},
		{
			name:     "maxLen 4 includes ellipsis",
			text:     "hello world",
			maxLen:   4,
			expected: "h...",
		},
		{
			name:     "empty text",
			text:     "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncatePrompt(tt.text, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncatePrompt(%q, %d) = %q, want %q", tt.text, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// TestSetImagegenProcessor tests the SetImagegenProcessor method.
func TestSetImagegenProcessor(t *testing.T) {
	logger := createTestLogger(t)

	// Create a minimal config for the client
	cfg := &core.Config{}

	// Create a minimal client (won't actually connect)
	client := &canvusapi.Client{}

	m := NewMonitor(client, cfg, logger)

	t.Run("processor initially nil", func(t *testing.T) {
		proc := m.getImagegenProcessor()
		if proc != nil {
			t.Error("processor should be nil initially")
		}
	})

	// Note: We can't easily create a real imagegen.Processor without the SD runtime,
	// so we just test the getter returns nil when not set.
	// Full integration testing would require the SD runtime to be available.
}

// TestRouteUpdateImagePrompt tests that routeUpdate correctly identifies image prompts.
func TestRouteUpdateImagePrompt(t *testing.T) {
	// This test verifies the routing logic without actually calling the handlers
	// by checking that parseImagePrompt correctly identifies image prompts.

	logger := createTestLogger(t)
	m := &Monitor{logger: logger}

	tests := []struct {
		name            string
		text            string
		expectImagePath bool
	}{
		{
			name:            "image prompt routes to image handler",
			text:            "{{image: a beautiful sunset}}",
			expectImagePath: true,
		},
		{
			name:            "regular prompt routes to text handler",
			text:            "{{explain quantum physics}}",
			expectImagePath: false,
		},
		{
			name:            "no trigger doesn't route anywhere",
			text:            "just plain text",
			expectImagePath: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := Update{"text": tt.text}
			_, matched := m.parseImagePrompt(update)

			if matched != tt.expectImagePath {
				t.Errorf("parseImagePrompt() for %q: matched = %v, want %v",
					tt.text, matched, tt.expectImagePath)
			}
		})
	}
}
