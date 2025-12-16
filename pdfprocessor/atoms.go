// Package pdfprocessor provides PDF processing functionality for CanvusLocalLLM.
// This package handles PDF text extraction, summarization, and token management.
package pdfprocessor

// EstimateTokenCount provides a rough estimate of tokens in a text.
// It uses an average of 4 characters per token as an approximation,
// which is a reasonable heuristic for English text with GPT-style tokenizers.
//
// This is a pure function with no dependencies - it simply performs
// character counting and division.
//
// Example:
//
//	tokens := EstimateTokenCount("Hello, world!") // Returns 3
//	tokens := EstimateTokenCount("")              // Returns 0
func EstimateTokenCount(text string) int {
	if len(text) == 0 {
		return 0
	}
	return len(text) / 4
}

// TruncateText truncates a text to a specified maximum length.
// If the text is shorter than or equal to maxLen, it is returned unchanged.
// If truncation occurs, the text is cut at exactly maxLen characters.
//
// This is a pure function with no dependencies - it simply performs
// length checking and slicing.
//
// Note: This function truncates by bytes, not runes. For proper Unicode
// handling with multi-byte characters, consider using TruncateTextRunes.
//
// Example:
//
//	result := TruncateText("Hello, world!", 5)  // Returns "Hello"
//	result := TruncateText("Hi", 10)            // Returns "Hi"
//	result := TruncateText("", 5)               // Returns ""
func TruncateText(text string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen]
}

// TruncateTextWithEllipsis truncates text to maxLen and appends "..." if truncated.
// The total length including ellipsis will not exceed maxLen.
// If maxLen is less than 4, no ellipsis is added (not enough room).
//
// This is a pure function with no dependencies.
//
// Example:
//
//	result := TruncateTextWithEllipsis("Hello, world!", 8)  // Returns "Hello..."
//	result := TruncateTextWithEllipsis("Hi", 10)            // Returns "Hi"
func TruncateTextWithEllipsis(text string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(text) <= maxLen {
		return text
	}
	// Need at least 4 chars for "x..."
	if maxLen < 4 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}
