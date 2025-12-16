package logging

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
	"go_backend/core"
)

// createMetricsTestLogger creates a Logger for testing that writes to a temp file.
func createMetricsTestLogger(t *testing.T) *Logger {
	t.Helper()
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewLogger(true, logPath)
	if err != nil {
		t.Fatalf("failed to create test logger: %v", err)
	}
	return logger
}

func TestNewMetricsLogger(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)
	if ml == nil {
		t.Fatal("NewMetricsLogger returned nil")
	}
	if ml.logger != logger {
		t.Error("MetricsLogger.logger does not match input logger")
	}
}

func TestMetricsLogger_StartEndInference(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)

	// Start inference
	timer := ml.StartInference("test-model")
	if timer == nil {
		t.Fatal("StartInference returned nil timer")
	}
	if timer.ModelName != "test-model" {
		t.Errorf("timer.ModelName = %q, want %q", timer.ModelName, "test-model")
	}
	if timer.StartTime.IsZero() {
		t.Error("timer.StartTime is zero")
	}

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	// End inference
	metrics := ml.EndInference(timer, 100, 200)

	if metrics.ModelName != "test-model" {
		t.Errorf("metrics.ModelName = %q, want %q", metrics.ModelName, "test-model")
	}
	if metrics.PromptTokens != 100 {
		t.Errorf("metrics.PromptTokens = %d, want 100", metrics.PromptTokens)
	}
	if metrics.CompletionTokens != 200 {
		t.Errorf("metrics.CompletionTokens = %d, want 200", metrics.CompletionTokens)
	}
	if metrics.TotalTokens != 300 {
		t.Errorf("metrics.TotalTokens = %d, want 300", metrics.TotalTokens)
	}
	if metrics.Duration < 10*time.Millisecond {
		t.Errorf("metrics.Duration = %v, want >= 10ms", metrics.Duration)
	}
	if metrics.TokensPerSecond <= 0 {
		t.Errorf("metrics.TokensPerSecond = %f, want > 0", metrics.TokensPerSecond)
	}
}

func TestMetricsLogger_StartInferenceWithGPU(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)
	gpu := core.GPUMetrics{
		VRAMUsedMB:     4096,
		VRAMTotalMB:    8192,
		GPUUtilization: 85.5,
		Temperature:    72.0,
	}

	timer := ml.StartInferenceWithGPU("test-model", gpu)
	if timer.GPU != gpu {
		t.Error("timer.GPU does not match input")
	}
}

func TestMetricsLogger_EndInferenceWithGPU(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)
	timer := ml.StartInference("test-model")

	gpu := core.GPUMetrics{
		VRAMUsedMB:     5000,
		VRAMTotalMB:    8192,
		GPUUtilization: 90.0,
		Temperature:    75.0,
	}

	metrics := ml.EndInferenceWithGPU(timer, 50, 100, gpu)

	if metrics.GPU != gpu {
		t.Error("metrics.GPU does not match input")
	}
}

func TestMetricsLogger_LogInference(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)
	gpu := core.GPUMetrics{
		VRAMUsedMB:     4096,
		VRAMTotalMB:    8192,
		GPUUtilization: 85.5,
		Temperature:    72.0,
	}

	metrics := ml.LogInference("test-model", 100, 200, 2*time.Second, gpu)

	if metrics.ModelName != "test-model" {
		t.Errorf("metrics.ModelName = %q, want %q", metrics.ModelName, "test-model")
	}
	if metrics.TotalTokens != 300 {
		t.Errorf("metrics.TotalTokens = %d, want 300", metrics.TotalTokens)
	}
	if metrics.Duration != 2*time.Second {
		t.Errorf("metrics.Duration = %v, want 2s", metrics.Duration)
	}
	// 300 tokens / 2 seconds = 150 tokens/sec
	if metrics.TokensPerSecond != 150 {
		t.Errorf("metrics.TokensPerSecond = %f, want 150", metrics.TokensPerSecond)
	}
}

func TestMetricsLogger_LogGPU(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)
	gpu := core.GPUMetrics{
		VRAMUsedMB:     4096,
		VRAMTotalMB:    8192,
		GPUUtilization: 85.5,
		Temperature:    72.0,
	}

	// Should not panic
	ml.LogGPU(gpu)
}

func TestMetricsLogger_LogGPUWarn(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)
	gpu := core.GPUMetrics{
		VRAMUsedMB:     7500,
		VRAMTotalMB:    8192,
		GPUUtilization: 95.0,
		Temperature:    85.0,
	}

	// Should not panic
	ml.LogGPUWarn(gpu, "high GPU utilization")
}

func TestMetricsLogger_LogTokens(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)

	// Should not panic
	ml.LogTokens("token update", 100, 200, 300)
}

func TestMetricsLogger_WithModel(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)
	modelLogger := ml.WithModel("llama-3.2-8b")

	if modelLogger == nil {
		t.Fatal("WithModel returned nil")
	}
	if modelLogger == ml {
		t.Error("WithModel should return a new MetricsLogger")
	}

	// Should not panic
	modelLogger.Info("test message")
}

func TestMetricsLogger_WithCanvas(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)
	canvasLogger := ml.WithCanvas("canvas-123", "widget-456")

	if canvasLogger == nil {
		t.Fatal("WithCanvas returned nil")
	}

	// Should not panic
	canvasLogger.Info("test message")
}

func TestMetricsLogger_WithCorrelation(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)
	corrLogger := ml.WithCorrelation("req-abc123")

	if corrLogger == nil {
		t.Fatal("WithCorrelation returned nil")
	}

	// Should not panic
	corrLogger.Info("test message")
}

func TestMetricsLogger_Logger(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)

	if ml.Logger() != logger {
		t.Error("Logger() should return the underlying logger")
	}
}

func TestMetricsLogger_LogMethods(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)

	// Test all log level methods don't panic
	ml.Debug("debug message", zap.String("key", "value"))
	ml.Info("info message", zap.Int("count", 42))
	ml.Warn("warn message", zap.Bool("alert", true))
	ml.Error("error message", zap.Error(os.ErrNotExist))
}

func TestNewInferenceMetrics(t *testing.T) {
	gpu := core.GPUMetrics{
		VRAMUsedMB:     4096,
		VRAMTotalMB:    8192,
		GPUUtilization: 85.5,
		Temperature:    72.0,
	}

	metrics := NewInferenceMetrics("test-model", 100, 200, 2*time.Second, gpu)

	if metrics.ModelName != "test-model" {
		t.Errorf("ModelName = %q, want %q", metrics.ModelName, "test-model")
	}
	if metrics.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", metrics.PromptTokens)
	}
	if metrics.CompletionTokens != 200 {
		t.Errorf("CompletionTokens = %d, want 200", metrics.CompletionTokens)
	}
	if metrics.TotalTokens != 300 {
		t.Errorf("TotalTokens = %d, want 300", metrics.TotalTokens)
	}
	if metrics.Duration != 2*time.Second {
		t.Errorf("Duration = %v, want 2s", metrics.Duration)
	}
	if metrics.TokensPerSecond != 150 {
		t.Errorf("TokensPerSecond = %f, want 150", metrics.TokensPerSecond)
	}
	if metrics.GPU != gpu {
		t.Error("GPU does not match input")
	}
}

func TestNewInferenceMetrics_ZeroDuration(t *testing.T) {
	metrics := NewInferenceMetrics("test", 100, 200, 0, core.GPUMetrics{})

	if metrics.TokensPerSecond != 0 {
		t.Errorf("TokensPerSecond with zero duration = %f, want 0", metrics.TokensPerSecond)
	}
}

func TestCalculateTokensPerSecond(t *testing.T) {
	tests := []struct {
		name        string
		totalTokens int
		duration    time.Duration
		want        float64
	}{
		{
			name:        "normal case",
			totalTokens: 300,
			duration:    2 * time.Second,
			want:        150,
		},
		{
			name:        "zero duration",
			totalTokens: 300,
			duration:    0,
			want:        0,
		},
		{
			name:        "negative duration",
			totalTokens: 300,
			duration:    -1 * time.Second,
			want:        0,
		},
		{
			name:        "zero tokens",
			totalTokens: 0,
			duration:    1 * time.Second,
			want:        0,
		},
		{
			name:        "high throughput",
			totalTokens: 1000,
			duration:    100 * time.Millisecond,
			want:        10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateTokensPerSecond(tt.totalTokens, tt.duration)
			if got != tt.want {
				t.Errorf("CalculateTokensPerSecond() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestVRAMUtilization(t *testing.T) {
	tests := []struct {
		name string
		gpu  core.GPUMetrics
		want float64
	}{
		{
			name: "50% usage",
			gpu:  core.GPUMetrics{VRAMUsedMB: 4096, VRAMTotalMB: 8192},
			want: 50.0,
		},
		{
			name: "100% usage",
			gpu:  core.GPUMetrics{VRAMUsedMB: 8192, VRAMTotalMB: 8192},
			want: 100.0,
		},
		{
			name: "0% usage",
			gpu:  core.GPUMetrics{VRAMUsedMB: 0, VRAMTotalMB: 8192},
			want: 0.0,
		},
		{
			name: "zero total",
			gpu:  core.GPUMetrics{VRAMUsedMB: 100, VRAMTotalMB: 0},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VRAMUtilization(tt.gpu)
			if got != tt.want {
				t.Errorf("VRAMUtilization() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestIsGPUHighUtilization(t *testing.T) {
	tests := []struct {
		name      string
		gpu       core.GPUMetrics
		threshold []float64
		want      bool
	}{
		{
			name: "high utilization default threshold",
			gpu:  core.GPUMetrics{GPUUtilization: 95.0},
			want: true,
		},
		{
			name: "low utilization default threshold",
			gpu:  core.GPUMetrics{GPUUtilization: 80.0},
			want: false,
		},
		{
			name:      "custom threshold exceeded",
			gpu:       core.GPUMetrics{GPUUtilization: 85.0},
			threshold: []float64{80.0},
			want:      true,
		},
		{
			name:      "custom threshold not exceeded",
			gpu:       core.GPUMetrics{GPUUtilization: 75.0},
			threshold: []float64{80.0},
			want:      false,
		},
		{
			name: "exactly at default threshold",
			gpu:  core.GPUMetrics{GPUUtilization: 90.0},
			want: false, // > not >=
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGPUHighUtilization(tt.gpu, tt.threshold...)
			if got != tt.want {
				t.Errorf("IsGPUHighUtilization() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsVRAMHighUsage(t *testing.T) {
	tests := []struct {
		name      string
		gpu       core.GPUMetrics
		threshold []float64
		want      bool
	}{
		{
			name: "high VRAM usage",
			gpu:  core.GPUMetrics{VRAMUsedMB: 7500, VRAMTotalMB: 8192}, // ~91.5%
			want: true,
		},
		{
			name: "low VRAM usage",
			gpu:  core.GPUMetrics{VRAMUsedMB: 4000, VRAMTotalMB: 8192}, // ~48.8%
			want: false,
		},
		{
			name:      "custom threshold",
			gpu:       core.GPUMetrics{VRAMUsedMB: 6500, VRAMTotalMB: 8192}, // ~79.3%
			threshold: []float64{75.0},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsVRAMHighUsage(tt.gpu, tt.threshold...)
			if got != tt.want {
				t.Errorf("IsVRAMHighUsage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsGPUHot(t *testing.T) {
	tests := []struct {
		name      string
		gpu       core.GPUMetrics
		threshold []float64
		want      bool
	}{
		{
			name: "hot GPU default threshold",
			gpu:  core.GPUMetrics{Temperature: 85.0},
			want: true,
		},
		{
			name: "cool GPU default threshold",
			gpu:  core.GPUMetrics{Temperature: 70.0},
			want: false,
		},
		{
			name:      "custom threshold exceeded",
			gpu:       core.GPUMetrics{Temperature: 75.0},
			threshold: []float64{70.0},
			want:      true,
		},
		{
			name: "exactly at default threshold",
			gpu:  core.GPUMetrics{Temperature: 80.0},
			want: false, // > not >=
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGPUHot(tt.gpu, tt.threshold...)
			if got != tt.want {
				t.Errorf("IsGPUHot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetricsLoggerChaining(t *testing.T) {
	logger := createMetricsTestLogger(t)
	defer logger.Sync()

	ml := NewMetricsLogger(logger)

	// Test method chaining
	chainedLogger := ml.
		WithModel("llama-3.2-8b").
		WithCanvas("canvas-123", "widget-456").
		WithCorrelation("req-abc123")

	if chainedLogger == nil {
		t.Fatal("chained logger is nil")
	}

	// Should not panic
	chainedLogger.Info("chained log message")
}
