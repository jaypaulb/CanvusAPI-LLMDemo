// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
// This file contains the health monitoring molecule for GPU and runtime health.
//
// The health module provides:
// - GPU detection and availability checking
// - Periodic GPU memory monitoring with logging
// - Interval-based health verification
// - Callback-based health status reporting
//
// Architecture:
// - Composes bindings.go (getGPUMemory, hasCUDA) and types.go (GPUInfo, HealthStatus)
// - Uses goroutines for periodic monitoring
// - Thread-safe with proper context cancellation
package llamaruntime

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// GPU Detection
// =============================================================================

// GPUDetectionResult contains the result of GPU detection.
type GPUDetectionResult struct {
	// Available indicates if a CUDA GPU is available.
	Available bool

	// GPUCount is the number of detected GPUs.
	GPUCount int

	// GPUs contains information about each detected GPU.
	GPUs []GPUInfo

	// TotalVRAM is the total VRAM across all GPUs in bytes.
	TotalVRAM int64

	// FreeVRAM is the available VRAM across all GPUs in bytes.
	FreeVRAM int64

	// CUDAVersion is the detected CUDA version (if available).
	CUDAVersion string

	// DriverVersion is the detected driver version (if available).
	DriverVersion string

	// Error is any error encountered during detection.
	Error error

	// DetectedAt is when the detection was performed.
	DetectedAt time.Time
}

// DetectGPU checks for available CUDA GPUs and returns their information.
// This is a point-in-time check - use GPUMonitor for continuous monitoring.
//
// Returns GPUDetectionResult with Available=false if no GPU is found
// or CUDA is not available. Never returns an error for "no GPU" -
// that's a valid result indicated by Available=false.
func DetectGPU() GPUDetectionResult {
	result := GPUDetectionResult{
		DetectedAt: time.Now(),
	}

	// Check if CUDA is available
	if !hasCUDA() {
		result.Available = false
		result.GPUCount = 0
		return result
	}

	// Get GPU memory information
	memInfo, err := getGPUMemory()
	if err != nil {
		result.Available = false
		result.Error = err
		return result
	}

	// Build GPU info from memory information
	gpuInfo := GPUInfo{
		Index:       0,
		TotalMemory: memInfo.Total,
		FreeMemory:  memInfo.Free,
		UsedMemory:  memInfo.Used,
		IsAvailable: true,
	}

	result.Available = true
	result.GPUCount = 1
	result.GPUs = []GPUInfo{gpuInfo}
	result.TotalVRAM = memInfo.Total
	result.FreeVRAM = memInfo.Free

	return result
}

// MustDetectGPU is like DetectGPU but returns an error if no GPU is available.
// Use this when GPU is required for the application to function.
func MustDetectGPU() (GPUDetectionResult, error) {
	result := DetectGPU()
	if !result.Available {
		if result.Error != nil {
			return result, fmt.Errorf("GPU detection failed: %w", result.Error)
		}
		return result, ErrGPUNotAvailable
	}
	return result, nil
}

// =============================================================================
// GPU Memory Monitor
// =============================================================================

// GPUMemoryCallback is called when GPU memory status is updated.
type GPUMemoryCallback func(info *GPUMemoryInfo)

// GPUMonitorConfig contains configuration for the GPU monitor.
type GPUMonitorConfig struct {
	// Interval is how often to check GPU memory.
	// Defaults to 5 seconds.
	Interval time.Duration

	// LogEnabled enables logging of GPU memory status.
	// Defaults to false.
	LogEnabled bool

	// LogPrefix is the prefix for log messages.
	// Defaults to "[GPU]".
	LogPrefix string

	// Logger is the logger to use. If nil, uses standard log package.
	Logger *log.Logger

	// Callback is called on each memory check. Optional.
	Callback GPUMemoryCallback

	// AlertThreshold is the VRAM usage percentage that triggers alerts.
	// Set to 0 to disable alerts. Defaults to 90 (90%).
	AlertThreshold float64
}

// DefaultGPUMonitorConfig returns a GPUMonitorConfig with sensible defaults.
func DefaultGPUMonitorConfig() GPUMonitorConfig {
	return GPUMonitorConfig{
		Interval:       5 * time.Second,
		LogEnabled:     false,
		LogPrefix:      "[GPU]",
		AlertThreshold: 90.0,
	}
}

// GPUMonitor periodically monitors GPU memory usage.
type GPUMonitor struct {
	config GPUMonitorConfig
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// State
	running    int32 // atomic
	lastInfo   *GPUMemoryInfo
	lastInfoMu sync.RWMutex
	alertCount int64 // atomic
	checkCount int64 // atomic
	errorCount int64 // atomic
	startTime  time.Time
}

// NewGPUMonitor creates a new GPU monitor with the given configuration.
func NewGPUMonitor(config GPUMonitorConfig) *GPUMonitor {
	// Apply defaults
	if config.Interval <= 0 {
		config.Interval = 5 * time.Second
	}
	if config.LogPrefix == "" {
		config.LogPrefix = "[GPU]"
	}
	if config.AlertThreshold == 0 {
		config.AlertThreshold = 90.0
	}

	return &GPUMonitor{
		config: config,
	}
}

// Start begins periodic GPU memory monitoring.
// The monitoring runs in a background goroutine until Stop is called
// or the context is cancelled.
//
// Start is safe to call multiple times - subsequent calls are no-ops.
func (m *GPUMonitor) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&m.running, 0, 1) {
		return nil // Already running
	}

	m.ctx, m.cancel = context.WithCancel(ctx)
	m.startTime = time.Now()

	m.wg.Add(1)
	go m.monitorLoop()

	return nil
}

// Stop stops the GPU monitor and waits for the monitoring goroutine to exit.
// Stop is safe to call multiple times.
func (m *GPUMonitor) Stop() {
	if !atomic.CompareAndSwapInt32(&m.running, 1, 0) {
		return // Not running
	}

	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
}

// IsRunning returns true if the monitor is currently running.
func (m *GPUMonitor) IsRunning() bool {
	return atomic.LoadInt32(&m.running) == 1
}

// LastInfo returns the most recent GPU memory information.
// Returns nil if no check has been performed yet.
func (m *GPUMonitor) LastInfo() *GPUMemoryInfo {
	m.lastInfoMu.RLock()
	defer m.lastInfoMu.RUnlock()
	return m.lastInfo
}

// Stats returns monitoring statistics.
func (m *GPUMonitor) Stats() GPUMonitorStats {
	m.lastInfoMu.RLock()
	lastInfo := m.lastInfo
	m.lastInfoMu.RUnlock()

	var uptime time.Duration
	if !m.startTime.IsZero() {
		uptime = time.Since(m.startTime)
	}

	return GPUMonitorStats{
		Running:    m.IsRunning(),
		CheckCount: atomic.LoadInt64(&m.checkCount),
		AlertCount: atomic.LoadInt64(&m.alertCount),
		ErrorCount: atomic.LoadInt64(&m.errorCount),
		LastInfo:   lastInfo,
		Uptime:     uptime,
	}
}

// GPUMonitorStats contains statistics from the GPU monitor.
type GPUMonitorStats struct {
	Running    bool
	CheckCount int64
	AlertCount int64
	ErrorCount int64
	LastInfo   *GPUMemoryInfo
	Uptime     time.Duration
}

// monitorLoop is the main monitoring goroutine.
func (m *GPUMonitor) monitorLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()

	// Do initial check immediately
	m.checkGPUMemory()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkGPUMemory()
		}
	}
}

// checkGPUMemory performs a single GPU memory check.
func (m *GPUMonitor) checkGPUMemory() {
	atomic.AddInt64(&m.checkCount, 1)

	info, err := getGPUMemory()
	if err != nil {
		atomic.AddInt64(&m.errorCount, 1)
		if m.config.LogEnabled {
			m.log("Error getting GPU memory: %v", err)
		}
		return
	}

	// Update last info
	m.lastInfoMu.Lock()
	m.lastInfo = info
	m.lastInfoMu.Unlock()

	// Log if enabled
	if m.config.LogEnabled {
		m.log("VRAM: %.1f%% used (%.2f GB / %.2f GB free)",
			info.UsedPct,
			float64(info.Used)/(1024*1024*1024),
			float64(info.Free)/(1024*1024*1024))
	}

	// Check alert threshold
	if m.config.AlertThreshold > 0 && info.UsedPct >= m.config.AlertThreshold {
		atomic.AddInt64(&m.alertCount, 1)
		if m.config.LogEnabled {
			m.log("WARNING: High VRAM usage (%.1f%% >= %.1f%% threshold)",
				info.UsedPct, m.config.AlertThreshold)
		}
	}

	// Invoke callback if provided
	if m.config.Callback != nil {
		m.config.Callback(info)
	}
}

// log outputs a log message with the configured prefix.
func (m *GPUMonitor) log(format string, args ...interface{}) {
	msg := fmt.Sprintf("%s %s", m.config.LogPrefix, fmt.Sprintf(format, args...))
	if m.config.Logger != nil {
		m.config.Logger.Println(msg)
	} else {
		log.Println(msg)
	}
}

// =============================================================================
// Periodic Health Check
// =============================================================================

// HealthCheckCallback is called when health status changes.
type HealthCheckCallback func(status *HealthStatus)

// HealthCheckerConfig contains configuration for the health checker.
type HealthCheckerConfig struct {
	// Interval is how often to perform health checks.
	// Defaults to 30 seconds.
	Interval time.Duration

	// Timeout is the maximum time for each health check.
	// Defaults to 10 seconds.
	Timeout time.Duration

	// Callback is called on each health check. Optional.
	Callback HealthCheckCallback

	// OnHealthy is called when status transitions to healthy.
	OnHealthy func()

	// OnUnhealthy is called when status transitions to unhealthy.
	OnUnhealthy func(reason string)

	// MinVRAMFree is the minimum free VRAM in bytes to consider healthy.
	// Defaults to 1 GB.
	MinVRAMFree int64

	// MaxErrorRate is the maximum error rate (0.0-1.0) to consider healthy.
	// Defaults to 0.1 (10%).
	MaxErrorRate float64
}

// DefaultHealthCheckerConfig returns a HealthCheckerConfig with sensible defaults.
func DefaultHealthCheckerConfig() HealthCheckerConfig {
	return HealthCheckerConfig{
		Interval:     30 * time.Second,
		Timeout:      10 * time.Second,
		MinVRAMFree:  1 * 1024 * 1024 * 1024, // 1 GB
		MaxErrorRate: 0.1,
	}
}

// HealthChecker performs periodic health verification.
type HealthChecker struct {
	config    HealthCheckerConfig
	client    *Client
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	// State
	running      int32 // atomic
	healthy      int32 // atomic (0 = unknown, 1 = healthy, 2 = unhealthy)
	lastStatus   *HealthStatus
	lastStatusMu sync.RWMutex
	checkCount   int64 // atomic
	failCount    int64 // atomic
	startTime    time.Time
}

// NewHealthChecker creates a new health checker for the given client.
func NewHealthChecker(client *Client, config HealthCheckerConfig) *HealthChecker {
	// Apply defaults
	if config.Interval <= 0 {
		config.Interval = 30 * time.Second
	}
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	if config.MinVRAMFree == 0 {
		config.MinVRAMFree = 1 * 1024 * 1024 * 1024
	}
	if config.MaxErrorRate == 0 {
		config.MaxErrorRate = 0.1
	}

	return &HealthChecker{
		config: config,
		client: client,
	}
}

// Start begins periodic health checking.
// The checking runs in a background goroutine until Stop is called
// or the context is cancelled.
func (h *HealthChecker) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&h.running, 0, 1) {
		return nil // Already running
	}

	h.ctx, h.cancel = context.WithCancel(ctx)
	h.startTime = time.Now()

	h.wg.Add(1)
	go h.checkLoop()

	return nil
}

// Stop stops the health checker and waits for the goroutine to exit.
func (h *HealthChecker) Stop() {
	if !atomic.CompareAndSwapInt32(&h.running, 1, 0) {
		return // Not running
	}

	if h.cancel != nil {
		h.cancel()
	}
	h.wg.Wait()
}

// IsRunning returns true if the health checker is currently running.
func (h *HealthChecker) IsRunning() bool {
	return atomic.LoadInt32(&h.running) == 1
}

// IsHealthy returns the current health status.
// Returns true if healthy, false if unhealthy or unknown.
func (h *HealthChecker) IsHealthy() bool {
	return atomic.LoadInt32(&h.healthy) == 1
}

// LastStatus returns the most recent health status.
// Returns nil if no check has been performed yet.
func (h *HealthChecker) LastStatus() *HealthStatus {
	h.lastStatusMu.RLock()
	defer h.lastStatusMu.RUnlock()
	return h.lastStatus
}

// Check performs an immediate health check (does not wait for interval).
// Returns the health status.
func (h *HealthChecker) Check() (*HealthStatus, error) {
	return h.performCheck()
}

// Stats returns health checking statistics.
func (h *HealthChecker) Stats() HealthCheckerStats {
	h.lastStatusMu.RLock()
	lastStatus := h.lastStatus
	h.lastStatusMu.RUnlock()

	var uptime time.Duration
	if !h.startTime.IsZero() {
		uptime = time.Since(h.startTime)
	}

	return HealthCheckerStats{
		Running:    h.IsRunning(),
		Healthy:    h.IsHealthy(),
		CheckCount: atomic.LoadInt64(&h.checkCount),
		FailCount:  atomic.LoadInt64(&h.failCount),
		LastStatus: lastStatus,
		Uptime:     uptime,
	}
}

// HealthCheckerStats contains statistics from the health checker.
type HealthCheckerStats struct {
	Running    bool
	Healthy    bool
	CheckCount int64
	FailCount  int64
	LastStatus *HealthStatus
	Uptime     time.Duration
}

// checkLoop is the main health checking goroutine.
func (h *HealthChecker) checkLoop() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()

	// Do initial check immediately
	h.performCheck()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.performCheck()
		}
	}
}

// performCheck performs a single health check.
func (h *HealthChecker) performCheck() (*HealthStatus, error) {
	atomic.AddInt64(&h.checkCount, 1)

	// Get health status from client
	status, err := h.client.HealthCheck()
	if err != nil {
		atomic.AddInt64(&h.failCount, 1)
		h.updateHealth(false, fmt.Sprintf("health check failed: %v", err))
		return nil, err
	}

	// Additional checks
	healthy := true
	var reason string

	// Check VRAM
	if status.GPUStatus != nil && status.GPUStatus.FreeMemory < h.config.MinVRAMFree {
		healthy = false
		reason = fmt.Sprintf("insufficient VRAM (free: %d bytes, required: %d bytes)",
			status.GPUStatus.FreeMemory, h.config.MinVRAMFree)
	}

	// Check error rate
	if status.Stats != nil && status.Stats.TotalInferences > 10 {
		errorRate := float64(status.Stats.ErrorCount) / float64(status.Stats.TotalInferences)
		if errorRate > h.config.MaxErrorRate {
			healthy = false
			reason = fmt.Sprintf("high error rate: %.2f%% (max: %.2f%%)",
				errorRate*100, h.config.MaxErrorRate*100)
		}
	}

	// Use client's health determination if we passed additional checks
	if healthy && !status.Healthy {
		healthy = false
		reason = status.Status
	}

	// Update status
	status.Healthy = healthy
	if !healthy && reason != "" {
		status.Status = reason
	}

	// Store and trigger callbacks
	h.lastStatusMu.Lock()
	h.lastStatus = status
	h.lastStatusMu.Unlock()

	h.updateHealth(healthy, reason)

	// Invoke callback if provided
	if h.config.Callback != nil {
		h.config.Callback(status)
	}

	return status, nil
}

// updateHealth updates the health status and triggers callbacks on transitions.
func (h *HealthChecker) updateHealth(healthy bool, reason string) {
	oldHealthy := atomic.LoadInt32(&h.healthy)

	var newHealthy int32 = 2 // unhealthy
	if healthy {
		newHealthy = 1 // healthy
	}

	if atomic.CompareAndSwapInt32(&h.healthy, oldHealthy, newHealthy) {
		// State changed
		if oldHealthy != 0 { // Skip callbacks on first check
			if healthy && h.config.OnHealthy != nil {
				h.config.OnHealthy()
			} else if !healthy && h.config.OnUnhealthy != nil {
				h.config.OnUnhealthy(reason)
			}
		}
	}
}

// =============================================================================
// Convenience Functions
// =============================================================================

// MonitorGPUMemory starts GPU memory monitoring with default settings
// and logs to the standard logger. Returns a stop function.
//
// Example:
//
//	stop := llamaruntime.MonitorGPUMemory(ctx)
//	defer stop()
func MonitorGPUMemory(ctx context.Context) func() {
	config := DefaultGPUMonitorConfig()
	config.LogEnabled = true

	monitor := NewGPUMonitor(config)
	monitor.Start(ctx)

	return monitor.Stop
}

// MonitorGPUMemoryWithCallback starts GPU memory monitoring with a callback.
// Returns a stop function.
func MonitorGPUMemoryWithCallback(ctx context.Context, interval time.Duration, callback GPUMemoryCallback) func() {
	config := DefaultGPUMonitorConfig()
	config.Interval = interval
	config.Callback = callback

	monitor := NewGPUMonitor(config)
	monitor.Start(ctx)

	return monitor.Stop
}

// PeriodicHealthCheck starts periodic health checking for a client.
// Returns a stop function and the health checker for querying status.
//
// Example:
//
//	stop, checker := llamaruntime.PeriodicHealthCheck(ctx, client, 30*time.Second)
//	defer stop()
//	if checker.IsHealthy() { ... }
func PeriodicHealthCheck(ctx context.Context, client *Client, interval time.Duration) (func(), *HealthChecker) {
	config := DefaultHealthCheckerConfig()
	config.Interval = interval

	checker := NewHealthChecker(client, config)
	checker.Start(ctx)

	return checker.Stop, checker
}

// PeriodicHealthCheckWithCallbacks starts periodic health checking with callbacks.
func PeriodicHealthCheckWithCallbacks(ctx context.Context, client *Client, config HealthCheckerConfig) (func(), *HealthChecker) {
	checker := NewHealthChecker(client, config)
	checker.Start(ctx)

	return checker.Stop, checker
}

// =============================================================================
// GPU Availability Check
// =============================================================================

// RequireGPU checks for GPU availability and returns an error if unavailable.
// This is a convenience function for startup checks.
func RequireGPU() error {
	result := DetectGPU()
	if !result.Available {
		if result.Error != nil {
			return &LlamaError{
				Op:      "RequireGPU",
				Code:    -1,
				Message: "GPU detection failed",
				Err:     result.Error,
			}
		}
		return ErrGPUNotAvailable
	}
	return nil
}

// CheckMinimumVRAM checks if minimum VRAM is available.
// Returns an error if VRAM is below the threshold.
func CheckMinimumVRAM(minBytes int64) error {
	result := DetectGPU()
	if !result.Available {
		return ErrGPUNotAvailable
	}

	if result.FreeVRAM < minBytes {
		return &LlamaError{
			Op:   "CheckMinimumVRAM",
			Code: -1,
			Message: fmt.Sprintf("insufficient VRAM: %d bytes available, %d bytes required",
				result.FreeVRAM, minBytes),
			Err: ErrInsufficientVRAM,
		}
	}

	return nil
}
