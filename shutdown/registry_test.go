package shutdown

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestShutdownRegistry_NewShutdownRegistry(t *testing.T) {
	registry := NewShutdownRegistry()
	if registry == nil {
		t.Fatal("NewShutdownRegistry returned nil")
	}
	if registry.Count() != 0 {
		t.Errorf("expected 0 entries, got %d", registry.Count())
	}
	if registry.IsClosed() {
		t.Error("new registry should not be closed")
	}
}

func TestShutdownRegistry_Register(t *testing.T) {
	registry := NewShutdownRegistry()

	registry.Register("test", 10, func(ctx context.Context) error {
		return nil
	})

	if registry.Count() != 1 {
		t.Errorf("expected 1 entry, got %d", registry.Count())
	}

	names := registry.Names()
	if len(names) != 1 || names[0] != "test" {
		t.Errorf("expected [test], got %v", names)
	}
}

func TestShutdownRegistry_PriorityOrdering(t *testing.T) {
	registry := NewShutdownRegistry()

	// Register in non-priority order
	registry.Register("third", 30, func(ctx context.Context) error { return nil })
	registry.Register("first", 10, func(ctx context.Context) error { return nil })
	registry.Register("second", 20, func(ctx context.Context) error { return nil })

	// Names should be in priority order
	names := registry.Names()
	expected := []string{"first", "second", "third"}

	if len(names) != len(expected) {
		t.Fatalf("expected %d names, got %d", len(expected), len(names))
	}

	for i, name := range names {
		if name != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], name)
		}
	}
}

func TestShutdownRegistry_ShutdownExecutesInOrder(t *testing.T) {
	registry := NewShutdownRegistry()

	var order []string
	var mu sync.Mutex

	appendOrder := func(name string) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			mu.Lock()
			order = append(order, name)
			mu.Unlock()
			return nil
		}
	}

	// Register in non-priority order
	registry.Register("third", 30, appendOrder("third"))
	registry.Register("first", 10, appendOrder("first"))
	registry.Register("second", 20, appendOrder("second"))

	errs := registry.Shutdown(context.Background())
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}

	expected := []string{"first", "second", "third"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d executions, got %d", len(expected), len(order))
	}

	for i, name := range order {
		if name != expected[i] {
			t.Errorf("execution %d: expected %s, got %s", i, expected[i], name)
		}
	}
}

func TestShutdownRegistry_ErrorCollection(t *testing.T) {
	registry := NewShutdownRegistry()

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	registry.Register("success", 10, func(ctx context.Context) error { return nil })
	registry.Register("fail1", 20, func(ctx context.Context) error { return err1 })
	registry.Register("fail2", 30, func(ctx context.Context) error { return err2 })

	errs := registry.Shutdown(context.Background())

	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(errs))
	}

	// Errors should be in execution order
	if errs[0] != err1 {
		t.Errorf("first error: expected %v, got %v", err1, errs[0])
	}
	if errs[1] != err2 {
		t.Errorf("second error: expected %v, got %v", err2, errs[1])
	}
}

func TestShutdownRegistry_ContinuesAfterError(t *testing.T) {
	registry := NewShutdownRegistry()

	var executed []string
	var mu sync.Mutex

	registry.Register("first", 10, func(ctx context.Context) error {
		mu.Lock()
		executed = append(executed, "first")
		mu.Unlock()
		return errors.New("first error")
	})
	registry.Register("second", 20, func(ctx context.Context) error {
		mu.Lock()
		executed = append(executed, "second")
		mu.Unlock()
		return nil
	})
	registry.Register("third", 30, func(ctx context.Context) error {
		mu.Lock()
		executed = append(executed, "third")
		mu.Unlock()
		return errors.New("third error")
	})

	errs := registry.Shutdown(context.Background())

	// All three should have executed despite errors
	if len(executed) != 3 {
		t.Errorf("expected 3 executions, got %d: %v", len(executed), executed)
	}

	// Should have 2 errors
	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errs))
	}
}

func TestShutdownRegistry_ShutdownOnlyOnce(t *testing.T) {
	registry := NewShutdownRegistry()

	var callCount int
	var mu sync.Mutex

	registry.Register("counter", 10, func(ctx context.Context) error {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil
	})

	// First shutdown
	errs := registry.Shutdown(context.Background())
	if len(errs) != 0 {
		t.Errorf("first shutdown: expected no errors, got %v", errs)
	}

	// Second shutdown should be no-op
	errs = registry.Shutdown(context.Background())
	if errs != nil {
		t.Errorf("second shutdown: expected nil, got %v", errs)
	}

	if callCount != 1 {
		t.Errorf("expected function called once, got %d", callCount)
	}

	if !registry.IsClosed() {
		t.Error("registry should be closed after shutdown")
	}
}

func TestShutdownRegistry_RegisterAfterShutdown(t *testing.T) {
	registry := NewShutdownRegistry()

	registry.Shutdown(context.Background())

	// Register after shutdown should be no-op
	registry.Register("late", 10, func(ctx context.Context) error {
		t.Error("late function should not be called")
		return nil
	})

	// Count should still be 0
	if registry.Count() != 0 {
		t.Errorf("expected 0 entries after late register, got %d", registry.Count())
	}
}

func TestShutdownRegistry_ContextCancellation(t *testing.T) {
	registry := NewShutdownRegistry()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var receivedCtx context.Context
	registry.Register("checker", 10, func(ctx context.Context) error {
		receivedCtx = ctx
		return ctx.Err()
	})

	errs := registry.Shutdown(ctx)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	if !errors.Is(errs[0], context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", errs[0])
	}

	if receivedCtx != ctx {
		t.Error("function did not receive the correct context")
	}
}

func TestShutdownRegistry_ContextTimeout(t *testing.T) {
	registry := NewShutdownRegistry()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	registry.Register("slow", 10, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	errs := registry.Shutdown(ctx)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	if !errors.Is(errs[0], context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", errs[0])
	}
}

func TestShutdownRegistry_SamePriority(t *testing.T) {
	registry := NewShutdownRegistry()

	// Register multiple functions with same priority
	registry.Register("a", 10, func(ctx context.Context) error { return nil })
	registry.Register("b", 10, func(ctx context.Context) error { return nil })
	registry.Register("c", 10, func(ctx context.Context) error { return nil })

	// All should be present
	if registry.Count() != 3 {
		t.Errorf("expected 3 entries, got %d", registry.Count())
	}

	// Should not error on shutdown
	errs := registry.Shutdown(context.Background())
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}
