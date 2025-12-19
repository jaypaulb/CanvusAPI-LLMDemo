package validation

import (
	"fmt"
	"os"
)

// FileExistsError indicates a file does not exist with a descriptive message
type FileExistsError struct {
	Path    string
	Message string
}

func (e *FileExistsError) Error() string {
	return e.Message
}

// CheckFileExists checks if a file exists at the given path.
// This is a pure function that only checks existence, no side effects.
//
// Returns nil if the file exists, or a *FileExistsError describing the failure.
func CheckFileExists(path string) error {
	if path == "" {
		return &FileExistsError{
			Path:    path,
			Message: "file path cannot be empty",
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &FileExistsError{
				Path:    path,
				Message: fmt.Sprintf("file not found: %s", path),
			}
		}
		return &FileExistsError{
			Path:    path,
			Message: fmt.Sprintf("error checking file %s: %v", path, err),
		}
	}

	if info.IsDir() {
		return &FileExistsError{
			Path:    path,
			Message: fmt.Sprintf("path is a directory, not a file: %s", path),
		}
	}

	return nil
}

// CheckEnvFileExists checks if the .env file exists in the current directory.
// This is a convenience wrapper around CheckFileExists for the common .env case.
func CheckEnvFileExists() error {
	return CheckFileExists(".env")
}
