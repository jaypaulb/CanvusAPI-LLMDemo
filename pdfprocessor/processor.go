// Package pdfprocessor provides PDF processing functionality for CanvusLocalLLM.
//
// processor.go implements the Processor organism that orchestrates all PDF processing.
// It composes the following molecules:
//   - extractor.go: Extractor for PDF text extraction
//   - chunker.go: Chunker for text chunking
//   - summarizer.go: Summarizer for AI-powered summarization
package pdfprocessor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
)

// ErrProcessorNotConfigured is returned when the processor is missing required configuration.
var ErrProcessorNotConfigured = errors.New("processor not properly configured")

// ProcessorConfig holds configuration for the PDF processor.
type ProcessorConfig struct {
	// Extractor configuration
	ExtractorConfig ExtractorConfig

	// Chunker configuration
	ChunkerConfig ChunkerConfig

	// Summarizer configuration
	SummarizerConfig SummarizerConfig
}

// DefaultProcessorConfig returns sensible default configuration.
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		ExtractorConfig:  DefaultExtractorConfig(),
		ChunkerConfig:    DefaultChunkerConfig(),
		SummarizerConfig: DefaultSummarizerConfig(),
	}
}

// ProcessResult contains the complete result of PDF processing.
type ProcessResult struct {
	// Summary is the AI-generated summary content
	Summary string

	// ExtractionResult contains details about text extraction
	ExtractionResult *ExtractionResult

	// ChunkerResult contains details about text chunking
	ChunkerResult *ChunkerResult

	// SummaryResult contains details about AI summarization
	SummaryResult *SummaryResult

	// ProcessingTime is the total time taken to process the PDF
	ProcessingTime time.Duration

	// Stages contains timing for each processing stage
	Stages ProcessingStages
}

// ProcessingStages contains timing information for each stage.
type ProcessingStages struct {
	ExtractionTime   time.Duration
	ChunkingTime     time.Duration
	SummarizingTime  time.Duration
}

// ProgressCallback is called to report processing progress.
// stage is the current stage name, progress is 0.0-1.0, message is a human-readable status.
type ProgressCallback func(stage string, progress float64, message string)

// Processor orchestrates PDF text extraction, chunking, and AI summarization.
type Processor struct {
	config     ProcessorConfig
	extractor  *Extractor
	chunker    *Chunker
	summarizer *Summarizer
	progress   ProgressCallback
}

// NewProcessor creates a new Processor with the given configuration and OpenAI client.
//
// Example:
//
//	client := openai.NewClient("api-key")
//	processor := NewProcessor(DefaultProcessorConfig(), client)
//	result, err := processor.Process(ctx, "/path/to/document.pdf")
func NewProcessor(config ProcessorConfig, client *openai.Client) *Processor {
	return &Processor{
		config:     config,
		extractor:  NewExtractor(config.ExtractorConfig),
		chunker:    NewChunker(config.ChunkerConfig),
		summarizer: NewSummarizer(config.SummarizerConfig, client),
	}
}

// NewProcessorWithProgress creates a Processor with a progress callback.
//
// Example:
//
//	client := openai.NewClient("api-key")
//	processor := NewProcessorWithProgress(DefaultProcessorConfig(), client, func(stage string, progress float64, msg string) {
//	    fmt.Printf("[%s] %.0f%% - %s\n", stage, progress*100, msg)
//	})
func NewProcessorWithProgress(config ProcessorConfig, client *openai.Client, progress ProgressCallback) *Processor {
	p := NewProcessor(config, client)
	p.progress = progress
	return p
}

// SetProgressCallback sets or updates the progress callback.
func (p *Processor) SetProgressCallback(progress ProgressCallback) {
	p.progress = progress
}

// Process extracts text from a PDF file, chunks it, and generates an AI summary.
// This is the main entry point for PDF processing.
//
// Example:
//
//	result, err := processor.Process(ctx, "/path/to/document.pdf")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Summary)
func (p *Processor) Process(ctx context.Context, pdfPath string) (*ProcessResult, error) {
	if p.extractor == nil || p.chunker == nil || p.summarizer == nil {
		return nil, ErrProcessorNotConfigured
	}

	start := time.Now()
	result := &ProcessResult{}

	// Stage 1: Extract text from PDF
	p.reportProgress("extraction", 0.0, "Starting PDF text extraction...")
	extractStart := time.Now()

	extractionResult, err := p.extractor.Extract(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}
	result.ExtractionResult = extractionResult
	result.Stages.ExtractionTime = time.Since(extractStart)

	p.reportProgress("extraction", 1.0, fmt.Sprintf("Extracted %d pages, ~%d tokens",
		extractionResult.ExtractedPages, extractionResult.EstimatedTokens))

	// Stage 2: Chunk the extracted text
	p.reportProgress("chunking", 0.0, "Splitting text into chunks...")
	chunkStart := time.Now()

	chunkerResult := p.chunker.SplitIntoChunks(extractionResult.Text)
	result.ChunkerResult = chunkerResult
	result.Stages.ChunkingTime = time.Since(chunkStart)

	p.reportProgress("chunking", 1.0, fmt.Sprintf("Created %d chunks",
		chunkerResult.TotalChunks))

	// Stage 3: Generate AI summary
	p.reportProgress("summarizing", 0.0, fmt.Sprintf("Sending %d chunks to AI...",
		chunkerResult.TotalChunks))
	summaryStart := time.Now()

	summaryResult, err := p.summarizer.SummarizeChunkerResult(ctx, chunkerResult)
	if err != nil {
		return nil, fmt.Errorf("summarization failed: %w", err)
	}
	result.SummaryResult = summaryResult
	result.Summary = summaryResult.Content
	result.Stages.SummarizingTime = time.Since(summaryStart)

	p.reportProgress("summarizing", 1.0, "Summary complete")

	result.ProcessingTime = time.Since(start)
	return result, nil
}

// ProcessText processes pre-extracted text (skipping extraction stage).
// Use this when you already have text and just need chunking and summarization.
//
// Example:
//
//	text := "Your long document text here..."
//	result, err := processor.ProcessText(ctx, text)
func (p *Processor) ProcessText(ctx context.Context, text string) (*ProcessResult, error) {
	if p.chunker == nil || p.summarizer == nil {
		return nil, ErrProcessorNotConfigured
	}

	start := time.Now()
	result := &ProcessResult{}

	// Stage 1: Chunk the text
	p.reportProgress("chunking", 0.0, "Splitting text into chunks...")
	chunkStart := time.Now()

	chunkerResult := p.chunker.SplitIntoChunks(text)
	result.ChunkerResult = chunkerResult
	result.Stages.ChunkingTime = time.Since(chunkStart)

	p.reportProgress("chunking", 1.0, fmt.Sprintf("Created %d chunks",
		chunkerResult.TotalChunks))

	// Stage 2: Generate AI summary
	p.reportProgress("summarizing", 0.0, fmt.Sprintf("Sending %d chunks to AI...",
		chunkerResult.TotalChunks))
	summaryStart := time.Now()

	summaryResult, err := p.summarizer.SummarizeChunkerResult(ctx, chunkerResult)
	if err != nil {
		return nil, fmt.Errorf("summarization failed: %w", err)
	}
	result.SummaryResult = summaryResult
	result.Summary = summaryResult.Content
	result.Stages.SummarizingTime = time.Since(summaryStart)

	p.reportProgress("summarizing", 1.0, "Summary complete")

	result.ProcessingTime = time.Since(start)
	return result, nil
}

// reportProgress calls the progress callback if set.
func (p *Processor) reportProgress(stage string, progress float64, message string) {
	if p.progress != nil {
		p.progress(stage, progress, message)
	}
}

// ExtractOnly extracts text from a PDF without chunking or summarizing.
// Use this when you only need the extracted text.
func (p *Processor) ExtractOnly(pdfPath string) (*ExtractionResult, error) {
	if p.extractor == nil {
		return nil, ErrProcessorNotConfigured
	}
	return p.extractor.Extract(pdfPath)
}

// ChunkOnly chunks text without extracting or summarizing.
// Use this when you already have text and only need it chunked.
func (p *Processor) ChunkOnly(text string) *ChunkerResult {
	if p.chunker == nil {
		return nil
	}
	return p.chunker.SplitIntoChunks(text)
}

// ProcessPDF is a convenience function for processing a PDF with default configuration.
// It creates a processor with the given OpenAI client and processes the PDF.
//
// Example:
//
//	client := openai.NewClient("api-key")
//	result, err := ProcessPDF(ctx, client, "/path/to/document.pdf")
func ProcessPDF(ctx context.Context, client *openai.Client, pdfPath string) (*ProcessResult, error) {
	processor := NewProcessor(DefaultProcessorConfig(), client)
	return processor.Process(ctx, pdfPath)
}

// ProcessPDFWithModel processes a PDF using a specific AI model.
//
// Example:
//
//	client := openai.NewClient("api-key")
//	result, err := ProcessPDFWithModel(ctx, client, "/path/to/document.pdf", "gpt-4-turbo")
func ProcessPDFWithModel(ctx context.Context, client *openai.Client, pdfPath string, model string) (*ProcessResult, error) {
	config := DefaultProcessorConfig()
	config.SummarizerConfig.Model = model
	processor := NewProcessor(config, client)
	return processor.Process(ctx, pdfPath)
}
