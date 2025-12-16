package core

// Version is the application version, set at build time via ldflags.
// To inject a version during build, use:
//
//	go build -ldflags "-X go_backend/core.Version=v1.0.0" .
//
// Or with git tag:
//
//	go build -ldflags "-X go_backend/core.Version=$(git describe --tags --always)" .
//
// If not set at build time, defaults to "dev".
var Version = "dev"

// BuildTime is the build timestamp, set at build time via ldflags.
// To inject build time during build, use:
//
//	go build -ldflags "-X go_backend/core.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" .
//
// If not set at build time, defaults to "unknown".
var BuildTime = "unknown"

// GitCommit is the git commit hash, set at build time via ldflags.
// To inject commit hash during build, use:
//
//	go build -ldflags "-X go_backend/core.GitCommit=$(git rev-parse --short HEAD)" .
//
// If not set at build time, defaults to "unknown".
var GitCommit = "unknown"

// GetVersion returns the application version string.
// This is a pure function that returns the compile-time injected version.
func GetVersion() string {
	return Version
}

// GetBuildTime returns the build timestamp.
// This is a pure function that returns the compile-time injected build time.
func GetBuildTime() string {
	return BuildTime
}

// GetGitCommit returns the git commit hash.
// This is a pure function that returns the compile-time injected commit hash.
func GetGitCommit() string {
	return GitCommit
}

// GetVersionInfo returns a formatted version information string.
// Includes version, build time, and git commit if available.
//
// Examples:
//   - "v1.0.0 (built 2024-01-15T10:30:00Z, commit abc1234)"
//   - "dev (built unknown, commit unknown)"
func GetVersionInfo() string {
	return Version + " (built " + BuildTime + ", commit " + GitCommit + ")"
}

// BuildLdflags returns the ldflags string for injecting version information.
// This is a helper for build scripts to construct the proper ldflags.
//
// Parameters:
//   - version: the version string (e.g., "v1.0.0")
//   - buildTime: the build timestamp (e.g., "2024-01-15T10:30:00Z")
//   - gitCommit: the git commit hash (e.g., "abc1234")
//
// Returns:
//   - string: the complete ldflags string for go build
//
// Example output:
//
//	"-X go_backend/core.Version=v1.0.0 -X go_backend/core.BuildTime=2024-01-15T10:30:00Z -X go_backend/core.GitCommit=abc1234"
func BuildLdflags(version, buildTime, gitCommit string) string {
	var flags string
	if version != "" {
		flags += "-X go_backend/core.Version=" + version
	}
	if buildTime != "" {
		if flags != "" {
			flags += " "
		}
		flags += "-X go_backend/core.BuildTime=" + buildTime
	}
	if gitCommit != "" {
		if flags != "" {
			flags += " "
		}
		flags += "-X go_backend/core.GitCommit=" + gitCommit
	}
	return flags
}
