package core

import (
	"os"
	"testing"
	"time"
)

func TestGetEnvOrDefault(t *testing.T) {
	const testKey = "TEST_GET_ENV_OR_DEFAULT"
	defer os.Unsetenv(testKey)

	tests := []struct {
		name         string
		envValue     string
		setEnv       bool
		defaultValue string
		want         string
	}{
		{
			name:         "returns env value when set",
			envValue:     "custom_value",
			setEnv:       true,
			defaultValue: "default",
			want:         "custom_value",
		},
		{
			name:         "returns default when not set",
			setEnv:       false,
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "returns default when empty",
			envValue:     "",
			setEnv:       true,
			defaultValue: "default",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(testKey)
			if tt.setEnv {
				os.Setenv(testKey, tt.envValue)
			}
			got := GetEnvOrDefault(testKey, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetEnvOrDefault() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseIntEnv(t *testing.T) {
	const testKey = "TEST_PARSE_INT_ENV"
	defer os.Unsetenv(testKey)

	tests := []struct {
		name         string
		envValue     string
		setEnv       bool
		defaultValue int
		want         int
	}{
		{
			name:         "parses valid integer",
			envValue:     "42",
			setEnv:       true,
			defaultValue: 0,
			want:         42,
		},
		{
			name:         "parses negative integer",
			envValue:     "-10",
			setEnv:       true,
			defaultValue: 0,
			want:         -10,
		},
		{
			name:         "returns default for invalid",
			envValue:     "not_a_number",
			setEnv:       true,
			defaultValue: 99,
			want:         99,
		},
		{
			name:         "returns default when not set",
			setEnv:       false,
			defaultValue: 55,
			want:         55,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(testKey)
			if tt.setEnv {
				os.Setenv(testKey, tt.envValue)
			}
			got := ParseIntEnv(testKey, tt.defaultValue)
			if got != tt.want {
				t.Errorf("ParseIntEnv() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseInt64Env(t *testing.T) {
	const testKey = "TEST_PARSE_INT64_ENV"
	defer os.Unsetenv(testKey)

	tests := []struct {
		name         string
		envValue     string
		setEnv       bool
		defaultValue int64
		want         int64
	}{
		{
			name:         "parses large integer",
			envValue:     "9223372036854775807",
			setEnv:       true,
			defaultValue: 0,
			want:         9223372036854775807,
		},
		{
			name:         "returns default for invalid",
			envValue:     "invalid",
			setEnv:       true,
			defaultValue: 100,
			want:         100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(testKey)
			if tt.setEnv {
				os.Setenv(testKey, tt.envValue)
			}
			got := ParseInt64Env(testKey, tt.defaultValue)
			if got != tt.want {
				t.Errorf("ParseInt64Env() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseFloat64Env(t *testing.T) {
	const testKey = "TEST_PARSE_FLOAT64_ENV"
	defer os.Unsetenv(testKey)

	tests := []struct {
		name         string
		envValue     string
		setEnv       bool
		defaultValue float64
		want         float64
	}{
		{
			name:         "parses float",
			envValue:     "3.14159",
			setEnv:       true,
			defaultValue: 0.0,
			want:         3.14159,
		},
		{
			name:         "parses integer as float",
			envValue:     "42",
			setEnv:       true,
			defaultValue: 0.0,
			want:         42.0,
		},
		{
			name:         "returns default for invalid",
			envValue:     "not_a_float",
			setEnv:       true,
			defaultValue: 2.5,
			want:         2.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(testKey)
			if tt.setEnv {
				os.Setenv(testKey, tt.envValue)
			}
			got := ParseFloat64Env(testKey, tt.defaultValue)
			if got != tt.want {
				t.Errorf("ParseFloat64Env() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestParseBoolEnv(t *testing.T) {
	const testKey = "TEST_PARSE_BOOL_ENV"
	defer os.Unsetenv(testKey)

	tests := []struct {
		name         string
		envValue     string
		setEnv       bool
		defaultValue bool
		want         bool
	}{
		// True values
		{name: "true lowercase", envValue: "true", setEnv: true, defaultValue: false, want: true},
		{name: "TRUE uppercase", envValue: "TRUE", setEnv: true, defaultValue: false, want: true},
		{name: "True mixed", envValue: "True", setEnv: true, defaultValue: false, want: true},
		{name: "1", envValue: "1", setEnv: true, defaultValue: false, want: true},
		{name: "yes", envValue: "yes", setEnv: true, defaultValue: false, want: true},
		{name: "YES", envValue: "YES", setEnv: true, defaultValue: false, want: true},
		{name: "on", envValue: "on", setEnv: true, defaultValue: false, want: true},
		{name: "ON", envValue: "ON", setEnv: true, defaultValue: false, want: true},
		// False values
		{name: "false", envValue: "false", setEnv: true, defaultValue: true, want: false},
		{name: "FALSE", envValue: "FALSE", setEnv: true, defaultValue: true, want: false},
		{name: "0", envValue: "0", setEnv: true, defaultValue: true, want: false},
		{name: "no", envValue: "no", setEnv: true, defaultValue: true, want: false},
		{name: "off", envValue: "off", setEnv: true, defaultValue: true, want: false},
		// Default values
		{name: "not set returns default true", setEnv: false, defaultValue: true, want: true},
		{name: "not set returns default false", setEnv: false, defaultValue: false, want: false},
		{name: "invalid returns default", envValue: "maybe", setEnv: true, defaultValue: true, want: true},
		{name: "whitespace handled", envValue: "  true  ", setEnv: true, defaultValue: false, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(testKey)
			if tt.setEnv {
				os.Setenv(testKey, tt.envValue)
			}
			got := ParseBoolEnv(testKey, tt.defaultValue)
			if got != tt.want {
				t.Errorf("ParseBoolEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDurationEnv(t *testing.T) {
	const testKey = "TEST_PARSE_DURATION_ENV"
	defer os.Unsetenv(testKey)

	tests := []struct {
		name           string
		envValue       string
		setEnv         bool
		defaultSeconds int
		want           time.Duration
	}{
		{
			name:           "parses seconds",
			envValue:       "30",
			setEnv:         true,
			defaultSeconds: 60,
			want:           30 * time.Second,
		},
		{
			name:           "returns default when not set",
			setEnv:         false,
			defaultSeconds: 120,
			want:           120 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(testKey)
			if tt.setEnv {
				os.Setenv(testKey, tt.envValue)
			}
			got := ParseDurationEnv(testKey, tt.defaultSeconds)
			if got != tt.want {
				t.Errorf("ParseDurationEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
