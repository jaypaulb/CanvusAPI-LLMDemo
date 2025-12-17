package canvasanalyzer

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

// ============================================================================
// Mock OpenAI Server
// ============================================================================

// mockOpenAIResponse creates a mock ChatCompletion response.
func mockOpenAIResponse(content string, promptTokens, completionTokens int) openai.ChatCompletionResponse {
	return openai.ChatCompletionResponse{
		ID:      "chatcmpl-test-123",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "gpt-4",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: content,
				},
				FinishReason: openai.FinishReasonStop,
			},
		},
		Usage: openai.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}
}

// mockOpenAIServer creates a test HTTP server that mocks OpenAI's chat completion endpoint.
func mockOpenAIServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(handler))
}

// defaultMockHandler returns a handler that responds with a standard analysis response.
func defaultMockHandler(t *testing.T, responseContent string) func(w http.ResponseWriter, r *http.Request) {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !strings.Contains(r.URL.Path, "chat/completions") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		// Parse request body to verify it's valid
		var req openai.ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// Respond with mock data
		response := mockOpenAIResponse(responseContent, 500, 200)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}
}

// createMockOpenAIClient creates an OpenAI client that points to the test server.
func createMockOpenAIClient(serverURL string) *openai.Client {
	config := openai.DefaultConfig("test-api-key")
	config.BaseURL = serverURL + "/v1"
	return openai.NewClientWithConfig(config)
}

// ============================================================================
// Integration Tests for Analyzer.Analyze
// ============================================================================

// TestIntegration_Analyzer_Analyze_Success tests the complete analysis workflow.
func TestIntegration_Analyzer_Analyze_Success(t *testing.T) {
	// Setup mock OpenAI server
	expectedAnalysis := `# Overview
This workspace contains project planning materials.

# Insights
The notes suggest a focus on technical implementation.

# Recommendations
Consider adding more visual elements to improve clarity.`

	server := mockOpenAIServer(t, defaultMockHandler(t, expectedAnalysis))
	defer server.Close()

	// Setup mock widget client with test data
	widgets := []map[string]interface{}{
		{"id": "note-1", "type": "note", "title": "Project Plan", "text": "Phase 1: Setup"},
		{"id": "note-2", "type": "note", "title": "Tasks", "text": "Task 1, Task 2"},
		{"id": "image-1", "type": "image"},
		{"id": "pdf-1", "type": "pdf", "title": "Documentation"},
	}
	client := &mockWidgetClient{widgets: widgets}

	// Create analyzer with mock dependencies
	openaiClient := createMockOpenAIClient(server.URL)
	config := DefaultAnalyzerConfig()
	config.ProcessorConfig.Model = "gpt-4"
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, openaiClient, config, logger)

	// Run analysis
	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, "")

	// Verify results
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if result == nil {
		t.Fatal("Analyze() returned nil result")
	}

	// Check FetchResult
	if result.FetchResult == nil {
		t.Error("FetchResult should not be nil")
	} else if result.FetchResult.FilteredCount != 4 {
		t.Errorf("FilteredCount = %v, want 4", result.FetchResult.FilteredCount)
	}

	// Check Analysis
	if result.Analysis == nil {
		t.Error("Analysis should not be nil")
	} else {
		if result.Analysis.Content == "" {
			t.Error("Analysis.Content should not be empty")
		}
		if !strings.Contains(result.Analysis.Content, "Overview") {
			t.Error("Analysis.Content should contain 'Overview'")
		}
		if result.Analysis.WidgetCount != 4 {
			t.Errorf("Analysis.WidgetCount = %v, want 4", result.Analysis.WidgetCount)
		}
		if result.Analysis.PromptTokens == 0 {
			t.Error("Analysis.PromptTokens should not be 0")
		}
		if result.Analysis.CompletionTokens == 0 {
			t.Error("Analysis.CompletionTokens should not be 0")
		}
	}

	// Check timing information
	if result.TotalDuration == 0 {
		t.Error("TotalDuration should not be 0")
	}
	if result.Stages.FetchDuration == 0 {
		t.Error("Stages.FetchDuration should not be 0")
	}
	if result.Stages.AnalysisDuration == 0 {
		t.Error("Stages.AnalysisDuration should not be 0")
	}
}

// TestIntegration_Analyzer_Analyze_WithTriggerExclusion tests trigger widget exclusion.
func TestIntegration_Analyzer_Analyze_WithTriggerExclusion(t *testing.T) {
	server := mockOpenAIServer(t, defaultMockHandler(t, "Analysis result"))
	defer server.Close()

	widgets := []map[string]interface{}{
		{"id": "trigger-widget", "type": "note", "title": "Trigger"},
		{"id": "note-1", "type": "note", "title": "Content"},
		{"id": "note-2", "type": "note", "title": "More Content"},
	}
	client := &mockWidgetClient{widgets: widgets}

	openaiClient := createMockOpenAIClient(server.URL)
	config := DefaultAnalyzerConfig()
	config.ExcludeTrigger = true
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, openaiClient, config, logger)

	result, err := analyzer.Analyze(context.Background(), "trigger-widget")

	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// With trigger exclusion, should have 2 widgets instead of 3
	if result.FetchResult.FilteredCount != 2 {
		t.Errorf("FilteredCount = %v, want 2 (trigger excluded)", result.FetchResult.FilteredCount)
	}
	if result.TriggerWidgetID != "trigger-widget" {
		t.Errorf("TriggerWidgetID = %v, want trigger-widget", result.TriggerWidgetID)
	}
}

// TestIntegration_Analyzer_Analyze_WithoutTriggerExclusion tests without trigger exclusion.
func TestIntegration_Analyzer_Analyze_WithoutTriggerExclusion(t *testing.T) {
	server := mockOpenAIServer(t, defaultMockHandler(t, "Analysis result"))
	defer server.Close()

	widgets := []map[string]interface{}{
		{"id": "trigger-widget", "type": "note"},
		{"id": "note-1", "type": "note"},
	}
	client := &mockWidgetClient{widgets: widgets}

	openaiClient := createMockOpenAIClient(server.URL)
	config := DefaultAnalyzerConfig()
	config.ExcludeTrigger = false
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, openaiClient, config, logger)

	result, err := analyzer.Analyze(context.Background(), "trigger-widget")

	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// Without trigger exclusion, should have all 2 widgets
	if result.FetchResult.FilteredCount != 2 {
		t.Errorf("FilteredCount = %v, want 2 (no exclusion)", result.FetchResult.FilteredCount)
	}
}

// TestIntegration_Analyzer_Analyze_ProgressCallback tests progress reporting.
func TestIntegration_Analyzer_Analyze_ProgressCallback(t *testing.T) {
	server := mockOpenAIServer(t, defaultMockHandler(t, "Analysis"))
	defer server.Close()

	widgets := []map[string]interface{}{
		{"id": "1", "type": "note"},
	}
	client := &mockWidgetClient{widgets: widgets}

	openaiClient := createMockOpenAIClient(server.URL)
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, openaiClient, config, logger)

	// Track progress callbacks
	var progressCalls []struct {
		stage   string
		message string
	}

	analyzer.SetProgressCallback(func(stage, message string) {
		progressCalls = append(progressCalls, struct {
			stage   string
			message string
		}{stage, message})
	})

	_, err := analyzer.Analyze(context.Background(), "")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// Should have at least fetch, analysis, and complete stages
	if len(progressCalls) < 3 {
		t.Errorf("Expected at least 3 progress calls, got %d", len(progressCalls))
	}

	// Check stages
	stages := make([]string, len(progressCalls))
	for i, call := range progressCalls {
		stages[i] = call.stage
	}

	expectedStages := []string{"fetch", "analysis", "complete"}
	for _, expected := range expectedStages {
		found := false
		for _, actual := range stages {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected stage %q in progress calls: %v", expected, stages)
		}
	}
}

// ============================================================================
// Integration Tests for Analyzer.AnalyzeWithPrompt
// ============================================================================

// TestIntegration_Analyzer_AnalyzeWithPrompt_Success tests custom prompt analysis.
func TestIntegration_Analyzer_AnalyzeWithPrompt_Success(t *testing.T) {
	// Capture the request to verify custom prompt was used
	var receivedRequest openai.ChatCompletionRequest

	server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedRequest); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		response := mockOpenAIResponse("Custom analysis result", 100, 50)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	widgets := []map[string]interface{}{
		{"id": "1", "type": "note", "text": "Test content"},
	}
	client := &mockWidgetClient{widgets: widgets}

	openaiClient := createMockOpenAIClient(server.URL)
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, openaiClient, config, logger)

	customPrompt := "You are a custom analyzer. Be brief and technical."

	result, err := analyzer.AnalyzeWithPrompt(context.Background(), "", customPrompt)

	if err != nil {
		t.Fatalf("AnalyzeWithPrompt() error = %v", err)
	}
	if result == nil {
		t.Fatal("AnalyzeWithPrompt() returned nil result")
	}

	// Verify custom prompt was used
	if len(receivedRequest.Messages) < 1 {
		t.Fatal("No messages in request")
	}
	systemMessage := receivedRequest.Messages[0].Content
	if systemMessage != customPrompt {
		t.Errorf("System prompt = %q, want %q", systemMessage, customPrompt)
	}

	// Verify original prompt is restored (test by checking config hasn't changed)
	gotConfig := analyzer.GetConfig()
	if gotConfig.ProcessorConfig.SystemPrompt != DefaultSystemPrompt {
		t.Error("Original system prompt should be restored after AnalyzeWithPrompt")
	}
}

// ============================================================================
// Integration Tests for Analyzer.AnalyzeWidgets
// ============================================================================

// TestIntegration_Analyzer_AnalyzeWidgets_Success tests direct widget analysis.
func TestIntegration_Analyzer_AnalyzeWidgets_Success(t *testing.T) {
	server := mockOpenAIServer(t, defaultMockHandler(t, "Direct widget analysis"))
	defer server.Close()

	// Create mock client (not used directly, but needed for Analyzer construction)
	client := &mockWidgetClient{}

	openaiClient := createMockOpenAIClient(server.URL)
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, openaiClient, config, logger)

	// Create widgets directly (bypassing Fetcher)
	widgets := []Widget{
		{"id": "w1", "type": "note", "title": "Widget 1", "text": "Content 1"},
		{"id": "w2", "type": "note", "title": "Widget 2", "text": "Content 2"},
		{"id": "w3", "type": "image"},
	}

	result, err := analyzer.AnalyzeWidgets(context.Background(), widgets)

	if err != nil {
		t.Fatalf("AnalyzeWidgets() error = %v", err)
	}
	if result == nil {
		t.Fatal("AnalyzeWidgets() returned nil result")
	}
	if result.Content == "" {
		t.Error("AnalysisResult.Content should not be empty")
	}
	if result.WidgetCount != 3 {
		t.Errorf("WidgetCount = %v, want 3", result.WidgetCount)
	}
}

// TestIntegration_Analyzer_AnalyzeWidgets_ProgressCallback tests progress during direct analysis.
func TestIntegration_Analyzer_AnalyzeWidgets_ProgressCallback(t *testing.T) {
	server := mockOpenAIServer(t, defaultMockHandler(t, "Analysis"))
	defer server.Close()

	client := &mockWidgetClient{}
	openaiClient := createMockOpenAIClient(server.URL)
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, openaiClient, config, logger)

	var progressCalls int
	analyzer.SetProgressCallback(func(stage, message string) {
		progressCalls++
	})

	widgets := []Widget{
		{"id": "w1", "type": "note"},
	}

	_, err := analyzer.AnalyzeWidgets(context.Background(), widgets)
	if err != nil {
		t.Fatalf("AnalyzeWidgets() error = %v", err)
	}

	// AnalyzeWidgets should report "analysis" and "complete" stages
	if progressCalls < 2 {
		t.Errorf("Expected at least 2 progress calls, got %d", progressCalls)
	}
}

// ============================================================================
// Integration Tests for Error Handling
// ============================================================================

// TestIntegration_Analyzer_Analyze_OpenAIError tests handling of OpenAI API errors.
func TestIntegration_Analyzer_Analyze_OpenAIError(t *testing.T) {
	// Server that returns an error
	server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Internal server error",
				"type":    "server_error",
			},
		})
	})
	defer server.Close()

	widgets := []map[string]interface{}{
		{"id": "1", "type": "note"},
	}
	client := &mockWidgetClient{widgets: widgets}

	openaiClient := createMockOpenAIClient(server.URL)
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, openaiClient, config, logger)

	result, err := analyzer.Analyze(context.Background(), "")

	if err == nil {
		t.Error("Expected error for OpenAI failure, got nil")
	}
	if result != nil {
		t.Error("Result should be nil on error")
	}
}

// TestIntegration_Analyzer_Analyze_EmptyOpenAIResponse tests handling of empty responses.
func TestIntegration_Analyzer_Analyze_EmptyOpenAIResponse(t *testing.T) {
	// Server that returns an empty choices array
	server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := openai.ChatCompletionResponse{
			ID:      "test",
			Choices: []openai.ChatCompletionChoice{}, // Empty
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	widgets := []map[string]interface{}{
		{"id": "1", "type": "note"},
	}
	client := &mockWidgetClient{widgets: widgets}

	openaiClient := createMockOpenAIClient(server.URL)
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, openaiClient, config, logger)

	result, err := analyzer.Analyze(context.Background(), "")

	if err == nil {
		t.Error("Expected error for empty response, got nil")
	}
	if result != nil {
		t.Error("Result should be nil on error")
	}
}

// TestIntegration_Analyzer_Analyze_ContextTimeout tests context timeout handling.
func TestIntegration_Analyzer_Analyze_ContextTimeout(t *testing.T) {
	// Server that delays response
	server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond) // Delay
		response := mockOpenAIResponse("Delayed", 100, 50)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	widgets := []map[string]interface{}{
		{"id": "1", "type": "note"},
	}
	client := &mockWidgetClient{widgets: widgets}

	openaiClient := createMockOpenAIClient(server.URL)
	config := DefaultAnalyzerConfig()
	config.ProcessorConfig.Timeout = 100 * time.Millisecond // Short timeout
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, openaiClient, config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := analyzer.Analyze(ctx, "")

	// Should get an error (either context timeout or analysis failure)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if result != nil {
		t.Error("Result should be nil on timeout")
	}
}

// ============================================================================
// Integration Tests for Complete Workflow
// ============================================================================

// TestIntegration_CompleteWorkflow tests a realistic end-to-end scenario.
func TestIntegration_CompleteWorkflow(t *testing.T) {
	// Track request for verification
	var requestCount int
	var lastRequest openai.ChatCompletionRequest

	server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		json.NewDecoder(r.Body).Decode(&lastRequest)

		response := mockOpenAIResponse(`# Overview
The workspace contains a mix of project planning notes, images, and documentation.

# Insights
- Notes focus on technical tasks and milestones
- Images likely serve as visual references
- The PDF contains supporting documentation

# Recommendations
1. Add due dates to task notes
2. Create connectors between related items
3. Group items by project phase`, 800, 300)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	// Realistic widget set
	widgets := []map[string]interface{}{
		{"id": "note-1", "type": "note", "title": "Sprint Goals", "text": "Complete API integration\nWrite tests\nDeploy to staging"},
		{"id": "note-2", "type": "note", "title": "Tech Stack", "text": "Go, React, PostgreSQL"},
		{"id": "image-1", "type": "image", "title": "Architecture Diagram"},
		{"id": "pdf-1", "type": "pdf", "title": "API Specification"},
		{"id": "browser-1", "type": "browser", "url": "https://docs.example.com"},
		{"id": "trigger-icon", "type": "icon", "title": "Analyze"},
	}
	client := &mockWidgetClient{widgets: widgets}

	openaiClient := createMockOpenAIClient(server.URL)
	config := DefaultAnalyzerConfig()
	config.ExcludeTrigger = true
	config.FetcherConfig.FilterTypes = []string{"note", "image", "pdf"} // Only analyze these types
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, openaiClient, config, logger)

	// Track progress
	var progressStages []string
	analyzer.SetProgressCallback(func(stage, message string) {
		progressStages = append(progressStages, stage)
		t.Logf("Progress: [%s] %s", stage, message)
	})

	// Run analysis
	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, "trigger-icon")

	if err != nil {
		t.Fatalf("Complete workflow failed: %v", err)
	}

	// Verify request was made
	if requestCount != 1 {
		t.Errorf("Expected 1 OpenAI request, got %d", requestCount)
	}

	// Verify widgets were filtered (browser and trigger excluded)
	expectedWidgetCount := 4 // note-1, note-2, image-1, pdf-1
	if result.FetchResult.FilteredCount != expectedWidgetCount {
		t.Errorf("FilteredCount = %v, want %d", result.FetchResult.FilteredCount, expectedWidgetCount)
	}

	// Verify analysis content
	if result.Analysis == nil {
		t.Fatal("Analysis should not be nil")
	}
	if !strings.Contains(result.Analysis.Content, "Overview") {
		t.Error("Analysis should contain 'Overview' section")
	}
	if !strings.Contains(result.Analysis.Content, "Recommendations") {
		t.Error("Analysis should contain 'Recommendations' section")
	}

	// Verify token usage
	if result.Analysis.PromptTokens < 100 {
		t.Errorf("PromptTokens = %d, expected more than 100", result.Analysis.PromptTokens)
	}

	// Verify progress was reported
	if len(progressStages) < 3 {
		t.Errorf("Expected at least 3 progress stages, got %d", len(progressStages))
	}

	// Verify timing
	t.Logf("Total duration: %v", result.TotalDuration)
	t.Logf("Fetch duration: %v", result.Stages.FetchDuration)
	t.Logf("Analysis duration: %v", result.Stages.AnalysisDuration)
}

// TestIntegration_MultipleAnalyses tests running multiple analyses in sequence.
func TestIntegration_MultipleAnalyses(t *testing.T) {
	var requestCount int

	server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		response := mockOpenAIResponse("Analysis "+string(rune('A'+requestCount-1)), 100, 50)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	widgets := []map[string]interface{}{
		{"id": "1", "type": "note"},
	}
	client := &mockWidgetClient{widgets: widgets}

	openaiClient := createMockOpenAIClient(server.URL)
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, openaiClient, config, logger)

	// Run multiple analyses
	for i := 0; i < 3; i++ {
		result, err := analyzer.Analyze(context.Background(), "")
		if err != nil {
			t.Fatalf("Analysis %d failed: %v", i+1, err)
		}
		if result == nil {
			t.Fatalf("Analysis %d returned nil", i+1)
		}
	}

	if requestCount != 3 {
		t.Errorf("Expected 3 OpenAI requests, got %d", requestCount)
	}
}

// TestIntegration_JSONWrappedResponse tests handling of JSON-wrapped AI responses.
func TestIntegration_JSONWrappedResponse(t *testing.T) {
	// Some AI models wrap responses in JSON
	jsonWrappedResponse := `{"content": "This is the actual analysis content", "metadata": {"processed": true}}`

	server := mockOpenAIServer(t, defaultMockHandler(t, jsonWrappedResponse))
	defer server.Close()

	widgets := []map[string]interface{}{
		{"id": "1", "type": "note"},
	}
	client := &mockWidgetClient{widgets: widgets}

	openaiClient := createMockOpenAIClient(server.URL)
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, openaiClient, config, logger)

	result, err := analyzer.Analyze(context.Background(), "")

	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// The processor should extract content from JSON
	if result.Analysis.Content == "" {
		t.Error("Analysis.Content should not be empty")
	}
	// Content could be either the extracted content or the raw response
	// depending on whether extractJSONContent succeeds
	t.Logf("Extracted content: %s", result.Analysis.Content)
}
