package shutdown

import (
	"sync"
	"testing"
)

func TestSignalCounter_NewSignalCounter(t *testing.T) {
	counter := NewSignalCounter(2, nil)
	if counter == nil {
		t.Fatal("NewSignalCounter returned nil")
	}
	if counter.Count() != 0 {
		t.Errorf("expected 0 count, got %d", counter.Count())
	}
}

func TestSignalCounter_Increment(t *testing.T) {
	counter := NewSignalCounter(3, nil)

	// First increment
	count := counter.Increment()
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}
	if counter.Count() != 1 {
		t.Errorf("expected Count() 1, got %d", counter.Count())
	}

	// Second increment
	count = counter.Increment()
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestSignalCounter_ForceCallback(t *testing.T) {
	var called bool
	counter := NewSignalCounter(2, func() {
		called = true
	})

	// First increment - should not trigger
	counter.Increment()
	if called {
		t.Error("callback should not be called on first increment")
	}

	// Second increment - should trigger
	counter.Increment()
	if !called {
		t.Error("callback should be called on second increment")
	}
}

func TestSignalCounter_ForceCallbackOnExactThreshold(t *testing.T) {
	var callCount int
	counter := NewSignalCounter(3, func() {
		callCount++
	})

	counter.Increment() // 1
	counter.Increment() // 2
	if callCount != 0 {
		t.Errorf("callback called too early, count: %d", callCount)
	}

	counter.Increment() // 3 - should trigger
	if callCount != 1 {
		t.Errorf("expected callback called once at threshold, got %d", callCount)
	}

	counter.Increment() // 4 - should trigger again (past threshold)
	if callCount != 2 {
		t.Errorf("expected callback called again, got %d", callCount)
	}
}

func TestSignalCounter_NilCallback(t *testing.T) {
	counter := NewSignalCounter(1, nil)

	// Should not panic with nil callback
	counter.Increment()
	counter.Increment()

	if counter.Count() != 2 {
		t.Errorf("expected count 2, got %d", counter.Count())
	}
}

func TestSignalCounter_Reset(t *testing.T) {
	var callCount int
	counter := NewSignalCounter(2, func() {
		callCount++
	})

	counter.Increment()
	counter.Increment() // Triggers callback
	if callCount != 1 {
		t.Errorf("expected 1 callback, got %d", callCount)
	}

	counter.Reset()
	if counter.Count() != 0 {
		t.Errorf("expected 0 after reset, got %d", counter.Count())
	}

	// Should trigger again after reset
	counter.Increment()
	counter.Increment()
	if callCount != 2 {
		t.Errorf("expected 2 callbacks after reset and re-trigger, got %d", callCount)
	}
}

func TestSignalCounter_SetForceCallback(t *testing.T) {
	var callback1Called, callback2Called bool

	counter := NewSignalCounter(2, func() {
		callback1Called = true
	})

	counter.Increment()

	// Change callback before threshold
	counter.SetForceCallback(func() {
		callback2Called = true
	})

	counter.Increment() // Should trigger new callback

	if callback1Called {
		t.Error("old callback should not be called")
	}
	if !callback2Called {
		t.Error("new callback should be called")
	}
}

func TestSignalCounter_ConcurrentIncrement(t *testing.T) {
	var callCount int
	var mu sync.Mutex

	counter := NewSignalCounter(50, func() {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	var wg sync.WaitGroup
	const goroutines = 100

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			counter.Increment()
		}()
	}

	wg.Wait()

	if counter.Count() != goroutines {
		t.Errorf("expected count %d, got %d", goroutines, counter.Count())
	}

	// Callback should have been called for increments 50-100 (51 times)
	expectedCalls := goroutines - 50 + 1 // 51
	mu.Lock()
	if callCount != expectedCalls {
		t.Errorf("expected %d callbacks, got %d", expectedCalls, callCount)
	}
	mu.Unlock()
}

func TestSignalCounter_ZeroThreshold(t *testing.T) {
	var called bool
	counter := NewSignalCounter(0, func() {
		called = true
	})

	// Even with threshold 0, callback requires increment
	if called {
		t.Error("callback should not be called before increment")
	}

	counter.Increment() // Should trigger (0 >= 0)
	if !called {
		t.Error("callback should be called when count >= threshold")
	}
}

func TestSignalCounter_NegativeThreshold(t *testing.T) {
	var callCount int
	counter := NewSignalCounter(-1, func() {
		callCount++
	})

	// Any increment should trigger (count >= -1)
	counter.Increment()
	if callCount != 1 {
		t.Errorf("expected 1 callback with negative threshold, got %d", callCount)
	}
}

func TestSignalCounter_Count(t *testing.T) {
	counter := NewSignalCounter(10, nil)

	for i := 1; i <= 5; i++ {
		result := counter.Increment()
		if result != i {
			t.Errorf("Increment() returned %d, expected %d", result, i)
		}
		if counter.Count() != i {
			t.Errorf("Count() returned %d, expected %d", counter.Count(), i)
		}
	}
}
