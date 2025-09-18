package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"go_backend/core"
)

func TestLocalLLMConnection(t *testing.T) {
	// Save original env vars
	originalBaseURL := os.Getenv("OPENAI_API_BASE_URL")
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		os.Setenv("OPENAI_API_BASE_URL", originalBaseURL)
		os.Setenv("OPENAI_API_KEY", originalAPIKey)
	}()

	// Set test env vars
	os.Setenv("OPENAI_API_BASE_URL", "http://localhost:1234/v1") // LM Studio default
	os.Setenv("OPENAI_API_KEY", "not-needed")                    // Local LLMs often don't need API key

	// Create core config with required fields for LLM
	cfg := &core.Config{
		OpenAIAPIKey:       os.Getenv("OPENAI_API_KEY"),
		OpenAINoteModel:    "gpt-3.5-turbo",
		NoteResponseTokens: 100,
		MaxRetries:         3,
		RetryDelay:         time.Second,
		AITimeout:          time.Second * 30,
	}

	// Test AI response generation
	prompt := "What is 2+2? Please respond with just the number."
	ctx := context.Background()
	response, err := core.TestAIResponse(ctx, cfg, prompt)
	if err != nil {
		t.Fatalf("Failed to generate AI response: %v", err)
	}

	t.Logf("Prompt: %s", prompt)
	t.Logf("Response: %s", response)

	// Basic validation of response
	if response == "" {
		t.Error("Response was empty")
	}
}
