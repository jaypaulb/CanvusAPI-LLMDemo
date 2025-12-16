package core

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// GetEnvOrDefault returns the value of an environment variable or a default value.
// This is a pure function with no side effects beyond reading env vars.
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ParseIntEnv parses an environment variable as an integer.
// Returns the default value if the variable is not set or cannot be parsed.
func ParseIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// ParseInt64Env parses an environment variable as an int64.
// Returns the default value if the variable is not set or cannot be parsed.
func ParseInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// ParseFloat64Env parses an environment variable as a float64.
// Returns the default value if the variable is not set or cannot be parsed.
func ParseFloat64Env(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// ParseBoolEnv parses an environment variable as a boolean.
// Accepts case-insensitive: "true", "1", "yes", "on" as true values.
// Accepts case-insensitive: "false", "0", "no", "off" as false values.
// Returns the default value if the variable is not set or cannot be parsed.
func ParseBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

// ParseDurationEnv parses an environment variable as a duration in seconds.
// Returns the default value if the variable is not set or cannot be parsed.
func ParseDurationEnv(key string, defaultSeconds int) time.Duration {
	return time.Duration(ParseIntEnv(key, defaultSeconds)) * time.Second
}
