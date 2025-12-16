package pdfprocessor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
)

// mockOpenAIServerForProcessor creates a test server that mimics OpenAI API responses.
func mockOpenAIServerForProcessor(responseContent string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

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
	}))
}

func TestDefaultProcessorConfig(t *testing.T) {
	config := DefaultProcessorConfig()

	// Verify extractor config
	if config.ExtractorConfig.SkipEmptyPages != true {
		t.Error("ExtractorConfig.SkipEmptyPages should default to true")
	}

	// Verify chunker config
	if config.ChunkerConfig.MaxChunkTokens != 20000 {
		t.Errorf("ChunkerConfig.MaxChunkTokens = %d, want %d", config.ChunkerConfig.MaxChunkTokens, 20000)
	}

	// Verify summarizer config
	if config.SummarizerConfig.Model != "gpt-4" {
		t.Errorf("SummarizerConfig.Model = %q, want %q", config.SummarizerConfig.Model, "gpt-4")
	}
}

func TestNewProcessor(t *testing.T) {
	clientConfig := openai.DefaultConfig("test-key")
	client := openai.NewClientWithConfig(clientConfig)

	processor := NewProcessor(DefaultProcessorConfig(), client)
	if processor == nil {
		t.Fatal("NewProcessor returned nil")
	}

	if processor.extractor == nil {
		t.Error("processor.extractor should not be nil")
	}
	if processor.chunker == nil {
		t.Error("processor.chunker should not be nil")
	}
	if processor.summarizer == nil {
		t.Error("processor.summarizer should not be nil")
	}
}

func TestNewProcessorWithProgress(t *testing.T) {
	clientConfig := openai.DefaultConfig("test-key")
	client := openai.NewClientWithConfig(clientConfig)

	called := false
	callback := func(stage string, progress float64, message string) {
		called = true
	}

	processor := NewProcessorWithProgress(DefaultProcessorConfig(), client, callback)
	if processor == nil {
		t.Fatal("NewProcessorWithProgress returned nil")
	}

	if processor.progress == nil {
		t.Error("processor.progress should not be nil")
	}

	// Trigger the callback
	processor.reportProgress("test", 0.5, "test message")
	if !called {
		t.Error("progress callback should have been called")
	}
}

func TestProcessor_SetProgressCallback(t *testing.T) {
	clientConfig := openai.DefaultConfig("test-key")
	client := openai.NewClientWithConfig(clientConfig)

	processor := NewProcessor(DefaultProcessorConfig(), client)

	if processor.progress != nil {
		t.Error("processor.progress should be nil initially")
	}

	called := false
	processor.SetProgressCallback(func(stage string, progress float64, message string) {
		called = true
	})

	processor.reportProgress("test", 0.5, "message")
	if !called {
		t.Error("progress callback should have been called after setting")
	}
}

func TestProcessor_Process_ValidPDF(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	validJSON := `{"type": "text", "content": "# Summary\nThis is a test summary of the document."}`
	server := mockOpenAIServerForProcessor(validJSON)
	defer server.Close()

	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(clientConfig)

	processor := NewProcessor(DefaultProcessorConfig(), client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := processor.Process(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}

	if result.Summary == "" {
		t.Error("Summary should not be empty")
	}

	if result.ExtractionResult == nil {
		t.Error("ExtractionResult should not be nil")
	}

	if result.ChunkerResult == nil {
		t.Error("ChunkerResult should not be nil")
	}

	if result.SummaryResult == nil {
		t.Error("SummaryResult should not be nil")
	}

	if result.ProcessingTime <= 0 {
		t.Error("ProcessingTime should be positive")
	}

	// Verify stages timing
	if result.Stages.ExtractionTime <= 0 {
		t.Error("ExtractionTime should be positive")
	}
	if result.Stages.ChunkingTime <= 0 {
		t.Error("ChunkingTime should be positive")
	}
	if result.Stages.SummarizingTime <= 0 {
		t.Error("SummarizingTime should be positive")
	}
}

func TestProcessor_Process_WithProgress(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	validJSON := `{"type": "text", "content": "Summary content"}`
	server := mockOpenAIServerForProcessor(validJSON)
	defer server.Close()

	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(clientConfig)

	var stages []string
	var mu sync.Mutex

	callback := func(stage string, progress float64, message string) {
		mu.Lock()
		stages = append(stages, stage)
		mu.Unlock()
	}

	processor := NewProcessorWithProgress(DefaultProcessorConfig(), client, callback)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := processor.Process(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify all stages were reported
	mu.Lock()
	defer mu.Unlock()

	if len(stages) < 6 { // 2 per stage (start + end)
		t.Errorf("Expected at least 6 progress callbacks, got %d", len(stages))
	}

	// Verify stages were called in order
	stagesSet := make(map[string]bool)
	for _, s := range stages {
		stagesSet[s] = true
	}

	if !stagesSet["extraction"] {
		t.Error("extraction stage should have been reported")
	}
	if !stagesSet["chunking"] {
		t.Error("chunking stage should have been reported")
	}
	if !stagesSet["summarizing"] {
		t.Error("summarizing stage should have been reported")
	}
}

func TestProcessor_Process_NonexistentFile(t *testing.T) {
	clientConfig := openai.DefaultConfig("test-key")
	client := openai.NewClientWithConfig(clientConfig)

	processor := NewProcessor(DefaultProcessorConfig(), client)

	ctx := context.Background()
	_, err := processor.Process(ctx, "/nonexistent/path/to/file.pdf")

	if err == nil {
		t.Error("Process should fail with nonexistent file")
	}
	if !strings.Contains(err.Error(), "extraction failed") {
		t.Errorf("error message should contain 'extraction failed', got: %v", err)
	}
}

func TestProcessor_ProcessText(t *testing.T) {
	validJSON := `{"type": "text", "content": "Text summary"}`
	server := mockOpenAIServerForProcessor(validJSON)
	defer server.Close()

	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(clientConfig)

	processor := NewProcessor(DefaultProcessorConfig(), client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	text := "This is a long document that needs to be summarized.\n\nIt has multiple paragraphs."
	result, err := processor.ProcessText(ctx, text)
	if err != nil {
		t.Fatalf("ProcessText failed: %v", err)
	}

	if result.Summary == "" {
		t.Error("Summary should not be empty")
	}

	// ExtractionResult should be nil since we skipped extraction
	if result.ExtractionResult != nil {
		t.Error("ExtractionResult should be nil for ProcessText")
	}

	if result.ChunkerResult == nil {
		t.Error("ChunkerResult should not be nil")
	}
}

func TestProcessor_ExtractOnly(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	clientConfig := openai.DefaultConfig("test-key")
	client := openai.NewClientWithConfig(clientConfig)

	processor := NewProcessor(DefaultProcessorConfig(), client)

	result, err := processor.ExtractOnly(pdfPath)
	if err != nil {
		t.Fatalf("ExtractOnly failed: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}

	if result.Text == "" {
		t.Error("Text should not be empty")
	}
}

func TestProcessor_ChunkOnly(t *testing.T) {
	clientConfig := openai.DefaultConfig("test-key")
	client := openai.NewClientWithConfig(clientConfig)

	processor := NewProcessor(DefaultProcessorConfig(), client)

	text := "Paragraph one.\n\nParagraph two.\n\nParagraph three."
	result := processor.ChunkOnly(text)

	if result == nil {
		t.Fatal("result is nil")
	}

	if result.TotalChunks == 0 {
		t.Error("TotalChunks should be > 0")
	}
}

func TestProcessPDF(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	validJSON := `{"type": "text", "content": "PDF Summary"}`
	server := mockOpenAIServerForProcessor(validJSON)
	defer server.Close()

	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(clientConfig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := ProcessPDF(ctx, client, pdfPath)
	if err != nil {
		t.Fatalf("ProcessPDF failed: %v", err)
	}

	if result.Summary != "PDF Summary" {
		t.Errorf("Summary = %q, want %q", result.Summary, "PDF Summary")
	}
}

func TestProcessPDFWithModel(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	validJSON := `{"type": "text", "content": "Custom model summary"}`
	server := mockOpenAIServerForProcessor(validJSON)
	defer server.Close()

	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(clientConfig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := ProcessPDFWithModel(ctx, client, pdfPath, "gpt-3.5-turbo")
	if err != nil {
		t.Fatalf("ProcessPDFWithModel failed: %v", err)
	}

	if result.Summary != "Custom model summary" {
		t.Errorf("Summary = %q, want %q", result.Summary, "Custom model summary")
	}
}

func TestProcessor_Process_NotConfigured(t *testing.T) {
	// Create processor with nil components
	processor := &Processor{}

	ctx := context.Background()
	_, err := processor.Process(ctx, "/some/path.pdf")

	if err != ErrProcessorNotConfigured {
		t.Errorf("Process with unconfigured processor error = %v, want ErrProcessorNotConfigured", err)
	}
}

func TestProcessor_ProcessText_NotConfigured(t *testing.T) {
	processor := &Processor{}

	ctx := context.Background()
	_, err := processor.ProcessText(ctx, "some text")

	if err != ErrProcessorNotConfigured {
		t.Errorf("ProcessText with unconfigured processor error = %v, want ErrProcessorNotConfigured", err)
	}
}

func TestProcessor_ExtractOnly_NotConfigured(t *testing.T) {
	processor := &Processor{}

	_, err := processor.ExtractOnly("/some/path.pdf")

	if err != ErrProcessorNotConfigured {
		t.Errorf("ExtractOnly with unconfigured processor error = %v, want ErrProcessorNotConfigured", err)
	}
}

func TestProcessor_ChunkOnly_NotConfigured(t *testing.T) {
	processor := &Processor{}

	result := processor.ChunkOnly("some text")

	if result != nil {
		t.Error("ChunkOnly with unconfigured processor should return nil")
	}
}

// Benchmark tests

func BenchmarkProcessor_Process(b *testing.B) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		b.Skip("Test PDF file not found, skipping benchmark")
	}

	validJSON := `{"type": "text", "content": "Benchmark summary"}`
	server := mockOpenAIServerForProcessor(validJSON)
	defer server.Close()

	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(clientConfig)

	processor := NewProcessor(DefaultProcessorConfig(), client)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = processor.Process(ctx, pdfPath)
	}
}

func BenchmarkProcessor_ProcessText(b *testing.B) {
	validJSON := `{"type": "text", "content": "Benchmark summary"}`
	server := mockOpenAIServerForProcessor(validJSON)
	defer server.Close()

	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = server.URL + "/v1"
	client := openai.NewClientWithConfig(clientConfig)

	processor := NewProcessor(DefaultProcessorConfig(), client)
	ctx := context.Background()

	text := strings.Repeat("This is a test paragraph with some content.\n\n", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = processor.ProcessText(ctx, text)
	}
}
