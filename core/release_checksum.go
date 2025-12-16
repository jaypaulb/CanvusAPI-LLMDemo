package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateChecksumFile computes the SHA256 checksum of a file and writes it to a .sha256 file.
// The output format is compatible with sha256sum (Linux) and can be verified with:
//   sha256sum -c filename.sha256
//
// Parameters:
//   - filePath: path to the file to checksum
//
// Returns:
//   - checksumFile: path to the generated .sha256 file
//   - checksum: the computed SHA256 hash
//   - error: if file cannot be read or checksum file cannot be written
//
// Output format: "hash  filename" (two spaces, filename only without path)
//
// Examples:
//   - GenerateChecksumFile("release/app-v1.0.0.tar.gz")
//     creates "release/app-v1.0.0.tar.gz.sha256" containing "abc123...  app-v1.0.0.tar.gz"
//
// This function has side effects (creates a file).
func GenerateChecksumFile(filePath string) (checksumFile string, checksum string, err error) {
	if filePath == "" {
		return "", "", fmt.Errorf("file path cannot be empty")
	}

	// Compute the checksum
	checksum, err = ComputeSHA256(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to compute checksum: %w", err)
	}

	// Generate checksum file path
	checksumFile = filePath + ".sha256"

	// Get just the filename (no path) for the checksum file content
	filename := filepath.Base(filePath)

	// Format: "hash  filename" (two spaces, compatible with sha256sum)
	content := fmt.Sprintf("%s  %s\n", checksum, filename)

	// Write the checksum file
	if err := os.WriteFile(checksumFile, []byte(content), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write checksum file: %w", err)
	}

	return checksumFile, checksum, nil
}

// GenerateChecksumFiles generates SHA256 checksum files for multiple release artifacts.
// Creates a .sha256 file alongside each artifact.
//
// Parameters:
//   - filePaths: slice of paths to files to checksum
//
// Returns:
//   - results: map of original file path to ChecksumResult
//   - error: first error encountered (processing continues for other files)
//
// This function has side effects (creates files).
func GenerateChecksumFiles(filePaths []string) (map[string]ChecksumResult, error) {
	results := make(map[string]ChecksumResult)
	var firstErr error

	for _, fp := range filePaths {
		checksumFile, checksum, err := GenerateChecksumFile(fp)
		results[fp] = ChecksumResult{
			ChecksumFile: checksumFile,
			Checksum:     checksum,
			Error:        err,
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return results, firstErr
}

// ChecksumResult holds the result of checksum generation for a single file.
type ChecksumResult struct {
	ChecksumFile string // Path to the generated .sha256 file
	Checksum     string // The computed SHA256 hash
	Error        error  // Any error that occurred
}

// FormatChecksumLine creates a single checksum line in sha256sum-compatible format.
// Format: "hash  filename" (two spaces between hash and filename)
//
// Parameters:
//   - checksum: the SHA256 hash (64 hex characters)
//   - filename: the filename (without path)
//
// Returns:
//   - string: formatted checksum line
//
// This is a pure function with no side effects.
func FormatChecksumLine(checksum, filename string) string {
	return fmt.Sprintf("%s  %s", checksum, filename)
}

// GenerateCombinedChecksumFile creates a single checksum file containing hashes for multiple files.
// This is useful for creating a CHECKSUMS.sha256 file for release archives.
//
// Parameters:
//   - outputPath: path where the combined checksum file will be written
//   - filePaths: slice of paths to files to include in the checksum file
//
// Returns:
//   - checksums: map of filename to checksum
//   - error: if any file cannot be read or output cannot be written
//
// Output format: Multiple lines of "hash  filename" (filenames only, no paths)
//
// This function has side effects (creates a file).
func GenerateCombinedChecksumFile(outputPath string, filePaths []string) (map[string]string, error) {
	if outputPath == "" {
		return nil, fmt.Errorf("output path cannot be empty")
	}
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no files provided")
	}

	checksums := make(map[string]string)
	var content string

	for _, fp := range filePaths {
		checksum, err := ComputeSHA256(fp)
		if err != nil {
			return nil, fmt.Errorf("failed to compute checksum for %q: %w", fp, err)
		}

		filename := filepath.Base(fp)
		checksums[filename] = checksum
		content += FormatChecksumLine(checksum, filename) + "\n"
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write checksum file: %w", err)
	}

	return checksums, nil
}

// GenerateWindowsChecksumCommand returns the Windows certutil command for generating a checksum.
// This is a helper for build scripts that need to generate checksums on Windows.
//
// Parameters:
//   - filePath: path to the file to checksum
//
// Returns:
//   - string: the certutil command to run
//
// Example output: "certutil -hashfile path/to/file.exe SHA256"
//
// This is a pure function with no side effects.
func GenerateWindowsChecksumCommand(filePath string) string {
	return fmt.Sprintf("certutil -hashfile %q SHA256", filePath)
}

// GenerateLinuxChecksumCommand returns the Linux sha256sum command for generating a checksum.
// This is a helper for build scripts that need to generate checksums on Linux.
//
// Parameters:
//   - filePath: path to the file to checksum
//
// Returns:
//   - string: the sha256sum command to run
//
// Example output: "sha256sum path/to/file.tar.gz"
//
// This is a pure function with no side effects.
func GenerateLinuxChecksumCommand(filePath string) string {
	return fmt.Sprintf("sha256sum %q", filePath)
}

// GenerateMacChecksumCommand returns the macOS shasum command for generating a checksum.
// This is a helper for build scripts that need to generate checksums on macOS.
//
// Parameters:
//   - filePath: path to the file to checksum
//
// Returns:
//   - string: the shasum command to run
//
// Example output: "shasum -a 256 path/to/file.dmg"
//
// This is a pure function with no side effects.
func GenerateMacChecksumCommand(filePath string) string {
	return fmt.Sprintf("shasum -a 256 %q", filePath)
}
