package llamaruntime

import (
	"testing"
	"time"
)

// =============================================================================
// Constants Tests
// =============================================================================

func TestDefaultConstants(t *testing.T) {
	// DOING: Verify default constants have sensible values
	// EXPECT: All constants should be within reasonable bounds

	tests := []struct {
		name string
		got  interface{}
		min  interface{}
		max  interface{}
	}{
		{"DefaultContextSize", DefaultContextSize, MinContextSize, MaxContextSize},
		{"DefaultBatchSize", DefaultBatchSize, MinBatchSize, MaxBatchSize},
		{"DefaultNumGPULayers", DefaultNumGPULayers, -1, 1000},
		{"DefaultNumThreads", DefaultNumThreads, 1, 256},
		{"DefaultMaxTokens", DefaultMaxTokens, 1, 4096},
		{"DefaultTopK", DefaultTopK, 1, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.got.(int)
			min := tt.min.(int)
			max := tt.max.(int)

			// Special case for -1 (all GPU layers)
			if min == -1 && got == -1 {
				return // Valid
			}

			if got < min || got > max {
				t.Errorf("%s = %d, want between %d and %d", tt.name, got, min, max)
			}
		})
	}

	// Float constants
	if DefaultTemperature < 0 || DefaultTemperature > 2.0 {
		t.Errorf("DefaultTemperature = %v, want between 0 and 2.0", DefaultTemperature)
	}
	if DefaultTopP < 0 || DefaultTopP > 1.0 {
		t.Errorf("DefaultTopP = %v, want between 0 and 1.0", DefaultTopP)
	}
	if DefaultRepeatPenalty < 1.0 || DefaultRepeatPenalty > 2.0 {
		t.Errorf("DefaultRepeatPenalty = %v, want between 1.0 and 2.0", DefaultRepeatPenalty)
	}

	// Timeout should be reasonable
	if DefaultTimeout < 10*time.Second || DefaultTimeout > 10*time.Minute {
		t.Errorf("DefaultTimeout = %v, want between 10s and 10m", DefaultTimeout)
	}
}

func TestContextSizeBounds(t *testing.T) {
	// DOING: Verify context size bounds are sensible
	// EXPECT: Min < Default < Max

	if MinContextSize >= MaxContextSize {
		t.Errorf("MinContextSize (%d) should be less than MaxContextSize (%d)",
			MinContextSize, MaxContextSize)
	}

	if DefaultContextSize < MinContextSize || DefaultContextSize > MaxContextSize {
		t.Errorf("DefaultContextSize (%d) should be between Min (%d) and Max (%d)",
			DefaultContextSize, MinContextSize, MaxContextSize)
	}
}

func TestBatchSizeBounds(t *testing.T) {
	// DOING: Verify batch size bounds are sensible
	// EXPECT: Min < Default < Max

	if MinBatchSize >= MaxBatchSize {
		t.Errorf("MinBatchSize (%d) should be less than MaxBatchSize (%d)",
			MinBatchSize, MaxBatchSize)
	}

	if DefaultBatchSize < MinBatchSize || DefaultBatchSize > MaxBatchSize {
		t.Errorf("DefaultBatchSize (%d) should be between Min (%d) and Max (%d)",
			DefaultBatchSize, MinBatchSize, MaxBatchSize)
	}
}

// =============================================================================
// Config Tests
// =============================================================================

func TestDefaultConfig(t *testing.T) {
	// DOING: Verify DefaultConfig returns valid configuration
	// EXPECT: All fields have sensible defaults

	cfg := DefaultConfig()

	// RESULT: Verify defaults
	if cfg.ContextSize != DefaultContextSize {
		t.Errorf("ContextSize = %d, want %d", cfg.ContextSize, DefaultContextSize)
	}
	if cfg.BatchSize != DefaultBatchSize {
		t.Errorf("BatchSize = %d, want %d", cfg.BatchSize, DefaultBatchSize)
	}
	if cfg.NumGPULayers != DefaultNumGPULayers {
		t.Errorf("NumGPULayers = %d, want %d", cfg.NumGPULayers, DefaultNumGPULayers)
	}
	if cfg.NumThreads != DefaultNumThreads {
		t.Errorf("NumThreads = %d, want %d", cfg.NumThreads, DefaultNumThreads)
	}
	if !cfg.UseMMap {
		t.Error("UseMMap should default to true")
	}
	if cfg.UseMlock {
		t.Error("UseMlock should default to false")
	}
	if cfg.VerboseLogging {
		t.Error("VerboseLogging should default to false")
	}
	if cfg.Seed != -1 {
		t.Errorf("Seed = %d, want -1 (random)", cfg.Seed)
	}
	if cfg.ModelPath != "" {
		t.Error("ModelPath should default to empty string")
	}
}

func TestConfigFields(t *testing.T) {
	// DOING: Verify Config struct can be populated
	// EXPECT: All fields are settable

	cfg := Config{
		ModelPath:      "/path/to/model.gguf",
		ContextSize:    4096,
		BatchSize:      256,
		NumGPULayers:   30,
		NumThreads:     8,
		UseMMap:        false,
		UseMlock:       true,
		VerboseLogging: true,
		Seed:           42,
	}

	// RESULT: Verify all fields were set correctly
	if cfg.ModelPath != "/path/to/model.gguf" {
		t.Error("ModelPath not set correctly")
	}
	if cfg.ContextSize != 4096 {
		t.Error("ContextSize not set correctly")
	}
	if cfg.BatchSize != 256 {
		t.Error("BatchSize not set correctly")
	}
	if cfg.NumGPULayers != 30 {
		t.Error("NumGPULayers not set correctly")
	}
	if cfg.NumThreads != 8 {
		t.Error("NumThreads not set correctly")
	}
	if cfg.UseMMap {
		t.Error("UseMMap not set correctly")
	}
	if !cfg.UseMlock {
		t.Error("UseMlock not set correctly")
	}
	if !cfg.VerboseLogging {
		t.Error("VerboseLogging not set correctly")
	}
	if cfg.Seed != 42 {
		t.Error("Seed not set correctly")
	}
}

// =============================================================================
// InferenceParams Tests
// =============================================================================

func TestDefaultInferenceParams(t *testing.T) {
	// DOING: Verify DefaultInferenceParams returns valid parameters
	// EXPECT: All fields have sensible defaults

	params := DefaultInferenceParams()

	// RESULT: Verify defaults
	if params.MaxTokens != DefaultMaxTokens {
		t.Errorf("MaxTokens = %d, want %d", params.MaxTokens, DefaultMaxTokens)
	}
	if params.Temperature != DefaultTemperature {
		t.Errorf("Temperature = %v, want %v", params.Temperature, DefaultTemperature)
	}
	if params.TopP != DefaultTopP {
		t.Errorf("TopP = %v, want %v", params.TopP, DefaultTopP)
	}
	if params.TopK != DefaultTopK {
		t.Errorf("TopK = %d, want %d", params.TopK, DefaultTopK)
	}
	if params.RepeatPenalty != DefaultRepeatPenalty {
		t.Errorf("RepeatPenalty = %v, want %v", params.RepeatPenalty, DefaultRepeatPenalty)
	}
	if params.Timeout != DefaultTimeout {
		t.Errorf("Timeout = %v, want %v", params.Timeout, DefaultTimeout)
	}
	if params.Prompt != "" {
		t.Error("Prompt should default to empty string")
	}
	if params.StopSequences != nil {
		t.Error("StopSequences should default to nil")
	}
}

func TestInferenceParamsFields(t *testing.T) {
	// DOING: Verify InferenceParams struct can be populated
	// EXPECT: All fields are settable

	params := InferenceParams{
		Prompt:        "Test prompt",
		MaxTokens:     256,
		Temperature:   0.5,
		TopP:          0.8,
		TopK:          30,
		RepeatPenalty: 1.2,
		StopSequences: []string{"</s>", "\n\n"},
		Timeout:       30 * time.Second,
	}

	// RESULT: Verify all fields were set correctly
	if params.Prompt != "Test prompt" {
		t.Error("Prompt not set correctly")
	}
	if params.MaxTokens != 256 {
		t.Error("MaxTokens not set correctly")
	}
	if params.Temperature != 0.5 {
		t.Error("Temperature not set correctly")
	}
	if params.TopP != 0.8 {
		t.Error("TopP not set correctly")
	}
	if params.TopK != 30 {
		t.Error("TopK not set correctly")
	}
	if params.RepeatPenalty != 1.2 {
		t.Error("RepeatPenalty not set correctly")
	}
	if len(params.StopSequences) != 2 {
		t.Error("StopSequences not set correctly")
	}
	if params.Timeout != 30*time.Second {
		t.Error("Timeout not set correctly")
	}
}

// =============================================================================
// VisionParams Tests
// =============================================================================

func TestDefaultVisionParams(t *testing.T) {
	// DOING: Verify DefaultVisionParams returns valid parameters
	// EXPECT: All fields have sensible defaults

	params := DefaultVisionParams()

	// RESULT: Verify defaults
	if params.MaxTokens != DefaultMaxTokens {
		t.Errorf("MaxTokens = %d, want %d", params.MaxTokens, DefaultMaxTokens)
	}
	if params.Temperature != DefaultTemperature {
		t.Errorf("Temperature = %v, want %v", params.Temperature, DefaultTemperature)
	}
	if params.Timeout != DefaultTimeout {
		t.Errorf("Timeout = %v, want %v", params.Timeout, DefaultTimeout)
	}
	if params.ImageData != nil {
		t.Error("ImageData should default to nil")
	}
	if params.ImagePath != "" {
		t.Error("ImagePath should default to empty string")
	}
	if params.Prompt != "" {
		t.Error("Prompt should default to empty string")
	}
}

// =============================================================================
// Result Types Tests
// =============================================================================

func TestInferenceResult(t *testing.T) {
	// DOING: Verify InferenceResult struct can be populated
	// EXPECT: All fields are settable

	result := InferenceResult{
		Text:            "Generated text",
		TokensGenerated: 100,
		TokensPrompt:    50,
		Duration:        2 * time.Second,
		TokensPerSecond: 50.0,
		StopReason:      "max_tokens",
	}

	// RESULT: Verify all fields were set correctly
	if result.Text != "Generated text" {
		t.Error("Text not set correctly")
	}
	if result.TokensGenerated != 100 {
		t.Error("TokensGenerated not set correctly")
	}
	if result.TokensPrompt != 50 {
		t.Error("TokensPrompt not set correctly")
	}
	if result.Duration != 2*time.Second {
		t.Error("Duration not set correctly")
	}
	if result.TokensPerSecond != 50.0 {
		t.Error("TokensPerSecond not set correctly")
	}
	if result.StopReason != "max_tokens" {
		t.Error("StopReason not set correctly")
	}
}

func TestInferenceStats(t *testing.T) {
	// DOING: Verify InferenceStats struct can be populated
	// EXPECT: All fields are settable

	stats := InferenceStats{
		TotalInferences:        100,
		TotalTokensGenerated:   10000,
		TotalTokensPrompt:      5000,
		TotalDuration:          100 * time.Second,
		AverageTokensPerSecond: 100.0,
		PeakMemoryUsage:        4 * 1024 * 1024 * 1024, // 4GB
		ErrorCount:             2,
	}

	// RESULT: Verify all fields were set correctly
	if stats.TotalInferences != 100 {
		t.Error("TotalInferences not set correctly")
	}
	if stats.TotalTokensGenerated != 10000 {
		t.Error("TotalTokensGenerated not set correctly")
	}
	if stats.TotalTokensPrompt != 5000 {
		t.Error("TotalTokensPrompt not set correctly")
	}
	if stats.TotalDuration != 100*time.Second {
		t.Error("TotalDuration not set correctly")
	}
	if stats.AverageTokensPerSecond != 100.0 {
		t.Error("AverageTokensPerSecond not set correctly")
	}
	if stats.PeakMemoryUsage != 4*1024*1024*1024 {
		t.Error("PeakMemoryUsage not set correctly")
	}
	if stats.ErrorCount != 2 {
		t.Error("ErrorCount not set correctly")
	}
}

// =============================================================================
// GPU Types Tests
// =============================================================================

func TestGPUInfo(t *testing.T) {
	// DOING: Verify GPUInfo struct can be populated
	// EXPECT: All fields are settable

	gpu := GPUInfo{
		Index:             0,
		Name:              "NVIDIA GeForce RTX 3080",
		TotalMemory:       10 * 1024 * 1024 * 1024, // 10GB
		FreeMemory:        8 * 1024 * 1024 * 1024,  // 8GB
		UsedMemory:        2 * 1024 * 1024 * 1024,  // 2GB
		ComputeCapability: "8.6",
		DriverVersion:     "535.104.05",
		CUDAVersion:       "12.2",
		IsAvailable:       true,
		Temperature:       65,
		Utilization:       30,
		PowerDraw:         150.0,
		PowerLimit:        320.0,
	}

	// RESULT: Verify all fields were set correctly
	if gpu.Index != 0 {
		t.Error("Index not set correctly")
	}
	if gpu.Name != "NVIDIA GeForce RTX 3080" {
		t.Error("Name not set correctly")
	}
	if gpu.TotalMemory != 10*1024*1024*1024 {
		t.Error("TotalMemory not set correctly")
	}
	if gpu.FreeMemory != 8*1024*1024*1024 {
		t.Error("FreeMemory not set correctly")
	}
	if gpu.UsedMemory != 2*1024*1024*1024 {
		t.Error("UsedMemory not set correctly")
	}
	if gpu.ComputeCapability != "8.6" {
		t.Error("ComputeCapability not set correctly")
	}
	if gpu.DriverVersion != "535.104.05" {
		t.Error("DriverVersion not set correctly")
	}
	if gpu.CUDAVersion != "12.2" {
		t.Error("CUDAVersion not set correctly")
	}
	if !gpu.IsAvailable {
		t.Error("IsAvailable not set correctly")
	}
	if gpu.Temperature != 65 {
		t.Error("Temperature not set correctly")
	}
	if gpu.Utilization != 30 {
		t.Error("Utilization not set correctly")
	}
	if gpu.PowerDraw != 150.0 {
		t.Error("PowerDraw not set correctly")
	}
	if gpu.PowerLimit != 320.0 {
		t.Error("PowerLimit not set correctly")
	}
}

func TestGPUStatus(t *testing.T) {
	// DOING: Verify GPUStatus struct can be populated
	// EXPECT: All fields are settable

	now := time.Now()
	status := GPUStatus{
		Available:   true,
		GPUCount:    1,
		GPUs:        []GPUInfo{{Index: 0, Name: "Test GPU"}},
		TotalMemory: 10 * 1024 * 1024 * 1024,
		FreeMemory:  8 * 1024 * 1024 * 1024,
		LastChecked: now,
	}

	// RESULT: Verify all fields were set correctly
	if !status.Available {
		t.Error("Available not set correctly")
	}
	if status.GPUCount != 1 {
		t.Error("GPUCount not set correctly")
	}
	if len(status.GPUs) != 1 {
		t.Error("GPUs not set correctly")
	}
	if status.GPUs[0].Name != "Test GPU" {
		t.Error("GPU name not set correctly")
	}
	if status.TotalMemory != 10*1024*1024*1024 {
		t.Error("TotalMemory not set correctly")
	}
	if status.FreeMemory != 8*1024*1024*1024 {
		t.Error("FreeMemory not set correctly")
	}
	if status.LastChecked != now {
		t.Error("LastChecked not set correctly")
	}
}

// =============================================================================
// Model Types Tests
// =============================================================================

func TestModelInfo(t *testing.T) {
	// DOING: Verify ModelInfo struct can be populated
	// EXPECT: All fields are settable

	now := time.Now()
	info := ModelInfo{
		Path:            "/models/bunny-v1.1-q4_k_m.gguf",
		Name:            "bunny-v1.1-q4_k_m",
		Size:            4 * 1024 * 1024 * 1024, // 4GB
		Format:          "GGUF",
		Quantization:    "Q4_K_M",
		Parameters:      8 * 1000 * 1000 * 1000, // 8B
		ContextLength:   2048,
		EmbeddingLength: 4096,
		VocabSize:       32000,
		IsMultimodal:    true,
		LoadedAt:        now,
		LoadDuration:    5 * time.Second,
	}

	// RESULT: Verify all fields were set correctly
	if info.Path != "/models/bunny-v1.1-q4_k_m.gguf" {
		t.Error("Path not set correctly")
	}
	if info.Name != "bunny-v1.1-q4_k_m" {
		t.Error("Name not set correctly")
	}
	if info.Size != 4*1024*1024*1024 {
		t.Error("Size not set correctly")
	}
	if info.Format != "GGUF" {
		t.Error("Format not set correctly")
	}
	if info.Quantization != "Q4_K_M" {
		t.Error("Quantization not set correctly")
	}
	if info.Parameters != 8*1000*1000*1000 {
		t.Error("Parameters not set correctly")
	}
	if info.ContextLength != 2048 {
		t.Error("ContextLength not set correctly")
	}
	if info.EmbeddingLength != 4096 {
		t.Error("EmbeddingLength not set correctly")
	}
	if info.VocabSize != 32000 {
		t.Error("VocabSize not set correctly")
	}
	if !info.IsMultimodal {
		t.Error("IsMultimodal not set correctly")
	}
	if info.LoadedAt != now {
		t.Error("LoadedAt not set correctly")
	}
	if info.LoadDuration != 5*time.Second {
		t.Error("LoadDuration not set correctly")
	}
}

// =============================================================================
// Health Types Tests
// =============================================================================

func TestHealthStatus(t *testing.T) {
	// DOING: Verify HealthStatus struct can be populated
	// EXPECT: All fields are settable

	now := time.Now()
	status := HealthStatus{
		Healthy:       true,
		Status:        "Ready for inference",
		ModelLoaded:   true,
		ModelInfo:     &ModelInfo{Name: "test-model"},
		GPUStatus:     &GPUStatus{Available: true, GPUCount: 1},
		Stats:         &InferenceStats{TotalInferences: 100},
		LastInference: now.Add(-1 * time.Minute),
		Uptime:        1 * time.Hour,
		CheckedAt:     now,
	}

	// RESULT: Verify all fields were set correctly
	if !status.Healthy {
		t.Error("Healthy not set correctly")
	}
	if status.Status != "Ready for inference" {
		t.Error("Status not set correctly")
	}
	if !status.ModelLoaded {
		t.Error("ModelLoaded not set correctly")
	}
	if status.ModelInfo == nil || status.ModelInfo.Name != "test-model" {
		t.Error("ModelInfo not set correctly")
	}
	if status.GPUStatus == nil || !status.GPUStatus.Available {
		t.Error("GPUStatus not set correctly")
	}
	if status.Stats == nil || status.Stats.TotalInferences != 100 {
		t.Error("Stats not set correctly")
	}
	if status.Uptime != 1*time.Hour {
		t.Error("Uptime not set correctly")
	}
	if status.CheckedAt != now {
		t.Error("CheckedAt not set correctly")
	}
}
