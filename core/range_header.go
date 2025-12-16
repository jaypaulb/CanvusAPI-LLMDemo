package core

import "fmt"

// BuildRangeHeader constructs an HTTP Range header for resumable downloads.
// The Range header allows requesting a specific byte range from a server,
// enabling download resumption from a specific offset.
//
// Parameters:
//   - resumeFrom: the byte offset to start downloading from (0-indexed)
//
// Returns:
//   - string: HTTP Range header value in the format "bytes=N-" where N is the offset
//
// Examples:
//   - BuildRangeHeader(0) returns "bytes=0-" (start from beginning)
//   - BuildRangeHeader(1024) returns "bytes=1024-" (resume from byte 1024)
//   - BuildRangeHeader(1048576) returns "bytes=1048576-" (resume from 1MB offset)
//
// Note: Negative values are treated as 0 (start from beginning).
//
// This is a pure function with no side effects.
func BuildRangeHeader(resumeFrom int64) string {
	if resumeFrom < 0 {
		resumeFrom = 0
	}
	return fmt.Sprintf("bytes=%d-", resumeFrom)
}

// BuildRangeHeaderWithEnd constructs an HTTP Range header with both start and end offsets.
// This is useful when requesting a specific chunk of a file.
//
// Parameters:
//   - start: the starting byte offset (0-indexed, inclusive)
//   - end: the ending byte offset (inclusive)
//
// Returns:
//   - string: HTTP Range header value in the format "bytes=start-end"
//
// Examples:
//   - BuildRangeHeaderWithEnd(0, 999) returns "bytes=0-999" (first 1000 bytes)
//   - BuildRangeHeaderWithEnd(1000, 1999) returns "bytes=1000-1999" (second 1000 bytes)
//
// Note: If start > end, they are swapped. Negative values are treated as 0.
//
// This is a pure function with no side effects.
func BuildRangeHeaderWithEnd(start, end int64) string {
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if start > end {
		start, end = end, start
	}
	return fmt.Sprintf("bytes=%d-%d", start, end)
}

// BuildRangeHeaderSuffix constructs an HTTP Range header requesting the last N bytes.
// This is useful for getting file trailers or metadata at the end of files.
//
// Parameters:
//   - suffixLength: number of bytes to request from the end of the file
//
// Returns:
//   - string: HTTP Range header value in the format "bytes=-N"
//
// Examples:
//   - BuildRangeHeaderSuffix(500) returns "bytes=-500" (last 500 bytes)
//   - BuildRangeHeaderSuffix(1024) returns "bytes=-1024" (last 1KB)
//
// Note: Values less than 1 are treated as 1.
//
// This is a pure function with no side effects.
func BuildRangeHeaderSuffix(suffixLength int64) string {
	if suffixLength < 1 {
		suffixLength = 1
	}
	return fmt.Sprintf("bytes=-%d", suffixLength)
}

// ParseContentRange parses a Content-Range header response to extract byte range info.
// Servers respond with Content-Range when honoring a Range request.
//
// Expected format: "bytes start-end/total" or "bytes start-end/*"
//
// Parameters:
//   - header: the Content-Range header value from the server response
//
// Returns:
//   - start: the starting byte of the returned range
//   - end: the ending byte of the returned range (inclusive)
//   - total: the total size of the resource (-1 if unknown, indicated by *)
//   - error: if the header format is invalid
//
// Examples:
//   - ParseContentRange("bytes 0-999/5000") returns (0, 999, 5000, nil)
//   - ParseContentRange("bytes 1000-1999/*") returns (1000, 1999, -1, nil)
//
// This is a pure function with no side effects.
func ParseContentRange(header string) (start, end, total int64, err error) {
	if header == "" {
		return 0, 0, 0, fmt.Errorf("empty Content-Range header")
	}

	// Parse "bytes start-end/total" format
	var totalStr string
	n, scanErr := fmt.Sscanf(header, "bytes %d-%d/%s", &start, &end, &totalStr)
	if scanErr != nil || n < 2 {
		return 0, 0, 0, fmt.Errorf("invalid Content-Range format: %q", header)
	}

	// Parse total (can be "*" for unknown)
	if totalStr == "*" {
		total = -1
	} else {
		_, parseErr := fmt.Sscanf(totalStr, "%d", &total)
		if parseErr != nil {
			return 0, 0, 0, fmt.Errorf("invalid total in Content-Range: %q", totalStr)
		}
	}

	return start, end, total, nil
}

// IsPartialContentSupported checks if a server response indicates support for range requests.
// The server indicates this via the Accept-Ranges header.
//
// Parameters:
//   - acceptRangesHeader: the Accept-Ranges header value from the server
//
// Returns:
//   - bool: true if the server accepts byte range requests
//
// Examples:
//   - IsPartialContentSupported("bytes") returns true
//   - IsPartialContentSupported("none") returns false
//   - IsPartialContentSupported("") returns false
//
// This is a pure function with no side effects.
func IsPartialContentSupported(acceptRangesHeader string) bool {
	return acceptRangesHeader == "bytes"
}
