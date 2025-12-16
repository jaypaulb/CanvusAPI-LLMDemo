package core

import (
	"os"
	"path/filepath"
	"runtime"
)

// AppName is the application name used in data directory paths.
const AppName = "CanvusLocalLLM"

// GetDataDirectory returns the platform-specific data directory path for the application.
// This is a pure function based on runtime.GOOS and environment variables.
//
// Paths by platform:
//   - Windows: %APPDATA%/CanvusLocalLLM (e.g., C:\Users\<user>\AppData\Roaming\CanvusLocalLLM)
//   - Linux/macOS: ~/.canvuslocallm (e.g., /home/<user>/.canvuslocallm)
//
// Does NOT create the directory - callers should use EnsureDataDirectory for that.
func GetDataDirectory() string {
	switch runtime.GOOS {
	case "windows":
		// Use APPDATA on Windows
		appData := os.Getenv("APPDATA")
		if appData == "" {
			// Fallback to user home if APPDATA not set
			home, err := os.UserHomeDir()
			if err != nil {
				return AppName
			}
			return filepath.Join(home, "AppData", "Roaming", AppName)
		}
		return filepath.Join(appData, AppName)
	default:
		// Linux, macOS, and other Unix-like systems
		home, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory if home cannot be determined
			return ".canvuslocallm"
		}
		return filepath.Join(home, ".canvuslocallm")
	}
}

// GetDataFilePath returns the full path for a file within the data directory.
// Example: GetDataFilePath("config.db") -> "/home/user/.canvuslocallm/config.db"
func GetDataFilePath(filename string) string {
	return filepath.Join(GetDataDirectory(), filename)
}

// EnsureDataDirectory creates the data directory if it doesn't exist.
// Returns the directory path and any error encountered.
func EnsureDataDirectory() (string, error) {
	dir := GetDataDirectory()
	err := os.MkdirAll(dir, 0700) // Secure permissions: owner read/write/execute only
	if err != nil {
		return "", err
	}
	return dir, nil
}
