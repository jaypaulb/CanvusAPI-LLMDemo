// Package webui provides the web-based user interface for CanvusLocalLLM.
// This file contains the FormatDuration atom for human-readable duration formatting.
package webui

import (
	"fmt"
	"time"
)

// FormatDuration converts a time.Duration to a human-readable string.
// This is a pure function with no side effects.
//
// Format rules:
//   - Sub-second: "0s" (minimum display)
//   - Seconds only: "45s"
//   - Minutes and seconds: "2m 30s"
//   - Hours and minutes: "2h 34m"
//   - Days and hours: "3d 5h"
//   - Weeks and days: "2w 3d"
//
// The function always shows at most two units to keep output concise.
// Negative durations are formatted with a leading minus sign.
//
// Examples:
//   - FormatDuration(0) returns "0s"
//   - FormatDuration(45 * time.Second) returns "45s"
//   - FormatDuration(2*time.Minute + 30*time.Second) returns "2m 30s"
//   - FormatDuration(2*time.Hour + 34*time.Minute) returns "2h 34m"
//   - FormatDuration(3*24*time.Hour + 5*time.Hour) returns "3d 5h"
//   - FormatDuration(-5 * time.Minute) returns "-5m 0s"
func FormatDuration(d time.Duration) string {
	// Handle negative durations
	if d < 0 {
		return "-" + FormatDuration(-d)
	}

	// Define time units in descending order
	const (
		day  = 24 * time.Hour
		week = 7 * day
	)

	// Handle zero duration
	if d == 0 {
		return "0s"
	}

	// Calculate each unit
	weeks := d / week
	d %= week

	days := d / day
	d %= day

	hours := d / time.Hour
	d %= time.Hour

	minutes := d / time.Minute
	d %= time.Minute

	seconds := d / time.Second

	// Build output with at most two units
	if weeks > 0 {
		return fmt.Sprintf("%dw %dd", weeks, days)
	}
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// FormatDurationCompact provides a shorter format for tight spaces.
// Only shows the largest non-zero unit.
//
// Examples:
//   - FormatDurationCompact(0) returns "0s"
//   - FormatDurationCompact(45 * time.Second) returns "45s"
//   - FormatDurationCompact(2 * time.Minute) returns "2m"
//   - FormatDurationCompact(3 * time.Hour) returns "3h"
//   - FormatDurationCompact(5 * 24 * time.Hour) returns "5d"
func FormatDurationCompact(d time.Duration) string {
	// Handle negative durations
	if d < 0 {
		return "-" + FormatDurationCompact(-d)
	}

	const (
		day  = 24 * time.Hour
		week = 7 * day
	)

	if d == 0 {
		return "0s"
	}

	// Return the largest non-zero unit
	if weeks := d / week; weeks > 0 {
		return fmt.Sprintf("%dw", weeks)
	}
	if days := d / day; days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	if hours := d / time.Hour; hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	if minutes := d / time.Minute; minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", d/time.Second)
}
