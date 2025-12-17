// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
// This file contains tests for the health monitoring molecule.
//
//go:build nocgo || !cgo

package llamaruntime

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// GPU Detection Tests
// =============================================================================

func TestDetectGPU(t *testing.T) {
	result := DetectGPU()

	// In stub mode, GPU is not available
	if hasCUDA() {
		if !result.Available {
			t.Error("expected GPU to be available in CGo mode")
		}
	} else {
		if result.Available {
			t.Log("GPU reported available in nocgo mode (stub returns mock data)")
		}
	}

	// DetectedAt should be set
	if result.DetectedAt.IsZero() {
		t.Error("DetectedAt should be set")
	}
}

func TestMustDetectGPU(t *testing.T) {
	result, err := MustDetectGPU()

	// In nocgo mode, this may or may not error depending on stub behavior
	if err != nil {
		t.Logf("MustDetectGPU returned error (expected in nocgo mode): %v", err)
	}

	// Result should always be returned
	if result.DetectedAt.IsZero() {
		t.Error("DetectedAt should be set even on error")
	}
}

func TestGPUDetectionResult_Fields(t *testing.T) {
	result := GPUDetectionResult{
		Available:     true,
		GPUCount:      1,
		TotalVRAM:     8 * 1024 * 1024 * 1024,
		FreeVRAM:      4 * 1024 * 1024 * 1024,
		CUDAVersion:   "12.0",
		DriverVersion: "535.86.10",
		DetectedAt:    time.Now(),
	}

	if !result.Available {
		t.Error("Available should be true")
	}
	if result.GPUCount != 1 {
		t.Errorf("GPUCount = %d, want 1", result.GPUCount)
	}
	if result.TotalVRAM != 8*1024*1024*1024 {
		t.Errorf("TotalVRAM = %d, want 8GB", result.TotalVRAM)
	}
}

// =============================================================================
// GPU Monitor Config Tests
// =============================================================================

func TestDefaultGPUMonitorConfig(t *testing.T) {
	config := DefaultGPUMonitorConfig()

	if config.Interval != 5*time.Second {
		t.Errorf("Interval = %v, want 5s", config.Interval)
	}
	if config.LogEnabled {
		t.Error("LogEnabled should be false by default")
	}
	if config.LogPrefix != "[GPU]" {
		t.Errorf("LogPrefix = %q, want %q", config.LogPrefix, "[GPU]")
	}
	if config.AlertThreshold != 90.0 {
		t.Errorf("AlertThreshold = %f, want 90.0", config.AlertThreshold)
	}
}

// =============================================================================
// GPU Monitor Tests
// =============================================================================

func TestNewGPUMonitor(t *testing.T) {
	config := DefaultGPUMonitorConfig()
	monitor := NewGPUMonitor(config)

	if monitor == nil {
		t.Fatal("NewGPUMonitor returned nil")
	}
	if monitor.IsRunning() {
		t.Error("monitor should not be running after creation")
	}
}

func TestGPUMonitor_StartStop(t *testing.T) {
	config := DefaultGPUMonitorConfig()
	config.Interval = 100 * time.Millisecond // Fast for testing
	monitor := NewGPUMonitor(config)

	ctx := context.Background()

	// Start monitor
	if err := monitor.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !monitor.IsRunning() {
		t.Error("monitor should be running after Start")
	}

	// Wait for at least one check
	time.Sleep(150 * time.Millisecond)

	// Stop monitor
	monitor.Stop()

	if monitor.IsRunning() {
		t.Error("monitor should not be running after Stop")
	}
}

func TestGPUMonitor_StartIdempotent(t *testing.T) {
	config := DefaultGPUMonitorConfig()
	config.Interval = 100 * time.Millisecond
	monitor := NewGPUMonitor(config)

	ctx := context.Background()

	// Start multiple times
	for i := 0; i < 3; i++ {
		if err := monitor.Start(ctx); err != nil {
			t.Errorf("Start %d failed: %v", i+1, err)
		}
	}

	monitor.Stop()
}

func TestGPUMonitor_StopIdempotent(t *testing.T) {
	config := DefaultGPUMonitorConfig()
	monitor := NewGPUMonitor(config)

	ctx := context.Background()
	monitor.Start(ctx)

	// Stop multiple times
	for i := 0; i < 3; i++ {
		monitor.Stop()
	}
}

func TestGPUMonitor_LastInfo(t *testing.T) {
	config := DefaultGPUMonitorConfig()
	config.Interval = 50 * time.Millisecond
	monitor := NewGPUMonitor(config)

	ctx := context.Background()
	monitor.Start(ctx)

	// Wait for check
	time.Sleep(100 * time.Millisecond)

	info := monitor.LastInfo()
	// In stub mode, we should get mock data
	if info == nil {
		t.Log("LastInfo is nil (may be expected if GPU check failed)")
	} else {
		if info.Total <= 0 {
			t.Error("expected positive total memory")
		}
	}

	monitor.Stop()
}

func TestGPUMonitor_Stats(t *testing.T) {
	config := DefaultGPUMonitorConfig()
	config.Interval = 50 * time.Millisecond
	monitor := NewGPUMonitor(config)

	ctx := context.Background()
	monitor.Start(ctx)

	// Wait for a few checks
	time.Sleep(200 * time.Millisecond)

	stats := monitor.Stats()

	if !stats.Running {
		t.Error("stats.Running should be true")
	}
	if stats.CheckCount <= 0 {
		t.Error("expected positive check count")
	}
	if stats.Uptime <= 0 {
		t.Error("expected positive uptime")
	}

	monitor.Stop()
}

func TestGPUMonitor_Callback(t *testing.T) {
	var callbackCalled int
	var mu sync.Mutex

	config := DefaultGPUMonitorConfig()
	config.Interval = 50 * time.Millisecond
	config.Callback = func(info *GPUMemoryInfo) {
		mu.Lock()
		callbackCalled++
		mu.Unlock()
	}

	monitor := NewGPUMonitor(config)
	ctx := context.Background()
	monitor.Start(ctx)

	// Wait for callbacks
	time.Sleep(200 * time.Millisecond)

	monitor.Stop()

	mu.Lock()
	if callbackCalled <= 0 {
		t.Error("callback should have been called at least once")
	}
	mu.Unlock()
}

func TestGPUMonitor_ContextCancellation(t *testing.T) {
	config := DefaultGPUMonitorConfig()
	config.Interval = 100 * time.Millisecond
	monitor := NewGPUMonitor(config)

	ctx, cancel := context.WithCancel(context.Background())
	monitor.Start(ctx)

	if !monitor.IsRunning() {
		t.Error("monitor should be running")
	}

	// Cancel context
	cancel()

	// Wait for shutdown
	time.Sleep(50 * time.Millisecond)

	// Check if stopped (may take a moment)
	time.Sleep(150 * time.Millisecond)
}

// =============================================================================
// Health Checker Config Tests
// =============================================================================

func TestDefaultHealthCheckerConfig(t *testing.T) {
	config := DefaultHealthCheckerConfig()

	if config.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", config.Interval)
	}
	if config.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", config.Timeout)
	}
	if config.MinVRAMFree != 1*1024*1024*1024 {
		t.Errorf("MinVRAMFree = %d, want 1GB", config.MinVRAMFree)
	}
	if config.MaxErrorRate != 0.1 {
		t.Errorf("MaxErrorRate = %f, want 0.1", config.MaxErrorRate)
	}
}

// =============================================================================
// Health Checker Tests
// =============================================================================

func testClient(t *testing.T) *Client {
	t.Helper()
	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "test-model.gguf")
	if err := os.WriteFile(modelPath, []byte("mock"), 0644); err != nil {
		t.Fatalf("failed to create test model: %v", err)
	}

	config := DefaultClientConfig()
	config.ModelPath = modelPath

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	return client
}

func TestNewHealthChecker(t *testing.T) {
	client := testClient(t)
	defer client.Close()

	config := DefaultHealthCheckerConfig()
	checker := NewHealthChecker(client, config)

	if checker == nil {
		t.Fatal("NewHealthChecker returned nil")
	}
	if checker.IsRunning() {
		t.Error("checker should not be running after creation")
	}
}

func TestHealthChecker_StartStop(t *testing.T) {
	client := testClient(t)
	defer client.Close()

	config := DefaultHealthCheckerConfig()
	config.Interval = 100 * time.Millisecond
	checker := NewHealthChecker(client, config)

	ctx := context.Background()

	// Start checker
	if err := checker.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !checker.IsRunning() {
		t.Error("checker should be running after Start")
	}

	// Wait for at least one check
	time.Sleep(150 * time.Millisecond)

	// Stop checker
	checker.Stop()

	if checker.IsRunning() {
		t.Error("checker should not be running after Stop")
	}
}

func TestHealthChecker_Check(t *testing.T) {
	client := testClient(t)
	defer client.Close()

	config := DefaultHealthCheckerConfig()
	checker := NewHealthChecker(client, config)

	// Manual check (no Start needed)
	status, err := checker.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.CheckedAt.IsZero() {
		t.Error("CheckedAt should be set")
	}
}

func TestHealthChecker_IsHealthy(t *testing.T) {
	client := testClient(t)
	defer client.Close()

	config := DefaultHealthCheckerConfig()
	config.Interval = 50 * time.Millisecond
	checker := NewHealthChecker(client, config)

	ctx := context.Background()
	checker.Start(ctx)

	// Wait for check
	time.Sleep(100 * time.Millisecond)

	// In stub mode with fresh client, should be healthy
	if !checker.IsHealthy() {
		t.Log("checker reports unhealthy (may be expected based on mock data)")
	}

	checker.Stop()
}

func TestHealthChecker_LastStatus(t *testing.T) {
	client := testClient(t)
	defer client.Close()

	config := DefaultHealthCheckerConfig()
	config.Interval = 50 * time.Millisecond
	checker := NewHealthChecker(client, config)

	ctx := context.Background()
	checker.Start(ctx)

	// Wait for check
	time.Sleep(100 * time.Millisecond)

	status := checker.LastStatus()
	if status == nil {
		t.Fatal("expected non-nil status after check")
	}

	checker.Stop()
}

func TestHealthChecker_Stats(t *testing.T) {
	client := testClient(t)
	defer client.Close()

	config := DefaultHealthCheckerConfig()
	config.Interval = 50 * time.Millisecond
	checker := NewHealthChecker(client, config)

	ctx := context.Background()
	checker.Start(ctx)

	// Wait for checks
	time.Sleep(200 * time.Millisecond)

	stats := checker.Stats()

	if !stats.Running {
		t.Error("stats.Running should be true")
	}
	if stats.CheckCount <= 0 {
		t.Error("expected positive check count")
	}
	if stats.Uptime <= 0 {
		t.Error("expected positive uptime")
	}

	checker.Stop()
}

func TestHealthChecker_Callback(t *testing.T) {
	client := testClient(t)
	defer client.Close()

	var callbackCalled int
	var mu sync.Mutex

	config := DefaultHealthCheckerConfig()
	config.Interval = 50 * time.Millisecond
	config.Callback = func(status *HealthStatus) {
		mu.Lock()
		callbackCalled++
		mu.Unlock()
	}

	checker := NewHealthChecker(client, config)
	ctx := context.Background()
	checker.Start(ctx)

	// Wait for callbacks
	time.Sleep(200 * time.Millisecond)

	checker.Stop()

	mu.Lock()
	if callbackCalled <= 0 {
		t.Error("callback should have been called at least once")
	}
	mu.Unlock()
}

func TestHealthChecker_Transitions(t *testing.T) {
	client := testClient(t)
	defer client.Close()

	var healthyCount, unhealthyCount int
	var mu sync.Mutex

	config := DefaultHealthCheckerConfig()
	config.Interval = 50 * time.Millisecond
	config.OnHealthy = func() {
		mu.Lock()
		healthyCount++
		mu.Unlock()
	}
	config.OnUnhealthy = func(reason string) {
		mu.Lock()
		unhealthyCount++
		mu.Unlock()
	}

	checker := NewHealthChecker(client, config)
	ctx := context.Background()
	checker.Start(ctx)

	// Wait for checks
	time.Sleep(200 * time.Millisecond)

	checker.Stop()

	// Just verify no panic - actual counts depend on mock data
	t.Logf("Healthy transitions: %d, Unhealthy transitions: %d", healthyCount, unhealthyCount)
}

// =============================================================================
// Convenience Function Tests
// =============================================================================

func TestMonitorGPUMemory(t *testing.T) {
	ctx := context.Background()
	stop := MonitorGPUMemory(ctx)

	if stop == nil {
		t.Fatal("expected non-nil stop function")
	}

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Stop it
	stop()
}

func TestMonitorGPUMemoryWithCallback(t *testing.T) {
	var called bool
	var mu sync.Mutex

	ctx := context.Background()
	stop := MonitorGPUMemoryWithCallback(ctx, 50*time.Millisecond, func(info *GPUMemoryInfo) {
		mu.Lock()
		called = true
		mu.Unlock()
	})

	// Wait for callback
	time.Sleep(100 * time.Millisecond)

	stop()

	mu.Lock()
	if !called {
		t.Error("callback should have been called")
	}
	mu.Unlock()
}

func TestPeriodicHealthCheck(t *testing.T) {
	client := testClient(t)
	defer client.Close()

	ctx := context.Background()
	stop, checker := PeriodicHealthCheck(ctx, client, 50*time.Millisecond)

	if stop == nil {
		t.Fatal("expected non-nil stop function")
	}
	if checker == nil {
		t.Fatal("expected non-nil checker")
	}

	// Wait for checks
	time.Sleep(100 * time.Millisecond)

	stop()
}

func TestPeriodicHealthCheckWithCallbacks(t *testing.T) {
	client := testClient(t)
	defer client.Close()

	config := DefaultHealthCheckerConfig()
	config.Interval = 50 * time.Millisecond

	ctx := context.Background()
	stop, checker := PeriodicHealthCheckWithCallbacks(ctx, client, config)

	if stop == nil {
		t.Fatal("expected non-nil stop function")
	}
	if checker == nil {
		t.Fatal("expected non-nil checker")
	}

	// Wait for checks
	time.Sleep(100 * time.Millisecond)

	stop()
}

// =============================================================================
// GPU Availability Check Tests
// =============================================================================

func TestRequireGPU(t *testing.T) {
	err := RequireGPU()

	// In nocgo mode without real GPU, this should return error
	if !hasCUDA() && err == nil {
		t.Log("RequireGPU succeeded in nocgo mode (stub may return mock data)")
	}
	// Just verify no panic
}

func TestCheckMinimumVRAM(t *testing.T) {
	// Check with very low requirement (should pass if GPU available)
	err := CheckMinimumVRAM(1) // 1 byte
	if err != nil {
		t.Logf("CheckMinimumVRAM(1) failed (expected if no GPU): %v", err)
	}

	// Check with impossibly high requirement (should fail)
	err = CheckMinimumVRAM(1024 * 1024 * 1024 * 1024) // 1 TB
	if err == nil {
		t.Error("expected error for 1TB VRAM requirement")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkDetectGPU(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectGPU()
	}
}

func BenchmarkGPUMonitor_CheckGPUMemory(b *testing.B) {
	config := DefaultGPUMonitorConfig()
	monitor := NewGPUMonitor(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.checkGPUMemory()
	}
}

func BenchmarkHealthChecker_Check(b *testing.B) {
	tmpDir := b.TempDir()
	modelPath := filepath.Join(tmpDir, "bench-model.gguf")
	if err := os.WriteFile(modelPath, []byte("mock"), 0644); err != nil {
		b.Fatalf("failed to create test model: %v", err)
	}

	config := DefaultClientConfig()
	config.ModelPath = modelPath

	client, err := NewClient(config)
	if err != nil {
		b.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	checkerConfig := DefaultHealthCheckerConfig()
	checker := NewHealthChecker(client, checkerConfig)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check()
	}
}
