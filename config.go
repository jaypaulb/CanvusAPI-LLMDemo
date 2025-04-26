package main

import "time"

type Config struct {
	MaxRetries        int
	RetryDelay        time.Duration
	AITimeout         time.Duration
	DownloadsDir      string
	MaxConcurrent     int
	ProcessingTimeout time.Duration
	MaxFileSize       int64
	GoogleVisionKey   string
	CanvusServer      string
	CanvasID          string
	CanvusAPIKey      string

	// OpenAI API Configuration
	OpenAIKey    string // Legacy field, kept for backward compatibility
	OpenAIAPIKey string // Main API key

	// Model Selection
	OpenAINoteModel   string // Model for note processing
	OpenAICanvasModel string // Model for canvas analysis
	OpenAIPDFModel    string // Model for PDF processing

	// LLM API Configuration
	BaseLLMURL  string // Default API endpoint for all LLM operations
	TextLLMURL  string // Optional override for text generation
	ImageLLMURL string // Optional override for image generation

	// Server Configuration
	CanvusServerURL      string
	CanvasName           string
	WebUIPassword        string
	Port                 int
	AllowSelfSignedCerts bool

	// Token limits for different operations
	PDFPrecisTokens       int64
	CanvasPrecisTokens    int64
	NoteResponseTokens    int64
	ImageAnalysisTokens   int64
	ErrorResponseTokens   int64
	PDFChunkSizeTokens    int64
	PDFMaxChunksTokens    int64
	PDFSummaryRatioTokens float64
}
