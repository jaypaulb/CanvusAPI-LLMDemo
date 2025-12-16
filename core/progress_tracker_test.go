package core

import (
	"sync"
	"testing"
	"time"
)

func TestNewProgressTracker(t *testing.T) {
	tests := []struct {
		name  string
		total int64
	}{
		{"zero total (unknown)", 0},
		{"1 KB", 1024},
		{"1 GB", 1024 * 1024 * 1024},
		{"8 GB (typical model)", 8 * 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewProgressTracker(tt.total)
			if tracker == nil {
				t.Fatal("NewProgressTracker returned nil")
			}
			if tracker.Total() != tt.total {
				t.Errorf("Total() = %d, want %d", tracker.Total(), tt.total)
			}
			if tracker.Downloaded() != 0 {
				t.Errorf("Downloaded() = %d, want 0", tracker.Downloaded())
			}
		})
	}
}

func TestProgressTracker_Update(t *testing.T) {
	tracker := NewProgressTracker(1024)

	// Update with 100 bytes
	tracker.Update(100)
	if tracker.Downloaded() != 100 {
		t.Errorf("After Update(100), Downloaded() = %d, want 100", tracker.Downloaded())
	}

	// Update with 200 more bytes
	tracker.Update(200)
	if tracker.Downloaded() != 300 {
		t.Errorf("After Update(200), Downloaded() = %d, want 300", tracker.Downloaded())
	}

	// Update with 0 bytes (should be ignored)
	tracker.Update(0)
	if tracker.Downloaded() != 300 {
		t.Errorf("After Update(0), Downloaded() = %d, want 300", tracker.Downloaded())
	}

	// Update with negative bytes (should be ignored)
	tracker.Update(-50)
	if tracker.Downloaded() != 300 {
		t.Errorf("After Update(-50), Downloaded() = %d, want 300", tracker.Downloaded())
	}
}

func TestProgressTracker_SetDownloaded(t *testing.T) {
	tracker := NewProgressTracker(1024)

	tracker.SetDownloaded(500)
	if tracker.Downloaded() != 500 {
		t.Errorf("SetDownloaded(500) => Downloaded() = %d, want 500", tracker.Downloaded())
	}

	// Setting to 0 should work
	tracker.SetDownloaded(0)
	if tracker.Downloaded() != 0 {
		t.Errorf("SetDownloaded(0) => Downloaded() = %d, want 0", tracker.Downloaded())
	}

	// Negative should be clamped to 0
	tracker.SetDownloaded(-100)
	if tracker.Downloaded() != 0 {
		t.Errorf("SetDownloaded(-100) => Downloaded() = %d, want 0", tracker.Downloaded())
	}
}

func TestProgressTracker_SetTotal(t *testing.T) {
	tracker := NewProgressTracker(1024)

	tracker.SetTotal(2048)
	if tracker.Total() != 2048 {
		t.Errorf("SetTotal(2048) => Total() = %d, want 2048", tracker.Total())
	}

	// Setting to 0 (unknown) should work
	tracker.SetTotal(0)
	if tracker.Total() != 0 {
		t.Errorf("SetTotal(0) => Total() = %d, want 0", tracker.Total())
	}

	// Negative should be clamped to 0
	tracker.SetTotal(-100)
	if tracker.Total() != 0 {
		t.Errorf("SetTotal(-100) => Total() = %d, want 0", tracker.Total())
	}
}

func TestProgressTracker_Progress_PercentCalculation(t *testing.T) {
	tests := []struct {
		name            string
		total           int64
		downloaded      int64
		expectedPercent float64
	}{
		{"0% complete", 1024, 0, 0},
		{"50% complete", 1024, 512, 50},
		{"100% complete", 1024, 1024, 100},
		{"over 100% (should cap at 100)", 1024, 2048, 100},
		{"unknown total (percent = -1)", 0, 500, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewProgressTracker(tt.total)
			tracker.SetDownloaded(tt.downloaded)

			info := tracker.Progress()
			if info.Percent != tt.expectedPercent {
				t.Errorf("Percent = %.2f, want %.2f", info.Percent, tt.expectedPercent)
			}
		})
	}
}

func TestProgressTracker_Progress_FormattedValues(t *testing.T) {
	tracker := NewProgressTracker(1024 * 1024) // 1 MB total
	tracker.SetDownloaded(512 * 1024)          // 512 KB downloaded

	info := tracker.Progress()

	// Check downloaded formatted
	expectedDownloaded := "512.00 KB"
	if info.DownloadedFormatted != expectedDownloaded {
		t.Errorf("DownloadedFormatted = %q, want %q", info.DownloadedFormatted, expectedDownloaded)
	}

	// Check total formatted
	expectedTotal := "1.00 MB"
	if info.TotalFormatted != expectedTotal {
		t.Errorf("TotalFormatted = %q, want %q", info.TotalFormatted, expectedTotal)
	}
}

func TestProgressTracker_Progress_UnknownTotal(t *testing.T) {
	tracker := NewProgressTracker(0) // Unknown total
	tracker.SetDownloaded(1024)

	info := tracker.Progress()

	if info.Percent != -1 {
		t.Errorf("Percent with unknown total = %.2f, want -1", info.Percent)
	}
	if info.TotalFormatted != "unknown" {
		t.Errorf("TotalFormatted with unknown total = %q, want %q", info.TotalFormatted, "unknown")
	}
}

func TestProgressTracker_IsComplete(t *testing.T) {
	tests := []struct {
		name       string
		total      int64
		downloaded int64
		expected   bool
	}{
		{"not complete", 1024, 512, false},
		{"exactly complete", 1024, 1024, true},
		{"over complete", 1024, 2048, true},
		{"unknown total (never complete)", 0, 1000, false},
		{"zero downloaded", 1024, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewProgressTracker(tt.total)
			tracker.SetDownloaded(tt.downloaded)

			if tracker.IsComplete() != tt.expected {
				t.Errorf("IsComplete() = %v, want %v", tracker.IsComplete(), tt.expected)
			}
		})
	}
}

func TestProgressTracker_Reset(t *testing.T) {
	tracker := NewProgressTracker(1024)
	tracker.Update(500)

	// Verify initial state
	if tracker.Downloaded() != 500 {
		t.Errorf("Before reset, Downloaded() = %d, want 500", tracker.Downloaded())
	}

	// Reset with new total
	tracker.Reset(2048)

	if tracker.Total() != 2048 {
		t.Errorf("After reset, Total() = %d, want 2048", tracker.Total())
	}
	if tracker.Downloaded() != 0 {
		t.Errorf("After reset, Downloaded() = %d, want 0", tracker.Downloaded())
	}
	if tracker.IsComplete() {
		t.Error("After reset, IsComplete() = true, want false")
	}
}

func TestProgressTracker_ThreadSafety(t *testing.T) {
	tracker := NewProgressTracker(10000)
	var wg sync.WaitGroup

	// Spawn multiple goroutines that update concurrently
	numGoroutines := 10
	updatesPerGoroutine := 100
	bytesPerUpdate := int64(10)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				tracker.Update(bytesPerUpdate)
			}
		}()
	}

	wg.Wait()

	expectedDownloaded := int64(numGoroutines * updatesPerGoroutine) * bytesPerUpdate
	if tracker.Downloaded() != expectedDownloaded {
		t.Errorf("After concurrent updates, Downloaded() = %d, want %d",
			tracker.Downloaded(), expectedDownloaded)
	}
}

func TestProgressTracker_ConcurrentReadWrite(t *testing.T) {
	tracker := NewProgressTracker(10000)
	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			tracker.Update(100)
			time.Sleep(time.Millisecond)
		}
	}()

	// Reader goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = tracker.Progress()
				_ = tracker.Downloaded()
				_ = tracker.Total()
				_ = tracker.IsComplete()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()

	// Should complete without panic or race condition
	if tracker.Downloaded() != 10000 {
		t.Errorf("After concurrent read/write, Downloaded() = %d, want 10000",
			tracker.Downloaded())
	}
}

func TestProgressTracker_ElapsedTime(t *testing.T) {
	tracker := NewProgressTracker(1024)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	info := tracker.Progress()
	if info.Elapsed < 50*time.Millisecond {
		t.Errorf("Elapsed = %v, want >= 50ms", info.Elapsed)
	}
}

func TestProgressTracker_SpeedCalculation(t *testing.T) {
	tracker := NewProgressTracker(10000)

	// Simulate download with known speed
	bytesPerUpdate := int64(1000)
	for i := 0; i < 5; i++ {
		tracker.Update(bytesPerUpdate)
		time.Sleep(100 * time.Millisecond)
	}

	info := tracker.Progress()

	// Speed should be approximately 1000 bytes / 0.1 seconds = 10000 bytes/sec
	// Allow wide tolerance due to timing variations
	if info.SpeedBytesPerSec < 1000 || info.SpeedBytesPerSec > 50000 {
		t.Errorf("SpeedBytesPerSec = %.2f, expected roughly 10000", info.SpeedBytesPerSec)
	}

	// SpeedFormatted should contain "/s"
	if len(info.SpeedFormatted) == 0 {
		t.Error("SpeedFormatted is empty")
	}
}

// Benchmark tests
func BenchmarkProgressTracker_Update(b *testing.B) {
	tracker := NewProgressTracker(int64(b.N) * 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.Update(100)
	}
}

func BenchmarkProgressTracker_Progress(b *testing.B) {
	tracker := NewProgressTracker(1024 * 1024)
	tracker.SetDownloaded(512 * 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.Progress()
	}
}

func BenchmarkProgressTracker_ConcurrentUpdate(b *testing.B) {
	tracker := NewProgressTracker(int64(b.N) * 100)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tracker.Update(100)
		}
	})
}
