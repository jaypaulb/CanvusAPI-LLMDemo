package shutdown

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestManager_NewManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
	if manager.Context() == nil {
		t.Error("Context should not be nil")
	}
	if manager.IsShuttingDown() {
		t.Error("New manager should not be shutting down")
	}
	if manager.ActiveOperations() != 0 {
		t.Errorf("expected 0 active operations, got %d", manager.ActiveOperations())
	}
}

func TestManager_WithTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	customTimeout := 30 * time.Second
	manager := NewManager(logger, WithTimeout(customTimeout))

	if manager.timeout != customTimeout {
		t.Errorf("expected timeout %v, got %v", customTimeout, manager.timeout)
	}
}

func TestManager_Register(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)

	// Register handlers with different priorities
	manager.Register("handler1", 10, func(ctx context.Context) error { return nil })
	manager.Register("handler2", 5, func(ctx context.Context) error { return nil })
	manager.Register("handler3", 20, func(ctx context.Context) error { return nil })

	handlers := manager.RegisteredHandlers()
	if len(handlers) != 3 {
		t.Fatalf("expected 3 handlers, got %d", len(handlers))
	}

	// Should be sorted by priority
	expected := []string{"handler2", "handler1", "handler3"}
	for i, name := range expected {
		if handlers[i] != name {
			t.Errorf("expected handler %d to be %q, got %q", i, name, handlers[i])
		}
	}
}

func TestManager_WrapOperation_Success(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)

	executed := false
	err := manager.WrapOperation(context.Background(), "test-op", func(ctx context.Context) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !executed {
		t.Error("operation should have been executed")
	}
}

func TestManager_WrapOperation_RejectsAfterClose(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)

	// Manually close the tracker (simulating shutdown)
	manager.tracker.Close()

	executed := false
	err := manager.WrapOperation(context.Background(), "test-op", func(ctx context.Context) error {
		executed = true
		return nil
	})

	if !errors.Is(err, ErrTrackerClosed) {
		t.Errorf("expected ErrTrackerClosed, got %v", err)
	}
	if executed {
		t.Error("operation should not have been executed")
	}
}

func TestManager_WrapOperation_TracksActive(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)

	started := make(chan struct{})
	done := make(chan struct{})

	go func() {
		_ = manager.WrapOperation(context.Background(), "long-op", func(ctx context.Context) error {
			close(started)
			<-done
			return nil
		})
	}()

	<-started // Wait for operation to start

	if manager.ActiveOperations() != 1 {
		t.Errorf("expected 1 active operation, got %d", manager.ActiveOperations())
	}

	close(done) // Let operation complete

	// Wait a bit for the operation to finish
	time.Sleep(10 * time.Millisecond)

	if manager.ActiveOperations() != 0 {
		t.Errorf("expected 0 active operations, got %d", manager.ActiveOperations())
	}
}

func TestManager_Shutdown_ExecutesHandlers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger, WithTimeout(5*time.Second))

	var order []string
	var mu sync.Mutex

	// Register handlers with different priorities
	manager.Register("first", 10, func(ctx context.Context) error {
		mu.Lock()
		order = append(order, "first")
		mu.Unlock()
		return nil
	})
	manager.Register("second", 20, func(ctx context.Context) error {
		mu.Lock()
		order = append(order, "second")
		mu.Unlock()
		return nil
	})
	manager.Register("zeroth", 5, func(ctx context.Context) error {
		mu.Lock()
		order = append(order, "zeroth")
		mu.Unlock()
		return nil
	})

	err := manager.Shutdown()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Check execution order (by priority: zeroth=5, first=10, second=20)
	expected := []string{"zeroth", "first", "second"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d handlers executed, got %d", len(expected), len(order))
	}
	for i, name := range expected {
		if order[i] != name {
			t.Errorf("expected order[%d] = %q, got %q", i, name, order[i])
		}
	}
}

func TestManager_Shutdown_ReportsErrors(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger, WithTimeout(5*time.Second))

	// Register one successful and one failing handler
	manager.Register("success", 10, func(ctx context.Context) error {
		return nil
	})
	manager.Register("failure", 20, func(ctx context.Context) error {
		return errors.New("cleanup failed")
	})

	err := manager.Shutdown()
	if err == nil {
		t.Error("expected error from failing handler")
	}
	if !contains(err.Error(), "1 errors") {
		t.Errorf("expected error message about 1 error, got %q", err.Error())
	}
}

func TestManager_Shutdown_WaitsForOperations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger, WithTimeout(5*time.Second))

	operationDone := make(chan struct{})
	var operationCompleted int32

	// Start a long-running operation
	go func() {
		_ = manager.WrapOperation(context.Background(), "long-op", func(ctx context.Context) error {
			<-operationDone
			atomic.StoreInt32(&operationCompleted, 1)
			return nil
		})
	}()

	// Give operation time to start
	time.Sleep(10 * time.Millisecond)

	// Start shutdown in background
	shutdownDone := make(chan error)
	go func() {
		shutdownDone <- manager.Shutdown()
	}()

	// Verify shutdown is waiting
	select {
	case <-shutdownDone:
		t.Fatal("shutdown should wait for in-flight operations")
	case <-time.After(50 * time.Millisecond):
		// Expected: shutdown is waiting
	}

	// Complete the operation
	close(operationDone)

	// Shutdown should now complete
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("shutdown should complete after operations finish")
	}

	if atomic.LoadInt32(&operationCompleted) != 1 {
		t.Error("operation should have completed before shutdown finished")
	}
}

func TestManager_Shutdown_Idempotent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger, WithTimeout(1*time.Second))

	var callCount int32
	manager.Register("counter", 10, func(ctx context.Context) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	})

	// Call shutdown multiple times
	for i := 0; i < 3; i++ {
		err := manager.Shutdown()
		if err != nil {
			t.Errorf("shutdown %d: expected no error, got %v", i, err)
		}
	}

	// Handler should only have been called once
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected handler called once, got %d", callCount)
	}
}

func TestManager_IsShuttingDown(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger, WithTimeout(1*time.Second))

	if manager.IsShuttingDown() {
		t.Error("should not be shutting down initially")
	}

	_ = manager.Shutdown()

	if !manager.IsShuttingDown() {
		t.Error("should be shutting down after Shutdown()")
	}
}

// ============================================================================
// Integration Tests - Additional tests for organism-level behavior
// ============================================================================

// TestManager_Shutdown_TimesOutWaitingForOperations verifies that shutdown
// properly times out when in-flight operations don't complete within the timeout.
func TestManager_Shutdown_TimesOutWaitingForOperations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	// Use a very short timeout for testing
	manager := NewManager(logger, WithTimeout(100*time.Millisecond))

	operationStarted := make(chan struct{})
	blockForever := make(chan struct{})

	// Start an operation that will never complete on its own
	go func() {
		_ = manager.WrapOperation(context.Background(), "blocking-op", func(ctx context.Context) error {
			close(operationStarted)
			<-blockForever // Block forever
			return nil
		})
	}()

	// Wait for operation to start
	<-operationStarted

	// Shutdown should timeout but still complete (with warning logged)
	start := time.Now()
	err := manager.Shutdown()
	elapsed := time.Since(start)

	// Shutdown should complete (no error returned for timeout, just logged warning)
	// The manager continues to cleanup handlers even if operations timeout
	if err != nil {
		t.Logf("shutdown returned error (acceptable): %v", err)
	}

	// Should have taken approximately the timeout duration
	if elapsed < 90*time.Millisecond {
		t.Errorf("shutdown completed too fast (%v), expected to wait for timeout", elapsed)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("shutdown took too long (%v), expected ~100ms", elapsed)
	}

	// Cleanup
	close(blockForever)
}

// TestManager_ForceShutdownOnSecondSignal verifies that the signal counter
// integration works correctly - second signal triggers force callback.
func TestManager_ForceShutdownOnSecondSignal(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)

	// Verify initial signal count is 0
	if manager.signals.Count() != 0 {
		t.Errorf("expected initial signal count 0, got %d", manager.signals.Count())
	}

	// Track if force callback was called
	var forceCallbackCalled int32

	// Replace the force callback with a testable one (instead of os.Exit)
	manager.signals.SetForceCallback(func() {
		atomic.StoreInt32(&forceCallbackCalled, 1)
	})

	// First signal should not trigger force callback
	count := manager.signals.Increment()
	if count != 1 {
		t.Errorf("expected count 1 after first signal, got %d", count)
	}
	if atomic.LoadInt32(&forceCallbackCalled) != 0 {
		t.Error("force callback should not be called after first signal")
	}

	// Second signal should trigger force callback
	count = manager.signals.Increment()
	if count != 2 {
		t.Errorf("expected count 2 after second signal, got %d", count)
	}
	if atomic.LoadInt32(&forceCallbackCalled) != 1 {
		t.Error("force callback should be called after second signal")
	}
}

// TestManager_WrapOperation_CancelledContext verifies that WrapOperation
// respects context cancellation and rejects operations appropriately.
func TestManager_WrapOperation_CancelledContext(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	executed := false
	err := manager.WrapOperation(ctx, "cancelled-op", func(ctx context.Context) error {
		executed = true
		return nil
	})

	// Should return context.Canceled
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
	if executed {
		t.Error("operation should not have been executed with cancelled context")
	}
}

// TestManager_WrapOperation_ManagerContextCancelled verifies that WrapOperation
// rejects operations when the manager's internal context is cancelled.
func TestManager_WrapOperation_ManagerContextCancelled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)

	// Cancel the manager's context directly
	manager.cancel()

	executed := false
	err := manager.WrapOperation(context.Background(), "after-cancel-op", func(ctx context.Context) error {
		executed = true
		return nil
	})

	// Should return context.Canceled
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
	if executed {
		t.Error("operation should not have been executed after manager context cancelled")
	}
}

// TestManager_ConcurrentOperationsDuringShutdown verifies that multiple
// concurrent operations are all tracked and waited for during shutdown.
func TestManager_ConcurrentOperationsDuringShutdown(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger, WithTimeout(5*time.Second))

	const numOperations = 5
	operationsStarted := make(chan struct{}, numOperations)
	operationsDone := make(chan struct{})
	var completedCount int32

	// Start multiple concurrent operations
	for i := 0; i < numOperations; i++ {
		go func(opNum int) {
			_ = manager.WrapOperation(context.Background(), "concurrent-op", func(ctx context.Context) error {
				operationsStarted <- struct{}{}
				<-operationsDone
				atomic.AddInt32(&completedCount, 1)
				return nil
			})
		}(i)
	}

	// Wait for all operations to start
	for i := 0; i < numOperations; i++ {
		<-operationsStarted
	}

	// Verify all operations are tracked
	activeCount := manager.ActiveOperations()
	if activeCount != numOperations {
		t.Errorf("expected %d active operations, got %d", numOperations, activeCount)
	}

	// Start shutdown in background
	shutdownDone := make(chan error)
	go func() {
		shutdownDone <- manager.Shutdown()
	}()

	// Shutdown should be waiting
	select {
	case <-shutdownDone:
		t.Fatal("shutdown should wait for all operations")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}

	// Release all operations
	close(operationsDone)

	// Shutdown should complete
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("shutdown should complete after all operations finish")
	}

	// All operations should have completed
	if atomic.LoadInt32(&completedCount) != numOperations {
		t.Errorf("expected %d completed operations, got %d", numOperations, completedCount)
	}
}

// TestManager_Start_Idempotent verifies that Start() can be safely called
// multiple times without issues.
func TestManager_Start_Idempotent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger)

	// Call Start multiple times
	manager.Start()
	manager.Start()
	manager.Start()

	// Should not panic or cause issues
	if !manager.started {
		t.Error("manager should be started")
	}

	// Cleanup
	_ = manager.Shutdown()
}

// TestManager_Shutdown_HandlerReceivesContext verifies that cleanup handlers
// receive a context with remaining timeout.
func TestManager_Shutdown_HandlerReceivesContext(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := NewManager(logger, WithTimeout(5*time.Second))

	var receivedCtx context.Context
	manager.Register("context-checker", 10, func(ctx context.Context) error {
		receivedCtx = ctx
		// Verify context has a deadline
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			t.Error("handler context should have a deadline")
		}
		return nil
	})

	err := manager.Shutdown()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if receivedCtx == nil {
		t.Fatal("handler should have received a context")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
