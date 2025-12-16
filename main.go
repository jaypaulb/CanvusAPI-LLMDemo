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
		os.Exit(core.ExitCodeError)
	}
	defer func() {
		if syncErr := logger.Sync(); syncErr != nil {
			fmt.Printf("Failed to sync logger: %v\n", syncErr)
		}
	}()

	// Run startup validation before heavy operations
	exitCode := runStartupValidation(logger, isDevelopment)
	if exitCode != core.ExitCodeSuccess {
		os.Exit(exitCode)
	}

	// Load configuration (safe to call after validation passes)
	config, err := core.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
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
		logger.Fatal("Failed to create downloads directory", zap.Error(err))
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

// runStartupValidation performs comprehensive startup validation.
// This includes configuration validation and optionally model availability checks.
//
// Returns the appropriate exit code:
//   - ExitCodeSuccess (0) if all validations pass
//   - ExitCodeError (1) if any validation fails
func runStartupValidation(logger *logging.Logger, isDevelopment bool) int {
	logger.Info("Starting startup validation...")

	// Determine if self-signed certs are allowed
	allowSelfSigned := os.Getenv("ALLOW_SELF_SIGNED_CERTS") == "true"

	// Run configuration validation suite
	suite := core.NewValidationSuite().
		WithAllowSelfSignedCerts(allowSelfSigned).
		WithShowProgress(true)

	result := suite.Validate()

	if !result.Success {
		logger.Error("Configuration validation failed",
			zap.Int("passed", result.PassedSteps),
			zap.Int("failed", result.FailedSteps),
			zap.Duration("duration", result.Duration),
		)

		// Log individual failures for debugging
		for _, step := range result.Steps {
			if step.Status == core.StepFailed {
				logger.Error("Validation step failed",
					zap.String("step", step.Name),
					zap.String("message", step.Message),
					zap.Error(step.Error),
				)
			}
		}

		return core.ExitCodeError
	}

	logger.Info("Configuration validation passed",
		zap.Int("checks_passed", result.PassedSteps),
		zap.Duration("duration", result.Duration),
	)

	// Model availability check (optional - only if model management is enabled)
	if shouldCheckModels() {
		if err := ensureModelsAvailable(logger); err != nil {
			logger.Error("Model availability check failed", zap.Error(err))
			return core.ExitCodeError
		}
	}

	logger.Info("Startup validation complete")
	return core.ExitCodeSuccess
}

// shouldCheckModels determines if model availability checks should be performed.
// Model checks are enabled when LOCAL_MODEL_DIR is configured and not empty.
func shouldCheckModels() bool {
	modelDir := os.Getenv("LOCAL_MODEL_DIR")
	return modelDir != ""
}

// ensureModelsAvailable checks that required AI models are available.
// If models are missing, attempts to download them.
func ensureModelsAvailable(logger *logging.Logger) error {
	modelDir := os.Getenv("LOCAL_MODEL_DIR")
	if modelDir == "" {
		// Model management not configured - skip
		return nil
	}

	logger.Info("Checking model availability...", zap.String("model_dir", modelDir))

	// Create model manager with default settings
	httpClient := core.GetHTTPClient(&core.Config{
		AllowSelfSignedCerts: os.Getenv("ALLOW_SELF_SIGNED_CERTS") == "true",
	}, 0) // No timeout for large downloads

	modelManager := core.NewModelManager(modelDir, httpClient)

	// Create context for model operations (can be cancelled via signal)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt during model download
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Warn("Model check interrupted, cancelling...")
		cancel()
	}()

	// Check which models to ensure based on configuration
	modelsToCheck := getRequiredModels()

	for _, modelName := range modelsToCheck {
		logger.Info("Ensuring model available", zap.String("model", modelName))

		if err := modelManager.EnsureModelAvailable(ctx, modelName); err != nil {
			return fmt.Errorf("model %q not available: %w", modelName, err)
		}

		modelPath, _ := modelManager.GetModelPath(modelName)
		logger.Info("Model ready", zap.String("model", modelName), zap.String("path", modelPath))
	}

	// Reset signal handling (will be set up again in main)
	signal.Reset(os.Interrupt, syscall.SIGTERM)

	return nil
}

// getRequiredModels returns the list of model names that should be checked.
// This reads from REQUIRED_MODELS environment variable (comma-separated)
// or defaults to checking the text model if LOCAL_MODEL_DIR is set.
func getRequiredModels() []string {
	// Check for explicit model list
	modelsEnv := os.Getenv("REQUIRED_MODELS")
	if modelsEnv != "" {
		var models []string
		for _, m := range splitAndTrim(modelsEnv, ",") {
			if m != "" {
				models = append(models, m)
			}
		}
		if len(models) > 0 {
			return models
		}
	}

	// Default: check text model if model dir is configured
	if os.Getenv("LOCAL_MODEL_DIR") != "" {
		return []string{core.DefaultTextModel.Name}
	}

	return nil
}

// splitAndTrim splits a string by separator and trims whitespace from each part.
func splitAndTrim(s string, sep string) []string {
	if s == "" {
		return nil
	}

	parts := make([]string, 0)
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// splitString splits a string by separator (simple implementation).
func splitString(s string, sep string) []string {
	var result []string
	current := ""
	sepLen := len(sep)

	for i := 0; i < len(s); i++ {
		if i+sepLen <= len(s) && s[i:i+sepLen] == sep {
			result = append(result, current)
			current = ""
			i += sepLen - 1
		} else {
			current += string(s[i])
		}
	}
	result = append(result, current)
	return result
}

// trimSpace removes leading and trailing whitespace from a string.
func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && isWhitespace(s[start]) {
		start++
	}
	for end > start && isWhitespace(s[end-1]) {
		end--
	}

	return s[start:end]
}

// isWhitespace returns true if the byte is a whitespace character.
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
