package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go_backend/canvusapi"
	"go_backend/core"
	"go_backend/logging"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// Use fmt here since logger isn't initialized yet
		fmt.Printf("Warning: .env file not found: %v\n", err)
	}

	// Determine if running in development mode
	isDevelopment := os.Getenv("DEV_MODE") == "true"

	// Initialize structured logger early
	logger, err := logging.NewLogger(isDevelopment, "app.log")
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if syncErr := logger.Sync(); syncErr != nil {
			fmt.Printf("Failed to sync logger: %v\n", syncErr)
		}
	}()

	// Load configuration
	config, err := core.LoadConfig()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Log configuration values
	logger.Info("Configuration loaded",
		zap.String("server", config.CanvusServerURL),
		zap.String("canvas", config.CanvasName),
		zap.String("canvas_id", config.CanvasID),
		zap.Int("max_retries", config.MaxRetries),
		zap.Duration("retry_delay", config.RetryDelay),
		zap.Duration("ai_timeout", config.AITimeout),
		zap.Duration("processing_timeout", config.ProcessingTimeout),
		zap.Int("max_concurrent", config.MaxConcurrent),
		zap.String("downloads_dir", config.DownloadsDir),
		zap.Bool("allow_self_signed_certs", config.AllowSelfSignedCerts),
		zap.Bool("dev_mode", isDevelopment),
	)

	// Create downloads directory
	if err := os.MkdirAll(config.DownloadsDir, 0755); err != nil {
		logger.Fatalf("Failed to create downloads directory: %v", err)
	}

	// Initialize Canvus client
	client := canvusapi.NewClient(
		config.CanvusServerURL,
		config.CanvasID,
		config.CanvusAPIKey,
		config.AllowSelfSignedCerts,
	)

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Received interrupt signal. Shutting down...")
		cancel()
	}()

	// Start monitoring with context
	monitor := NewMonitor(client, config, logger)
	go monitor.Start(ctx)

	// Block until context is cancelled
	<-ctx.Done()
	logger.Info("Goodbye!")
}
