package canvasanalyzer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ErrFetchFailed is returned when widget fetching fails after all retries.
var ErrFetchFailed = errors.New("canvasanalyzer: failed to fetch widgets")

// ErrNoWidgets is returned when the canvas has no widgets.
var ErrNoWidgets = errors.New("canvasanalyzer: no widgets found on canvas")

// WidgetClient is the interface for fetching widgets from the Canvus API.
// This enables dependency injection and testing without a real API client.
type WidgetClient interface {
	// GetWidgets retrieves all widgets from the canvas.
	// If subscribe is true, it establishes a streaming connection.
	GetWidgets(subscribe bool) ([]map[string]interface{}, error)
}

// FetcherConfig holds configuration for the Widget Fetcher.
type FetcherConfig struct {
	// MaxRetries is the maximum number of retry attempts (default: 3)
	MaxRetries int

	// RetryDelay is the delay between retry attempts (default: 2s)
	RetryDelay time.Duration

	// ExcludeIDs is a list of widget IDs to exclude from results
	ExcludeIDs []string

	// FilterTypes limits results to specific widget types (empty = all types)
	FilterTypes []string
}

// DefaultFetcherConfig returns sensible default configuration.
func DefaultFetcherConfig() FetcherConfig {
	return FetcherConfig{
		MaxRetries: 3,
		RetryDelay: 2 * time.Second,
		ExcludeIDs: nil,
		FilterTypes: nil,
	}
}

// FetchResult contains the result of a widget fetch operation.
type FetchResult struct {
	// Widgets is the list of fetched widgets
	Widgets []Widget

	// TotalCount is the number of widgets before filtering
	TotalCount int

	// FilteredCount is the number of widgets after filtering
	FilteredCount int

	// Attempts is the number of fetch attempts made
	Attempts int

	// Duration is the time taken to fetch widgets
	Duration time.Duration
}

// Fetcher retrieves canvas widgets with retry logic and filtering.
type Fetcher struct {
	client WidgetClient
	config FetcherConfig
	logger *zap.Logger
}

// NewFetcher creates a new Fetcher with the given client and configuration.
//
// Example:
//
//	fetcher := NewFetcher(canvusClient, DefaultFetcherConfig(), logger)
//	result, err := fetcher.Fetch(ctx)
func NewFetcher(client WidgetClient, config FetcherConfig, logger *zap.Logger) *Fetcher {
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = 2 * time.Second
	}

	return &Fetcher{
		client: client,
		config: config,
		logger: logger,
	}
}

// Fetch retrieves widgets from the canvas with retry logic.
// The context can be used to cancel the operation.
//
// Returns ErrFetchFailed if all retry attempts fail.
// Returns ErrNoWidgets if the canvas has no widgets.
func (f *Fetcher) Fetch(ctx context.Context) (*FetchResult, error) {
	start := time.Now()
	var lastErr error

	for attempt := 1; attempt <= f.config.MaxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		f.logger.Debug("fetching widgets",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", f.config.MaxRetries))

		rawWidgets, err := f.client.GetWidgets(false)
		if err == nil {
			// Success - process the widgets
			result := f.processWidgets(rawWidgets, attempt, time.Since(start))

			f.logger.Info("widgets fetched successfully",
				zap.Int("total", result.TotalCount),
				zap.Int("filtered", result.FilteredCount),
				zap.Int("attempts", result.Attempts),
				zap.Duration("duration", result.Duration))

			return result, nil
		}

		// Log failure and retry
		lastErr = err
		f.logger.Warn("widget fetch attempt failed",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", f.config.MaxRetries),
			zap.Error(err))

		// Wait before retry (unless this was the last attempt)
		if attempt < f.config.MaxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(f.config.RetryDelay):
			}
		}
	}

	return nil, fmt.Errorf("%w after %d attempts: %v",
		ErrFetchFailed, f.config.MaxRetries, lastErr)
}

// FetchWithExclusions is a convenience method that fetches widgets while excluding
// the specified widget IDs. This is useful for excluding trigger widgets from analysis.
func (f *Fetcher) FetchWithExclusions(ctx context.Context, excludeIDs ...string) (*FetchResult, error) {
	// Create a copy of config with additional exclusions
	originalExclusions := f.config.ExcludeIDs
	f.config.ExcludeIDs = append(f.config.ExcludeIDs, excludeIDs...)

	defer func() {
		f.config.ExcludeIDs = originalExclusions
	}()

	return f.Fetch(ctx)
}

// processWidgets converts raw widgets and applies filtering.
func (f *Fetcher) processWidgets(rawWidgets []map[string]interface{}, attempts int, duration time.Duration) *FetchResult {
	// Convert to Widget type
	widgets := make([]Widget, len(rawWidgets))
	for i, raw := range rawWidgets {
		widgets[i] = Widget(raw)
	}

	totalCount := len(widgets)

	// Apply ID exclusions
	if len(f.config.ExcludeIDs) > 0 {
		widgets = FilterWidgets(widgets, f.config.ExcludeIDs...)
	}

	// Apply type filter
	if len(f.config.FilterTypes) > 0 {
		widgets = FilterWidgetsByType(widgets, f.config.FilterTypes...)
	}

	return &FetchResult{
		Widgets:       widgets,
		TotalCount:    totalCount,
		FilteredCount: len(widgets),
		Attempts:      attempts,
		Duration:      duration,
	}
}

// GetConfig returns a copy of the current configuration.
func (f *Fetcher) GetConfig() FetcherConfig {
	return f.config
}

// SetExcludeIDs updates the list of widget IDs to exclude.
func (f *Fetcher) SetExcludeIDs(ids []string) {
	f.config.ExcludeIDs = ids
}

// SetFilterTypes updates the list of widget types to include.
func (f *Fetcher) SetFilterTypes(types []string) {
	f.config.FilterTypes = types
}
