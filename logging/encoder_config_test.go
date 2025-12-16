package logging

import (
	"testing"
	"time"
)

func TestNewEncoderConfig(t *testing.T) {
	config := NewEncoderConfig()

	t.Run("has correct field keys", func(t *testing.T) {
		if config.TimeKey != FieldTimestamp {
			t.Errorf("TimeKey = %q, want %q", config.TimeKey, FieldTimestamp)
		}
		if config.LevelKey != FieldLevel {
			t.Errorf("LevelKey = %q, want %q", config.LevelKey, FieldLevel)
		}
		if config.MessageKey != FieldMessage {
			t.Errorf("MessageKey = %q, want %q", config.MessageKey, FieldMessage)
		}
		if config.StacktraceKey != FieldStacktrace {
			t.Errorf("StacktraceKey = %q, want %q", config.StacktraceKey, FieldStacktrace)
		}
		if config.CallerKey != FieldCaller {
			t.Errorf("CallerKey = %q, want %q", config.CallerKey, FieldCaller)
		}
	})

	t.Run("has required encoders set", func(t *testing.T) {
		if config.EncodeLevel == nil {
			t.Error("EncodeLevel should not be nil")
		}
		if config.EncodeTime == nil {
			t.Error("EncodeTime should not be nil")
		}
		if config.EncodeDuration == nil {
			t.Error("EncodeDuration should not be nil")
		}
		if config.EncodeCaller == nil {
			t.Error("EncodeCaller should not be nil")
		}
	})

	t.Run("is consistent across calls", func(t *testing.T) {
		config2 := NewEncoderConfig()
		if config.TimeKey != config2.TimeKey {
			t.Error("NewEncoderConfig should return consistent results")
		}
		if config.LevelKey != config2.LevelKey {
			t.Error("NewEncoderConfig should return consistent results")
		}
	})
}

func TestNewConsoleEncoderConfig(t *testing.T) {
	config := NewConsoleEncoderConfig()

	t.Run("has correct field keys", func(t *testing.T) {
		if config.TimeKey != FieldTimestamp {
			t.Errorf("TimeKey = %q, want %q", config.TimeKey, FieldTimestamp)
		}
		if config.LevelKey != FieldLevel {
			t.Errorf("LevelKey = %q, want %q", config.LevelKey, FieldLevel)
		}
	})

	t.Run("has required encoders set", func(t *testing.T) {
		if config.EncodeLevel == nil {
			t.Error("EncodeLevel should not be nil")
		}
		if config.EncodeTime == nil {
			t.Error("EncodeTime should not be nil")
		}
	})
}

func TestShortTimeEncoder(t *testing.T) {
	// Create a mock encoder to capture output
	var encoded string
	mockEncoder := mockPrimitiveArrayEncoder{
		appendString: func(s string) {
			encoded = s
		},
	}

	testTime := time.Date(2024, 1, 15, 14, 30, 45, 123000000, time.UTC)
	shortTimeEncoder(testTime, &mockEncoder)

	expected := "14:30:45.123"
	if encoded != expected {
		t.Errorf("shortTimeEncoder encoded %q, want %q", encoded, expected)
	}
}

// mockPrimitiveArrayEncoder implements zapcore.PrimitiveArrayEncoder for testing
type mockPrimitiveArrayEncoder struct {
	appendString func(string)
}

func (m *mockPrimitiveArrayEncoder) AppendBool(v bool)             {}
func (m *mockPrimitiveArrayEncoder) AppendByteString(v []byte)     {}
func (m *mockPrimitiveArrayEncoder) AppendComplex128(v complex128) {}
func (m *mockPrimitiveArrayEncoder) AppendComplex64(v complex64)   {}
func (m *mockPrimitiveArrayEncoder) AppendFloat64(v float64)       {}
func (m *mockPrimitiveArrayEncoder) AppendFloat32(v float32)       {}
func (m *mockPrimitiveArrayEncoder) AppendInt(v int)               {}
func (m *mockPrimitiveArrayEncoder) AppendInt64(v int64)           {}
func (m *mockPrimitiveArrayEncoder) AppendInt32(v int32)           {}
func (m *mockPrimitiveArrayEncoder) AppendInt16(v int16)           {}
func (m *mockPrimitiveArrayEncoder) AppendInt8(v int8)             {}
func (m *mockPrimitiveArrayEncoder) AppendString(v string) {
	if m.appendString != nil {
		m.appendString(v)
	}
}
func (m *mockPrimitiveArrayEncoder) AppendUint(v uint)       {}
func (m *mockPrimitiveArrayEncoder) AppendUint64(v uint64)   {}
func (m *mockPrimitiveArrayEncoder) AppendUint32(v uint32)   {}
func (m *mockPrimitiveArrayEncoder) AppendUint16(v uint16)   {}
func (m *mockPrimitiveArrayEncoder) AppendUint8(v uint8)     {}
func (m *mockPrimitiveArrayEncoder) AppendUintptr(v uintptr) {}
