// Package webui provides the web-based user interface for CanvusLocalLLM.
// This file contains the CircularBuffer atom for fixed-size historical data storage.
package webui

import "sync"

// CircularBuffer is a thread-safe, fixed-size buffer that overwrites oldest entries
// when full. It maintains FIFO ordering for historical data storage.
//
// This is a pure data structure atom with no external dependencies.
// Thread safety is provided via sync.RWMutex.
//
// Use cases:
//   - Recent task history (last N tasks)
//   - GPU metrics snapshots (last 1 hour at 5-second intervals = 720 samples)
//   - Activity logs with bounded memory usage
//
// Note: This implementation uses []interface{} for simplicity.
// For type-safe usage, wrap this buffer or use Go generics when available.
type CircularBuffer struct {
	mu       sync.RWMutex
	data     []interface{}
	capacity int
	size     int
	head     int // Index where next element will be written
	tail     int // Index of oldest element
}

// NewCircularBuffer creates a new CircularBuffer with the specified capacity.
// The capacity must be at least 1.
//
// Parameters:
//   - capacity: Maximum number of elements the buffer can hold
//
// Returns:
//   - *CircularBuffer: Ready-to-use buffer
//
// Panics if capacity is less than 1.
func NewCircularBuffer(capacity int) *CircularBuffer {
	if capacity < 1 {
		panic("CircularBuffer capacity must be at least 1")
	}
	return &CircularBuffer{
		data:     make([]interface{}, capacity),
		capacity: capacity,
		size:     0,
		head:     0,
		tail:     0,
	}
}

// Push adds an element to the buffer. If the buffer is full,
// the oldest element is overwritten.
//
// Parameters:
//   - item: The element to add (can be any type)
func (b *CircularBuffer) Push(item interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.data[b.head] = item
	b.head = (b.head + 1) % b.capacity

	if b.size < b.capacity {
		b.size++
	} else {
		// Buffer is full, move tail forward (oldest element overwritten)
		b.tail = (b.tail + 1) % b.capacity
	}
}

// GetAll returns all elements in the buffer, ordered from oldest to newest.
// Returns an empty slice if the buffer is empty.
//
// The returned slice is a copy and safe to modify.
func (b *CircularBuffer) GetAll() []interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.size == 0 {
		return []interface{}{}
	}

	result := make([]interface{}, b.size)
	for i := 0; i < b.size; i++ {
		idx := (b.tail + i) % b.capacity
		result[i] = b.data[idx]
	}
	return result
}

// GetLast returns the N most recent elements, ordered from oldest to newest.
// If n exceeds the current size, all available elements are returned.
// If n is 0 or negative, an empty slice is returned.
//
// Parameters:
//   - n: Maximum number of elements to return
//
// Returns:
//   - []interface{}: The most recent elements
func (b *CircularBuffer) GetLast(n int) []interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if n <= 0 || b.size == 0 {
		return []interface{}{}
	}

	// Limit n to current size
	if n > b.size {
		n = b.size
	}

	result := make([]interface{}, n)
	// Calculate starting index for the last n elements
	startOffset := b.size - n
	for i := 0; i < n; i++ {
		idx := (b.tail + startOffset + i) % b.capacity
		result[i] = b.data[idx]
	}
	return result
}

// Peek returns the most recent element without removing it.
// Returns nil if the buffer is empty.
func (b *CircularBuffer) Peek() interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.size == 0 {
		return nil
	}

	// Most recent element is at head - 1 (with wraparound)
	idx := (b.head - 1 + b.capacity) % b.capacity
	return b.data[idx]
}

// PeekOldest returns the oldest element without removing it.
// Returns nil if the buffer is empty.
func (b *CircularBuffer) PeekOldest() interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.size == 0 {
		return nil
	}

	return b.data[b.tail]
}

// Size returns the current number of elements in the buffer.
func (b *CircularBuffer) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.size
}

// Capacity returns the maximum number of elements the buffer can hold.
func (b *CircularBuffer) Capacity() int {
	return b.capacity // Immutable, no lock needed
}

// IsFull returns true if the buffer is at capacity.
func (b *CircularBuffer) IsFull() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.size == b.capacity
}

// IsEmpty returns true if the buffer contains no elements.
func (b *CircularBuffer) IsEmpty() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.size == 0
}

// Clear removes all elements from the buffer.
func (b *CircularBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Clear references to allow GC
	for i := range b.data {
		b.data[i] = nil
	}
	b.size = 0
	b.head = 0
	b.tail = 0
}
