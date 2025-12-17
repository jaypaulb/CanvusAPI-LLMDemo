package webui

import (
	"sync"
	"testing"
)

func TestNewCircularBuffer(t *testing.T) {
	buf := NewCircularBuffer(5)
	if buf.Capacity() != 5 {
		t.Errorf("Capacity() = %d, want 5", buf.Capacity())
	}
	if buf.Size() != 0 {
		t.Errorf("Size() = %d, want 0", buf.Size())
	}
	if !buf.IsEmpty() {
		t.Error("IsEmpty() = false, want true")
	}
}

func TestNewCircularBuffer_PanicsOnZeroCapacity(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewCircularBuffer(0) should panic")
		}
	}()
	NewCircularBuffer(0)
}

func TestCircularBuffer_Push(t *testing.T) {
	buf := NewCircularBuffer(3)

	buf.Push(1)
	if buf.Size() != 1 {
		t.Errorf("Size() = %d, want 1", buf.Size())
	}
	if buf.IsEmpty() {
		t.Error("IsEmpty() = true, want false")
	}

	buf.Push(2)
	buf.Push(3)
	if buf.Size() != 3 {
		t.Errorf("Size() = %d, want 3", buf.Size())
	}
	if !buf.IsFull() {
		t.Error("IsFull() = false, want true")
	}
}

func TestCircularBuffer_PushOverwrite(t *testing.T) {
	buf := NewCircularBuffer(3)

	buf.Push(1)
	buf.Push(2)
	buf.Push(3)
	buf.Push(4) // Overwrites 1

	if buf.Size() != 3 {
		t.Errorf("Size() = %d, want 3", buf.Size())
	}

	all := buf.GetAll()
	expected := []interface{}{2, 3, 4}
	if !sliceEqual(all, expected) {
		t.Errorf("GetAll() = %v, want %v", all, expected)
	}
}

func TestCircularBuffer_GetAll(t *testing.T) {
	buf := NewCircularBuffer(5)

	// Empty buffer
	all := buf.GetAll()
	if len(all) != 0 {
		t.Errorf("GetAll() on empty = %v, want []", all)
	}

	// Partially filled
	buf.Push("a")
	buf.Push("b")
	buf.Push("c")

	all = buf.GetAll()
	expected := []interface{}{"a", "b", "c"}
	if !sliceEqual(all, expected) {
		t.Errorf("GetAll() = %v, want %v", all, expected)
	}
}

func TestCircularBuffer_GetAll_AfterWrap(t *testing.T) {
	buf := NewCircularBuffer(3)

	// Fill and wrap around
	buf.Push(1)
	buf.Push(2)
	buf.Push(3)
	buf.Push(4)
	buf.Push(5)

	all := buf.GetAll()
	expected := []interface{}{3, 4, 5}
	if !sliceEqual(all, expected) {
		t.Errorf("GetAll() = %v, want %v", all, expected)
	}
}

func TestCircularBuffer_GetLast(t *testing.T) {
	buf := NewCircularBuffer(5)

	buf.Push(1)
	buf.Push(2)
	buf.Push(3)
	buf.Push(4)
	buf.Push(5)

	tests := []struct {
		n        int
		expected []interface{}
	}{
		{0, []interface{}{}},
		{-1, []interface{}{}},
		{1, []interface{}{5}},
		{2, []interface{}{4, 5}},
		{3, []interface{}{3, 4, 5}},
		{5, []interface{}{1, 2, 3, 4, 5}},
		{10, []interface{}{1, 2, 3, 4, 5}}, // Exceeds size
	}

	for _, tt := range tests {
		result := buf.GetLast(tt.n)
		if !sliceEqual(result, tt.expected) {
			t.Errorf("GetLast(%d) = %v, want %v", tt.n, result, tt.expected)
		}
	}
}

func TestCircularBuffer_GetLast_AfterWrap(t *testing.T) {
	buf := NewCircularBuffer(3)

	buf.Push(1)
	buf.Push(2)
	buf.Push(3)
	buf.Push(4)
	buf.Push(5)

	result := buf.GetLast(2)
	expected := []interface{}{4, 5}
	if !sliceEqual(result, expected) {
		t.Errorf("GetLast(2) = %v, want %v", result, expected)
	}
}

func TestCircularBuffer_Peek(t *testing.T) {
	buf := NewCircularBuffer(3)

	// Empty buffer
	if buf.Peek() != nil {
		t.Errorf("Peek() on empty = %v, want nil", buf.Peek())
	}

	buf.Push("first")
	if buf.Peek() != "first" {
		t.Errorf("Peek() = %v, want 'first'", buf.Peek())
	}

	buf.Push("second")
	if buf.Peek() != "second" {
		t.Errorf("Peek() = %v, want 'second'", buf.Peek())
	}

	buf.Push("third")
	buf.Push("fourth") // Overwrites "first"
	if buf.Peek() != "fourth" {
		t.Errorf("Peek() = %v, want 'fourth'", buf.Peek())
	}
}

func TestCircularBuffer_PeekOldest(t *testing.T) {
	buf := NewCircularBuffer(3)

	// Empty buffer
	if buf.PeekOldest() != nil {
		t.Errorf("PeekOldest() on empty = %v, want nil", buf.PeekOldest())
	}

	buf.Push("first")
	if buf.PeekOldest() != "first" {
		t.Errorf("PeekOldest() = %v, want 'first'", buf.PeekOldest())
	}

	buf.Push("second")
	if buf.PeekOldest() != "first" {
		t.Errorf("PeekOldest() = %v, want 'first'", buf.PeekOldest())
	}

	buf.Push("third")
	buf.Push("fourth") // Overwrites "first"
	if buf.PeekOldest() != "second" {
		t.Errorf("PeekOldest() = %v, want 'second'", buf.PeekOldest())
	}
}

func TestCircularBuffer_Clear(t *testing.T) {
	buf := NewCircularBuffer(5)

	buf.Push(1)
	buf.Push(2)
	buf.Push(3)

	buf.Clear()

	if !buf.IsEmpty() {
		t.Error("IsEmpty() after Clear() = false, want true")
	}
	if buf.Size() != 0 {
		t.Errorf("Size() after Clear() = %d, want 0", buf.Size())
	}
	if buf.Peek() != nil {
		t.Errorf("Peek() after Clear() = %v, want nil", buf.Peek())
	}

	// Buffer should be reusable after clear
	buf.Push("new")
	if buf.Size() != 1 {
		t.Errorf("Size() after Push = %d, want 1", buf.Size())
	}
}

func TestCircularBuffer_ThreadSafety(t *testing.T) {
	buf := NewCircularBuffer(100)
	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				buf.Push(id*1000 + j)
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				buf.GetAll()
				buf.GetLast(10)
				buf.Peek()
				buf.PeekOldest()
				buf.Size()
			}
		}()
	}

	wg.Wait()

	// Buffer should be full with most recent entries
	if !buf.IsFull() {
		t.Error("Buffer should be full after concurrent writes")
	}
	if buf.Size() != 100 {
		t.Errorf("Size() = %d, want 100", buf.Size())
	}
}

func TestCircularBuffer_SingleElement(t *testing.T) {
	buf := NewCircularBuffer(1)

	buf.Push("only")
	if buf.Size() != 1 {
		t.Errorf("Size() = %d, want 1", buf.Size())
	}
	if buf.Peek() != "only" {
		t.Errorf("Peek() = %v, want 'only'", buf.Peek())
	}

	buf.Push("replaced")
	if buf.Size() != 1 {
		t.Errorf("Size() after overwrite = %d, want 1", buf.Size())
	}
	if buf.Peek() != "replaced" {
		t.Errorf("Peek() after overwrite = %v, want 'replaced'", buf.Peek())
	}
}

// Helper function to compare slices
func sliceEqual(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Benchmark tests
func BenchmarkCircularBuffer_Push(b *testing.B) {
	buf := NewCircularBuffer(1000)
	for i := 0; i < b.N; i++ {
		buf.Push(i)
	}
}

func BenchmarkCircularBuffer_GetAll(b *testing.B) {
	buf := NewCircularBuffer(1000)
	for i := 0; i < 1000; i++ {
		buf.Push(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.GetAll()
	}
}

func BenchmarkCircularBuffer_GetLast(b *testing.B) {
	buf := NewCircularBuffer(1000)
	for i := 0; i < 1000; i++ {
		buf.Push(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.GetLast(100)
	}
}
