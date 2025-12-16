package db

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestAsyncWriterBasicWrite tests basic write and processing functionality.
func TestAsyncWriterBasicWrite(t *testing.T) {
	var processed int64
	var receivedData []interface{}
	var mu sync.Mutex

	handler := func(op WriteOperation) error {
		atomic.AddInt64(&processed, 1)
		mu.Lock()
		receivedData = append(receivedData, op.Data)
		mu.Unlock()
		return nil
	}

	writer := NewAsyncWriter(handler)
	writer.Start()

	// Write some data
	testData := []string{"first", "second", "third"}
	for _, data := range testData {
		if !writer.Write(data) {
			t.Errorf("Write() returned false, expected true")
		}
	}

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	writer.Stop()

	if atomic.LoadInt64(&processed) != 3 {
		t.Errorf("processed = %d, want 3", processed)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(receivedData) != 3 {
		t.Errorf("receivedData length = %d, want 3", len(receivedData))
	}
}

// TestAsyncWriterNonBlocking tests that writes are non-blocking.
func TestAsyncWriterNonBlocking(t *testing.T) {
	// Use a slow handler to simulate blocking
	handler := func(op WriteOperation) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	config := AsyncWriterConfig{
		ChannelCapacity: 10,
		DrainTimeout:    5 * time.Second,
	}
	writer := NewAsyncWriterWithConfig(handler, config)
	writer.Start()
	defer writer.Stop()

	// These should all return quickly (non-blocking)
	start := time.Now()
	for i := 0; i < 10; i++ {
		writer.Write(i)
	}
	elapsed := time.Since(start)

	// Writes should complete in < 10ms since they're buffered
	if elapsed > 10*time.Millisecond {
		t.Errorf("Writes took %v, expected < 10ms (non-blocking)", elapsed)
	}
}

// TestAsyncWriterChannelFull tests behavior when channel is full.
func TestAsyncWriterChannelFull(t *testing.T) {
	// Use a signal to know when handler is processing
	processing := make(chan struct{})
	blocker := make(chan struct{})

	handler := func(op WriteOperation) error {
		// Signal that we're processing
		select {
		case processing <- struct{}{}:
		default:
		}
		<-blocker
		return nil
	}

	config := AsyncWriterConfig{
		ChannelCapacity: 3,
		DrainTimeout:    1 * time.Second,
	}
	writer := NewAsyncWriterWithConfig(handler, config)
	writer.Start()

	// Write first item - goroutine will pick this up
	writer.Write("first")

	// Wait for handler to start processing
	<-processing

	// Now fill the channel buffer (capacity = 3)
	for i := 0; i < 3; i++ {
		if !writer.Write(i) {
			t.Errorf("Write %d should succeed (buffer not full yet)", i)
		}
	}

	// Next write should fail (non-blocking return false)
	if writer.Write("overflow") {
		t.Error("Write() should return false when channel is full")
	}

	close(blocker)
	writer.Stop()
}

// TestAsyncWriterGracefulDrain tests that pending ops are processed on stop.
func TestAsyncWriterGracefulDrain(t *testing.T) {
	var processed int64

	handler := func(op WriteOperation) error {
		atomic.AddInt64(&processed, 1)
		time.Sleep(5 * time.Millisecond) // Slow processing
		return nil
	}

	writer := NewAsyncWriter(handler)
	writer.Start()

	// Queue several writes
	for i := 0; i < 10; i++ {
		writer.Write(i)
	}

	// Stop should wait for drain
	writer.Stop()

	// All operations should have been processed
	if atomic.LoadInt64(&processed) != 10 {
		t.Errorf("processed = %d, want 10 (graceful drain)", processed)
	}
}

// TestAsyncWriterPending tests the Pending() count.
func TestAsyncWriterPending(t *testing.T) {
	// Handler that blocks
	blocker := make(chan struct{})
	handler := func(op WriteOperation) error {
		<-blocker
		return nil
	}

	config := AsyncWriterConfig{
		ChannelCapacity: 100,
		DrainTimeout:    1 * time.Second,
	}
	writer := NewAsyncWriterWithConfig(handler, config)
	writer.Start()

	// Write some data
	for i := 0; i < 5; i++ {
		writer.Write(i)
	}

	// Give the goroutine time to pick up first item
	time.Sleep(10 * time.Millisecond)

	// One is being processed (blocked), rest are pending
	pending := writer.Pending()
	if pending < 3 || pending > 5 {
		t.Errorf("Pending() = %d, expected 3-5", pending)
	}

	close(blocker)
	writer.Stop()
}

// TestAsyncWriterWriteWithTimeout tests timeout-based writes.
func TestAsyncWriterWriteWithTimeout(t *testing.T) {
	// Use a signal to know when handler is processing
	processing := make(chan struct{})
	blocker := make(chan struct{})

	handler := func(op WriteOperation) error {
		select {
		case processing <- struct{}{}:
		default:
		}
		<-blocker
		return nil
	}

	config := AsyncWriterConfig{
		ChannelCapacity: 2,
		DrainTimeout:    1 * time.Second,
	}
	writer := NewAsyncWriterWithConfig(handler, config)
	writer.Start()

	// Write first item - goroutine will pick this up
	writer.Write("first")

	// Wait for handler to start processing
	<-processing

	// Fill the buffer (capacity = 2)
	writer.Write("a")
	writer.Write("b")

	// Channel is now full, this should timeout
	start := time.Now()
	success := writer.WriteWithTimeout("overflow", 50*time.Millisecond)
	elapsed := time.Since(start)

	if success {
		t.Error("WriteWithTimeout should return false when timed out")
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("WriteWithTimeout returned too fast: %v (expected ~50ms timeout)", elapsed)
	}

	close(blocker)
	writer.Stop()
}

// TestDefaultAsyncWriterConfig verifies default configuration values.
func TestDefaultAsyncWriterConfig(t *testing.T) {
	config := DefaultAsyncWriterConfig()

	if config.ChannelCapacity != DefaultChannelCapacity {
		t.Errorf("ChannelCapacity = %d, want %d", config.ChannelCapacity, DefaultChannelCapacity)
	}
	if config.DrainTimeout != DefaultDrainTimeout {
		t.Errorf("DrainTimeout = %v, want %v", config.DrainTimeout, DefaultDrainTimeout)
	}
}

// TestAsyncWriterDoubleStart tests that Start is idempotent.
func TestAsyncWriterDoubleStart(t *testing.T) {
	var count int64
	handler := func(op WriteOperation) error {
		atomic.AddInt64(&count, 1)
		return nil
	}

	writer := NewAsyncWriter(handler)
	writer.Start()
	writer.Start() // Second call should be no-op

	writer.Write("test")
	time.Sleep(20 * time.Millisecond)
	writer.Stop()

	// Should only have one goroutine processing
	if atomic.LoadInt64(&count) != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

// TestAsyncWriterIsStarted tests the IsStarted() method.
func TestAsyncWriterIsStarted(t *testing.T) {
	handler := func(op WriteOperation) error { return nil }
	writer := NewAsyncWriter(handler)

	if writer.IsStarted() {
		t.Error("IsStarted() should be false before Start()")
	}

	writer.Start()
	if !writer.IsStarted() {
		t.Error("IsStarted() should be true after Start()")
	}

	writer.Stop()
}
