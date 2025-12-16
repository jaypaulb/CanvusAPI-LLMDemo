package handlers

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go_backend/canvusapi"
	"go_backend/core"
	"go_backend/logging"
	"go_backend/shutdown"

	"go.uber.org/zap/zaptest"
)

// TestRequestWrapper_Execute_Success verifies that Execute runs the handler
// and returns nil when everything succeeds.
func TestRequestWrapper_Execute_Success(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := shutdown.NewManager(logger)
	wrapper := NewRequestWrapper(manager, logger)

	handlerCalled := false
	handler := func(ctx context.Context, update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) error {
		handlerCalled = true
		return nil
	}

	err := wrapper.Execute(
		context.Background(),
		"test-operation",
		handler,
		map[string]interface{}{"id": "test"},
		nil, // client not needed for this test
		nil, // config not needed for this test
		nil, // logger not needed for this test
	)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !handlerCalled {
		t.Error("handler should have been called")
	}
}

// TestRequestWrapper_Execute_RejectsWhenShuttingDown verifies that Execute
// returns ErrShuttingDown when the system is shutting down.
func TestRequestWrapper_Execute_RejectsWhenShuttingDown(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := shutdown.NewManager(logger)
	wrapper := NewRequestWrapper(manager, logger)

	// Trigger shutdown
	_ = manager.Shutdown()

	handlerCalled := false
	handler := func(ctx context.Context, update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) error {
		handlerCalled = true
		return nil
	}

	err := wrapper.Execute(
		context.Background(),
		"test-operation",
		handler,
		map[string]interface{}{"id": "test"},
		nil, nil, nil,
	)

	if !errors.Is(err, ErrShuttingDown) {
		t.Errorf("expected ErrShuttingDown, got %v", err)
	}
	if handlerCalled {
		t.Error("handler should not have been called during shutdown")
	}
}

// TestRequestWrapper_Execute_PropagatesHandlerError verifies that handler
// errors are properly propagated.
func TestRequestWrapper_Execute_PropagatesHandlerError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := shutdown.NewManager(logger)
	wrapper := NewRequestWrapper(manager, logger)

	expectedErr := errors.New("handler failed")
	handler := func(ctx context.Context, update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) error {
		return expectedErr
	}

	err := wrapper.Execute(
		context.Background(),
		"failing-operation",
		handler,
		map[string]interface{}{"id": "test"},
		nil, nil, nil,
	)

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected handler error, got %v", err)
	}
}

// TestRequestWrapper_Execute_TracksOperations verifies that active operations
// are properly tracked during handler execution.
func TestRequestWrapper_Execute_TracksOperations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := shutdown.NewManager(logger)
	wrapper := NewRequestWrapper(manager, logger)

	started := make(chan struct{})
	done := make(chan struct{})

	handler := func(ctx context.Context, update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) error {
		close(started)
		<-done
		return nil
	}

	// Start handler in background
	go func() {
		_ = wrapper.Execute(
			context.Background(),
			"tracked-operation",
			handler,
			map[string]interface{}{"id": "test"},
			nil, nil, nil,
		)
	}()

	<-started // Wait for handler to start

	// Check active count
	if wrapper.ActiveOperations() != 1 {
		t.Errorf("expected 1 active operation, got %d", wrapper.ActiveOperations())
	}

	close(done) // Let handler complete

	// Wait for completion
	time.Sleep(10 * time.Millisecond)

	if wrapper.ActiveOperations() != 0 {
		t.Errorf("expected 0 active operations after completion, got %d", wrapper.ActiveOperations())
	}
}

// TestRequestWrapper_ExecuteAsync_StartsHandler verifies that ExecuteAsync
// launches the handler in a goroutine.
func TestRequestWrapper_ExecuteAsync_StartsHandler(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := shutdown.NewManager(logger)
	wrapper := NewRequestWrapper(manager, logger)

	var handlerCalled int32
	handlerDone := make(chan struct{})

	handler := func(ctx context.Context, update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) error {
		atomic.StoreInt32(&handlerCalled, 1)
		close(handlerDone)
		return nil
	}

	started := wrapper.ExecuteAsync(
		context.Background(),
		"async-operation",
		handler,
		map[string]interface{}{"id": "test"},
		nil, nil, nil,
	)

	if !started {
		t.Error("ExecuteAsync should return true when handler is started")
	}

	// Wait for handler to complete
	select {
	case <-handlerDone:
		// Handler completed
	case <-time.After(1 * time.Second):
		t.Fatal("handler did not complete in time")
	}

	if atomic.LoadInt32(&handlerCalled) != 1 {
		t.Error("handler should have been called")
	}
}

// TestRequestWrapper_ExecuteAsync_RejectsWhenShuttingDown verifies that
// ExecuteAsync returns false and doesn't start a goroutine during shutdown.
func TestRequestWrapper_ExecuteAsync_RejectsWhenShuttingDown(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := shutdown.NewManager(logger)
	wrapper := NewRequestWrapper(manager, logger)

	// Trigger shutdown
	_ = manager.Shutdown()

	handlerCalled := false
	handler := func(ctx context.Context, update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) error {
		handlerCalled = true
		return nil
	}

	started := wrapper.ExecuteAsync(
		context.Background(),
		"rejected-async",
		handler,
		map[string]interface{}{"id": "test"},
		nil, nil, nil,
	)

	if started {
		t.Error("ExecuteAsync should return false during shutdown")
	}

	// Give some time to ensure no goroutine was started
	time.Sleep(50 * time.Millisecond)

	if handlerCalled {
		t.Error("handler should not have been called during shutdown")
	}
}

// TestRequestWrapper_IsShuttingDown returns correct status.
func TestRequestWrapper_IsShuttingDown(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := shutdown.NewManager(logger)
	wrapper := NewRequestWrapper(manager, logger)

	if wrapper.IsShuttingDown() {
		t.Error("should not be shutting down initially")
	}

	_ = manager.Shutdown()

	if !wrapper.IsShuttingDown() {
		t.Error("should be shutting down after Shutdown()")
	}
}

// TestRequestWrapper_Stats returns correct statistics.
func TestRequestWrapper_Stats(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := shutdown.NewManager(logger)
	wrapper := NewRequestWrapper(manager, logger)

	stats := wrapper.Stats()
	if stats.ActiveCount != 0 {
		t.Errorf("expected 0 active operations, got %d", stats.ActiveCount)
	}
	if stats.IsShuttingDown {
		t.Error("should not be shutting down initially")
	}
}

// TestWrapLegacyHandler correctly adapts legacy handlers.
func TestWrapLegacyHandler(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := shutdown.NewManager(logger)
	wrapper := NewRequestWrapper(manager, logger)

	legacyCalled := false
	legacy := func(update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) {
		legacyCalled = true
	}

	wrapped := WrapLegacyHandler(legacy)

	err := wrapper.Execute(
		context.Background(),
		"legacy-operation",
		wrapped,
		map[string]interface{}{"id": "test"},
		nil, nil, nil,
	)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !legacyCalled {
		t.Error("legacy handler should have been called")
	}
}

// TestWrapLegacyHandler_RespectsContextCancellation verifies that wrapped
// legacy handlers check context before executing.
func TestWrapLegacyHandler_RespectsContextCancellation(t *testing.T) {
	legacyCalled := false
	legacy := func(update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) {
		legacyCalled = true
	}

	wrapped := WrapLegacyHandler(legacy)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := wrapped(ctx, map[string]interface{}{"id": "test"}, nil, nil, nil)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if legacyCalled {
		t.Error("legacy handler should not have been called with cancelled context")
	}
}

// TestRequestWrapper_ConcurrentOperations verifies that multiple concurrent
// operations are properly tracked.
func TestRequestWrapper_ConcurrentOperations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := shutdown.NewManager(logger, shutdown.WithTimeout(5*time.Second))
	wrapper := NewRequestWrapper(manager, logger)

	const numOperations = 5
	allStarted := make(chan struct{}, numOperations)
	releaseAll := make(chan struct{})
	var completedCount int32

	handler := func(ctx context.Context, update map[string]interface{}, client *canvusapi.Client, config *core.Config, logger *logging.Logger) error {
		allStarted <- struct{}{}
		<-releaseAll
		atomic.AddInt32(&completedCount, 1)
		return nil
	}

	// Start multiple operations
	var wg sync.WaitGroup
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(opNum int) {
			defer wg.Done()
			_ = wrapper.Execute(
				context.Background(),
				"concurrent-op",
				handler,
				map[string]interface{}{"id": "test"},
				nil, nil, nil,
			)
		}(i)
	}

	// Wait for all to start
	for i := 0; i < numOperations; i++ {
		<-allStarted
	}

	// Verify all are tracked
	if wrapper.ActiveOperations() != numOperations {
		t.Errorf("expected %d active operations, got %d", numOperations, wrapper.ActiveOperations())
	}

	// Release all operations
	close(releaseAll)

	// Wait for completion
	wg.Wait()

	if atomic.LoadInt32(&completedCount) != numOperations {
		t.Errorf("expected %d completed operations, got %d", numOperations, completedCount)
	}

	if wrapper.ActiveOperations() != 0 {
		t.Errorf("expected 0 active operations after all complete, got %d", wrapper.ActiveOperations())
	}
}
