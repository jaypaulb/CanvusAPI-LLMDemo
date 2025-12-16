package logging

import (
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
	"go_backend/core"
)

// mockObjectEncoder is a test implementation of zapcore.ObjectEncoder
// that captures encoded values for verification.
type mockObjectEncoder struct {
	strings  map[string]string
	ints     map[string]int
	int64s   map[string]int64
	float64s map[string]float64
	objects  map[string]zapcore.ObjectMarshaler
}

func newMockObjectEncoder() *mockObjectEncoder {
	return &mockObjectEncoder{
		strings:  make(map[string]string),
		ints:     make(map[string]int),
		int64s:   make(map[string]int64),
		float64s: make(map[string]float64),
		objects:  make(map[string]zapcore.ObjectMarshaler),
	}
}

func (m *mockObjectEncoder) AddString(key, value string)    { m.strings[key] = value }
func (m *mockObjectEncoder) AddInt(key string, value int)   { m.ints[key] = value }
func (m *mockObjectEncoder) AddInt64(key string, value int64) { m.int64s[key] = value }
func (m *mockObjectEncoder) AddFloat64(key string, value float64) { m.float64s[key] = value }
func (m *mockObjectEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	m.objects[key] = obj
	// Actually marshal the object to test nested encoding
	return obj.MarshalLogObject(m)
}

// Implement remaining ObjectEncoder interface methods as no-ops
func (m *mockObjectEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error { return nil }
func (m *mockObjectEncoder) AddBinary(key string, value []byte)                    {}
func (m *mockObjectEncoder) AddBool(key string, value bool)                        {}
func (m *mockObjectEncoder) AddByteString(key string, value []byte)                {}
func (m *mockObjectEncoder) AddComplex128(key string, value complex128)            {}
func (m *mockObjectEncoder) AddComplex64(key string, value complex64)              {}
func (m *mockObjectEncoder) AddDuration(key string, value time.Duration)           {}
func (m *mockObjectEncoder) AddFloat32(key string, value float32)                  {}
func (m *mockObjectEncoder) AddInt32(key string, value int32)                      {}
func (m *mockObjectEncoder) AddInt16(key string, value int16)                      {}
func (m *mockObjectEncoder) AddInt8(key string, value int8)                        {}
func (m *mockObjectEncoder) AddReflected(key string, value interface{}) error      { return nil }
func (m *mockObjectEncoder) AddTime(key string, value time.Time)                   {}
func (m *mockObjectEncoder) AddUint(key string, value uint)                        {}
func (m *mockObjectEncoder) AddUint64(key string, value uint64)                    {}
func (m *mockObjectEncoder) AddUint32(key string, value uint32)                    {}
func (m *mockObjectEncoder) AddUint16(key string, value uint16)                    {}
func (m *mockObjectEncoder) AddUint8(key string, value uint8)                      {}
func (m *mockObjectEncoder) AddUintptr(key string, value uintptr)                  {}
func (m *mockObjectEncoder) OpenNamespace(key string)                              {}

func TestInferenceMetrics_MarshalLogObject(t *testing.T) {
	metrics := InferenceMetrics{
		ModelName:        "llama-3.2-8b",
		PromptTokens:     150,
		CompletionTokens: 200,
		TotalTokens:      350,
		Duration:         2500 * time.Millisecond,
		TokensPerSecond:  140.0,
		GPU: core.GPUMetrics{
			VRAMUsedMB:     4096,
			VRAMTotalMB:    8192,
			GPUUtilization: 85.5,
			Temperature:    72.0,
		},
	}

	enc := newMockObjectEncoder()
	err := metrics.MarshalLogObject(enc)

	if err != nil {
		t.Fatalf("MarshalLogObject returned error: %v", err)
	}

	// Verify string fields
	t.Run("model_name", func(t *testing.T) {
		if got, want := enc.strings["model_name"], "llama-3.2-8b"; got != want {
			t.Errorf("model_name = %q, want %q", got, want)
		}
	})

	// Verify int fields
	t.Run("prompt_tokens", func(t *testing.T) {
		if got, want := enc.ints["prompt_tokens"], 150; got != want {
			t.Errorf("prompt_tokens = %d, want %d", got, want)
		}
	})

	t.Run("completion_tokens", func(t *testing.T) {
		if got, want := enc.ints["completion_tokens"], 200; got != want {
			t.Errorf("completion_tokens = %d, want %d", got, want)
		}
	})

	t.Run("total_tokens", func(t *testing.T) {
		if got, want := enc.ints["total_tokens"], 350; got != want {
			t.Errorf("total_tokens = %d, want %d", got, want)
		}
	})

	// Verify duration in milliseconds
	t.Run("duration_ms", func(t *testing.T) {
		if got, want := enc.int64s["duration_ms"], int64(2500); got != want {
			t.Errorf("duration_ms = %d, want %d", got, want)
		}
	})

	// Verify float64 fields
	t.Run("tokens_per_second", func(t *testing.T) {
		if got, want := enc.float64s["tokens_per_second"], 140.0; got != want {
			t.Errorf("tokens_per_second = %f, want %f", got, want)
		}
	})

	// Verify GPU metrics were embedded (from nested MarshalLogObject call)
	t.Run("gpu_vram_used_mb", func(t *testing.T) {
		if got, want := enc.int64s["vram_used_mb"], int64(4096); got != want {
			t.Errorf("vram_used_mb = %d, want %d", got, want)
		}
	})

	t.Run("gpu_vram_total_mb", func(t *testing.T) {
		if got, want := enc.int64s["vram_total_mb"], int64(8192); got != want {
			t.Errorf("vram_total_mb = %d, want %d", got, want)
		}
	})

	t.Run("gpu_utilization", func(t *testing.T) {
		if got, want := enc.float64s["gpu_utilization"], 85.5; got != want {
			t.Errorf("gpu_utilization = %f, want %f", got, want)
		}
	})

	t.Run("gpu_temperature", func(t *testing.T) {
		if got, want := enc.float64s["temperature"], 72.0; got != want {
			t.Errorf("temperature = %f, want %f", got, want)
		}
	})
}

func TestInferenceMetrics_ZeroValues(t *testing.T) {
	// Test with zero/empty values to ensure no panics
	metrics := InferenceMetrics{}

	enc := newMockObjectEncoder()
	err := metrics.MarshalLogObject(enc)

	if err != nil {
		t.Fatalf("MarshalLogObject with zero values returned error: %v", err)
	}

	// Verify zero values are encoded correctly
	if got := enc.strings["model_name"]; got != "" {
		t.Errorf("model_name with zero value = %q, want empty string", got)
	}

	if got := enc.ints["total_tokens"]; got != 0 {
		t.Errorf("total_tokens with zero value = %d, want 0", got)
	}

	if got := enc.int64s["duration_ms"]; got != 0 {
		t.Errorf("duration_ms with zero value = %d, want 0", got)
	}

	if got := enc.float64s["tokens_per_second"]; got != 0.0 {
		t.Errorf("tokens_per_second with zero value = %f, want 0.0", got)
	}
}

func TestInferenceMetrics_DurationConversion(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		wantMs   int64
	}{
		{
			name:     "zero duration",
			duration: 0,
			wantMs:   0,
		},
		{
			name:     "1 second",
			duration: time.Second,
			wantMs:   1000,
		},
		{
			name:     "500 milliseconds",
			duration: 500 * time.Millisecond,
			wantMs:   500,
		},
		{
			name:     "2.5 seconds",
			duration: 2500 * time.Millisecond,
			wantMs:   2500,
		},
		{
			name:     "microseconds (less than 1ms)",
			duration: 500 * time.Microsecond,
			wantMs:   0, // Truncated to 0 since less than 1ms
		},
		{
			name:     "1 minute",
			duration: time.Minute,
			wantMs:   60000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := InferenceMetrics{
				ModelName: "test-model",
				Duration:  tt.duration,
			}

			enc := newMockObjectEncoder()
			err := metrics.MarshalLogObject(enc)

			if err != nil {
				t.Fatalf("MarshalLogObject returned error: %v", err)
			}

			if got := enc.int64s["duration_ms"]; got != tt.wantMs {
				t.Errorf("duration_ms = %d, want %d", got, tt.wantMs)
			}
		})
	}
}
