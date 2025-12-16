package core

import (
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	// GetVersion should return the Version variable
	result := GetVersion()
	if result != Version {
		t.Errorf("GetVersion() = %q, want %q", result, Version)
	}
}

func TestGetBuildTime(t *testing.T) {
	// GetBuildTime should return the BuildTime variable
	result := GetBuildTime()
	if result != BuildTime {
		t.Errorf("GetBuildTime() = %q, want %q", result, BuildTime)
	}
}

func TestGetGitCommit(t *testing.T) {
	// GetGitCommit should return the GitCommit variable
	result := GetGitCommit()
	if result != GitCommit {
		t.Errorf("GetGitCommit() = %q, want %q", result, GitCommit)
	}
}

func TestGetVersionInfo(t *testing.T) {
	result := GetVersionInfo()

	// Should contain the version
	if !strings.Contains(result, Version) {
		t.Errorf("GetVersionInfo() = %q, should contain version %q", result, Version)
	}

	// Should contain the build time
	if !strings.Contains(result, BuildTime) {
		t.Errorf("GetVersionInfo() = %q, should contain build time %q", result, BuildTime)
	}

	// Should contain the git commit
	if !strings.Contains(result, GitCommit) {
		t.Errorf("GetVersionInfo() = %q, should contain git commit %q", result, GitCommit)
	}

	// Should have expected format
	if !strings.Contains(result, "built") || !strings.Contains(result, "commit") {
		t.Errorf("GetVersionInfo() = %q, should contain 'built' and 'commit' labels", result)
	}
}

func TestBuildLdflags(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		buildTime string
		gitCommit string
		expected  string
	}{
		{
			name:      "all values set",
			version:   "v1.0.0",
			buildTime: "2024-01-15T10:30:00Z",
			gitCommit: "abc1234",
			expected:  "-X go_backend/core.Version=v1.0.0 -X go_backend/core.BuildTime=2024-01-15T10:30:00Z -X go_backend/core.GitCommit=abc1234",
		},
		{
			name:      "only version",
			version:   "v2.0.0",
			buildTime: "",
			gitCommit: "",
			expected:  "-X go_backend/core.Version=v2.0.0",
		},
		{
			name:      "version and commit",
			version:   "v1.5.0",
			buildTime: "",
			gitCommit: "def5678",
			expected:  "-X go_backend/core.Version=v1.5.0 -X go_backend/core.GitCommit=def5678",
		},
		{
			name:      "no values",
			version:   "",
			buildTime: "",
			gitCommit: "",
			expected:  "",
		},
		{
			name:      "only build time",
			version:   "",
			buildTime: "2024-12-16T00:00:00Z",
			gitCommit: "",
			expected:  "-X go_backend/core.BuildTime=2024-12-16T00:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildLdflags(tt.version, tt.buildTime, tt.gitCommit)
			if result != tt.expected {
				t.Errorf("BuildLdflags(%q, %q, %q) = %q, want %q",
					tt.version, tt.buildTime, tt.gitCommit, result, tt.expected)
			}
		})
	}
}
