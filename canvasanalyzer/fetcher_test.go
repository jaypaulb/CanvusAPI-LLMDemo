package canvasanalyzer

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

// mockWidgetClient implements WidgetClient for testing.
type mockWidgetClient struct {
	widgets []map[string]interface{}
	err     error
	calls   int
	failN   int // Fail first N calls
}

func (m *mockWidgetClient) GetWidgets(subscribe bool) ([]map[string]interface{}, error) {
	m.calls++
	if m.failN > 0 && m.calls <= m.failN {
		return nil, m.err
	}
	return m.widgets, nil
}

func newTestLogger() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel) // Reduce noise in tests
	logger, _ := cfg.Build()
	return logger
}

func TestDefaultFetcherConfig(t *testing.T) {
	config := DefaultFetcherConfig()

	if config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %v, want 3", config.MaxRetries)
	}
	if config.RetryDelay != 2*time.Second {
		t.Errorf("RetryDelay = %v, want 2s", config.RetryDelay)
	}
	if config.ExcludeIDs != nil {
		t.Errorf("ExcludeIDs should be nil by default")
	}
	if config.FilterTypes != nil {
		t.Errorf("FilterTypes should be nil by default")
	}
}

func TestNewFetcher(t *testing.T) {
	client := &mockWidgetClient{}
	logger := newTestLogger()

	t.Run("with valid config", func(t *testing.T) {
		config := DefaultFetcherConfig()
		fetcher := NewFetcher(client, config, logger)
		if fetcher == nil {
			t.Error("NewFetcher returned nil")
		}
	})

	t.Run("with zero retries defaults to 3", func(t *testing.T) {
		config := FetcherConfig{MaxRetries: 0}
		fetcher := NewFetcher(client, config, logger)
		if fetcher.config.MaxRetries != 3 {
			t.Errorf("MaxRetries = %v, want 3", fetcher.config.MaxRetries)
		}
	})

	t.Run("with zero delay defaults to 2s", func(t *testing.T) {
		config := FetcherConfig{RetryDelay: 0}
		fetcher := NewFetcher(client, config, logger)
		if fetcher.config.RetryDelay != 2*time.Second {
			t.Errorf("RetryDelay = %v, want 2s", fetcher.config.RetryDelay)
		}
	})
}

func TestFetcher_Fetch_Success(t *testing.T) {
	widgets := []map[string]interface{}{
		{"id": "1", "type": "note", "title": "Note 1"},
		{"id": "2", "type": "image"},
		{"id": "3", "type": "pdf"},
	}

	client := &mockWidgetClient{widgets: widgets}
	config := DefaultFetcherConfig()
	logger := newTestLogger()

	fetcher := NewFetcher(client, config, logger)
	result, err := fetcher.Fetch(context.Background())

	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if result == nil {
		t.Fatal("Fetch() returned nil result")
	}
	if result.TotalCount != 3 {
		t.Errorf("TotalCount = %v, want 3", result.TotalCount)
	}
	if result.FilteredCount != 3 {
		t.Errorf("FilteredCount = %v, want 3", result.FilteredCount)
	}
	if result.Attempts != 1 {
		t.Errorf("Attempts = %v, want 1", result.Attempts)
	}
	if len(result.Widgets) != 3 {
		t.Errorf("Widgets count = %v, want 3", len(result.Widgets))
	}
}

func TestFetcher_Fetch_WithExclusions(t *testing.T) {
	widgets := []map[string]interface{}{
		{"id": "1", "type": "note"},
		{"id": "2", "type": "image"},
		{"id": "3", "type": "pdf"},
	}

	client := &mockWidgetClient{widgets: widgets}
	config := FetcherConfig{
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
		ExcludeIDs: []string{"2"},
	}
	logger := newTestLogger()

	fetcher := NewFetcher(client, config, logger)
	result, err := fetcher.Fetch(context.Background())

	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if result.TotalCount != 3 {
		t.Errorf("TotalCount = %v, want 3", result.TotalCount)
	}
	if result.FilteredCount != 2 {
		t.Errorf("FilteredCount = %v, want 2", result.FilteredCount)
	}

	// Verify ID "2" is excluded
	for _, w := range result.Widgets {
		if w.GetID() == "2" {
			t.Error("Widget with ID 2 should have been excluded")
		}
	}
}

func TestFetcher_Fetch_WithTypeFilter(t *testing.T) {
	widgets := []map[string]interface{}{
		{"id": "1", "type": "note"},
		{"id": "2", "type": "image"},
		{"id": "3", "type": "pdf"},
		{"id": "4", "type": "note"},
	}

	client := &mockWidgetClient{widgets: widgets}
	config := FetcherConfig{
		MaxRetries:  3,
		RetryDelay:  10 * time.Millisecond,
		FilterTypes: []string{"note"},
	}
	logger := newTestLogger()

	fetcher := NewFetcher(client, config, logger)
	result, err := fetcher.Fetch(context.Background())

	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if result.FilteredCount != 2 {
		t.Errorf("FilteredCount = %v, want 2", result.FilteredCount)
	}

	// Verify only notes are returned
	for _, w := range result.Widgets {
		if w.GetType() != "note" {
			t.Errorf("Expected only notes, got type %q", w.GetType())
		}
	}
}

func TestFetcher_Fetch_RetrySuccess(t *testing.T) {
	widgets := []map[string]interface{}{
		{"id": "1", "type": "note"},
	}

	client := &mockWidgetClient{
		widgets: widgets,
		err:     errors.New("temporary error"),
		failN:   2, // Fail first 2 calls, succeed on 3rd
	}
	config := FetcherConfig{
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
	}
	logger := newTestLogger()

	fetcher := NewFetcher(client, config, logger)
	result, err := fetcher.Fetch(context.Background())

	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if result.Attempts != 3 {
		t.Errorf("Attempts = %v, want 3", result.Attempts)
	}
	if client.calls != 3 {
		t.Errorf("client.calls = %v, want 3", client.calls)
	}
}

func TestFetcher_Fetch_AllRetriesFail(t *testing.T) {
	client := &mockWidgetClient{
		err:   errors.New("persistent error"),
		failN: 999, // Always fail
	}
	config := FetcherConfig{
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
	}
	logger := newTestLogger()

	fetcher := NewFetcher(client, config, logger)
	result, err := fetcher.Fetch(context.Background())

	if err == nil {
		t.Error("Fetch() should have returned error")
	}
	if !errors.Is(err, ErrFetchFailed) {
		t.Errorf("error = %v, want ErrFetchFailed", err)
	}
	if result != nil {
		t.Error("result should be nil on error")
	}
	if client.calls != 3 {
		t.Errorf("client.calls = %v, want 3", client.calls)
	}
}

func TestFetcher_Fetch_ContextCancellation(t *testing.T) {
	client := &mockWidgetClient{
		err:   errors.New("error"),
		failN: 999,
	}
	config := FetcherConfig{
		MaxRetries: 10,
		RetryDelay: 1 * time.Second, // Long delay
	}
	logger := newTestLogger()

	ctx, cancel := context.WithCancel(context.Background())
	fetcher := NewFetcher(client, config, logger)

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result, err := fetcher.Fetch(ctx)

	if err == nil {
		t.Error("Fetch() should have returned error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	if result != nil {
		t.Error("result should be nil on cancellation")
	}
}

func TestFetcher_FetchWithExclusions(t *testing.T) {
	widgets := []map[string]interface{}{
		{"id": "1", "type": "note"},
		{"id": "2", "type": "image"},
		{"id": "3", "type": "pdf"},
	}

	client := &mockWidgetClient{widgets: widgets}
	config := FetcherConfig{
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
		ExcludeIDs: []string{"1"}, // Pre-configured exclusion
	}
	logger := newTestLogger()

	fetcher := NewFetcher(client, config, logger)

	// Add additional exclusions
	result, err := fetcher.FetchWithExclusions(context.Background(), "2")

	if err != nil {
		t.Fatalf("FetchWithExclusions() error = %v", err)
	}
	if result.FilteredCount != 1 {
		t.Errorf("FilteredCount = %v, want 1", result.FilteredCount)
	}

	// Only ID "3" should remain
	if len(result.Widgets) != 1 || result.Widgets[0].GetID() != "3" {
		t.Error("Only widget with ID 3 should remain")
	}

	// Verify original config is restored
	if len(fetcher.config.ExcludeIDs) != 1 || fetcher.config.ExcludeIDs[0] != "1" {
		t.Error("Original ExcludeIDs should be restored")
	}
}

func TestFetcher_GetConfig(t *testing.T) {
	client := &mockWidgetClient{}
	config := FetcherConfig{
		MaxRetries:  5,
		RetryDelay:  3 * time.Second,
		ExcludeIDs:  []string{"a", "b"},
		FilterTypes: []string{"note"},
	}
	logger := newTestLogger()

	fetcher := NewFetcher(client, config, logger)
	got := fetcher.GetConfig()

	if got.MaxRetries != 5 {
		t.Errorf("MaxRetries = %v, want 5", got.MaxRetries)
	}
	if got.RetryDelay != 3*time.Second {
		t.Errorf("RetryDelay = %v, want 3s", got.RetryDelay)
	}
}

func TestFetcher_SetExcludeIDs(t *testing.T) {
	client := &mockWidgetClient{}
	config := DefaultFetcherConfig()
	logger := newTestLogger()

	fetcher := NewFetcher(client, config, logger)
	fetcher.SetExcludeIDs([]string{"x", "y", "z"})

	if len(fetcher.config.ExcludeIDs) != 3 {
		t.Errorf("ExcludeIDs length = %v, want 3", len(fetcher.config.ExcludeIDs))
	}
}

func TestFetcher_SetFilterTypes(t *testing.T) {
	client := &mockWidgetClient{}
	config := DefaultFetcherConfig()
	logger := newTestLogger()

	fetcher := NewFetcher(client, config, logger)
	fetcher.SetFilterTypes([]string{"note", "image"})

	if len(fetcher.config.FilterTypes) != 2 {
		t.Errorf("FilterTypes length = %v, want 2", len(fetcher.config.FilterTypes))
	}
}

func TestFetcher_EmptyCanvas(t *testing.T) {
	client := &mockWidgetClient{widgets: []map[string]interface{}{}}
	config := DefaultFetcherConfig()
	logger := newTestLogger()

	fetcher := NewFetcher(client, config, logger)
	result, err := fetcher.Fetch(context.Background())

	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if result.TotalCount != 0 {
		t.Errorf("TotalCount = %v, want 0", result.TotalCount)
	}
	if result.FilteredCount != 0 {
		t.Errorf("FilteredCount = %v, want 0", result.FilteredCount)
	}
}
