// Package pdfprocessor provides PDF processing functionality for CanvusLocalLLM.
//
// chunker.go implements the Chunker molecule that splits text into chunks
// respecting token limits. It composes:
//   - atoms.go: EstimateTokenCount for token estimation
package pdfprocessor

import (
	"strings"
)

// ChunkerConfig holds configuration for text chunking.
type ChunkerConfig struct {
	// MaxChunkTokens is the maximum number of tokens per chunk.
	// Uses EstimateTokenCount for estimation (4 chars per token).
	MaxChunkTokens int

	// MaxChunks limits the total number of chunks produced.
	// Set to 0 for unlimited.
	MaxChunks int

	// OverlapTokens is the number of tokens to overlap between chunks.
	// This helps maintain context across chunk boundaries.
	// Set to 0 for no overlap.
	OverlapTokens int

	// PreserveParagraphs when true attempts to keep paragraphs intact.
	// When false, chunks can split mid-paragraph to hit token limits.
	PreserveParagraphs bool

	// ParagraphSeparator is the string used to split paragraphs.
	// Defaults to "\n\n" if empty.
	ParagraphSeparator string
}

// DefaultChunkerConfig returns sensible default configuration.
func DefaultChunkerConfig() ChunkerConfig {
	return ChunkerConfig{
		MaxChunkTokens:     20000,
		MaxChunks:          10,
		OverlapTokens:      0,
		PreserveParagraphs: true,
		ParagraphSeparator: "\n\n",
	}
}

// ChunkResult represents a single chunk with metadata.
type ChunkResult struct {
	// Text is the chunk content
	Text string

	// Index is the 0-based chunk index
	Index int

	// EstimatedTokens is the estimated token count for this chunk
	EstimatedTokens int

	// StartOffset is the character offset in the original text
	StartOffset int

	// EndOffset is the ending character offset in the original text
	EndOffset int
}

// ChunkerResult contains all chunks and metadata from chunking operation.
type ChunkerResult struct {
	// Chunks is the list of text chunks
	Chunks []ChunkResult

	// TotalChunks is the number of chunks produced
	TotalChunks int

	// TotalTokensEstimate is the total estimated tokens across all chunks
	TotalTokensEstimate int

	// Truncated is true if MaxChunks limit caused truncation
	Truncated bool

	// OriginalTokensEstimate is the estimated tokens in the original text
	OriginalTokensEstimate int
}

// Chunker splits text into chunks respecting token limits.
//
// Thread-Safety:
//   - Chunker is safe for concurrent use (stateless)
type Chunker struct {
	config ChunkerConfig
}

// NewChunker creates a new Chunker with the given configuration.
//
// Parameters:
//   - config: chunking configuration
//
// Returns a Chunker instance.
func NewChunker(config ChunkerConfig) *Chunker {
	// Set default separator if empty
	if config.ParagraphSeparator == "" {
		config.ParagraphSeparator = "\n\n"
	}

	return &Chunker{
		config: config,
	}
}

// SplitIntoChunks divides text into chunks respecting token limits.
//
// The chunking strategy depends on configuration:
//   - When PreserveParagraphs is true, chunks try to end at paragraph boundaries
//   - When PreserveParagraphs is false, chunks are split at exact token limits
//   - OverlapTokens adds repeated content at chunk boundaries for context
//   - MaxChunks limits total output; Truncated flag indicates if content was cut
//
// Parameters:
//   - text: the text to chunk
//
// Returns ChunkerResult with chunks and metadata, never returns error.
// Empty input returns empty result.
func (c *Chunker) SplitIntoChunks(text string) *ChunkerResult {
	result := &ChunkerResult{
		Chunks:                 make([]ChunkResult, 0),
		OriginalTokensEstimate: EstimateTokenCount(text),
	}

	if text == "" {
		return result
	}

	if c.config.PreserveParagraphs {
		c.chunkByParagraphs(text, result)
	} else {
		c.chunkByTokens(text, result)
	}

	// Apply max chunks limit
	if c.config.MaxChunks > 0 && len(result.Chunks) > c.config.MaxChunks {
		result.Chunks = result.Chunks[:c.config.MaxChunks]
		result.Truncated = true
	}

	// Update totals
	result.TotalChunks = len(result.Chunks)
	totalTokens := 0
	for _, chunk := range result.Chunks {
		totalTokens += chunk.EstimatedTokens
	}
	result.TotalTokensEstimate = totalTokens

	return result
}

// chunkByParagraphs splits text preserving paragraph boundaries.
func (c *Chunker) chunkByParagraphs(text string, result *ChunkerResult) {
	paragraphs := strings.Split(text, c.config.ParagraphSeparator)

	var currentChunk strings.Builder
	currentTokens := 0
	currentOffset := 0
	chunkStartOffset := 0
	chunkIndex := 0

	for i, para := range paragraphs {
		paraTokens := EstimateTokenCount(para)
		separatorLen := len(c.config.ParagraphSeparator)

		// Check if adding this paragraph exceeds limit
		if currentTokens > 0 && currentTokens+paraTokens > c.config.MaxChunkTokens {
			// Finalize current chunk
			chunk := ChunkResult{
				Text:            strings.TrimSuffix(currentChunk.String(), c.config.ParagraphSeparator),
				Index:           chunkIndex,
				EstimatedTokens: currentTokens,
				StartOffset:     chunkStartOffset,
				EndOffset:       currentOffset,
			}
			result.Chunks = append(result.Chunks, chunk)

			// Handle overlap
			overlapText := ""
			if c.config.OverlapTokens > 0 {
				overlapText = c.getOverlapText(chunk.Text)
			}

			// Reset for next chunk
			currentChunk.Reset()
			if overlapText != "" {
				currentChunk.WriteString(overlapText)
				currentChunk.WriteString(c.config.ParagraphSeparator)
				currentTokens = EstimateTokenCount(overlapText)
			} else {
				currentTokens = 0
			}
			chunkStartOffset = currentOffset
			chunkIndex++
		}

		// Add paragraph to current chunk
		currentChunk.WriteString(para)
		currentTokens += paraTokens

		// Add separator unless it's the last paragraph
		if i < len(paragraphs)-1 {
			currentChunk.WriteString(c.config.ParagraphSeparator)
			currentOffset += len(para) + separatorLen
			currentTokens += EstimateTokenCount(c.config.ParagraphSeparator)
		} else {
			currentOffset += len(para)
		}
	}

	// Add final chunk if any content remains
	if currentChunk.Len() > 0 {
		chunk := ChunkResult{
			Text:            currentChunk.String(),
			Index:           chunkIndex,
			EstimatedTokens: currentTokens,
			StartOffset:     chunkStartOffset,
			EndOffset:       currentOffset,
		}
		result.Chunks = append(result.Chunks, chunk)
	}
}

// chunkByTokens splits text at exact token boundaries.
func (c *Chunker) chunkByTokens(text string, result *ChunkerResult) {
	// Convert max tokens to approximate character count
	maxChars := c.config.MaxChunkTokens * 4 // 4 chars per token approximation
	overlapChars := c.config.OverlapTokens * 4

	chunkIndex := 0
	offset := 0
	textLen := len(text)

	for offset < textLen {
		// Calculate chunk end
		endOffset := offset + maxChars
		if endOffset > textLen {
			endOffset = textLen
		}

		// Extract chunk
		chunkText := text[offset:endOffset]

		chunk := ChunkResult{
			Text:            chunkText,
			Index:           chunkIndex,
			EstimatedTokens: EstimateTokenCount(chunkText),
			StartOffset:     offset,
			EndOffset:       endOffset,
		}
		result.Chunks = append(result.Chunks, chunk)

		// Move to next chunk with overlap
		if overlapChars > 0 && endOffset < textLen {
			offset = endOffset - overlapChars
			if offset < 0 {
				offset = 0
			}
		} else {
			offset = endOffset
		}

		chunkIndex++
	}
}

// getOverlapText extracts the last OverlapTokens worth of text.
func (c *Chunker) getOverlapText(text string) string {
	overlapChars := c.config.OverlapTokens * 4
	if overlapChars >= len(text) {
		return text
	}
	return text[len(text)-overlapChars:]
}

// EstimateChunkCount returns an estimate of how many chunks the text will produce.
// This is useful for progress indicators before actually chunking.
//
// Parameters:
//   - text: the text to estimate
//
// Returns estimated number of chunks.
func (c *Chunker) EstimateChunkCount(text string) int {
	if text == "" {
		return 0
	}

	totalTokens := EstimateTokenCount(text)
	if totalTokens <= c.config.MaxChunkTokens {
		return 1
	}

	// Simple division for estimate
	estimate := (totalTokens + c.config.MaxChunkTokens - 1) / c.config.MaxChunkTokens

	if c.config.MaxChunks > 0 && estimate > c.config.MaxChunks {
		return c.config.MaxChunks
	}

	return estimate
}

// ChunksToStrings extracts just the text content from a ChunkerResult.
// This is a convenience method for when metadata is not needed.
//
// Parameters:
//   - result: the chunker result
//
// Returns slice of chunk text strings.
func ChunksToStrings(result *ChunkerResult) []string {
	if result == nil || len(result.Chunks) == 0 {
		return nil
	}

	texts := make([]string, len(result.Chunks))
	for i, chunk := range result.Chunks {
		texts[i] = chunk.Text
	}
	return texts
}
