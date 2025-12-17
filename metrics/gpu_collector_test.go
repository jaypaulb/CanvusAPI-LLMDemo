package metrics

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDefaultGPUCollectorConfig(t *testing.T) {
	config := DefaultGPUCollectorConfig()

	if config.CollectionInterval != 5*time.Second {
		t.Errorf("expected CollectionInterval 5s, got %v", config.CollectionInterval)
	}

	if config.HistorySize != 720 {
		t.Errorf("expected HistorySize 720, got %d", config.HistorySize)
	}

	if config.NvidiaSMIPath != "nvidia-smi" {
		t.Errorf("expected NvidiaSMIPath 'nvidia-smi', got %s", config.NvidiaSMIPath)
	}
}

func TestNewGPUCollector(t *testing.T) {
	t.Run("creates collector with default config", func(t *testing.T) {
		config := DefaultGPUCollectorConfig()
		collector := NewGPUCollector(config, nil)
		defer collector.Stop()

		if collector == nil {
			t.Fatal("expected non-nil collector")
		}

		if collector.config.HistorySize != 720 {
			t.Errorf("expected HistorySize 720, got %d", collector.config.HistorySize)
		}
	})

	t.Run("handles invalid collection interval", func(t *testing.T) {
		config := GPUCollectorConfig{
			CollectionInterval: 100 * time.Millisecond, // Too short
			HistorySize:        10,
		}
		collector := NewGPUCollector(config, nil)
		defer collector.Stop()

		// Should default to 5 seconds
		if collector.config.CollectionInterval != 5*time.Second {
			t.Errorf("expected CollectionInterval 5s, got %v", collector.config.CollectionInterval)
		}
	})

	t.Run("handles zero history size", func(t *testing.T) {
		config := GPUCollectorConfig{
			CollectionInterval: 5 * time.Second,
			HistorySize:        0,
		}
		collector := NewGPUCollector(config, nil)
		defer collector.Stop()

		// Should default to 720
		if collector.config.HistorySize != 720 {
			t.Errorf("expected HistorySize 720, got %d", collector.config.HistorySize)
		}
	})
}

func TestGPUCollector_WithMockReader(t *testing.T) {
	expectedMetrics := GPUMetrics{
		Utilization: 75.0,
		Temperature: 65.0,
		MemoryTotal: 8 * 1024 * 1024 * 1024, // 8 GB
		MemoryUsed:  4 * 1024 * 1024 * 1024, // 4 GB
		MemoryFree:  4 * 1024 * 1024 * 1024, // 4 GB
	}

	mockReader := NewMockGPUReader(expectedMetrics)

	var receivedMetrics GPUMetrics
	var callbackCalled atomic.Int32
	callback := func(m GPUMetrics) {
		receivedMetrics = m
		callbackCalled.Add(1)
	}

	config := GPUCollectorConfig{
		CollectionInterval: 50 * time.Millisecond,
		HistorySize:        10,
	}

	collector := NewGPUCollectorWithReader(config, mockReader, callback)
	collector.Start()

	// Wait for at least one collection
	time.Sleep(100 * time.Millisecond)

	collector.Stop()

	if callbackCalled.Load() == 0 {
		t.Fatal("expected callback to be called at least once")
	}

	if !collector.IsAvailable() {
		t.Error("expected GPU to be available")
	}

	if collector.GetLastError() != nil {
		t.Errorf("expected no error, got %v", collector.GetLastError())
	}

	current := collector.GetCurrentMetrics()
	if current.Utilization != expectedMetrics.Utilization {
		t.Errorf("expected utilization %f, got %f", expectedMetrics.Utilization, current.Utilization)
	}

	if receivedMetrics.Utilization != expectedMetrics.Utilization {
		t.Errorf("callback received wrong utilization: got %f, want %f",
			receivedMetrics.Utilization, expectedMetrics.Utilization)
	}
}

func TestGPUCollector_History(t *testing.T) {
	t.Run("stores metrics in circular buffer", func(t *testing.T) {
		mockReader := NewMockGPUReader(GPUMetrics{Utilization: 50.0})

		config := GPUCollectorConfig{
			CollectionInterval: 10 * time.Millisecond,
			HistorySize:        5,
		}

		collector := NewGPUCollectorWithReader(config, mockReader, nil)
		collector.Start()

		// Wait for multiple collections
		time.Sleep(100 * time.Millisecond)

		collector.Stop()

		histSize := collector.GetHistorySize()
		if histSize == 0 {
			t.Fatal("expected non-empty history")
		}

		history := collector.GetHistory(histSize)
		if len(history) != histSize {
			t.Errorf("expected %d history entries, got %d", histSize, len(history))
		}
	})

	t.Run("respects history limit", func(t *testing.T) {
		mockReader := NewMockGPUReader(GPUMetrics{Utilization: 50.0})

		config := GPUCollectorConfig{
			CollectionInterval: 10 * time.Millisecond,
			HistorySize:        3,
		}

		collector := NewGPUCollectorWithReader(config, mockReader, nil)
		collector.Start()

		// Wait for more collections than history size
		time.Sleep(150 * time.Millisecond)

		collector.Stop()

		// History should be capped at 3
		histSize := collector.GetHistorySize()
		if histSize > 3 {
			t.Errorf("expected history size <= 3, got %d", histSize)
		}
	})

	t.Run("GetHistory handles various limits", func(t *testing.T) {
		mockReader := NewMockGPUReader(GPUMetrics{Utilization: 50.0})

		config := GPUCollectorConfig{
			CollectionInterval: 10 * time.Millisecond,
			HistorySize:        10,
		}

		collector := NewGPUCollectorWithReader(config, mockReader, nil)
		collector.Start()
		time.Sleep(100 * time.Millisecond)
		collector.Stop()

		// Zero limit
		history := collector.GetHistory(0)
		if len(history) != 0 {
			t.Errorf("expected 0 entries for limit 0, got %d", len(history))
		}

		// Negative limit
		history = collector.GetHistory(-1)
		if len(history) != 0 {
			t.Errorf("expected 0 entries for negative limit, got %d", len(history))
		}

		// Limit exceeds available
		histSize := collector.GetHistorySize()
		history = collector.GetHistory(histSize + 100)
		if len(history) != histSize {
			t.Errorf("expected %d entries (all available), got %d", histSize, len(history))
		}
	})
}

func TestGPUCollector_ErrorHandling(t *testing.T) {
	t.Run("handles reader error gracefully", func(t *testing.T) {
		mockReader := NewMockGPUReader(GPUMetrics{})
		mockReader.SetError(errors.New("GPU not available"))

		var callbackCalled atomic.Int32
		callback := func(m GPUMetrics) {
			callbackCalled.Add(1)
		}

		config := GPUCollectorConfig{
			CollectionInterval: 10 * time.Millisecond,
			HistorySize:        5,
		}

		collector := NewGPUCollectorWithReader(config, mockReader, callback)
		collector.Start()

		time.Sleep(50 * time.Millisecond)

		collector.Stop()

		// Should not be available due to error
		if collector.IsAvailable() {
			t.Error("expected GPU to not be available")
		}

		// Should have an error
		if collector.GetLastError() == nil {
			t.Error("expected error to be set")
		}

		// Callback should not have been called (error case)
		if callbackCalled.Load() != 0 {
			t.Errorf("expected callback not called on error, but was called %d times",
				callbackCalled.Load())
		}
	})

	t.Run("recovers from error when GPU becomes available", func(t *testing.T) {
		mockReader := NewMockGPUReader(GPUMetrics{})
		mockReader.SetError(errors.New("GPU not available"))

		var callbackCalled atomic.Int32
		callback := func(m GPUMetrics) {
			callbackCalled.Add(1)
		}

		config := GPUCollectorConfig{
			CollectionInterval: 10 * time.Millisecond,
			HistorySize:        5,
		}

		collector := NewGPUCollectorWithReader(config, mockReader, callback)
		collector.Start()

		// Wait for error state with polling
		deadline := time.Now().Add(200 * time.Millisecond)
		for time.Now().Before(deadline) && collector.IsAvailable() {
			time.Sleep(5 * time.Millisecond)
		}

		if collector.IsAvailable() {
			t.Error("expected GPU to not be available initially")
		}

		// Clear error and set metrics
		mockReader.SetError(nil)
		mockReader.SetMetrics(GPUMetrics{Utilization: 80.0})

		// Wait for recovery with polling
		deadline = time.Now().Add(200 * time.Millisecond)
		for time.Now().Before(deadline) && !collector.IsAvailable() {
			time.Sleep(5 * time.Millisecond)
		}

		collector.Stop()

		// Should now be available
		if !collector.IsAvailable() {
			t.Error("expected GPU to be available after recovery")
		}

		// Error should be cleared
		if collector.GetLastError() != nil {
			t.Errorf("expected no error after recovery, got %v", collector.GetLastError())
		}

		// Callback should have been called at least once
		if callbackCalled.Load() == 0 {
			t.Error("expected callback to be called after recovery")
		}
	})
}

func TestGPUCollector_Concurrency(t *testing.T) {
	mockReader := NewMockGPUReader(GPUMetrics{Utilization: 50.0})

	config := GPUCollectorConfig{
		CollectionInterval: 5 * time.Millisecond,
		HistorySize:        100,
	}

	collector := NewGPUCollectorWithReader(config, mockReader, nil)
	collector.Start()

	var wg sync.WaitGroup
	// Start multiple concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = collector.GetCurrentMetrics()
				_ = collector.IsAvailable()
				_ = collector.GetLastError()
				_ = collector.GetHistory(10)
				_ = collector.GetHistorySize()
			}
		}()
	}

	wg.Wait()
	collector.Stop()

	// If we got here without panic, concurrency is safe
}

func TestParseNvidiaSMIOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    GPUMetrics
		wantErr bool
	}{
		{
			name:   "valid output",
			output: "75, 65, 4096, 8192",
			want: GPUMetrics{
				Utilization: 75.0,
				Temperature: 65.0,
				MemoryTotal: 8192 * 1024 * 1024,
				MemoryUsed:  4096 * 1024 * 1024,
				MemoryFree:  4096 * 1024 * 1024,
			},
			wantErr: false,
		},
		{
			name:   "valid output with whitespace",
			output: "  50 , 70 , 2048 , 4096  \n",
			want: GPUMetrics{
				Utilization: 50.0,
				Temperature: 70.0,
				MemoryTotal: 4096 * 1024 * 1024,
				MemoryUsed:  2048 * 1024 * 1024,
				MemoryFree:  2048 * 1024 * 1024,
			},
			wantErr: false,
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			output:  "   \n  ",
			wantErr: true,
		},
		{
			name:    "insufficient fields",
			output:  "75, 65, 4096",
			wantErr: true,
		},
		{
			name:    "invalid utilization",
			output:  "abc, 65, 4096, 8192",
			wantErr: true,
		},
		{
			name:    "invalid temperature",
			output:  "75, xyz, 4096, 8192",
			wantErr: true,
		},
		{
			name:    "invalid memory used",
			output:  "75, 65, bad, 8192",
			wantErr: true,
		},
		{
			name:    "invalid memory total",
			output:  "75, 65, 4096, bad",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNvidiaSMIOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNvidiaSMIOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Utilization != tt.want.Utilization {
					t.Errorf("Utilization = %v, want %v", got.Utilization, tt.want.Utilization)
				}
				if got.Temperature != tt.want.Temperature {
					t.Errorf("Temperature = %v, want %v", got.Temperature, tt.want.Temperature)
				}
				if got.MemoryTotal != tt.want.MemoryTotal {
					t.Errorf("MemoryTotal = %v, want %v", got.MemoryTotal, tt.want.MemoryTotal)
				}
				if got.MemoryUsed != tt.want.MemoryUsed {
					t.Errorf("MemoryUsed = %v, want %v", got.MemoryUsed, tt.want.MemoryUsed)
				}
				if got.MemoryFree != tt.want.MemoryFree {
					t.Errorf("MemoryFree = %v, want %v", got.MemoryFree, tt.want.MemoryFree)
				}
			}
		})
	}
}

func TestMockGPUReader(t *testing.T) {
	t.Run("returns configured metrics", func(t *testing.T) {
		expected := GPUMetrics{
			Utilization: 42.0,
			Temperature: 55.0,
			MemoryTotal: 16 * 1024 * 1024 * 1024,
			MemoryUsed:  8 * 1024 * 1024 * 1024,
			MemoryFree:  8 * 1024 * 1024 * 1024,
		}

		mock := NewMockGPUReader(expected)
		got, err := mock.ReadGPUMetrics()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if got.Utilization != expected.Utilization {
			t.Errorf("Utilization = %v, want %v", got.Utilization, expected.Utilization)
		}
	})

	t.Run("returns configured error", func(t *testing.T) {
		mock := NewMockGPUReader(GPUMetrics{})
		expectedErr := errors.New("test error")
		mock.SetError(expectedErr)

		_, err := mock.ReadGPUMetrics()

		if err == nil {
			t.Error("expected error, got nil")
		}

		if err.Error() != expectedErr.Error() {
			t.Errorf("error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("tracks call count", func(t *testing.T) {
		mock := NewMockGPUReader(GPUMetrics{Utilization: 50.0})

		if mock.CallCount() != 0 {
			t.Errorf("initial call count = %d, want 0", mock.CallCount())
		}

		mock.ReadGPUMetrics()
		mock.ReadGPUMetrics()
		mock.ReadGPUMetrics()

		if mock.CallCount() != 3 {
			t.Errorf("call count = %d, want 3", mock.CallCount())
		}
	})

	t.Run("allows updating metrics", func(t *testing.T) {
		mock := NewMockGPUReader(GPUMetrics{Utilization: 50.0})

		got1, _ := mock.ReadGPUMetrics()
		if got1.Utilization != 50.0 {
			t.Errorf("initial utilization = %v, want 50.0", got1.Utilization)
		}

		mock.SetMetrics(GPUMetrics{Utilization: 75.0})

		got2, _ := mock.ReadGPUMetrics()
		if got2.Utilization != 75.0 {
			t.Errorf("updated utilization = %v, want 75.0", got2.Utilization)
		}
	})
}

func TestGPUCollector_StopIdempotent(t *testing.T) {
	mockReader := NewMockGPUReader(GPUMetrics{Utilization: 50.0})

	config := GPUCollectorConfig{
		CollectionInterval: 50 * time.Millisecond,
		HistorySize:        5,
	}

	collector := NewGPUCollectorWithReader(config, mockReader, nil)
	collector.Start()

	time.Sleep(100 * time.Millisecond)

	// Multiple stops should be safe
	collector.Stop()
	collector.Stop()
	collector.Stop()
}

func TestGPUCollector_CallbackNotCalledAfterStop(t *testing.T) {
	mockReader := NewMockGPUReader(GPUMetrics{Utilization: 50.0})

	var callbackCalled atomic.Int32
	callback := func(m GPUMetrics) {
		callbackCalled.Add(1)
	}

	config := GPUCollectorConfig{
		CollectionInterval: 10 * time.Millisecond,
		HistorySize:        5,
	}

	collector := NewGPUCollectorWithReader(config, mockReader, callback)
	collector.Start()

	time.Sleep(50 * time.Millisecond)

	collector.Stop()

	countAtStop := callbackCalled.Load()

	// Wait a bit more to ensure no more callbacks
	time.Sleep(50 * time.Millisecond)

	if callbackCalled.Load() != countAtStop {
		t.Errorf("callback called after stop: was %d at stop, now %d",
			countAtStop, callbackCalled.Load())
	}
}
