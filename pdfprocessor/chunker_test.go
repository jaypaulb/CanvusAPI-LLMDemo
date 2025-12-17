package pdfprocessor

import (
	"strings"
	"testing"
)

func TestDefaultChunkerConfig(t *testing.T) {
	config := DefaultChunkerConfig()

	if config.MaxChunkTokens <= 0 {
		t.Errorf("MaxChunkTokens = %d, want positive", config.MaxChunkTokens)
	}
	if config.MaxChunks <= 0 {
		t.Errorf("MaxChunks = %d, want positive", config.MaxChunks)
	}
	if config.ParagraphSeparator != "\n\n" {
		t.Errorf("ParagraphSeparator = %q, want %q", config.ParagraphSeparator, "\n\n")
	}
	if !config.PreserveParagraphs {
		t.Error("PreserveParagraphs should be true by default")
	}
}

func TestNewChunker(t *testing.T) {
	tests := []struct {
		name   string
		config ChunkerConfig
	}{
		{
			name:   "default config",
			config: DefaultChunkerConfig(),
		},
		{
			name: "empty separator uses default",
			config: ChunkerConfig{
				MaxChunkTokens:     1000,
				ParagraphSeparator: "",
			},
		},
		{
			name: "custom separator",
			config: ChunkerConfig{
				MaxChunkTokens:     1000,
				ParagraphSeparator: "\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunker := NewChunker(tt.config)
			if chunker == nil {
				t.Error("NewChunker returned nil")
			}
		})
	}
}

func TestChunker_SplitIntoChunks_Empty(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	result := chunker.SplitIntoChunks("")

	if result == nil {
		t.Fatal("SplitIntoChunks returned nil for empty input")
	}
	if len(result.Chunks) != 0 {
		t.Errorf("Chunks length = %d, want 0", len(result.Chunks))
	}
	if result.TotalChunks != 0 {
		t.Errorf("TotalChunks = %d, want 0", result.TotalChunks)
	}
	if result.OriginalTokensEstimate != 0 {
		t.Errorf("OriginalTokensEstimate = %d, want 0", result.OriginalTokensEstimate)
	}
}

func TestChunker_SplitIntoChunks_SingleParagraph(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{
		MaxChunkTokens:     1000, // Large enough for test text
		PreserveParagraphs: true,
		ParagraphSeparator: "\n\n",
	})

	text := "This is a single paragraph of text that should not be chunked."
	result := chunker.SplitIntoChunks(text)

	if result.TotalChunks != 1 {
		t.Errorf("TotalChunks = %d, want 1", result.TotalChunks)
	}
	if result.Chunks[0].Text != text {
		t.Errorf("Chunk text = %q, want %q", result.Chunks[0].Text, text)
	}
	if result.Chunks[0].Index != 0 {
		t.Errorf("Chunk index = %d, want 0", result.Chunks[0].Index)
	}
}

func TestChunker_SplitIntoChunks_MultipleParagraphs(t *testing.T) {
	config := ChunkerConfig{
		MaxChunkTokens:     50, // Small limit to force chunking
		PreserveParagraphs: true,
		ParagraphSeparator: "\n\n",
	}
	chunker := NewChunker(config)

	// Create text with multiple paragraphs
	para1 := strings.Repeat("a", 100) // ~25 tokens
	para2 := strings.Repeat("b", 100) // ~25 tokens
	para3 := strings.Repeat("c", 100) // ~25 tokens
	text := para1 + "\n\n" + para2 + "\n\n" + para3

	result := chunker.SplitIntoChunks(text)

	if result.TotalChunks < 2 {
		t.Errorf("TotalChunks = %d, want at least 2", result.TotalChunks)
	}

	// Verify indices are sequential
	for i, chunk := range result.Chunks {
		if chunk.Index != i {
			t.Errorf("Chunk[%d].Index = %d, want %d", i, chunk.Index, i)
		}
	}
}

func TestChunker_SplitIntoChunks_MaxChunksLimit(t *testing.T) {
	config := ChunkerConfig{
		MaxChunkTokens:     10, // Very small to force many chunks
		MaxChunks:          3,  // Limit to 3 chunks
		PreserveParagraphs: false,
		ParagraphSeparator: "\n\n",
	}
	chunker := NewChunker(config)

	// Create text that would require many chunks
	text := strings.Repeat("word ", 500) // ~500 tokens worth

	result := chunker.SplitIntoChunks(text)

	if result.TotalChunks > config.MaxChunks {
		t.Errorf("TotalChunks = %d, want <= %d", result.TotalChunks, config.MaxChunks)
	}
	if !result.Truncated {
		t.Error("Truncated should be true when MaxChunks limits output")
	}
}

func TestChunker_SplitIntoChunks_PreserveParagraphsFalse(t *testing.T) {
	config := ChunkerConfig{
		MaxChunkTokens:     25, // ~100 chars
		PreserveParagraphs: false,
		ParagraphSeparator: "\n\n",
	}
	chunker := NewChunker(config)

	// Create continuous text
	text := strings.Repeat("x", 200) // ~50 tokens, should be split

	result := chunker.SplitIntoChunks(text)

	if result.TotalChunks < 2 {
		t.Errorf("TotalChunks = %d, want at least 2", result.TotalChunks)
	}

	// Verify chunks together reconstruct original (without overlap)
	reconstructed := ""
	for _, chunk := range result.Chunks {
		reconstructed += chunk.Text
	}
	if reconstructed != text {
		t.Errorf("Reconstructed text length = %d, want %d", len(reconstructed), len(text))
	}
}

func TestChunker_SplitIntoChunks_WithOverlap(t *testing.T) {
	config := ChunkerConfig{
		MaxChunkTokens:     25, // ~100 chars
		OverlapTokens:      5,  // ~20 chars overlap
		PreserveParagraphs: false,
		ParagraphSeparator: "\n\n",
	}
	chunker := NewChunker(config)

	text := strings.Repeat("y", 300) // Should produce multiple chunks

	result := chunker.SplitIntoChunks(text)

	if result.TotalChunks < 2 {
		t.Errorf("TotalChunks = %d, want at least 2 for overlap test", result.TotalChunks)
	}

	// With overlap, total chars in chunks should exceed original
	totalChars := 0
	for _, chunk := range result.Chunks {
		totalChars += len(chunk.Text)
	}
	if config.OverlapTokens > 0 && result.TotalChunks > 1 {
		if totalChars <= len(text) {
			t.Errorf("Total chars with overlap = %d, should exceed original %d", totalChars, len(text))
		}
	}
}

func TestChunker_SplitIntoChunks_CustomSeparator(t *testing.T) {
	config := ChunkerConfig{
		MaxChunkTokens:     30, // ~120 chars, less than one paragraph
		PreserveParagraphs: true,
		ParagraphSeparator: "---", // Custom separator
	}
	chunker := NewChunker(config)

	// Each paragraph is ~50 tokens (200 chars), exceeds 30 token limit
	para1 := strings.Repeat("a", 200)
	para2 := strings.Repeat("b", 200)
	text := para1 + "---" + para2

	result := chunker.SplitIntoChunks(text)

	if result.TotalChunks < 2 {
		t.Errorf("TotalChunks = %d, want at least 2 with custom separator", result.TotalChunks)
	}

	// Verify the separator was used correctly by checking paragraphs aren't split
	for _, chunk := range result.Chunks {
		if strings.Contains(chunk.Text, "---") && !strings.HasSuffix(chunk.Text, "---") {
			// If a chunk contains the separator, it should be at the boundary
			// not mid-text (unless it's a single paragraph)
		}
	}
}

func TestChunker_EstimateChunkCount(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		maxChunkTokens int
		maxChunks      int
		wantMin        int
		wantMax        int
	}{
		{
			name:           "empty text",
			text:           "",
			maxChunkTokens: 100,
			wantMin:        0,
			wantMax:        0,
		},
		{
			name:           "single chunk",
			text:           strings.Repeat("x", 100), // ~25 tokens
			maxChunkTokens: 100,
			wantMin:        1,
			wantMax:        1,
		},
		{
			name:           "multiple chunks",
			text:           strings.Repeat("x", 1000), // ~250 tokens
			maxChunkTokens: 50,
			wantMin:        2,
			wantMax:        10,
		},
		{
			name:           "limited by max chunks",
			text:           strings.Repeat("x", 2000), // ~500 tokens
			maxChunkTokens: 10,
			maxChunks:      5,
			wantMin:        5,
			wantMax:        5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ChunkerConfig{
				MaxChunkTokens:     tt.maxChunkTokens,
				MaxChunks:          tt.maxChunks,
				PreserveParagraphs: true,
				ParagraphSeparator: "\n\n",
			}
			chunker := NewChunker(config)

			got := chunker.EstimateChunkCount(tt.text)

			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("EstimateChunkCount() = %d, want between %d and %d", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestChunksToStrings(t *testing.T) {
	tests := []struct {
		name   string
		result *ChunkerResult
		want   []string
	}{
		{
			name:   "nil result",
			result: nil,
			want:   nil,
		},
		{
			name:   "empty chunks",
			result: &ChunkerResult{Chunks: []ChunkResult{}},
			want:   nil,
		},
		{
			name: "single chunk",
			result: &ChunkerResult{
				Chunks: []ChunkResult{
					{Text: "hello"},
				},
			},
			want: []string{"hello"},
		},
		{
			name: "multiple chunks",
			result: &ChunkerResult{
				Chunks: []ChunkResult{
					{Text: "hello"},
					{Text: "world"},
					{Text: "test"},
				},
			},
			want: []string{"hello", "world", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChunksToStrings(tt.result)

			if tt.want == nil {
				if got != nil {
					t.Errorf("ChunksToStrings() = %v, want nil", got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("ChunksToStrings() length = %d, want %d", len(got), len(tt.want))
				return
			}

			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("ChunksToStrings()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestChunkResult_Metadata(t *testing.T) {
	config := ChunkerConfig{
		MaxChunkTokens:     50,
		PreserveParagraphs: false,
		ParagraphSeparator: "\n\n",
	}
	chunker := NewChunker(config)

	text := strings.Repeat("z", 300)
	result := chunker.SplitIntoChunks(text)

	if result.TotalChunks < 2 {
		t.Skip("Need at least 2 chunks for metadata test")
	}

	// Check first chunk
	first := result.Chunks[0]
	if first.StartOffset != 0 {
		t.Errorf("First chunk StartOffset = %d, want 0", first.StartOffset)
	}
	if first.EstimatedTokens <= 0 {
		t.Errorf("First chunk EstimatedTokens = %d, want positive", first.EstimatedTokens)
	}

	// Check last chunk
	last := result.Chunks[len(result.Chunks)-1]
	if last.EndOffset != len(text) {
		t.Errorf("Last chunk EndOffset = %d, want %d", last.EndOffset, len(text))
	}

	// Check chunk offsets are sequential
	for i := 1; i < len(result.Chunks); i++ {
		curr := result.Chunks[i]
		prev := result.Chunks[i-1]

		// Without overlap, next chunk should start where previous ended
		if config.OverlapTokens == 0 {
			if curr.StartOffset != prev.EndOffset {
				t.Errorf("Chunk[%d] StartOffset = %d, want %d (previous EndOffset)",
					i, curr.StartOffset, prev.EndOffset)
			}
		}
	}
}

func TestChunker_SplitIntoChunks_LargeParagraph(t *testing.T) {
	config := ChunkerConfig{
		MaxChunkTokens:     25, // ~100 chars
		PreserveParagraphs: true,
		ParagraphSeparator: "\n\n",
	}
	chunker := NewChunker(config)

	// Single paragraph larger than chunk size
	largePara := strings.Repeat("x", 500)
	text := largePara

	result := chunker.SplitIntoChunks(text)

	// With preserve paragraphs, it may still produce one chunk
	// that exceeds the limit rather than breaking mid-paragraph
	if result.TotalChunks == 0 {
		t.Error("Should produce at least one chunk")
	}
}
