// Package metrics provides the GPUCollector organism for GPU metrics collection.
// This file contains the GPUCollector which periodically collects GPU metrics
// and updates the MetricsStore.
package metrics

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GPUReader is the interface for reading GPU metrics.
// This abstraction allows for mock implementations during testing.
type GPUReader interface {
	// ReadGPUMetrics reads the current GPU metrics.
	// Returns an error if the GPU is unavailable or metrics cannot be read.
	ReadGPUMetrics() (GPUMetrics, error)
}

// GPUCollectorConfig configures the GPUCollector behavior.
type GPUCollectorConfig struct {
	// CollectionInterval is how often to collect GPU metrics
	CollectionInterval time.Duration

	// HistorySize is the number of samples to retain (720 = 1 hour at 5s intervals)
	HistorySize int

	// NvidiaSMIPath is the path to the nvidia-smi executable
	// If empty, uses "nvidia-smi" and relies on PATH
	NvidiaSMIPath string
}

// DefaultGPUCollectorConfig returns a default configuration.
func DefaultGPUCollectorConfig() GPUCollectorConfig {
	return GPUCollectorConfig{
		CollectionInterval: 5 * time.Second,
		HistorySize:        720, // 1 hour at 5s intervals
		NvidiaSMIPath:      "nvidia-smi",
	}
}

// GPUCollector is an organism that periodically collects GPU metrics.
// It uses nvidia-smi to query GPU state and stores historical samples.
//
// This organism composes:
// - GPUMetrics atoms for data representation
// - MetricsStore for metrics aggregation (via callback)
// - CircularBuffer pattern for history storage
type GPUCollector struct {
	mu sync.RWMutex

	config GPUCollectorConfig
	reader GPUReader

	// History storage (circular buffer)
	history   []GPUMetrics
	histHead  int
	histSize  int
	histCap   int

	// Current state
	lastMetrics GPUMetrics
	available   bool
	lastError   error

	// Callback to update MetricsStore
	onMetrics func(GPUMetrics)

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewGPUCollector creates a new GPUCollector with the specified configuration.
// The onMetrics callback is invoked each time new metrics are collected.
func NewGPUCollector(config GPUCollectorConfig, onMetrics func(GPUMetrics)) *GPUCollector {
	if config.CollectionInterval < time.Second {
		config.CollectionInterval = 5 * time.Second
	}
	if config.HistorySize < 1 {
		config.HistorySize = 720
	}
	if config.NvidiaSMIPath == "" {
		config.NvidiaSMIPath = "nvidia-smi"
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &GPUCollector{
		config:    config,
		history:   make([]GPUMetrics, config.HistorySize),
		histCap:   config.HistorySize,
		onMetrics: onMetrics,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// NewGPUCollectorWithReader creates a GPUCollector with a custom GPUReader.
// This is primarily used for testing.
func NewGPUCollectorWithReader(config GPUCollectorConfig, reader GPUReader, onMetrics func(GPUMetrics)) *GPUCollector {
	c := NewGPUCollector(config, onMetrics)
	c.reader = reader
	return c
}

// Start begins the periodic GPU metrics collection.
// This method is non-blocking; metrics are collected in a background goroutine.
func (c *GPUCollector) Start() {
	c.wg.Add(1)
	go c.collectLoop()
}

// Stop halts the GPU metrics collection.
// This method blocks until the collection goroutine has stopped.
func (c *GPUCollector) Stop() {
	c.cancel()
	c.wg.Wait()
}

// IsAvailable returns true if the GPU is available for metrics collection.
func (c *GPUCollector) IsAvailable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.available
}

// GetLastError returns the most recent error from metrics collection.
// Returns nil if no error occurred.
func (c *GPUCollector) GetLastError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastError
}

// GetCurrentMetrics returns the most recently collected GPU metrics.
func (c *GPUCollector) GetCurrentMetrics() GPUMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastMetrics
}

// GetHistory returns the last N samples of GPU metrics.
// If limit exceeds available samples, all available samples are returned.
// Samples are ordered oldest-first.
func (c *GPUCollector) GetHistory(limit int) []GPUMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 || c.histSize == 0 {
		return []GPUMetrics{}
	}

	if limit > c.histSize {
		limit = c.histSize
	}

	result := make([]GPUMetrics, limit)
	for i := 0; i < limit; i++ {
		// Calculate index to read oldest-first within the requested range
		idx := (c.histHead - c.histSize + i + c.histCap) % c.histCap
		result[i] = c.history[idx]
	}

	return result
}

// GetHistorySize returns the current number of samples in history.
func (c *GPUCollector) GetHistorySize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.histSize
}

// collectLoop is the main collection goroutine.
func (c *GPUCollector) collectLoop() {
	defer c.wg.Done()

	// Collect immediately on start
	c.collectOnce()

	ticker := time.NewTicker(c.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collectOnce()
		}
	}
}

// collectOnce performs a single metrics collection.
func (c *GPUCollector) collectOnce() {
	var metrics GPUMetrics
	var err error

	if c.reader != nil {
		metrics, err = c.reader.ReadGPUMetrics()
	} else {
		metrics, err = c.readNvidiaSMI()
	}

	c.mu.Lock()
	if err != nil {
		c.available = false
		c.lastError = err
		// Keep last valid metrics but don't add to history
	} else {
		c.available = true
		c.lastError = nil
		c.lastMetrics = metrics

		// Add to circular buffer
		c.history[c.histHead] = metrics
		c.histHead = (c.histHead + 1) % c.histCap
		if c.histSize < c.histCap {
			c.histSize++
		}
	}
	currentMetrics := c.lastMetrics
	c.mu.Unlock()

	// Invoke callback outside of lock
	if c.onMetrics != nil && err == nil {
		c.onMetrics(currentMetrics)
	}
}

// readNvidiaSMI queries nvidia-smi for GPU metrics.
func (c *GPUCollector) readNvidiaSMI() (GPUMetrics, error) {
	// Query: utilization, temperature, memory used, memory total
	// Format: CSV with headers
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.config.NvidiaSMIPath,
		"--query-gpu=utilization.gpu,temperature.gpu,memory.used,memory.total",
		"--format=csv,noheader,nounits")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return GPUMetrics{}, fmt.Errorf("nvidia-smi failed: %w (stderr: %s)", err, stderr.String())
	}

	return parseNvidiaSMIOutput(stdout.String())
}

// parseNvidiaSMIOutput parses the CSV output from nvidia-smi.
func parseNvidiaSMIOutput(output string) (GPUMetrics, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return GPUMetrics{}, fmt.Errorf("empty nvidia-smi output")
	}

	reader := csv.NewReader(strings.NewReader(output))
	record, err := reader.Read()
	if err != nil {
		return GPUMetrics{}, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(record) < 4 {
		return GPUMetrics{}, fmt.Errorf("unexpected field count: got %d, expected 4", len(record))
	}

	// Parse utilization (%)
	util, err := strconv.ParseFloat(strings.TrimSpace(record[0]), 64)
	if err != nil {
		return GPUMetrics{}, fmt.Errorf("failed to parse utilization: %w", err)
	}

	// Parse temperature (C)
	temp, err := strconv.ParseFloat(strings.TrimSpace(record[1]), 64)
	if err != nil {
		return GPUMetrics{}, fmt.Errorf("failed to parse temperature: %w", err)
	}

	// Parse memory used (MiB)
	memUsedMiB, err := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)
	if err != nil {
		return GPUMetrics{}, fmt.Errorf("failed to parse memory used: %w", err)
	}

	// Parse memory total (MiB)
	memTotalMiB, err := strconv.ParseFloat(strings.TrimSpace(record[3]), 64)
	if err != nil {
		return GPUMetrics{}, fmt.Errorf("failed to parse memory total: %w", err)
	}

	// Convert MiB to bytes
	const mibToBytes = 1024 * 1024
	memTotal := int64(memTotalMiB * mibToBytes)
	memUsed := int64(memUsedMiB * mibToBytes)
	memFree := memTotal - memUsed

	return GPUMetrics{
		Utilization: util,
		Temperature: temp,
		MemoryTotal: memTotal,
		MemoryUsed:  memUsed,
		MemoryFree:  memFree,
	}, nil
}

// MockGPUReader is a mock implementation of GPUReader for testing.
type MockGPUReader struct {
	mu      sync.Mutex
	metrics GPUMetrics
	err     error
	calls   int
}

// NewMockGPUReader creates a new mock GPU reader with the specified metrics.
func NewMockGPUReader(metrics GPUMetrics) *MockGPUReader {
	return &MockGPUReader{metrics: metrics}
}

// SetMetrics updates the metrics returned by this mock.
func (m *MockGPUReader) SetMetrics(metrics GPUMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = metrics
}

// SetError sets an error to be returned by ReadGPUMetrics.
func (m *MockGPUReader) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// ReadGPUMetrics returns the configured mock metrics or error.
func (m *MockGPUReader) ReadGPUMetrics() (GPUMetrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.err != nil {
		return GPUMetrics{}, m.err
	}
	return m.metrics, nil
}

// CallCount returns the number of times ReadGPUMetrics was called.
func (m *MockGPUReader) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}
