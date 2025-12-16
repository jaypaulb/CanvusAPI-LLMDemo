package shutdown

import (
	"sync"
	"testing"
	"time"
)

func TestOperationTracker_NewOperationTracker(t *testing.T) {
	tracker := NewOperationTracker()
	if tracker == nil {
		t.Fatal("NewOperationTracker returned nil")
	}
	if tracker.ActiveCount() != 0 {
		t.Errorf("expected 0 active operations, got %d", tracker.ActiveCount())
	}
	if tracker.IsClosed() {
		t.Error("new tracker should not be closed")
	}
}

func TestOperationTracker_StartDone(t *testing.T) {
	tracker := NewOperationTracker()

	// Start should succeed
	if !tracker.Start() {
		t.Error("Start should return true on open tracker")
	}
	if tracker.ActiveCount() != 1 {
		t.Errorf("expected 1 active operation, got %d", tracker.ActiveCount())
	}

	// Done should decrement
	tracker.Done()
	if tracker.ActiveCount() != 0 {
		t.Errorf("expected 0 active operations after Done, got %d", tracker.ActiveCount())
	}
}

func TestOperationTracker_MultipleOperations(t *testing.T) {
	tracker := NewOperationTracker()

	// Start multiple operations
	for i := 0; i < 5; i++ {
		if !tracker.Start() {
			t.Errorf("Start %d should succeed", i)
		}
	}

	if tracker.ActiveCount() != 5 {
		t.Errorf("expected 5 active operations, got %d", tracker.ActiveCount())
	}

	// Complete them
	for i := 0; i < 5; i++ {
		tracker.Done()
	}

	if tracker.ActiveCount() != 0 {
		t.Errorf("expected 0 active operations, got %d", tracker.ActiveCount())
	}
}

func TestOperationTracker_CloseRejectsNewOps(t *testing.T) {
	tracker := NewOperationTracker()

	tracker.Close()

	if !tracker.IsClosed() {
		t.Error("tracker should be closed after Close()")
	}

	// Start should fail after close
	if tracker.Start() {
		t.Error("Start should return false on closed tracker")
	}

	if tracker.ActiveCount() != 0 {
		t.Errorf("expected 0 active operations, got %d", tracker.ActiveCount())
	}
}

func TestOperationTracker_CloseAllowsInFlightToComplete(t *testing.T) {
	tracker := NewOperationTracker()

	// Start an operation
	if !tracker.Start() {
		t.Fatal("Start should succeed")
	}

	// Close while operation in progress
	tracker.Close()

	// Should still have 1 active
	if tracker.ActiveCount() != 1 {
		t.Errorf("expected 1 active operation, got %d", tracker.ActiveCount())
	}

	// New operation should be rejected
	if tracker.Start() {
		t.Error("Start should return false on closed tracker")
	}

	// Complete the in-flight operation
	tracker.Done()

	if tracker.ActiveCount() != 0 {
		t.Errorf("expected 0 active operations, got %d", tracker.ActiveCount())
	}
}

func TestOperationTracker_WaitCompletes(t *testing.T) {
	tracker := NewOperationTracker()

	// Start an operation
	if !tracker.Start() {
		t.Fatal("Start should succeed")
	}

	// Complete it in background
	go func() {
		time.Sleep(10 * time.Millisecond)
		tracker.Done()
	}()

	// Wait should complete without timeout
	err := tracker.Wait(1 * time.Second)
	if err != nil {
		t.Errorf("Wait should succeed, got error: %v", err)
	}
}

func TestOperationTracker_WaitTimeout(t *testing.T) {
	tracker := NewOperationTracker()

	// Start an operation but don't complete it
	if !tracker.Start() {
		t.Fatal("Start should succeed")
	}

	// Wait should timeout
	err := tracker.Wait(50 * time.Millisecond)
	if err != ErrWaitTimeout {
		t.Errorf("expected ErrWaitTimeout, got: %v", err)
	}

	// Clean up
	tracker.Done()
}

func TestOperationTracker_WaitNoOperations(t *testing.T) {
	tracker := NewOperationTracker()

	// Wait with no operations should complete immediately
	err := tracker.Wait(100 * time.Millisecond)
	if err != nil {
		t.Errorf("Wait should succeed with no operations, got error: %v", err)
	}
}

func TestOperationTracker_ConcurrentStartDone(t *testing.T) {
	tracker := NewOperationTracker()
	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if tracker.Start() {
				time.Sleep(time.Millisecond)
				tracker.Done()
			}
		}()
	}

	wg.Wait()

	if tracker.ActiveCount() != 0 {
		t.Errorf("expected 0 active operations after all goroutines complete, got %d", tracker.ActiveCount())
	}
}

func TestOperationTracker_ConcurrentStartWithClose(t *testing.T) {
	tracker := NewOperationTracker()
	const goroutines = 100

	var wg sync.WaitGroup
	var started int64
	var rejected int64
	var mu sync.Mutex

	// Start goroutines that try to start operations
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if tracker.Start() {
				mu.Lock()
				started++
				mu.Unlock()
				time.Sleep(time.Millisecond)
				tracker.Done()
			} else {
				mu.Lock()
				rejected++
				mu.Unlock()
			}
		}()
	}

	// Close after a short delay
	time.Sleep(500 * time.Microsecond)
	tracker.Close()

	wg.Wait()

	// All started operations should have completed
	if tracker.ActiveCount() != 0 {
		t.Errorf("expected 0 active operations, got %d", tracker.ActiveCount())
	}

	// Should have some started and some rejected
	t.Logf("started: %d, rejected: %d", started, rejected)
	if started+rejected != goroutines {
		t.Errorf("expected %d total, got %d", goroutines, started+rejected)
	}
}
