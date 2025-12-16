package canvasanalyzer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// ErrNotConfigured is returned when the Analyzer is missing required configuration.
var ErrNotConfigured = errors.New("canvasanalyzer: analyzer not properly configured")

// ErrAnalyzerCancelled is returned when analysis is cancelled via context.
var ErrAnalyzerCancelled = errors.New("canvasanalyzer: analysis cancelled")

// AnalyzerConfig holds configuration for the complete Analyzer organism.
type AnalyzerConfig struct {
	// FetcherConfig configures the widget fetcher
	FetcherConfig FetcherConfig

	// ProcessorConfig configures the AI processor
	ProcessorConfig ProcessorConfig

	// ExcludeTrigger controls whether to automatically exclude the trigger widget
	ExcludeTrigger bool
}

// DefaultAnalyzerConfig returns sensible default configuration.
func DefaultAnalyzerConfig() AnalyzerConfig {
	return AnalyzerConfig{
		FetcherConfig:   DefaultFetcherConfig(),
		ProcessorConfig: DefaultProcessorConfig(),
		ExcludeTrigger:  true,
	}
}

// AnalyzeResult contains the complete result of a canvas analysis operation.
type AnalyzeResult struct {
	// Analysis is the AI-generated analysis result
	Analysis *AnalysisResult

	// FetchResult contains details about widget fetching
	FetchResult *FetchResult

	// TotalDuration is the total time for the complete operation
	TotalDuration time.Duration

	// TriggerWidgetID is the ID of the widget that triggered the analysis (if any)
	TriggerWidgetID string

	// Stages contains timing for each stage
	Stages AnalyzeStages
}

// AnalyzeStages contains timing information for each analysis stage.
type AnalyzeStages struct {
	FetchDuration    time.Duration
	AnalysisDuration time.Duration
}

// ProgressCallback is called to report progress during analysis.
// stage is the current stage name, message is a human-readable status.
type ProgressCallback func(stage string, message string)

// Analyzer orchestrates canvas analysis by combining the Fetcher and Processor molecules.
// It provides a simple interface for performing end-to-end canvas analysis.
type Analyzer struct {
	config    AnalyzerConfig
	fetcher   *Fetcher
	processor *Processor
	logger    *zap.Logger
	progress  ProgressCallback
}

// NewAnalyzer creates a new Analyzer with the given configuration.
//
// The client must be a valid WidgetClient for fetching canvas widgets.
// The openaiClient must be a valid OpenAI client for AI analysis.
//
// Example:
//
//	analyzer := NewAnalyzer(
//	    canvusClient,
//	    openaiClient,
//	    DefaultAnalyzerConfig(),
//	    logger,
//	)
//	result, err := analyzer.Analyze(ctx, "trigger-widget-id")
func NewAnalyzer(
	client WidgetClient,
	openaiClient *openai.Client,
	config AnalyzerConfig,
	logger *zap.Logger,
) *Analyzer {
	fetcher := NewFetcher(client, config.FetcherConfig, logger)
	processor := NewProcessor(config.ProcessorConfig, openaiClient, logger)

	return &Analyzer{
		config:    config,
		fetcher:   fetcher,
		processor: processor,
		logger:    logger,
	}
}

// NewAnalyzerWithComponents creates an Analyzer using pre-built Fetcher and Processor.
// This is useful for testing or when you need custom molecule configurations.
//
// Example:
//
//	fetcher := NewFetcher(client, customFetcherConfig, logger)
//	processor := NewProcessor(customProcessorConfig, openaiClient, logger)
//	analyzer := NewAnalyzerWithComponents(fetcher, processor, config, logger)
func NewAnalyzerWithComponents(
	fetcher *Fetcher,
	processor *Processor,
	config AnalyzerConfig,
	logger *zap.Logger,
) *Analyzer {
	return &Analyzer{
		config:    config,
		fetcher:   fetcher,
		processor: processor,
		logger:    logger,
	}
}

// SetProgressCallback sets a callback function for progress updates.
func (a *Analyzer) SetProgressCallback(callback ProgressCallback) {
	a.progress = callback
}

// Analyze performs a complete canvas analysis.
//
// If triggerWidgetID is provided and ExcludeTrigger is true (default), that widget
// will be excluded from the analysis. This prevents the trigger widget (e.g., an icon
// the user clicked) from appearing in the analysis results.
//
// The operation can be cancelled via the context.
func (a *Analyzer) Analyze(ctx context.Context, triggerWidgetID string) (*AnalyzeResult, error) {
	start := time.Now()

	a.logger.Info("starting canvas analysis",
		zap.String("trigger_widget_id", triggerWidgetID),
		zap.Bool("exclude_trigger", a.config.ExcludeTrigger))

	result := &AnalyzeResult{
		TriggerWidgetID: triggerWidgetID,
	}

	// Stage 1: Fetch widgets
	a.reportProgress("fetch", "Fetching canvas widgets...")

	fetchStart := time.Now()
	var fetchResult *FetchResult
	var err error

	if triggerWidgetID != "" && a.config.ExcludeTrigger {
		fetchResult, err = a.fetcher.FetchWithExclusions(ctx, triggerWidgetID)
	} else {
		fetchResult, err = a.fetcher.Fetch(ctx)
	}

	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, ErrAnalyzerCancelled
		}
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	result.FetchResult = fetchResult
	result.Stages.FetchDuration = time.Since(fetchStart)

	a.logger.Info("widgets fetched",
		zap.Int("count", fetchResult.FilteredCount),
		zap.Duration("duration", result.Stages.FetchDuration))

	// Check for empty canvas
	if len(fetchResult.Widgets) == 0 {
		a.logger.Warn("no widgets to analyze")
		// Return a result with no analysis rather than error
		result.TotalDuration = time.Since(start)
		return result, nil
	}

	// Stage 2: Generate AI analysis
	a.reportProgress("analysis", fmt.Sprintf("Analyzing %d widgets...", fetchResult.FilteredCount))

	analysisStart := time.Now()
	analysis, err := a.processor.Analyze(ctx, fetchResult.Widgets)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, ErrAnalyzerCancelled
		}
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	result.Analysis = analysis
	result.Stages.AnalysisDuration = time.Since(analysisStart)
	result.TotalDuration = time.Since(start)

	a.logger.Info("canvas analysis completed",
		zap.Int("widget_count", fetchResult.FilteredCount),
		zap.Int("prompt_tokens", analysis.PromptTokens),
		zap.Int("completion_tokens", analysis.CompletionTokens),
		zap.Duration("total_duration", result.TotalDuration))

	a.reportProgress("complete", "Analysis complete")

	return result, nil
}

// AnalyzeWithPrompt performs canvas analysis using a custom system prompt.
func (a *Analyzer) AnalyzeWithPrompt(ctx context.Context, triggerWidgetID, systemPrompt string) (*AnalyzeResult, error) {
	// Temporarily override the system prompt
	originalPrompt := a.processor.config.SystemPrompt
	a.processor.SetSystemPrompt(systemPrompt)
	defer a.processor.SetSystemPrompt(originalPrompt)

	return a.Analyze(ctx, triggerWidgetID)
}

// AnalyzeWidgets performs AI analysis on a pre-fetched list of widgets.
// This is useful when you've already fetched widgets and want to analyze them directly.
func (a *Analyzer) AnalyzeWidgets(ctx context.Context, widgets []Widget) (*AnalysisResult, error) {
	a.reportProgress("analysis", fmt.Sprintf("Analyzing %d widgets...", len(widgets)))

	analysis, err := a.processor.Analyze(ctx, widgets)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, ErrAnalyzerCancelled
		}
		return nil, err
	}

	a.reportProgress("complete", "Analysis complete")
	return analysis, nil
}

// reportProgress calls the progress callback if set.
func (a *Analyzer) reportProgress(stage, message string) {
	if a.progress != nil {
		a.progress(stage, message)
	}
}

// GetConfig returns a copy of the current configuration.
func (a *Analyzer) GetConfig() AnalyzerConfig {
	return a.config
}

// SetExcludeTrigger updates whether to exclude the trigger widget from analysis.
func (a *Analyzer) SetExcludeTrigger(exclude bool) {
	a.config.ExcludeTrigger = exclude
}

// GetFetcher returns the underlying Fetcher molecule.
// This allows direct access for advanced configuration.
func (a *Analyzer) GetFetcher() *Fetcher {
	return a.fetcher
}

// GetProcessor returns the underlying Processor molecule.
// This allows direct access for advanced configuration.
func (a *Analyzer) GetProcessor() *Processor {
	return a.processor
}
