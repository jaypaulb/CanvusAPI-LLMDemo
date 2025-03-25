package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go_backend/canvusapi"

	"github.com/joho/godotenv"
)

// loadConfig loads and validates configuration from environment
func loadConfig() (*Config, error) {
	envPath := ".env"
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("❌ Error loading .env file from %s: %v", envPath, err)
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	log.Println("✅ Successfully loaded .env file")

	// Parse token limits with defaults
	pdfPrecisTokens := parseInt64Env("OPENAI_PDF_PRECIS_TOKENS", 1000)
	canvasPrecisTokens := parseInt64Env("OPENAI_CANVAS_PRECIS_TOKENS", 600)
	noteResponseTokens := parseInt64Env("OPENAI_NOTE_RESPONSE_TOKENS", 400)
	imageAnalysisTokens := parseInt64Env("OPENAI_IMAGE_ANALYSIS_TOKENS", 16384)
	errorResponseTokens := parseInt64Env("OPENAI_ERROR_RESPONSE_TOKENS", 200)
	pdfChunkSizeTokens := parseInt64Env("OPENAI_PDF_CHUNK_SIZE_TOKENS", 20000)
	pdfMaxChunksTokens := parseInt64Env("OPENAI_PDF_MAX_CHUNKS_TOKENS", 10)
	pdfSummaryRatioTokens := parseFloat64Env("OPENAI_PDF_SUMMARY_RATIO", 0.3)

	config := &Config{
		MaxRetries:        3,
		RetryDelay:        time.Second,
		AITimeout:         30 * time.Second,
		DownloadsDir:      "./downloads",
		MaxConcurrent:     5,
		ProcessingTimeout: 5 * time.Minute,
		MaxFileSize:       50 * 1024 * 1024,
		GoogleVisionKey:   os.Getenv("GOOGLE_VISION_API_KEY"),
		CanvusServer:      os.Getenv("CANVUS_SERVER"),
		CanvasID:          os.Getenv("CANVAS_ID"),
		CanvusAPIKey:      os.Getenv("CANVUS_API_KEY"),
		OpenAIKey:         os.Getenv("OPENAI_API_KEY"),
		OpenAINoteModel:   os.Getenv("OPENAI_NOTE_MODEL"),
		OpenAICanvasModel: os.Getenv("OPENAI_CANVAS_MODEL"),
		OpenAIPDFModel:    os.Getenv("OPENAI_PDF_MODEL"),
		// Token limits
		PDFPrecisTokens:       pdfPrecisTokens,
		CanvasPrecisTokens:    canvasPrecisTokens,
		NoteResponseTokens:    noteResponseTokens,
		ImageAnalysisTokens:   imageAnalysisTokens,
		ErrorResponseTokens:   errorResponseTokens,
		PDFChunkSizeTokens:    pdfChunkSizeTokens,
		PDFMaxChunksTokens:    pdfMaxChunksTokens,
		PDFSummaryRatioTokens: pdfSummaryRatioTokens,
	}

	// Validate required fields with detailed logging
	missingVars := []string{}
	if config.CanvusServer == "" {
		log.Println("❌ Missing environment variable: CANVUS_SERVER")
		missingVars = append(missingVars, "CANVUS_SERVER")
	}
	if config.CanvasID == "" {
		log.Println("❌ Missing environment variable: CANVAS_ID")
		missingVars = append(missingVars, "CANVAS_ID")
	}
	if config.CanvusAPIKey == "" {
		log.Println("❌ Missing environment variable: CANVUS_API_KEY")
		missingVars = append(missingVars, "CANVUS_API_KEY")
	}
	if config.OpenAIKey == "" {
		log.Println("❌ Missing environment variable: OPENAI_API_KEY")
		missingVars = append(missingVars, "OPENAI_API_KEY")
	}
	if config.GoogleVisionKey == "" {
		log.Println("❌ Missing environment variable: GOOGLE_VISION_API_KEY")
		missingVars = append(missingVars, "GOOGLE_VISION_API_KEY")
	}

	if len(missingVars) > 0 {
		errorMsg := fmt.Sprintf("❌ Missing required environment variables: %v", missingVars)
		log.Println(errorMsg)
		return nil, fmt.Errorf(errorMsg)
	}

	log.Println("✅ All required environment variables found")
	return config, nil
}

// Helper functions to parse environment variables
func parseInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func parseFloat64Env(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// setupLogging initializes application logging
func setupLogging() (*os.File, error) {
	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	return logFile, nil
}

func main() {
	// Load environment variables from .env in the project root
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("❌ Failed to load .env file from .env: %v\n", err)
		fmt.Printf("❌ Failed to load .env file from .env: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	logFile, err := setupLogging()
	if err != nil {
		fmt.Printf("Failed to setup logging: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Printf("❌ Failed to load configuration: %v", err)
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		fmt.Println("👉 Please ensure your .env file exists and contains all required variables")
		os.Exit(1)
	}

	// Create downloads directory
	if err := os.MkdirAll(cfg.DownloadsDir, 0755); err != nil {
		log.Fatalf("Failed to create downloads directory: %v", err)
	}

	// Initialize Canvus client
	client := canvusapi.NewClient(
		cfg.CanvusServer,
		cfg.CanvasID,
		cfg.CanvusAPIKey,
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start monitoring with context
	monitor := NewMonitor(client, cfg)
	go monitor.Start(ctx)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("AI Handler started successfully")
	fmt.Println("Press Ctrl+C to exit")

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received shutdown signal: %v", sig)
	fmt.Println("\nInitiating graceful shutdown...")

	// Cancel context to notify all goroutines
	cancel()

	// Allow time for cleanup
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Wait for cleanup or timeout
	select {
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout reached")
	case <-monitor.Done():
		log.Println("Clean shutdown completed")
	}

	// Call handlers.Cleanup() instead of directly dealing with downloads
	Cleanup()
}
