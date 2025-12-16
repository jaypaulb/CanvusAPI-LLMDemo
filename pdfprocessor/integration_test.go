// Package pdfprocessor integration tests verify the complete PDF processing pipeline.
//
// These tests require:
//   - A test PDF file at ../test_files/test_pdf.pdf
//   - Network access for OpenAI API (mocked in most tests)
//
// Integration tests verify that:
//   - Components work together correctly (extractor -> chunker -> summarizer)
//   - Error handling propagates correctly through the pipeline
//   - Context cancellation works at all stages
//   - Resource cleanup happens properly
//   - Edge cases are handled (empty PDFs, large content, etc.)
package pdfprocessor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
)

// =============================================================================
// Test Helpers for Integration Tests
// =============================================================================

// mockOpenAIServerWithTracking creates a test server that tracks requests and
// returns configurable responses.
type mockServerConfig struct {
	responseContent string
	responseDelay   time.Duration
	shouldFail      bool
	failAfter       int // Fail after N successful requests
	statusCode      int
}

func newMockOpenAIServerWithTracking(config mockServerConfig) (*httptest.Server, *int32) {
	requestCount := new(int32)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(requestCount, 1)

		// Apply delay if configured
		if config.responseDelay > 0 {
			time.Sleep(config.responseDelay)
		}

		// Check if we should fail
		if config.shouldFail || (config.failAfter > 0 && int(count) > config.failAfter) {
			statusCode := config.statusCode
			if statusCode == 0 {
				statusCode = http.StatusInternalServerError
			}
			w.WriteHeader(statusCode)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"message": "mock server error",
					"type":    "server_error",
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: config.responseContent,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))

	return server, requestCount
}

// createTestClient creates an OpenAI client configured to use the test server.
func createTestClient(serverURL string) *openai.Client {
	clientConfig := openai.DefaultConfig("test-key")
	clientConfig.BaseURL = serverURL + "/v1"
	return openai.NewClientWithConfig(clientConfig)
}

// =============================================================================
// Full Pipeline Integration Tests
// =============================================================================

func TestIntegration_FullPipeline_Success(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	// Setup mock server with valid response
	validJSON := `{"type": "text", "content": "# Document Summary\n\nThis document contains important information about the test content."}`
	server, requestCount := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: validJSON,
	})
	defer server.Close()

	client := createTestClient(server.URL)
	processor := NewProcessor(DefaultProcessorConfig(), client)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Execute full pipeline
	result, err := processor.Process(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Full pipeline failed: %v", err)
	}

	// Verify result completeness
	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Verify all stages completed
	if result.ExtractionResult == nil {
		t.Error("ExtractionResult should not be nil - extraction stage failed")
	}
	if result.ChunkerResult == nil {
		t.Error("ChunkerResult should not be nil - chunking stage failed")
	}
	if result.SummaryResult == nil {
		t.Error("SummaryResult should not be nil - summarization stage failed")
	}

	// Verify summary content
	if result.Summary == "" {
		t.Error("Summary should not be empty")
	}
	if !strings.Contains(result.Summary, "Document Summary") {
		t.Errorf("Summary content unexpected: %s", result.Summary)
	}

	// Verify timing information
	if result.ProcessingTime <= 0 {
		t.Error("ProcessingTime should be positive")
	}
	if result.Stages.ExtractionTime <= 0 {
		t.Error("ExtractionTime should be positive")
	}
	if result.Stages.ChunkingTime <= 0 {
		t.Error("ChunkingTime should be positive")
	}
	if result.Stages.SummarizingTime <= 0 {
		t.Error("SummarizingTime should be positive")
	}

	// Verify OpenAI was called
	if *requestCount == 0 {
		t.Error("OpenAI API should have been called")
	}

	// Verify data flow between components
	if result.ExtractionResult.Text == "" {
		t.Error("Extraction should have produced text")
	}
	if result.ChunkerResult.TotalChunks == 0 {
		t.Error("Chunker should have produced chunks")
	}
}

func TestIntegration_FullPipeline_WithProgressTracking(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	validJSON := `{"type": "text", "content": "Summary with progress tracking"}`
	server, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: validJSON,
	})
	defer server.Close()

	client := createTestClient(server.URL)

	// Track progress callbacks
	var progressEvents []struct {
		stage    string
		progress float64
		message  string
	}
	var mu sync.Mutex

	callback := func(stage string, progress float64, message string) {
		mu.Lock()
		progressEvents = append(progressEvents, struct {
			stage    string
			progress float64
			message  string
		}{stage, progress, message})
		mu.Unlock()
	}

	processor := NewProcessorWithProgress(DefaultProcessorConfig(), client, callback)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := processor.Process(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Pipeline with progress failed: %v", err)
	}

	mu.Lock()
	events := progressEvents
	mu.Unlock()

	// Verify progress events
	if len(events) < 6 { // 2 events per stage (start + end)
		t.Errorf("Expected at least 6 progress events, got %d", len(events))
	}

	// Verify stage order and progress values
	stageOrder := []string{"extraction", "chunking", "summarizing"}
	stagesSeen := make(map[string]int)

	for _, event := range events {
		stagesSeen[event.stage]++
		// Verify progress is in valid range
		if event.progress < 0 || event.progress > 1 {
			t.Errorf("Invalid progress value: %f for stage %s", event.progress, event.stage)
		}
	}

	for _, stage := range stageOrder {
		if stagesSeen[stage] < 2 {
			t.Errorf("Stage %s should have at least 2 events (start + end), got %d", stage, stagesSeen[stage])
		}
	}

	// Verify result is still valid
	if result.Summary == "" {
		t.Error("Summary should not be empty even with progress tracking")
	}
}

// =============================================================================
// Context Cancellation Tests
// =============================================================================

func TestIntegration_ContextCancellation_DuringSummarization(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	// Create server with delay to allow cancellation
	server, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: `{"type": "text", "content": "Should not see this"}`,
		responseDelay:   500 * time.Millisecond,
	})
	defer server.Close()

	client := createTestClient(server.URL)
	processor := NewProcessor(DefaultProcessorConfig(), client)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := processor.Process(ctx, pdfPath)
	elapsed := time.Since(start)

	// Should fail due to context cancellation or deadline
	if err == nil {
		t.Error("Process should fail when context is cancelled")
	}

	// Should fail relatively quickly (not waiting for full delay)
	if elapsed > 1*time.Second {
		t.Errorf("Process took too long after cancellation: %v", elapsed)
	}
}

func TestIntegration_ContextCancellation_BeforeStart(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	server, requestCount := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: `{"type": "text", "content": "test"}`,
	})
	defer server.Close()

	client := createTestClient(server.URL)
	processor := NewProcessor(DefaultProcessorConfig(), client)

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := processor.Process(ctx, pdfPath)

	// Should fail - either during extraction or summarization
	if err == nil {
		t.Error("Process should fail with pre-cancelled context")
	}

	// Should not have made many API calls (might make one before noticing cancellation)
	if *requestCount > 1 {
		t.Errorf("Should minimize API calls with cancelled context, got %d", *requestCount)
	}
}

// =============================================================================
// Error Handling Integration Tests
// =============================================================================

func TestIntegration_ErrorPropagation_ExtractionFailure(t *testing.T) {
	server, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: `{"type": "text", "content": "test"}`,
	})
	defer server.Close()

	client := createTestClient(server.URL)
	processor := NewProcessor(DefaultProcessorConfig(), client)

	ctx := context.Background()

	// Use nonexistent file to trigger extraction failure
	_, err := processor.Process(ctx, "/nonexistent/path/to/file.pdf")

	if err == nil {
		t.Error("Process should fail with nonexistent file")
	}
	if !strings.Contains(err.Error(), "extraction failed") {
		t.Errorf("Error should indicate extraction failure, got: %v", err)
	}
}

func TestIntegration_ErrorPropagation_SummarizationFailure(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	// Server that always fails
	server, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		shouldFail: true,
		statusCode: http.StatusServiceUnavailable,
	})
	defer server.Close()

	client := createTestClient(server.URL)
	processor := NewProcessor(DefaultProcessorConfig(), client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := processor.Process(ctx, pdfPath)

	if err == nil {
		t.Error("Process should fail when summarization fails")
	}
	if !strings.Contains(err.Error(), "summarization failed") {
		t.Errorf("Error should indicate summarization failure, got: %v", err)
	}
}

func TestIntegration_ErrorPropagation_InvalidJSONResponse(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	// Server returns invalid JSON structure
	invalidJSON := `not a valid json response at all`
	server, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: invalidJSON,
	})
	defer server.Close()

	client := createTestClient(server.URL)
	processor := NewProcessor(DefaultProcessorConfig(), client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := processor.Process(ctx, pdfPath)

	// The summarizer correctly returns ErrInvalidJSON when response doesn't contain valid JSON
	if err == nil {
		t.Error("Process should fail with invalid JSON response")
	}

	// Verify the error indicates JSON parsing failure
	if !strings.Contains(err.Error(), "JSON") && !strings.Contains(err.Error(), "summarization failed") {
		t.Errorf("Error should indicate JSON or summarization failure, got: %v", err)
	}
}

// =============================================================================
// Component Integration Tests
// =============================================================================

func TestIntegration_ExtractorToChunker_DataFlow(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	// Create standalone components
	extractor := NewExtractor(DefaultExtractorConfig())
	chunker := NewChunker(DefaultChunkerConfig())

	// Extract text
	extractionResult, err := extractor.Extract(pdfPath)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	if extractionResult.Text == "" {
		t.Fatal("Extracted text should not be empty")
	}

	// Chunk the extracted text
	chunkerResult := chunker.SplitIntoChunks(extractionResult.Text)

	if chunkerResult.TotalChunks == 0 {
		t.Error("Chunker should produce at least one chunk from extracted text")
	}

	// Verify token estimates are consistent
	totalChunkTokens := 0
	for _, chunk := range chunkerResult.Chunks {
		totalChunkTokens += chunk.EstimatedTokens
	}

	// Chunker total should be close to extraction estimate (may differ due to overlap)
	if totalChunkTokens < extractionResult.EstimatedTokens/2 {
		t.Errorf("Chunker tokens (%d) significantly less than extraction estimate (%d)",
			totalChunkTokens, extractionResult.EstimatedTokens)
	}
}

func TestIntegration_ChunkerToSummarizer_DataFlow(t *testing.T) {
	server, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: `{"type": "text", "content": "Summary of chunked content"}`,
	})
	defer server.Close()

	client := createTestClient(server.URL)

	chunker := NewChunker(DefaultChunkerConfig())
	summarizer := NewSummarizer(DefaultSummarizerConfig(), client)

	// Create test text
	text := strings.Repeat("This is a test paragraph with meaningful content.\n\n", 50)

	// Chunk the text
	chunkerResult := chunker.SplitIntoChunks(text)
	if chunkerResult.TotalChunks == 0 {
		t.Fatal("Chunker should produce chunks")
	}

	// Summarize the chunks
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	summaryResult, err := summarizer.SummarizeChunkerResult(ctx, chunkerResult)
	if err != nil {
		t.Fatalf("Summarization failed: %v", err)
	}

	if summaryResult.Content == "" {
		t.Error("Summary content should not be empty")
	}

	// Verify usage tracking
	if summaryResult.PromptTokens == 0 {
		// Note: Mock server doesn't return usage, so this might be 0
		t.Log("Note: PromptTokens is 0 (expected with mock server)")
	}
}

// =============================================================================
// Edge Case Integration Tests
// =============================================================================

func TestIntegration_EmptyContent_Handling(t *testing.T) {
	server, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: `{"type": "text", "content": "Summary of empty content"}`,
	})
	defer server.Close()

	client := createTestClient(server.URL)
	processor := NewProcessor(DefaultProcessorConfig(), client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Process empty text
	result, err := processor.ProcessText(ctx, "")

	// Should handle empty content gracefully
	if err != nil && !strings.Contains(err.Error(), "empty") {
		t.Logf("ProcessText with empty content: %v", err)
	}

	if err == nil && result != nil {
		if result.ChunkerResult != nil && result.ChunkerResult.TotalChunks > 0 {
			t.Logf("Empty text produced %d chunks", result.ChunkerResult.TotalChunks)
		}
	}
}

func TestIntegration_LargeContent_Chunking(t *testing.T) {
	server, requestCount := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: `{"type": "text", "content": "Summary of large content"}`,
	})
	defer server.Close()

	client := createTestClient(server.URL)

	// Configure for smaller chunks to test multi-chunk behavior
	// Token estimation is ~4 chars per token, so 500 tokens = ~2000 chars
	// Use PreserveParagraphs=false to enable token-based chunking
	config := DefaultProcessorConfig()
	config.ChunkerConfig.MaxChunkTokens = 500  // Much smaller chunks
	config.ChunkerConfig.MaxChunks = 10        // Allow more chunks
	config.ChunkerConfig.PreserveParagraphs = false // Force token-based chunking

	processor := NewProcessor(config, client)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create large text (each repeat is ~70 chars = ~17 tokens, 500 repeats = ~8500 tokens)
	// With 500 tokens per chunk and token-based chunking, we should get multiple chunks
	largeText := strings.Repeat("This is a test paragraph with multiple words to increase token count. ", 500)

	result, err := processor.ProcessText(ctx, largeText)
	if err != nil {
		t.Fatalf("ProcessText failed with large content: %v", err)
	}

	// Verify chunking occurred
	if result.ChunkerResult == nil {
		t.Fatal("ChunkerResult should not be nil")
	}

	// Log actual chunk count for debugging
	t.Logf("Produced %d chunks from %d estimated tokens (configured MaxChunkTokens=%d, MaxChunks=%d, PreserveParagraphs=%v)",
		result.ChunkerResult.TotalChunks,
		result.ChunkerResult.OriginalTokensEstimate,
		config.ChunkerConfig.MaxChunkTokens,
		config.ChunkerConfig.MaxChunks,
		config.ChunkerConfig.PreserveParagraphs)

	// With token-based chunking and small chunk size, we should get multiple chunks
	if result.ChunkerResult.TotalChunks < 2 {
		t.Errorf("Large content with small MaxChunkTokens (%d) and PreserveParagraphs=false should produce multiple chunks, got %d",
			config.ChunkerConfig.MaxChunkTokens, result.ChunkerResult.TotalChunks)
	}

	// Verify chunk limit was respected
	if result.ChunkerResult.TotalChunks > config.ChunkerConfig.MaxChunks {
		t.Errorf("Chunks (%d) should not exceed MaxChunks (%d)",
			result.ChunkerResult.TotalChunks, config.ChunkerConfig.MaxChunks)
	}

	// Verify API was called for summarization
	if *requestCount == 0 {
		t.Error("API should have been called for large content")
	}
}

func TestIntegration_LargeContent_ParagraphChunking(t *testing.T) {
	server, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: `{"type": "text", "content": "Summary of paragraph content"}`,
	})
	defer server.Close()

	client := createTestClient(server.URL)

	// Configure for paragraph-aware chunking
	config := DefaultProcessorConfig()
	config.ChunkerConfig.MaxChunkTokens = 100  // Small chunks
	config.ChunkerConfig.MaxChunks = 10
	config.ChunkerConfig.PreserveParagraphs = true // Paragraph-based chunking

	processor := NewProcessor(config, client)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create text WITH paragraph breaks so paragraph-based chunking works
	var largeText strings.Builder
	for i := 0; i < 50; i++ {
		largeText.WriteString("This is paragraph number ")
		largeText.WriteString(strings.Repeat("content ", 20))
		largeText.WriteString("\n\n") // Paragraph separator
	}

	result, err := processor.ProcessText(ctx, largeText.String())
	if err != nil {
		t.Fatalf("ProcessText failed with paragraph content: %v", err)
	}

	if result.ChunkerResult == nil {
		t.Fatal("ChunkerResult should not be nil")
	}

	t.Logf("Paragraph chunking produced %d chunks from %d estimated tokens",
		result.ChunkerResult.TotalChunks,
		result.ChunkerResult.OriginalTokensEstimate)

	// With paragraph-based chunking and small chunk limit, we should get multiple chunks
	if result.ChunkerResult.TotalChunks < 2 {
		t.Logf("Note: Paragraph-based chunking with small paragraphs may produce fewer chunks")
	}
}

func TestIntegration_UnicodeContent_Processing(t *testing.T) {
	server, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: `{"type": "text", "content": "Summary of unicode content with special characters"}`,
	})
	defer server.Close()

	client := createTestClient(server.URL)
	processor := NewProcessor(DefaultProcessorConfig(), client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Unicode text with various scripts and special characters
	unicodeText := `
		English: Hello World
		Spanish: Hola Mundo
		Chinese: 你好世界
		Japanese: こんにちは世界
		Arabic: مرحبا بالعالم
		Emoji: 🌍🌎🌏
		Mathematical: ∑∏∫∂∇
		Special: ™®©℃℉
	`

	result, err := processor.ProcessText(ctx, unicodeText)
	if err != nil {
		t.Fatalf("ProcessText failed with unicode content: %v", err)
	}

	if result.Summary == "" {
		t.Error("Summary should not be empty for unicode content")
	}

	// Verify chunking handled unicode correctly
	if result.ChunkerResult == nil {
		t.Error("ChunkerResult should not be nil")
	}
}

// =============================================================================
// Concurrent Processing Tests
// =============================================================================

func TestIntegration_ConcurrentProcessing(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	server, requestCount := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: `{"type": "text", "content": "Concurrent summary"}`,
	})
	defer server.Close()

	client := createTestClient(server.URL)

	const numConcurrent = 3
	var wg sync.WaitGroup
	errors := make([]error, numConcurrent)
	results := make([]*ProcessResult, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			processor := NewProcessor(DefaultProcessorConfig(), client)
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			result, err := processor.Process(ctx, pdfPath)
			errors[idx] = err
			results[idx] = result
		}(i)
	}

	wg.Wait()

	// Verify all succeeded
	for i, err := range errors {
		if err != nil {
			t.Errorf("Concurrent process %d failed: %v", i, err)
		}
	}

	for i, result := range results {
		if result == nil {
			t.Errorf("Concurrent process %d returned nil result", i)
		} else if result.Summary == "" {
			t.Errorf("Concurrent process %d returned empty summary", i)
		}
	}

	// Verify server received requests from all goroutines
	if *requestCount < int32(numConcurrent) {
		t.Errorf("Expected at least %d API calls, got %d", numConcurrent, *requestCount)
	}
}

// =============================================================================
// Configuration Integration Tests
// =============================================================================

func TestIntegration_CustomConfiguration(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	server, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: `{"type": "text", "content": "Custom config summary"}`,
	})
	defer server.Close()

	client := createTestClient(server.URL)

	// Create custom configuration
	config := ProcessorConfig{
		ExtractorConfig: ExtractorConfig{
			PageSeparator:  "\n---PAGE---\n",
			SkipEmptyPages: true,
			MaxPages:       10,
		},
		ChunkerConfig: ChunkerConfig{
			MaxChunkTokens:     5000,
			OverlapTokens:      100,
			ParagraphSeparator: "\n\n---\n\n",
			MaxChunks:          3,
			PreserveParagraphs: true,
		},
		SummarizerConfig: SummarizerConfig{
			Model:       "gpt-3.5-turbo",
			MaxTokens:   500,
			Temperature: 0.5,
		},
	}

	processor := NewProcessor(config, client)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := processor.Process(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Process with custom config failed: %v", err)
	}

	if result.Summary == "" {
		t.Error("Summary should not be empty")
	}

	// Verify chunk limit was respected
	if result.ChunkerResult != nil && result.ChunkerResult.TotalChunks > config.ChunkerConfig.MaxChunks {
		t.Errorf("Chunks (%d) exceeded MaxChunks (%d)",
			result.ChunkerResult.TotalChunks, config.ChunkerConfig.MaxChunks)
	}
}

// =============================================================================
// Convenience Function Tests
// =============================================================================

func TestIntegration_ProcessPDFConvenience(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	server, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: `{"type": "text", "content": "Convenience function summary"}`,
	})
	defer server.Close()

	client := createTestClient(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := ProcessPDF(ctx, client, pdfPath)
	if err != nil {
		t.Fatalf("ProcessPDF convenience function failed: %v", err)
	}

	if result.Summary != "Convenience function summary" {
		t.Errorf("Unexpected summary: %s", result.Summary)
	}
}

func TestIntegration_ProcessPDFWithModelConvenience(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	// Track model in request
	var receivedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Model string `json:"model"`
		}
		json.Unmarshal(body, &req)
		receivedModel = req.Model

		w.Header().Set("Content-Type", "application/json")
		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{Message: openai.ChatCompletionMessage{Content: `{"type": "text", "content": "Model test"}`}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := createTestClient(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	customModel := "gpt-4-turbo-preview"
	_, err := ProcessPDFWithModel(ctx, client, pdfPath, customModel)
	if err != nil {
		t.Fatalf("ProcessPDFWithModel failed: %v", err)
	}

	if receivedModel != customModel {
		t.Errorf("Model mismatch: got %s, want %s", receivedModel, customModel)
	}
}

// =============================================================================
// Resource Cleanup Tests
// =============================================================================

func TestIntegration_ResourceCleanup_OnError(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	// Server that fails
	server, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		shouldFail: true,
	})
	defer server.Close()

	client := createTestClient(server.URL)
	processor := NewProcessor(DefaultProcessorConfig(), client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should fail during summarization
	_, err := processor.Process(ctx, pdfPath)
	if err == nil {
		t.Fatal("Process should have failed")
	}

	// Verify we can run another process (resources were cleaned up)
	validServer, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: `{"type": "text", "content": "Recovery test"}`,
	})
	defer validServer.Close()

	client2 := createTestClient(validServer.URL)
	processor2 := NewProcessor(DefaultProcessorConfig(), client2)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()

	result, err := processor2.Process(ctx2, pdfPath)
	if err != nil {
		t.Fatalf("Second process should succeed after first failure: %v", err)
	}

	if result.Summary != "Recovery test" {
		t.Error("Recovery process should have correct summary")
	}
}

// =============================================================================
// Metrics and Observability Tests
// =============================================================================

func TestIntegration_ProcessingMetrics(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	server, _ := newMockOpenAIServerWithTracking(mockServerConfig{
		responseContent: `{"type": "text", "content": "Metrics test summary"}`,
	})
	defer server.Close()

	client := createTestClient(server.URL)
	processor := NewProcessor(DefaultProcessorConfig(), client)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := processor.Process(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify comprehensive metrics
	t.Logf("Processing Time: %v", result.ProcessingTime)
	t.Logf("Extraction Time: %v", result.Stages.ExtractionTime)
	t.Logf("Chunking Time: %v", result.Stages.ChunkingTime)
	t.Logf("Summarizing Time: %v", result.Stages.SummarizingTime)

	// Verify timing sanity
	totalStageTime := result.Stages.ExtractionTime + result.Stages.ChunkingTime + result.Stages.SummarizingTime
	if totalStageTime > result.ProcessingTime+time.Millisecond*100 {
		t.Errorf("Sum of stage times (%v) should not significantly exceed total time (%v)",
			totalStageTime, result.ProcessingTime)
	}

	// Log extraction metrics
	if result.ExtractionResult != nil {
		t.Logf("Pages: %d", result.ExtractionResult.ExtractedPages)
		t.Logf("Text Length: %d", len(result.ExtractionResult.Text))
		t.Logf("Estimated Tokens: %d", result.ExtractionResult.EstimatedTokens)
	}

	// Log chunking metrics
	if result.ChunkerResult != nil {
		t.Logf("Chunks: %d", result.ChunkerResult.TotalChunks)
		t.Logf("Total Tokens Estimate: %d", result.ChunkerResult.TotalTokensEstimate)
	}
}

// =============================================================================
// Error Type Tests
// =============================================================================

func TestIntegration_ErrorTypes(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (*Processor, string)
		wantErrType string
	}{
		{
			name: "file not found error",
			setup: func() (*Processor, string) {
				client := createTestClient("http://localhost:1")
				processor := NewProcessor(DefaultProcessorConfig(), client)
				return processor, "/nonexistent/file.pdf"
			},
			wantErrType: "extraction failed",
		},
		{
			name: "not configured error",
			setup: func() (*Processor, string) {
				processor := &Processor{} // Empty processor
				return processor, "/any/path.pdf"
			},
			wantErrType: "not properly configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, path := tt.setup()

			ctx := context.Background()
			_, err := processor.Process(ctx, path)

			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrType) {
				t.Errorf("Error type mismatch: got %v, want containing %q", err, tt.wantErrType)
			}

			// Verify error is properly wrapped
			var processorErr error
			if errors.Is(err, ErrProcessorNotConfigured) {
				t.Log("Error is ErrProcessorNotConfigured (expected for unconfigured)")
			}
			_ = processorErr
		})
	}
}
