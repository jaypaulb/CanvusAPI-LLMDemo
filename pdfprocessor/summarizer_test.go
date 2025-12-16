package pdfprocessor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
)

// mockOpenAIServer creates a test server that mimics OpenAI API responses.
func mockOpenAIServer(t *testing.T, responseContent string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if !strings.Contains(r.URL.Path, "/chat/completions") {
			t.Errorf("Expected /chat/completions path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		if statusCode == http.StatusOK {
			resp := openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: responseContent,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "test error",
				},
			})
		}
	}))
}

func TestDefaultSummarizerConfig(t *testing.T) {
	config := DefaultSummarizerConfig()

	if config.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", config.Model, "gpt-4")
	}
	if config.MaxTokens != 2000 {
		t.Errorf("MaxTokens = %d, want %d", config.MaxTokens, 2000)
	}
	if config.Temperature != 0.3 {
		t.Errorf("Temperature = %f, want %f", config.Temperature, 0.3)
	}
	if config.SystemPromptTemplate == "" {
		t.Error("SystemPromptTemplate should not be empty")
	}
	if config.ChunkTemplate == "" {
		t.Error("ChunkTemplate should not be empty")
	}
	if config.FinalPrompt == "" {
		t.Error("FinalPrompt should not be empty")
	}
}

func TestNewSummarizer(t *testing.T) {
	config := DefaultSummarizerConfig()
	clientConfig := openai.DefaultConfig("test-key")
	client := openai.NewClientWithConfig(clientConfig)

	s := NewSummarizer(config, client)
	if s == nil {
		t.Fatal("NewSummarizer returned nil")
	}
	if s.config.Model != config.Model {
		t.Errorf("config.Model = %q, want %q", s.config.Model, config.Model)
	}
}

func TestSummarizer_buildMessages(t *testing.T) {
	config := DefaultSummarizerConfig()
	clientConfig := openai.DefaultConfig("test-key")
	client := openai.NewClientWithConfig(clientConfig)
	s := NewSummarizer(config, client)

	chunks := []string{"chunk one", "chunk two", "chunk three"}
	messages := s.buildMessages(chunks)

	// Expected: 1 system + 3 chunks + 1 final = 5 messages
	expectedCount := 5
	if len(messages) != expectedCount {
		t.Fatalf("messages length = %d, want %d", len(messages), expectedCount)
	}

	// First message should be system message
	if messages[0].Role != openai.ChatMessageRoleSystem {
		t.Errorf("First message role = %q, want %q", messages[0].Role, openai.ChatMessageRoleSystem)
	}

	// Verify system message contains chunk count
	if !strings.Contains(messages[0].Content, "3") {
		t.Error("System message should contain chunk count")
	}

	// Chunk messages should be user role
	for i := 1; i <= 3; i++ {
		if messages[i].Role != openai.ChatMessageRoleUser {
			t.Errorf("Chunk message %d role = %q, want %q", i, messages[i].Role, openai.ChatMessageRoleUser)
		}
	}

	// Last message should be final prompt
	if messages[4].Role != openai.ChatMessageRoleUser {
		t.Errorf("Final message role = %q, want %q", messages[4].Role, openai.ChatMessageRoleUser)
	}
	if !strings.Contains(messages[4].Content, "JSON") {
		t.Error("Final message should mention JSON format")
	}
}

func TestSummarizer_estimatePromptTokens(t *testing.T) {
	config := DefaultSummarizerConfig()
	clientConfig := openai.DefaultConfig("test-key")
	client := openai.NewClientWithConfig(clientConfig)
	s := NewSummarizer(config, client)

	chunks := []string{"short chunk", "another short chunk"}
	tokens := s.estimatePromptTokens(chunks)

	if tokens <= 0 {
		t.Error("estimatePromptTokens should return positive value")
	}
}

func TestSummarizer_Summarize_NoChunks(t *testing.T) {
	config := DefaultSummarizerConfig()
	clientConfig := openai.DefaultConfig("test-key")
	client := openai.NewClientWithConfig(clientConfig)
	s := NewSummarizer(config, client)

	_, err := s.Summarize(context.Background(), []string{})
	if err != ErrNoChunks {
		t.Errorf("Summarize with no chunks error = %v, want ErrNoChunks", err)
	}

	_, err = s.Summarize(context.Background(), nil)
	if err != ErrNoChunks {
		t.Errorf("Summarize with nil chunks error = %v, want ErrNoChunks", err)
	}
}

func TestSummarizer_Summarize_Success(t *testing.T) {
	validJSON := `{"type": "text", "content": "# Overview\nThis is a test summary."}`
	server := mockOpenAIServer(t, validJSON, http.StatusOK)
	defer server.Close()

	config := DefaultSummarizerConfig()
	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(clientConfig)

	s := NewSummarizer(config, client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	chunks := []string{"Test chunk one", "Test chunk two"}
	result, err := s.Summarize(ctx, chunks)
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}

	if result.Content != "# Overview\nThis is a test summary." {
		t.Errorf("Content = %q, want %q", result.Content, "# Overview\nThis is a test summary.")
	}

	if result.ChunksProcessed != 2 {
		t.Errorf("ChunksProcessed = %d, want %d", result.ChunksProcessed, 2)
	}

	if result.RawResponse != validJSON {
		t.Errorf("RawResponse = %q, want %q", result.RawResponse, validJSON)
	}

	if result.PromptTokens <= 0 {
		t.Error("PromptTokens should be positive")
	}

	if result.CompletionTokens <= 0 {
		t.Error("CompletionTokens should be positive")
	}
}

func TestSummarizer_Summarize_InvalidJSON(t *testing.T) {
	server := mockOpenAIServer(t, "This is not JSON at all", http.StatusOK)
	defer server.Close()

	config := DefaultSummarizerConfig()
	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(clientConfig)

	s := NewSummarizer(config, client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.Summarize(ctx, []string{"test chunk"})
	if err == nil {
		t.Error("Summarize should fail with invalid JSON response")
	}
	if err != ErrInvalidJSON {
		t.Errorf("error = %v, want ErrInvalidJSON", err)
	}
}

func TestSummarizer_Summarize_EmptyContent(t *testing.T) {
	server := mockOpenAIServer(t, `{"type": "text", "content": ""}`, http.StatusOK)
	defer server.Close()

	config := DefaultSummarizerConfig()
	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(clientConfig)

	s := NewSummarizer(config, client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.Summarize(ctx, []string{"test chunk"})
	if err == nil {
		t.Error("Summarize should fail with empty content")
	}
}

func TestSummarizer_SummarizeChunkerResult(t *testing.T) {
	validJSON := `{"type": "text", "content": "Summary of chunked text"}`
	server := mockOpenAIServer(t, validJSON, http.StatusOK)
	defer server.Close()

	config := DefaultSummarizerConfig()
	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(clientConfig)

	s := NewSummarizer(config, client)

	// Create a mock ChunkerResult
	chunkerResult := &ChunkerResult{
		Chunks: []ChunkResult{
			{Text: "First chunk", Index: 0},
			{Text: "Second chunk", Index: 1},
		},
		TotalChunks: 2,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := s.SummarizeChunkerResult(ctx, chunkerResult)
	if err != nil {
		t.Fatalf("SummarizeChunkerResult failed: %v", err)
	}

	if result.Content != "Summary of chunked text" {
		t.Errorf("Content = %q, want %q", result.Content, "Summary of chunked text")
	}
}

func TestSummarizer_SummarizeChunkerResult_Nil(t *testing.T) {
	config := DefaultSummarizerConfig()
	clientConfig := openai.DefaultConfig("test-key")
	client := openai.NewClientWithConfig(clientConfig)

	s := NewSummarizer(config, client)

	_, err := s.SummarizeChunkerResult(context.Background(), nil)
	if err != ErrNoChunks {
		t.Errorf("SummarizeChunkerResult(nil) error = %v, want ErrNoChunks", err)
	}
}

func TestSummarizer_SummarizeChunkerResult_EmptyChunks(t *testing.T) {
	config := DefaultSummarizerConfig()
	clientConfig := openai.DefaultConfig("test-key")
	client := openai.NewClientWithConfig(clientConfig)

	s := NewSummarizer(config, client)

	chunkerResult := &ChunkerResult{
		Chunks:      []ChunkResult{},
		TotalChunks: 0,
	}

	_, err := s.SummarizeChunkerResult(context.Background(), chunkerResult)
	if err != ErrNoChunks {
		t.Errorf("SummarizeChunkerResult with empty chunks error = %v, want ErrNoChunks", err)
	}
}

func TestExtractJSONContent(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantContent string
		wantErr     bool
	}{
		{
			name:        "valid JSON",
			input:       `{"type": "text", "content": "Hello world"}`,
			wantContent: "Hello world",
			wantErr:     false,
		},
		{
			name:        "JSON with surrounding text",
			input:       `Here is my response: {"type": "text", "content": "Summary here"} That's all.`,
			wantContent: "Summary here",
			wantErr:     false,
		},
		{
			name:        "JSON with markdown content",
			input:       `{"type": "text", "content": "# Heading\n- Item 1\n- Item 2"}`,
			wantContent: "# Heading\n- Item 1\n- Item 2",
			wantErr:     false,
		},
		{
			name:        "no JSON",
			input:       "This is just plain text",
			wantContent: "",
			wantErr:     true,
		},
		{
			name:        "malformed JSON",
			input:       `{type: text, content: broken}`,
			wantContent: "",
			wantErr:     true,
		},
		{
			name:        "empty content field",
			input:       `{"type": "text", "content": ""}`,
			wantContent: "",
			wantErr:     true,
		},
		{
			name:        "missing content field",
			input:       `{"type": "text"}`,
			wantContent: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := extractJSONContent(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("extractJSONContent should return error")
				}
			} else {
				if err != nil {
					t.Errorf("extractJSONContent error = %v", err)
				}
				if content != tt.wantContent {
					t.Errorf("content = %q, want %q", content, tt.wantContent)
				}
			}
		})
	}
}

func TestSummarizeText(t *testing.T) {
	validJSON := `{"type": "text", "content": "Complete summary"}`
	server := mockOpenAIServer(t, validJSON, http.StatusOK)
	defer server.Close()

	config := DefaultSummarizerConfig()
	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(clientConfig)

	summarizer := NewSummarizer(config, client)
	chunker := NewChunker(DefaultChunkerConfig())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create text that will be chunked
	text := "This is a test paragraph.\n\nThis is another paragraph."

	result, err := SummarizeText(ctx, summarizer, chunker, text)
	if err != nil {
		t.Fatalf("SummarizeText failed: %v", err)
	}

	if result.Content != "Complete summary" {
		t.Errorf("Content = %q, want %q", result.Content, "Complete summary")
	}
}
