package core

import (
	"os"
	"testing"
)

func TestParseCanvasIDs(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected []string
	}{
		{
			name:     "empty string returns nil",
			envValue: "",
			expected: nil,
		},
		{
			name:     "single ID",
			envValue: "canvas-1",
			expected: []string{"canvas-1"},
		},
		{
			name:     "multiple IDs",
			envValue: "canvas-1,canvas-2,canvas-3",
			expected: []string{"canvas-1", "canvas-2", "canvas-3"},
		},
		{
			name:     "IDs with whitespace",
			envValue: " canvas-1 , canvas-2 , canvas-3 ",
			expected: []string{"canvas-1", "canvas-2", "canvas-3"},
		},
		{
			name:     "empty parts ignored",
			envValue: "canvas-1,,canvas-2,",
			expected: []string{"canvas-1", "canvas-2"},
		},
		{
			name:     "only commas returns nil",
			envValue: ",,,",
			expected: nil,
		},
		{
			name:     "UUIDs",
			envValue: "6eaba5df-e5b7-4786-ab95-06b3eb67f40a,a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			expected: []string{"6eaba5df-e5b7-4786-ab95-06b3eb67f40a", "a1b2c3d4-e5f6-7890-abcd-ef1234567890"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the environment variable
			if tt.envValue != "" {
				os.Setenv("TEST_CANVAS_IDS", tt.envValue)
				defer os.Unsetenv("TEST_CANVAS_IDS")
			} else {
				os.Unsetenv("TEST_CANVAS_IDS")
			}

			result := parseCanvasIDs("TEST_CANVAS_IDS")

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d items, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("item %d: expected %q, got %q", i, exp, result[i])
				}
			}
		})
	}
}

func TestConfigCanvasHelpers(t *testing.T) {
	t.Run("GetCanvasCount", func(t *testing.T) {
		cfg := &Config{
			CanvasConfigs: []CanvasConfig{
				{ID: "canvas-1"},
				{ID: "canvas-2"},
			},
		}
		if cfg.GetCanvasCount() != 2 {
			t.Errorf("expected 2, got %d", cfg.GetCanvasCount())
		}

		emptyCfg := &Config{}
		if emptyCfg.GetCanvasCount() != 0 {
			t.Errorf("expected 0, got %d", emptyCfg.GetCanvasCount())
		}
	})

	t.Run("GetCanvasIDs", func(t *testing.T) {
		cfg := &Config{
			CanvasConfigs: []CanvasConfig{
				{ID: "canvas-1"},
				{ID: "canvas-2"},
				{ID: "canvas-3"},
			},
		}
		ids := cfg.GetCanvasIDs()
		if len(ids) != 3 {
			t.Errorf("expected 3 IDs, got %d", len(ids))
		}
		expected := []string{"canvas-1", "canvas-2", "canvas-3"}
		for i, exp := range expected {
			if ids[i] != exp {
				t.Errorf("ID %d: expected %q, got %q", i, exp, ids[i])
			}
		}
	})

	t.Run("GetCanvasConfig found", func(t *testing.T) {
		cfg := &Config{
			CanvasConfigs: []CanvasConfig{
				{ID: "canvas-1", Name: "First Canvas"},
				{ID: "canvas-2", Name: "Second Canvas"},
			},
		}
		canvas := cfg.GetCanvasConfig("canvas-2")
		if canvas == nil {
			t.Fatal("expected to find canvas-2")
		}
		if canvas.Name != "Second Canvas" {
			t.Errorf("expected 'Second Canvas', got %q", canvas.Name)
		}
	})

	t.Run("GetCanvasConfig not found", func(t *testing.T) {
		cfg := &Config{
			CanvasConfigs: []CanvasConfig{
				{ID: "canvas-1"},
			},
		}
		canvas := cfg.GetCanvasConfig("non-existent")
		if canvas != nil {
			t.Error("expected nil for non-existent canvas")
		}
	})

	t.Run("GetPrimaryCanvasID multi-canvas", func(t *testing.T) {
		cfg := &Config{
			CanvasID: "old-canvas",
			CanvasConfigs: []CanvasConfig{
				{ID: "canvas-1"},
				{ID: "canvas-2"},
			},
		}
		primary := cfg.GetPrimaryCanvasID()
		if primary != "canvas-1" {
			t.Errorf("expected 'canvas-1', got %q", primary)
		}
	})

	t.Run("GetPrimaryCanvasID single-canvas fallback", func(t *testing.T) {
		cfg := &Config{
			CanvasID:      "legacy-canvas",
			CanvasConfigs: []CanvasConfig{},
		}
		primary := cfg.GetPrimaryCanvasID()
		if primary != "legacy-canvas" {
			t.Errorf("expected 'legacy-canvas', got %q", primary)
		}
	})

	t.Run("IsMultiCanvasMode", func(t *testing.T) {
		singleCfg := &Config{
			CanvasConfigs: []CanvasConfig{
				{ID: "canvas-1"},
			},
		}
		if singleCfg.IsMultiCanvasMode() {
			t.Error("expected false for single canvas")
		}

		multiCfg := &Config{
			CanvasConfigs: []CanvasConfig{
				{ID: "canvas-1"},
				{ID: "canvas-2"},
			},
		}
		if !multiCfg.IsMultiCanvasMode() {
			t.Error("expected true for multiple canvases")
		}

		emptyCfg := &Config{}
		if emptyCfg.IsMultiCanvasMode() {
			t.Error("expected false for empty config")
		}
	})
}

func TestCanvasConfigStruct(t *testing.T) {
	t.Run("fields are accessible", func(t *testing.T) {
		cfg := CanvasConfig{
			ID:        "test-id",
			Name:      "Test Canvas",
			ServerURL: "https://test.example.com",
			APIKey:    "test-api-key",
		}

		if cfg.ID != "test-id" {
			t.Errorf("ID: expected 'test-id', got %q", cfg.ID)
		}
		if cfg.Name != "Test Canvas" {
			t.Errorf("Name: expected 'Test Canvas', got %q", cfg.Name)
		}
		if cfg.ServerURL != "https://test.example.com" {
			t.Errorf("ServerURL: expected 'https://test.example.com', got %q", cfg.ServerURL)
		}
		if cfg.APIKey != "test-api-key" {
			t.Errorf("APIKey: expected 'test-api-key', got %q", cfg.APIKey)
		}
	})
}

// Note: LoadConfig tests require setting all required environment variables
// and would be better suited for integration tests with proper test fixtures.
// The following test demonstrates the pattern but is skipped by default.

func TestLoadConfig_MultiCanvasMode(t *testing.T) {
	// Skip if we don't want to mess with environment
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Save and restore environment
	saveEnv := func(keys []string) map[string]string {
		saved := make(map[string]string)
		for _, key := range keys {
			saved[key] = os.Getenv(key)
		}
		return saved
	}

	restoreEnv := func(saved map[string]string) {
		for key, value := range saved {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}

	envKeys := []string{
		"CANVUS_SERVER",
		"CANVAS_NAME",
		"CANVAS_ID",
		"CANVAS_IDS",
		"OPENAI_API_KEY",
		"CANVUS_API_KEY",
		"WEBUI_PWD",
	}

	t.Run("single canvas mode", func(t *testing.T) {
		saved := saveEnv(envKeys)
		defer restoreEnv(saved)

		os.Setenv("CANVUS_SERVER", "https://test.example.com")
		os.Setenv("CANVAS_NAME", "Test Canvas")
		os.Setenv("CANVAS_ID", "single-canvas-id")
		os.Unsetenv("CANVAS_IDS")
		os.Setenv("OPENAI_API_KEY", "test-key")
		os.Setenv("CANVUS_API_KEY", "test-canvus-key")
		os.Setenv("WEBUI_PWD", "test-pwd")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.GetCanvasCount() != 1 {
			t.Errorf("expected 1 canvas, got %d", cfg.GetCanvasCount())
		}

		if cfg.GetPrimaryCanvasID() != "single-canvas-id" {
			t.Errorf("expected 'single-canvas-id', got %q", cfg.GetPrimaryCanvasID())
		}

		if cfg.IsMultiCanvasMode() {
			t.Error("expected single canvas mode")
		}
	})

	t.Run("multi canvas mode", func(t *testing.T) {
		saved := saveEnv(envKeys)
		defer restoreEnv(saved)

		os.Setenv("CANVUS_SERVER", "https://test.example.com")
		os.Unsetenv("CANVAS_NAME")
		os.Unsetenv("CANVAS_ID")
		os.Setenv("CANVAS_IDS", "canvas-1,canvas-2,canvas-3")
		os.Setenv("OPENAI_API_KEY", "test-key")
		os.Setenv("CANVUS_API_KEY", "test-canvus-key")
		os.Setenv("WEBUI_PWD", "test-pwd")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.GetCanvasCount() != 3 {
			t.Errorf("expected 3 canvases, got %d", cfg.GetCanvasCount())
		}

		expectedIDs := []string{"canvas-1", "canvas-2", "canvas-3"}
		ids := cfg.GetCanvasIDs()
		for i, exp := range expectedIDs {
			if ids[i] != exp {
				t.Errorf("canvas %d: expected %q, got %q", i, exp, ids[i])
			}
		}

		if !cfg.IsMultiCanvasMode() {
			t.Error("expected multi canvas mode")
		}
	})

	t.Run("missing canvas ID fails", func(t *testing.T) {
		saved := saveEnv(envKeys)
		defer restoreEnv(saved)

		os.Setenv("CANVUS_SERVER", "https://test.example.com")
		os.Unsetenv("CANVAS_NAME")
		os.Unsetenv("CANVAS_ID")
		os.Unsetenv("CANVAS_IDS")
		os.Setenv("OPENAI_API_KEY", "test-key")
		os.Setenv("CANVUS_API_KEY", "test-canvus-key")
		os.Setenv("WEBUI_PWD", "test-pwd")

		_, err := LoadConfig()
		if err == nil {
			t.Error("expected error when neither CANVAS_ID nor CANVAS_IDS is set")
		}
	})
}
