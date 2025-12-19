package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go_backend/canvusapi"
	"go_backend/core"
	"go_backend/core/modelmanager"
	"go_backend/core/validation"
	"go_backend/db"
	"go_backend/imagegen"
	"go_backend/llamaruntime"
	"go_backend/logging"
	"go_backend/metrics"
	"go_backend/sdruntime"
	"go_backend/shutdown"
	"go_backend/webui"
	"go_backend/webui/auth"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Default timeouts for the HTTP server
const (
	// DefaultReadTimeout is the maximum duration for reading the entire request.
	DefaultReadTimeout = 15 * time.Second

	// DefaultWriteTimeout is the maximum duration before timing out writes of the response.
	DefaultWriteTimeout = 15 * time.Second

	// DefaultIdleTimeout is the maximum amount of time to wait for the next request.
	DefaultIdleTimeout = 60 * time.Second

	// DefaultShutdownTimeout is the maximum time to wait for server shutdown.
	DefaultShutdownTimeout = 10 * time.Second
)

func main() {
	// Track which signal caused shutdown (if any)
	var shutdownSignal os.Signal

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

	// Run startup validation before heavy operations
	exitCode := runStartupValidation(logger, isDevelopment)
	if exitCode != core.ExitCodeSuccess {
		// Sync logger before exit
		if syncErr := logger.Sync(); syncErr != nil {
			fmt.Printf("Failed to sync logger: %v\n", syncErr)
		}
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
		zap.Int("webui_port", config.Port),
	)

	// Create downloads directory
	if err := os.MkdirAll(config.DownloadsDir, 0755); err != nil {
		logger.Fatal("Failed to create downloads directory", zap.Error(err))
	}

	// Initialize database
	// Determine database path from environment or use default in user's home
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			logger.Fatal("Failed to determine home directory", zap.Error(err))
		}
		dbPath = filepath.Join(homeDir, ".canvuslocallm", "data.db")
	}

	logger.Info("Initializing database", zap.String("path", dbPath))
	database, err := db.NewDatabase(dbPath)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Create repository first (without async writer)
	tempRepo := db.NewRepository(database, nil)

	// Create and start async writer with handler from repository
	asyncWriter := db.NewAsyncWriter(tempRepo.CreateAsyncWriteHandler())
	asyncWriter.Start()

	// Create final repository with async writer attached
	repository := db.NewRepository(database, asyncWriter)
	logger.Info("Database and repository initialized")

	// Initialize Canvus client
	client := canvusapi.NewClient(
		config.CanvusServerURL,
		config.CanvasID,
		config.CanvusAPIKey,
		config.AllowSelfSignedCerts,
	)

	// Initialize shutdown manager with 60-second timeout
	shutdownManager := shutdown.NewManager(logger.Zap(), shutdown.WithTimeout(60*time.Second))

	// Register logger sync as highest priority (runs first during shutdown)
	shutdownManager.Register("logger-sync", 5, func(ctx context.Context) error {
		logger.Info("Syncing logger...")
		if syncErr := logger.Sync(); syncErr != nil {
			logger.Warn("Failed to sync logger during shutdown", zap.Error(syncErr))
			return syncErr
		}
		logger.Info("Logger synced")
		return nil
	})

	// Register async writer shutdown (priority 10 - before database)
	shutdownManager.Register("async-writer", 10, func(ctx context.Context) error {
		logger.Info("Stopping async writer...")
		asyncWriter.Stop()
		logger.Info("Async writer stopped")
		return nil
	})

	// Register database close (priority 15 - after async writer)
	shutdownManager.Register("database", 15, func(ctx context.Context) error {
		logger.Info("Closing database...")
		if closeErr := database.Close(); closeErr != nil {
			logger.Error("Failed to close database", zap.Error(closeErr))
			return closeErr
		}
		logger.Info("Database closed")
		return nil
	})

	// Initialize SD runtime and imagegen processor (optional)
	var sdPool *sdruntime.ContextPool
	var imageProcessor *imagegen.Processor

	sdPool, imageProcessor, err = initializeSDRuntime(logger, client, config)
	if err != nil {
		// Log the error but continue - SD is optional
		logger.Warn("SD runtime initialization failed, image generation disabled",
			zap.Error(err))
	} else if sdPool != nil {
		// Register SD pool shutdown (priority 30 - resource cleanup)
		shutdownManager.Register("sd-pool", 30, func(ctx context.Context) error {
			logger.Info("Shutting down SD context pool...")
			if closeErr := sdPool.Close(); closeErr != nil {
				logger.Error("Failed to close SD pool", zap.Error(closeErr))
				return closeErr
			}
			logger.Info("SD context pool closed")
			return nil
		})
	}

	// Initialize llamaruntime (LLM inference) - optional
	var llamaClient *llamaruntime.Client
	var llamaHealthChecker *llamaruntime.HealthChecker
	var llamaGPUMonitor *llamaruntime.GPUMonitor

	llamaClient, llamaHealthChecker, llamaGPUMonitor, err = initializeLlamaRuntime(logger, shutdownManager.Context())
	if err != nil {
		// Log the error but continue - llamaruntime is optional
		logger.Warn("llamaruntime initialization failed, local LLM inference disabled",
			zap.Error(err))
	} else if llamaClient != nil {
		// Register llamaruntime shutdown (priority 35 - after SD pool)
		shutdownManager.Register("llamaruntime", 35, func(ctx context.Context) error {
			logger.Info("Shutting down llamaruntime...")

			// Stop health checker first
			if llamaHealthChecker != nil {
				llamaHealthChecker.Stop()
				logger.Info("llamaruntime health checker stopped")
			}

			// Stop GPU monitor
			if llamaGPUMonitor != nil {
				llamaGPUMonitor.Stop()
				logger.Info("llamaruntime GPU monitor stopped")
			}

			// Close the client
			if closeErr := llamaClient.Close(); closeErr != nil {
				logger.Error("Failed to close llamaruntime client", zap.Error(closeErr))
				return closeErr
			}
			logger.Info("llamaruntime client closed")
			return nil
		})
	}

	// Initialize MetricsStore for dashboard metrics
	metricsConfig := metrics.StoreConfig{
		TaskHistoryCapacity: 100,
		Version:             "1.0.0",
	}
	metricsStore := metrics.NewMetricsStore(metricsConfig, time.Now())
	logger.Info("MetricsStore initialized")

	// Initialize GPUCollector for GPU metrics
	gpuConfig := metrics.DefaultGPUCollectorConfig()
	gpuCollector := metrics.NewGPUCollector(gpuConfig, func(gpuMetrics metrics.GPUMetrics) {
		// Update metrics store with GPU data
		metricsStore.UpdateGPUMetrics(gpuMetrics)
	})
	logger.Info("GPUCollector initialized",
		zap.Duration("interval", gpuConfig.CollectionInterval),
		zap.Int("history_size", gpuConfig.HistorySize),
	)

	// Start GPU collector goroutine (uses internal context)
	gpuCollector.Start()

	// Register GPU collector shutdown (priority 25 - after web server)
	shutdownManager.Register("gpu-collector", 25, func(ctx context.Context) error {
		logger.Info("Stopping GPU collector...")
		gpuCollector.Stop()
		logger.Info("GPU collector stopped")
		return nil
	})

	// Start monitoring with context from shutdown manager
	monitor := NewMonitor(client, config, logger, repository)

	// Wire metrics store into monitor for task recording
	monitor.SetMetricsStore(metricsStore)

	// Wire in the imagegen processor if available
	if imageProcessor != nil {
		monitor.SetImagegenProcessor(imageProcessor)
		logger.Info("Image generation enabled via SD runtime")
	}

	// Wire in the llamaruntime client if available
	if llamaClient != nil {
		monitor.SetLlamaClient(llamaClient)
		logger.Info("Local LLM inference enabled via llamaruntime")
	}

	go monitor.Start(shutdownManager.Context())

	// Initialize WebUIServer with the real components
	serverConfig := webui.ServerConfig{
		Port:            config.Port,
		Host:            "", // Bind to all interfaces
		ReadTimeout:     DefaultReadTimeout,
		WriteTimeout:    DefaultWriteTimeout,
		IdleTimeout:     DefaultIdleTimeout,
		ShutdownTimeout: DefaultShutdownTimeout,
		StaticConfig:    webui.DefaultStaticAssetConfig(),
		LogSkipPaths:    []string{"/health", "/api/status"},
		VersionInfo: webui.VersionInfo{
			Version: "1.0.0",
		},
	}

	// Create auth provider
	authProvider, err := createAuthProvider(config, logger)
	if err != nil {
		logger.Fatal("Failed to create auth provider", zap.Error(err))
	}

	// Create WebUIServer with all dependencies wired together
	webServer, err := webui.NewServer(
		serverConfig,
		metricsStore,
		gpuCollector,
		authProvider,
		logger.Zap(),
	)
	if err != nil {
		logger.Fatal("Failed to setup web server", zap.Error(err))
	}
	logger.Info("WebUIServer initialized",
		zap.Int("port", config.Port),
		zap.Bool("auth_enabled", authProvider != nil),
	)

	// Wire WebSocket broadcaster into monitor for real-time task updates
	if broadcaster := webServer.GetBroadcaster(); broadcaster != nil {
		monitor.SetTaskBroadcaster(broadcaster)
		// Also wire into handlers.go for direct task recording
		SetDashboardMetrics(metricsStore, broadcaster)
		logger.Info("Task broadcaster wired for real-time dashboard updates")
	}

	// Register WebUI server shutdown (priority 20 - service cleanup)
	shutdownManager.Register("webui-server", 20, func(ctx context.Context) error {
		logger.Info("Shutting down WebUI server...")
		if err := webServer.Shutdown(ctx); err != nil {
			logger.Error("WebUI server shutdown error", zap.Error(err))
			return err
		}
		logger.Info("WebUI server shutdown complete")
		return nil
	})

	// Register temp file cleanup (priority 45 - final cleanup)
	shutdownManager.Register("cleanup-downloads", 45, shutdown.CleanupDownloads(logger.Zap(), config.DownloadsDir))

	// Start shutdown manager (signal handling)
	shutdownManager.Start()

	// Capture which signal triggered shutdown for exit code determination
	// We need to intercept signals before the manager to capture them
	sigChan := make(chan os.Signal, 1)
	signalNotify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		shutdownSignal = sig
		logger.Info("Received shutdown signal",
			zap.String("signal", sig.String()),
		)
	}()

	// Start web server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		logger.Info("Starting WebUI server",
			zap.String("addr", webServer.Addr()),
			zap.String("login_url", fmt.Sprintf("http://localhost:%d/login", config.Port)),
		)
		if err := webServer.Start(shutdownManager.Context()); err != nil && err != http.ErrServerClosed {
			serverErrChan <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-shutdownManager.Context().Done():
		// Normal shutdown via signal
		logger.Info("Shutdown initiated")
	case err := <-serverErrChan:
		// Server error - set exit code to error
		logger.Error("Web server error", zap.Error(err))
		exitCode = core.ExitCodeError
	}

	// Execute graceful shutdown sequence
	if shutdownErr := shutdownManager.Shutdown(); shutdownErr != nil {
		logger.Error("Shutdown completed with errors", zap.Error(shutdownErr))
		// Only override exit code if we don't already have an error
		if exitCode == core.ExitCodeSuccess {
			exitCode = core.ExitCodeError
		}
	}

	// Determine final exit code based on shutdown signal
	// Only set signal-based exit codes if no error occurred
	if exitCode == core.ExitCodeSuccess && shutdownSignal != nil {
		switch shutdownSignal {
		case os.Interrupt:
			exitCode = core.ExitCodeSIGINT
		case syscall.SIGTERM:
			exitCode = core.ExitCodeSIGTERM
		}
	}

	logger.Info("Goodbye!",
		zap.Int("exit_code", exitCode),
		zap.String("exit_reason", core.ExitCodeName(exitCode)),
	)

	// Final logger sync before exit
	if syncErr := logger.Sync(); syncErr != nil {
		fmt.Printf("Failed to sync logger: %v\n", syncErr)
	}

	os.Exit(exitCode)
}

// signalNotify is a wrapper around signal.Notify for easier testing.
// It can be replaced with a mock in tests.
var signalNotify = func(c chan<- os.Signal, sig ...os.Signal) {
	// Use a different channel from shutdown manager's internal channel
	// to capture the signal type before manager handles it
	signal.Notify(c, sig...)
}

// authProviderAdapter adapts auth.AuthMiddleware to implement webui.AuthProvider.
// This allows the WebUIServer to remain decoupled from the auth package.
type authProviderAdapter struct {
	middleware *auth.AuthMiddleware
}

// Middleware wraps an http.Handler with authentication.
func (a *authProviderAdapter) Middleware(next http.Handler) http.Handler {
	return a.middleware.Middleware(next)
}

// MiddlewareFunc wraps an http.HandlerFunc with authentication.
func (a *authProviderAdapter) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return a.middleware.RequireAuth(next)
}

// LoginHandler returns a handler for the login page.
func (a *authProviderAdapter) LoginHandler() http.HandlerFunc {
	return auth.LoginHandler(a.middleware)
}

// LogoutHandler returns a handler for logout.
func (a *authProviderAdapter) LogoutHandler() http.HandlerFunc {
	return auth.LogoutHandler(a.middleware)
}

// createAuthProvider creates an authentication provider from the configuration.
// Returns nil if no password is configured (unauthenticated mode).
func createAuthProvider(config *core.Config, logger *logging.Logger) (webui.AuthProvider, error) {
	// Check if authentication is configured
	if config.WebUIPassword == "" {
		logger.Warn("WebUI password not configured, running in unauthenticated mode")
		return nil, nil
	}

	// Create authentication middleware
	authMiddleware, err := auth.NewAuthMiddleware(config.WebUIPassword, logger.Zap())
	if err != nil {
		return nil, fmt.Errorf("failed to create auth middleware: %w", err)
	}

	logger.Info("Auth middleware initialized")
	return &authProviderAdapter{middleware: authMiddleware}, nil
}

// initializeSDRuntime initializes the Stable Diffusion runtime and image processor.
// Returns (nil, nil, nil) if SD is not configured (no model path).
// Returns (nil, nil, error) if SD is configured but initialization fails.
// Returns (pool, processor, nil) on success.
//
// This is a molecule that composes:
//   - sdruntime.LoadSDConfig (atom)
//   - sdruntime.VerifyModelChecksum (molecule)
//   - sdruntime.NewContextPool (molecule)
//   - imagegen.NewProcessor (organism)
func initializeSDRuntime(logger *logging.Logger, client *canvusapi.Client, config *core.Config) (*sdruntime.ContextPool, *imagegen.Processor, error) {
	// Load SD configuration
	sdConfig := sdruntime.LoadSDConfig()

	// Check if SD is configured
	if sdConfig.ModelPath == "" {
		logger.Info("SD model path not configured, image generation disabled")
		return nil, nil, nil
	}

	logger.Info("Initializing SD runtime",
		zap.String("model_path", sdConfig.ModelPath),
		zap.Int("max_concurrent", sdConfig.MaxConcurrent),
		zap.Int("image_size", sdConfig.ImageSize),
		zap.Int("inference_steps", sdConfig.InferenceSteps),
		zap.Float64("guidance_scale", sdConfig.GuidanceScale),
		zap.Duration("timeout", sdConfig.Timeout),
	)

	// Verify model exists
	if _, err := os.Stat(sdConfig.ModelPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("SD model file not found: %s", sdConfig.ModelPath)
		}
		return nil, nil, fmt.Errorf("failed to access SD model file: %w", err)
	}

	// Verify model integrity (optional - only if checksum is registered)
	if err := sdruntime.VerifyModelChecksum(sdConfig.ModelPath); err != nil {
		if errors.Is(err, sdruntime.ErrModelCorrupted) {
			return nil, nil, fmt.Errorf("SD model file corrupted: %w", err)
		}
		// Log warning for other errors but continue
		logger.Warn("SD model checksum verification skipped",
			zap.Error(err))
	} else {
		logger.Info("SD model checksum verified")
	}

	// Create context pool
	pool, err := sdruntime.NewContextPool(sdConfig.MaxConcurrent, sdConfig.ModelPath)
	if err != nil {
		if errors.Is(err, sdruntime.ErrCUDANotAvailable) {
			return nil, nil, fmt.Errorf("CUDA not available for SD: %w", err)
		}
		return nil, nil, fmt.Errorf("failed to create SD context pool: %w", err)
	}

	logger.Info("SD context pool created",
		zap.Int("max_size", pool.MaxSize()))

	// Create imagegen processor
	processorConfig := imagegen.ProcessorConfig{
		DownloadsDir:    config.DownloadsDir,
		DefaultWidth:    sdConfig.ImageSize,
		DefaultHeight:   sdConfig.ImageSize,
		DefaultSteps:    sdConfig.InferenceSteps,
		DefaultCFGScale: sdConfig.GuidanceScale,
		PlacementConfig: imagegen.DefaultPlacementConfig(),
		ProcessingNote:  imagegen.DefaultProcessingNoteConfig(),
	}

	processor, err := imagegen.NewProcessor(pool, client, logger, processorConfig)
	if err != nil {
		// Clean up the pool if processor creation fails
		pool.Close()
		return nil, nil, fmt.Errorf("failed to create image processor: %w", err)
	}

	logger.Info("Image generation processor initialized")

	return pool, processor, nil
}

// initializeLlamaRuntime initializes the llamaruntime LLM client.
// Returns (nil, nil, nil, nil) if llamaruntime is not configured (no model path).
// Returns (nil, nil, nil, error) if llamaruntime is configured but initialization fails.
// Returns (client, healthChecker, gpuMonitor, nil) on success.
//
// This is a molecule that composes:
//   - llamaruntime.NewModelLoader (molecule)
//   - llamaruntime.NewHealthChecker (molecule)
//   - llamaruntime.NewGPUMonitor (molecule)
func initializeLlamaRuntime(logger *logging.Logger, ctx context.Context) (*llamaruntime.Client, *llamaruntime.HealthChecker, *llamaruntime.GPUMonitor, error) {
	// Check if llamaruntime is configured
	modelPath := os.Getenv("LLAMA_MODEL_PATH")
	if modelPath == "" {
		logger.Info("LLAMA_MODEL_PATH not configured, local LLM inference disabled")
		return nil, nil, nil, nil
	}

	logger.Info("Initializing llamaruntime",
		zap.String("model_path", modelPath),
	)

	// Configure model loader
	loaderConfig := llamaruntime.DefaultModelLoaderConfig()
	loaderConfig.ModelPath = modelPath
	loaderConfig.ModelsDir = os.Getenv("LLAMA_MODELS_DIR")
	if loaderConfig.ModelsDir == "" {
		loaderConfig.ModelsDir = "./models"
	}
	loaderConfig.AllowDownload = os.Getenv("LLAMA_AUTO_DOWNLOAD") == "true"
	loaderConfig.ModelURL = os.Getenv("LLAMA_MODEL_URL")
	loaderConfig.RunStartupTest = true

	// Create model loader
	loader := llamaruntime.NewModelLoader(loaderConfig)

	// Load the model and create client
	loadCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	client, err := loader.Load(loadCtx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load model: %w", err)
	}

	// Get model metadata for logging
	if metadata := loader.Metadata(); metadata != nil {
		logger.Info("Model loaded",
			zap.String("name", metadata.Name),
			zap.String("size", metadata.SizeHuman),
			zap.Int("vocab_size", metadata.VocabSize),
			zap.Int("context_size", metadata.ContextSize),
			zap.Bool("startup_test_passed", metadata.StartupTestPassed),
		)
	}

	// Start GPU monitoring if GPU is available
	var gpuMonitor *llamaruntime.GPUMonitor
	gpuResult := llamaruntime.DetectGPU()
	if gpuResult.Available {
		logger.Info("GPU detected for llamaruntime",
			zap.Int("gpu_count", gpuResult.GPUCount),
			zap.Int64("total_vram_bytes", gpuResult.TotalVRAM),
		)

		// Create GPU monitor with logging callback
		gpuMonitorConfig := llamaruntime.DefaultGPUMonitorConfig()
		gpuMonitorConfig.Interval = 30 * time.Second // Check every 30 seconds
		gpuMonitorConfig.AlertThreshold = 90.0       // Alert at 90% VRAM usage
		gpuMonitorConfig.Callback = func(info *llamaruntime.GPUMemoryInfo) {
			logger.Debug("llamaruntime GPU memory",
				zap.Int64("used_bytes", info.Used),
				zap.Int64("total_bytes", info.Total),
				zap.Float64("used_pct", info.UsedPct),
			)
		}
		gpuMonitor = llamaruntime.NewGPUMonitor(gpuMonitorConfig)
		gpuMonitor.Start(ctx)
		logger.Info("llamaruntime GPU monitor started")
	} else {
		logger.Info("No GPU detected for llamaruntime, running in CPU mode")
	}

	// Create health checker
	healthConfig := llamaruntime.DefaultHealthCheckerConfig()
	healthConfig.Interval = 60 * time.Second     // Check every minute
	healthConfig.MinVRAMFree = 512 * 1024 * 1024 // Alert if less than 512MB free
	healthConfig.OnHealthy = func() {
		logger.Debug("llamaruntime health check: healthy")
	}
	healthConfig.OnUnhealthy = func(reason string) {
		logger.Warn("llamaruntime health check: unhealthy",
			zap.String("reason", reason),
		)
	}

	healthChecker := llamaruntime.NewHealthChecker(client, healthConfig)
	healthChecker.Start(ctx)
	logger.Info("llamaruntime health checker started")

	return client, healthChecker, gpuMonitor, nil
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
	suite := validation.NewValidationSuite().
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
			if step.Status == validation.StepFailed {
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

	modelManager := modelmanager.NewModelManager(modelDir, httpClient)

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
		return []string{modelmanager.DefaultTextModel.Name}
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
