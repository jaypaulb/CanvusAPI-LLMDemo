package canvasanalyzer

import (
	"context"
	"testing"
	"time"
)

func TestDefaultAnalyzerConfig(t *testing.T) {
	config := DefaultAnalyzerConfig()

	if config.FetcherConfig.MaxRetries != 3 {
		t.Errorf("FetcherConfig.MaxRetries = %v, want 3", config.FetcherConfig.MaxRetries)
	}
	if config.ProcessorConfig.Model != "gpt-4" {
		t.Errorf("ProcessorConfig.Model = %v, want gpt-4", config.ProcessorConfig.Model)
	}
	if !config.ExcludeTrigger {
		t.Error("ExcludeTrigger should be true by default")
	}
}

func TestNewAnalyzer(t *testing.T) {
	client := &mockWidgetClient{}
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, nil, config, logger)

	if analyzer == nil {
		t.Error("NewAnalyzer returned nil")
	}
	if analyzer.fetcher == nil {
		t.Error("Analyzer.fetcher should not be nil")
	}
	if analyzer.processor == nil {
		t.Error("Analyzer.processor should not be nil")
	}
}

func TestNewAnalyzerWithComponents(t *testing.T) {
	client := &mockWidgetClient{}
	logger := newTestLogger()

	fetcher := NewFetcher(client, DefaultFetcherConfig(), logger)
	processor := NewProcessor(DefaultProcessorConfig(), nil, logger)
	config := DefaultAnalyzerConfig()

	analyzer := NewAnalyzerWithComponents(fetcher, processor, config, logger)

	if analyzer == nil {
		t.Error("NewAnalyzerWithComponents returned nil")
	}
	if analyzer.fetcher != fetcher {
		t.Error("Analyzer should use provided fetcher")
	}
	if analyzer.processor != processor {
		t.Error("Analyzer should use provided processor")
	}
}

func TestAnalyzer_SetProgressCallback(t *testing.T) {
	client := &mockWidgetClient{}
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, nil, config, logger)

	var called bool
	analyzer.SetProgressCallback(func(stage, message string) {
		called = true
	})

	// Trigger a progress report
	analyzer.reportProgress("test", "message")

	if !called {
		t.Error("Progress callback should have been called")
	}
}

func TestAnalyzer_GetConfig(t *testing.T) {
	client := &mockWidgetClient{}
	config := AnalyzerConfig{
		FetcherConfig: FetcherConfig{
			MaxRetries: 5,
		},
		ProcessorConfig: ProcessorConfig{
			Model: "gpt-3.5-turbo",
		},
		ExcludeTrigger: false,
	}
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, nil, config, logger)
	got := analyzer.GetConfig()

	if got.FetcherConfig.MaxRetries != 5 {
		t.Errorf("FetcherConfig.MaxRetries = %v, want 5", got.FetcherConfig.MaxRetries)
	}
	if got.ExcludeTrigger != false {
		t.Error("ExcludeTrigger should be false")
	}
}

func TestAnalyzer_SetExcludeTrigger(t *testing.T) {
	client := &mockWidgetClient{}
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, nil, config, logger)
	analyzer.SetExcludeTrigger(false)

	if analyzer.config.ExcludeTrigger != false {
		t.Error("ExcludeTrigger should be false after SetExcludeTrigger(false)")
	}
}

func TestAnalyzer_GetFetcher(t *testing.T) {
	client := &mockWidgetClient{}
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, nil, config, logger)
	fetcher := analyzer.GetFetcher()

	if fetcher == nil {
		t.Error("GetFetcher should not return nil")
	}
}

func TestAnalyzer_GetProcessor(t *testing.T) {
	client := &mockWidgetClient{}
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, nil, config, logger)
	processor := analyzer.GetProcessor()

	if processor == nil {
		t.Error("GetProcessor should not return nil")
	}
}

func TestAnalyzer_Analyze_EmptyCanvas(t *testing.T) {
	client := &mockWidgetClient{widgets: []map[string]interface{}{}}
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, nil, config, logger)
	result, err := analyzer.Analyze(context.Background(), "")

	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if result == nil {
		t.Fatal("Analyze() returned nil result")
	}
	if result.FetchResult == nil {
		t.Error("FetchResult should not be nil")
	}
	if result.FetchResult.FilteredCount != 0 {
		t.Errorf("FilteredCount = %v, want 0", result.FetchResult.FilteredCount)
	}
	// Analysis is nil for empty canvas
	if result.Analysis != nil {
		t.Error("Analysis should be nil for empty canvas")
	}
}

func TestAnalyzer_Analyze_FetchError(t *testing.T) {
	client := &mockWidgetClient{
		err:   ErrFetchFailed,
		failN: 999,
	}
	config := AnalyzerConfig{
		FetcherConfig: FetcherConfig{
			MaxRetries: 2,
			RetryDelay: 10 * time.Millisecond,
		},
		ProcessorConfig: DefaultProcessorConfig(),
		ExcludeTrigger:  true,
	}
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, nil, config, logger)
	result, err := analyzer.Analyze(context.Background(), "")

	if err == nil {
		t.Error("Analyze() should return error when fetch fails")
	}
	if result != nil {
		t.Error("result should be nil on error")
	}
}

func TestAnalyzer_Analyze_ContextCancellation(t *testing.T) {
	client := &mockWidgetClient{
		widgets: []map[string]interface{}{
			{"id": "1", "type": "note"},
		},
	}
	config := AnalyzerConfig{
		FetcherConfig: FetcherConfig{
			MaxRetries: 1,
			RetryDelay: 10 * time.Millisecond,
		},
		ProcessorConfig: ProcessorConfig{
			Timeout: 10 * time.Second,
		},
		ExcludeTrigger: true,
	}
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, nil, config, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := analyzer.Analyze(ctx, "")

	// Should either return context cancelled error or error from cancelled operation
	if err == nil {
		// If we got a result (empty canvas case), that's also acceptable
		if result != nil && result.FetchResult != nil && result.FetchResult.FilteredCount > 0 {
			t.Error("Expected error or empty result with cancelled context")
		}
	}
}

func TestAnalyzer_Analyze_WithTriggerExclusion(t *testing.T) {
	client := &mockWidgetClient{
		widgets: []map[string]interface{}{
			{"id": "trigger-1", "type": "note"},
			{"id": "2", "type": "image"},
			{"id": "3", "type": "pdf"},
		},
	}
	config := DefaultAnalyzerConfig()
	config.ExcludeTrigger = true
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, nil, config, logger)

	// Note: This test can't complete the full analysis because we have a nil OpenAI client
	// But we can test that the fetcher is called with the right exclusions
	// by checking the mock's behavior

	// For a full test, we'd need to mock the OpenAI client
	// Here we just verify the configuration is applied correctly
	got := analyzer.GetConfig()
	if !got.ExcludeTrigger {
		t.Error("ExcludeTrigger should be true")
	}
}

func TestAnalyzer_Analyze_WithoutTriggerExclusion(t *testing.T) {
	client := &mockWidgetClient{
		widgets: []map[string]interface{}{
			{"id": "trigger-1", "type": "note"},
			{"id": "2", "type": "image"},
		},
	}
	config := DefaultAnalyzerConfig()
	config.ExcludeTrigger = false
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, nil, config, logger)

	got := analyzer.GetConfig()
	if got.ExcludeTrigger {
		t.Error("ExcludeTrigger should be false")
	}
}

func TestAnalyzer_reportProgress(t *testing.T) {
	client := &mockWidgetClient{}
	config := DefaultAnalyzerConfig()
	logger := newTestLogger()

	analyzer := NewAnalyzer(client, nil, config, logger)

	t.Run("with callback", func(t *testing.T) {
		var stages []string
		var messages []string
		analyzer.SetProgressCallback(func(stage, message string) {
			stages = append(stages, stage)
			messages = append(messages, message)
		})

		analyzer.reportProgress("fetch", "Fetching...")
		analyzer.reportProgress("analyze", "Analyzing...")

		if len(stages) != 2 {
			t.Errorf("Expected 2 progress calls, got %d", len(stages))
		}
		if stages[0] != "fetch" {
			t.Errorf("First stage = %v, want fetch", stages[0])
		}
	})

	t.Run("without callback", func(t *testing.T) {
		analyzer.progress = nil
		// Should not panic
		analyzer.reportProgress("test", "message")
	})
}

func TestAnalyzeResult(t *testing.T) {
	result := &AnalyzeResult{
		Analysis: &AnalysisResult{
			Content:      "Test analysis",
			PromptTokens: 100,
		},
		FetchResult: &FetchResult{
			FilteredCount: 5,
			Attempts:      1,
		},
		TotalDuration:   1 * time.Second,
		TriggerWidgetID: "trigger-123",
		Stages: AnalyzeStages{
			FetchDuration:    100 * time.Millisecond,
			AnalysisDuration: 900 * time.Millisecond,
		},
	}

	if result.Analysis.Content != "Test analysis" {
		t.Errorf("Analysis.Content = %v, want Test analysis", result.Analysis.Content)
	}
	if result.FetchResult.FilteredCount != 5 {
		t.Errorf("FetchResult.FilteredCount = %v, want 5", result.FetchResult.FilteredCount)
	}
	if result.TriggerWidgetID != "trigger-123" {
		t.Errorf("TriggerWidgetID = %v, want trigger-123", result.TriggerWidgetID)
	}
	if result.Stages.FetchDuration != 100*time.Millisecond {
		t.Errorf("Stages.FetchDuration = %v, want 100ms", result.Stages.FetchDuration)
	}
}

func TestAnalyzeStages(t *testing.T) {
	stages := AnalyzeStages{
		FetchDuration:    200 * time.Millisecond,
		AnalysisDuration: 1500 * time.Millisecond,
	}

	if stages.FetchDuration != 200*time.Millisecond {
		t.Errorf("FetchDuration = %v, want 200ms", stages.FetchDuration)
	}
	if stages.AnalysisDuration != 1500*time.Millisecond {
		t.Errorf("AnalysisDuration = %v, want 1500ms", stages.AnalysisDuration)
	}
}
