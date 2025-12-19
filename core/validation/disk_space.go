package validation

import (
	"go_backend/core"
	"fmt"
	"os"
	"runtime"
)

// DiskSpaceInfo contains information about disk space.
type DiskSpaceInfo struct {
	// Path that was checked
	Path string
	// Total disk space in bytes
	Total int64
	// Free disk space in bytes
	Free int64
	// Used disk space in bytes
	Used int64
	// Human-readable total
	TotalFormatted string
	// Human-readable free
	FreeFormatted string
	// Human-readable used
	UsedFormatted string
	// Percentage used (0-100)
	UsedPercent float64
}

// core.DiskSpaceError indicates a disk space problem.
type DiskSpaceError struct {
	// Path that was checked
	Path string
	// Required space in bytes
	Required int64
	// Available space in bytes
	Available int64
	// Human-readable message
	Message string
}

func (e *DiskSpaceError) Error() string {
	return e.Message
}

// GetDiskSpace returns disk space information for the given path.
// The path can be a file or directory; the function will check the filesystem
// containing that path.
func GetDiskSpace(path string) (*DiskSpaceInfo, error) {
	// Ensure the path exists by trying to stat it
	// If it doesn't exist, try the parent directory
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Try parent directory
			parentPath := getParentPath(path)
			if parentPath != "" && parentPath != path {
				return GetDiskSpace(parentPath)
			}
		}
		return nil, fmt.Errorf("cannot access path %s: %w", path, err)
	}

	// If it's a file, use its parent directory
	if !info.IsDir() {
		parentPath := getParentPath(path)
		if parentPath != "" {
			path = parentPath
		}
	}

	// Get disk space using platform-specific implementation
	total, free, err := getDiskSpace(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk space for %s: %w", path, err)
	}

	used := total - free
	var usedPercent float64
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100
	}

	return &DiskSpaceInfo{
		Path:           path,
		Total:          total,
		Free:           free,
		Used:           used,
		TotalFormatted: core.FormatBytes(total),
		FreeFormatted:  core.FormatBytes(free),
		UsedFormatted:  core.FormatBytes(used),
		UsedPercent:    usedPercent,
	}, nil
}

// CheckDiskSpace verifies there is sufficient disk space at the given path.
// Returns nil if there is enough space, or a core.DiskSpaceError if not.
// requiredBytes is the minimum required free space in bytes.
func CheckDiskSpace(path string, requiredBytes int64) error {
	info, err := GetDiskSpace(path)
	if err != nil {
		return err
	}

	if info.Free < requiredBytes {
		return &DiskSpaceError{
			Path:      path,
			Required:  requiredBytes,
			Available: info.Free,
			Message: fmt.Sprintf("insufficient disk space at %s: need %s, have %s free",
				path, core.FormatBytes(requiredBytes), info.FreeFormatted),
		}
	}

	return nil
}

// CheckDiskSpaceForModel checks if there's enough space to download a model.
// Uses default model size requirements (~8GB for typical LLM model).
// bufferPercent is additional buffer space as a percentage (e.g., 10 for 10% extra).
func core.CheckDiskSpaceForModel(path string, modelSizeBytes int64, bufferPercent int) error {
	// Add buffer for extraction, temp files, etc.
	buffer := modelSizeBytes * int64(bufferPercent) / 100
	required := modelSizeBytes + buffer

	return CheckDiskSpace(path, required)
}

// DefaultModelSizeBytes is the typical size requirement for an LLM model (~8GB).
const DefaultModelSizeBytes int64 = 8 * core.BytesPerGB

// core.DefaultBufferPercent is the default buffer percentage to add for temporary files.
const core.DefaultBufferPercent = 10

// CheckDiskSpaceForDefaultModel checks disk space for a typical LLM model download.
// Uses DefaultModelSizeBytes (8GB) with core.DefaultBufferPercent (10%) buffer.
func CheckDiskSpaceForDefaultModel(path string) error {
	return core.CheckDiskSpaceForModel(path, DefaultModelSizeBytes, core.DefaultBufferPercent)
}

// getParentPath returns the parent directory of a path.
// Returns "" if the path has no parent (e.g., "/" or ".").
func getParentPath(path string) string {
	// Handle special cases
	if path == "" || path == "." || path == "/" {
		return ""
	}

	// Handle different OS path separators
	if runtime.GOOS == "windows" {
		// Handle Windows paths like C:\foo\bar
		// Remove trailing separator if present
		if len(path) > 1 && (path[len(path)-1] == '\\' || path[len(path)-1] == '/') {
			path = path[:len(path)-1]
		}
		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '\\' || path[i] == '/' {
				if i == 2 && len(path) > 2 && path[1] == ':' {
					// Return drive root like "C:\"
					return path[:3]
				}
				if i == 0 {
					return ""
				}
				return path[:i]
			}
		}
		return ""
	}

	// Unix-like paths
	// Remove trailing separator if present
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	sep := string(os.PathSeparator)
	for i := len(path) - 1; i >= 0; i-- {
		if string(path[i]) == sep {
			if i == 0 {
				return "/" // Parent of /foo is /
			}
			return path[:i]
		}
	}
	return ""
}
