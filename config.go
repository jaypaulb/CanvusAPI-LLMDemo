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
	OpenAIKey         string
	OpenAINoteModel   string
	OpenAICanvasModel string
	OpenAIPDFModel    string
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
