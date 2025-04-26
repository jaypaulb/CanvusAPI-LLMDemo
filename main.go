package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go_backend/canvusapi"
	"go_backend/core"

	"github.com/joho/godotenv"
)

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
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Load configuration
	config, err := core.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logging
	logFile, err := setupLogging()
	if err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}
	defer logFile.Close()

	// Log configuration values
	fmt.Printf("📝 Configuration loaded:\n")
	fmt.Printf("  Server: %s\n", config.CanvusServerURL)
	fmt.Printf("  Canvas: %s (ID: %s)\n", config.CanvasName, config.CanvasID)
	fmt.Printf("  Max Retries: %d\n", config.MaxRetries)
	fmt.Printf("  Retry Delay: %v\n", config.RetryDelay)
	fmt.Printf("  AI Timeout: %v\n", config.AITimeout)
	fmt.Printf("  Processing Timeout: %v\n", config.ProcessingTimeout)
	fmt.Printf("  Max Concurrent: %d\n", config.MaxConcurrent)
	fmt.Printf("  Downloads Directory: %s\n", config.DownloadsDir)
	fmt.Printf("  Allow Self-Signed Certs: %v\n", config.AllowSelfSignedCerts)

	// Create downloads directory
	if err := os.MkdirAll(config.DownloadsDir, 0755); err != nil {
		log.Fatalf("Failed to create downloads directory: %v", err)
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
		fmt.Println("\n🛑 Received interrupt signal. Shutting down...")
		cancel()
	}()

	// Start monitoring with context
	monitor := NewMonitor(client, config)
	go monitor.Start(ctx)

	// Block until context is cancelled
	<-ctx.Done()
	fmt.Println("👋 Goodbye!")
}
