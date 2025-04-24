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

// Helper function to get environment variable with default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Helper function to parse integer environment variable with default value
func parseIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Helper function to parse int64 environment variable with default value
func parseInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Helper function to parse float64 environment variable with default value
func parseFloat64Env(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// loadConfig loads and validates configuration from environment
func loadConfig(envPath string) (*Config, error) {
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Load OpenAI API base URL with default value
	openAIAPIBaseURL := getEnvOrDefault("OPENAI_API_BASE_URL", "https://api.openai.com/v1")

	// Load OpenAI model configurations with default values
	openAINoteModel := getEnvOrDefault("OPENAI_NOTE_MODEL", "gpt-3.5-turbo")
	openAICanvasModel := getEnvOrDefault("OPENAI_CANVAS_MODEL", "gpt-4")
	openAIPDFModel := getEnvOrDefault("OPENAI_PDF_MODEL", "gpt-4")

	// Load token limits with default values
	pdfPrecisTokens := parseInt64Env("OPENAI_PDF_PRECIS_TOKENS", 1000)
	canvasPrecisTokens := parseInt64Env("OPENAI_CANVAS_PRECIS_TOKENS", 600)
	noteResponseTokens := parseInt64Env("OPENAI_NOTE_RESPONSE_TOKENS", 400)
	imageAnalysisTokens := parseInt64Env("OPENAI_IMAGE_ANALYSIS_TOKENS", 16384)
	errorResponseTokens := parseInt64Env("OPENAI_ERROR_RESPONSE_TOKENS", 200)
	pdfChunkSizeTokens := parseInt64Env("OPENAI_PDF_CHUNK_SIZE_TOKENS", 20000)
	pdfMaxChunksTokens := parseInt64Env("OPENAI_PDF_MAX_CHUNKS_TOKENS", 10)
	pdfSummaryRatio := parseFloat64Env("OPENAI_PDF_SUMMARY_RATIO", 0.3)

	// Load other configurations
	port := parseIntEnv("PORT", 3000)
	maxConcurrent := parseIntEnv("MAX_CONCURRENT", 5)
	processingTimeout := time.Duration(parseIntEnv("PROCESSING_TIMEOUT", 300)) * time.Second
	maxFileSize := parseInt64Env("MAX_FILE_SIZE", 52428800) // 50MB
	downloadsDir := getEnvOrDefault("DOWNLOADS_DIR", "./downloads")
	allowSelfSignedCerts := getEnvOrDefault("ALLOW_SELF_SIGNED_CERTS", "false") == "true"

	// Validate required environment variables
	requiredVars := []string{
		"CANVUS_SERVER",
		"CANVAS_NAME",
		"CANVAS_ID",
		"OPENAI_API_KEY",
		"CANVUS_API_KEY",
		"WEBUI_PWD",
	}

	var missingVars []string
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			missingVars = append(missingVars, v)
		}
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	return &Config{
		CanvusServerURL:      os.Getenv("CANVUS_SERVER"),
		CanvasName:           os.Getenv("CANVAS_NAME"),
		CanvasID:             os.Getenv("CANVAS_ID"),
		OpenAIAPIKey:         os.Getenv("OPENAI_API_KEY"),
		OpenAIAPIBaseURL:     openAIAPIBaseURL,
		OpenAINoteModel:      openAINoteModel,
		OpenAICanvasModel:    openAICanvasModel,
		OpenAIPDFModel:       openAIPDFModel,
		GoogleVisionKey:      os.Getenv("GOOGLE_VISION_API_KEY"),
		CanvusAPIKey:         os.Getenv("CANVUS_API_KEY"),
		WebUIPassword:        os.Getenv("WEBUI_PWD"),
		Port:                 port,
		MaxConcurrent:        maxConcurrent,
		ProcessingTimeout:    processingTimeout,
		MaxFileSize:          maxFileSize,
		DownloadsDir:         downloadsDir,
		AllowSelfSignedCerts: allowSelfSignedCerts,
		// Token limits
		PDFPrecisTokens:       pdfPrecisTokens,
		CanvasPrecisTokens:    canvasPrecisTokens,
		NoteResponseTokens:    noteResponseTokens,
		ImageAnalysisTokens:   imageAnalysisTokens,
		ErrorResponseTokens:   errorResponseTokens,
		PDFChunkSizeTokens:    pdfChunkSizeTokens,
		PDFMaxChunksTokens:    pdfMaxChunksTokens,
		PDFSummaryRatioTokens: pdfSummaryRatio,
	}, nil
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
	cfg, err := loadConfig(".env")
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
		cfg.CanvusServerURL,
		cfg.CanvasID,
		cfg.CanvusAPIKey,
		cfg.AllowSelfSignedCerts,
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
