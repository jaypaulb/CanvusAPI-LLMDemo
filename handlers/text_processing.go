// Package handlers provides request handling utilities including text processing atoms.
package handlers

import (
	"strings"

	"github.com/google/uuid"
)

// GenerateCorrelationID creates a unique 8-character ID for request tracing.
// Uses UUID v4 and truncates to first 8 characters for brevity while
// maintaining sufficient uniqueness for correlation purposes.
//
// This is a pure atom function with no external dependencies beyond UUID generation.
//
// Example:
//
//	correlationID := handlers.GenerateCorrelationID()
//	log.Printf("Processing request: %s", correlationID)
func GenerateCorrelationID() string {
	return uuid.New().String()[:8]
}

// TruncateText truncates a string to a specified maximum length.
// If the text is shorter than the limit, it is returned unchanged.
//
// This is a pure atom function with no side effects.
//
// Example:
//
//	preview := handlers.TruncateText(longContent, 50)
func TruncateText(text string, maxLength int) string {
	if len(text) > maxLength {
		return text[:maxLength]
	}
	return text
}

// ExtractAIPrompt removes AI trigger markers {{ and }} from note text.
// This extracts the raw prompt content from a note that triggered AI processing.
//
// This is a pure atom function.
//
// Example:
//
//	prompt := handlers.ExtractAIPrompt("{{Generate a haiku}}")
//	// Returns: "Generate a haiku"
func ExtractAIPrompt(noteText string) string {
	return strings.ReplaceAll(strings.ReplaceAll(noteText, "{{", ""), "}}", "")
}

// HasAITrigger checks if text contains an AI trigger pattern ({{ }}).
// Used to determine if a note update should trigger AI processing.
//
// This is a pure atom function.
//
// Example:
//
//	if handlers.HasAITrigger(noteText) {
//	    processAIRequest(noteText)
//	}
func HasAITrigger(text string) bool {
	return strings.Contains(text, "{{") && strings.Contains(text, "}}")
}

// IsAzureOpenAIEndpoint checks if an endpoint URL is an Azure OpenAI endpoint.
// Azure endpoints contain "openai.azure.com" or "cognitiveservices.azure.com".
//
// This is a pure atom function used for endpoint routing.
//
// Example:
//
//	if handlers.IsAzureOpenAIEndpoint(endpoint) {
//	    useAzureClient()
//	}
func IsAzureOpenAIEndpoint(endpoint string) bool {
	lowerEndpoint := strings.ToLower(endpoint)
	return strings.Contains(lowerEndpoint, "openai.azure.com") ||
		strings.Contains(lowerEndpoint, "cognitiveservices.azure.com")
}

// EstimateTokenCount provides a rough estimate of tokens in text.
// Uses a simple approximation of 4 characters per token.
// This is suitable for quick estimates but not precise tokenization.
//
// This is a pure atom function.
//
// Example:
//
//	tokens := handlers.EstimateTokenCount(prompt)
//	if tokens > maxTokens {
//	    prompt = truncateByTokens(prompt, maxTokens)
//	}
func EstimateTokenCount(text string) int {
	return len(text) / 4
}

// SplitIntoChunks splits text into chunks based on paragraph boundaries.
// Attempts to keep paragraphs together within the size limit.
// Returns empty slice if input text is empty.
//
// This is a pure atom function useful for processing large documents.
//
// Example:
//
//	chunks := handlers.SplitIntoChunks(pdfText, 4000)
//	for _, chunk := range chunks {
//	    processPDFChunk(chunk)
//	}
func SplitIntoChunks(text string, maxChunkSize int) []string {
	var chunks []string
	paragraphs := strings.Split(text, "\n\n")

	var currentChunk strings.Builder
	currentSize := 0

	for _, para := range paragraphs {
		paraSize := len(para)

		if currentSize+paraSize > maxChunkSize {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentSize = 0
			}
		}

		currentChunk.WriteString(para)
		currentChunk.WriteString("\n\n")
		currentSize += paraSize + 2
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// PDFChunkPrompt returns the system message for PDF chunk analysis.
// This provides consistent instructions for AI when processing PDF sections.
//
// This is a pure atom function returning a constant string.
//
// Example:
//
//	systemPrompt := handlers.PDFChunkPrompt()
func PDFChunkPrompt() string {
	return `You are analyzing a section of a document. Focus on:
1. Main ideas and key points
2. Important details and evidence
3. Connections to other sections
4. Technical accuracy and academic tone
Format your response as: {"type": "text", "content": "your analysis"}`
}
