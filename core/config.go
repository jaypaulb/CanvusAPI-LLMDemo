package core

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// CanvasConfig holds configuration for a single canvas
type CanvasConfig struct {
	ID        string // Canvas UUID
	Name      string // Human-readable name (optional, derived from API if not set)
	ServerURL string // Canvus server URL (uses default if empty)
	APIKey    string // API key (uses default if empty)
}

// Config holds all configuration values
type Config struct {
	// API Keys (all optional - cloud fallback only)
	OpenAIAPIKey    string
	GoogleVisionKey string
	CanvusAPIKey    string

	// Server Configuration
	CanvusServerURL      string
	CanvasName           string
	CanvasID             string         // Primary canvas ID (backward compatibility)
	CanvasConfigs        []CanvasConfig // Multi-canvas configuration
	WebUIPassword        string
	Port                 int
	AllowSelfSignedCerts bool

	// LLM API Configuration (defaults to local inference)
	BaseLLMURL  string // Default API endpoint for all LLM operations
	TextLLMURL  string // Optional override for text generation
	ImageLLMURL string // Optional override for image generation

	// Local LLM (llama.cpp) Configuration
	LlamaModelPath    string // Path to GGUF model file for local inference
	LlamaModelURL     string // Optional URL to download model if not found
	LlamaModelsDir    string // Directory for storing models (default: ./models)
	LlamaAutoDownload bool   // Enable auto-download of model if not found

	// Stable Diffusion (local image generation) Configuration
	SDModelPath      string  // Path to SD model file (.safetensors, .ckpt, or .gguf)
	SDImageSize      int     // Image output size in pixels (default: 512, must be divisible by 8)
	SDInferenceSteps int     // Number of denoising steps (default: 20, range: 1-100)
	SDGuidanceScale  float64 // CFG scale (default: 7.0, range: 1.0-30.0)
	SDNegativePrompt string  // Default negative prompt for generation
	SDTimeoutSeconds int     // Generation timeout in seconds (default: 120)
	SDMaxConcurrent  int     // Maximum concurrent generations (default: 2, adjust for VRAM)
	SDMaxImageSize   int     // Maximum image size in pixels (default: 1024)

	// Azure OpenAI Configuration (optional cloud fallback)
	AzureOpenAIEndpoint   string // Azure OpenAI endpoint (e.g., https://your-resource.openai.azure.com/)
	AzureOpenAIDeployment string // Azure deployment name for image generation
	AzureOpenAIApiVersion string // Azure API version (default: 2024-02-15-preview)

	// Model Selection (optional - local models don't need OpenAI identifiers)
	OpenAINoteModel   string
	OpenAICanvasModel string
	OpenAIPDFModel    string
	OpenAIImageModel  string

	// Token Limits (sensible defaults for local inference)
	PDFPrecisTokens       int64
	CanvasPrecisTokens    int64
	NoteResponseTokens    int64
	ImageAnalysisTokens   int64
	ErrorResponseTokens   int64
	PDFChunkSizeTokens    int64
	PDFMaxChunksTokens    int64
	PDFSummaryRatioTokens float64

	// Processing Configuration (optimized for local GPU)
	MaxRetries        int
	RetryDelay        time.Duration
	AITimeout         time.Duration
	ProcessingTimeout time.Duration
	MaxConcurrent     int
	MaxFileSize       int64
	DownloadsDir      string
}

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

// parseCanvasIDs parses the CANVAS_IDS environment variable (comma-separated).
// Returns nil if not set or empty. Each ID is trimmed of whitespace.
func parseCanvasIDs(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}

	// Split by comma and trim each ID
	var result []string
	for _, part := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// LoadConfig loads configuration from environment variables with sensible defaults
// for zero-config local AI deployment. Only Canvus credentials are required.
func LoadConfig() (*Config, error) {
	// Load API keys (all optional - only needed for cloud fallback)
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		openAIKey = os.Getenv("OPENAI_KEY") // Legacy support
	}

	// Load LLM URLs with local-first defaults
	// Default to local llamaruntime server for all operations
	baseLLMURL := getEnvOrDefault("BASE_LLM_URL", "http://127.0.0.1:1234/v1")
	textLLMURL := os.Getenv("TEXT_LLM_URL") // Optional override
	// Default to empty for image generation (triggers local SD generation)
	imageLLMURL := os.Getenv("IMAGE_LLM_URL")

	// Load Azure OpenAI configuration (optional cloud fallback)
	azureOpenAIEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	azureOpenAIDeployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")
	azureOpenAIApiVersion := getEnvOrDefault("AZURE_OPENAI_API_VERSION", "2024-02-15-preview")

	// Load Local LLM (llama.cpp) configuration
	llamaModelPath := os.Getenv("LLAMA_MODEL_PATH")
	llamaModelURL := os.Getenv("LLAMA_MODEL_URL")
	llamaModelsDir := getEnvOrDefault("LLAMA_MODELS_DIR", "./models")
	llamaAutoDownload := getEnvOrDefault("LLAMA_AUTO_DOWNLOAD", "false") == "true"

	// Load Stable Diffusion configuration
	sdModelPath := os.Getenv("SD_MODEL_PATH")
	sdImageSize := parseIntEnv("SD_IMAGE_SIZE", 512)
	sdInferenceSteps := parseIntEnv("SD_INFERENCE_STEPS", 20)
	sdGuidanceScale := parseFloat64Env("SD_GUIDANCE_SCALE", 7.0)
	sdNegativePrompt := os.Getenv("SD_NEGATIVE_PROMPT")
	sdTimeoutSeconds := parseIntEnv("SD_TIMEOUT_SECONDS", 120)
	sdMaxConcurrent := parseIntEnv("SD_MAX_CONCURRENT", 2)
	sdMaxImageSize := parseIntEnv("SD_MAX_IMAGE_SIZE", 1024)

	// Validate SD configuration if model path is set
	if sdModelPath != "" {
		// Validate image size is divisible by 8
		if sdImageSize%8 != 0 {
			return nil, fmt.Errorf("SD_IMAGE_SIZE must be divisible by 8, got %d", sdImageSize)
		}
		// Validate image size range
		if sdImageSize < 128 || sdImageSize > sdMaxImageSize {
			return nil, fmt.Errorf("SD_IMAGE_SIZE must be between 128 and %d, got %d", sdMaxImageSize, sdImageSize)
		}
		// Validate inference steps
		if sdInferenceSteps < 1 || sdInferenceSteps > 150 {
			return nil, fmt.Errorf("SD_INFERENCE_STEPS must be between 1 and 150, got %d", sdInferenceSteps)
		}
		// Validate guidance scale
		if sdGuidanceScale < 1.0 || sdGuidanceScale > 20.0 {
			return nil, fmt.Errorf("SD_GUIDANCE_SCALE must be between 1.0 and 20.0, got %.2f", sdGuidanceScale)
		}
		// Validate timeout
		if sdTimeoutSeconds < 10 {
			return nil, fmt.Errorf("SD_TIMEOUT_SECONDS must be at least 10, got %d", sdTimeoutSeconds)
		}
		// Validate max concurrent
		if sdMaxConcurrent < 1 || sdMaxConcurrent > 10 {
			return nil, fmt.Errorf("SD_MAX_CONCURRENT must be between 1 and 10, got %d", sdMaxConcurrent)
		}
	}

	// Load model names (optional - local models don't need OpenAI identifiers)
	noteModel := getEnvOrDefault("OPENAI_NOTE_MODEL", "")
	canvasModel := getEnvOrDefault("OPENAI_CANVAS_MODEL", "")
	pdfModel := getEnvOrDefault("OPENAI_PDF_MODEL", "")
	imageModel := getEnvOrDefault("IMAGE_GEN_MODEL", "")

	// Load token limits with sensible defaults optimized for local inference
	// Lower token limits improve response time for quick operations
	noteResponseTokens := parseInt64Env("OPENAI_NOTE_RESPONSE_TOKENS", 400)
	// Higher token limits for complex analysis tasks
	pdfPrecisTokens := parseInt64Env("OPENAI_PDF_PRECIS_TOKENS", 1000)
	canvasPrecisTokens := parseInt64Env("OPENAI_CANVAS_PRECIS_TOKENS", 600)
	// Very high limit for vision models which need large context windows
	imageAnalysisTokens := parseInt64Env("OPENAI_IMAGE_ANALYSIS_TOKENS", 16384)
	errorResponseTokens := parseInt64Env("OPENAI_ERROR_RESPONSE_TOKENS", 200)
	// PDF chunking settings balance memory usage and processing thoroughness
	pdfChunkSizeTokens := parseInt64Env("OPENAI_PDF_CHUNK_SIZE_TOKENS", 20000)
	pdfMaxChunksTokens := parseInt64Env("OPENAI_PDF_MAX_CHUNKS_TOKENS", 10)
	pdfSummaryRatio := parseFloat64Env("OPENAI_PDF_SUMMARY_RATIO", 0.3)

	// Load processing configuration optimized for local GPU inference
	// 3 retries with 1s delay handles transient issues without excessive wait
	maxRetries := parseIntEnv("MAX_RETRIES", 3)
	retryDelay := time.Duration(parseIntEnv("RETRY_DELAY", 1)) * time.Second
	// 60s AI timeout accommodates slower models while preventing hangs
	aiTimeout := time.Duration(parseIntEnv("AI_TIMEOUT", 60)) * time.Second
	// 300s processing timeout allows complex multi-step operations to complete
	processingTimeout := time.Duration(parseIntEnv("PROCESSING_TIMEOUT", 300)) * time.Second
	// 5 concurrent operations balances throughput and GPU memory usage
	maxConcurrent := parseIntEnv("MAX_CONCURRENT", 5)
	// 50MB limit handles most PDFs and images while preventing abuse
	maxFileSize := parseInt64Env("MAX_FILE_SIZE", 52428800)
	downloadsDir := getEnvOrDefault("DOWNLOADS_DIR", "./downloads")
	allowSelfSignedCerts := getEnvOrDefault("ALLOW_SELF_SIGNED_CERTS", "false") == "true"

	// Parse multi-canvas configuration
	canvasIDs := parseCanvasIDs("CANVAS_IDS")
	singleCanvasID := os.Getenv("CANVAS_ID")
	canvusServerURL := os.Getenv("CANVUS_SERVER")
	canvusAPIKey := os.Getenv("CANVUS_API_KEY")

	// Build canvas configs from either CANVAS_IDS or CANVAS_ID
	var canvasConfigs []CanvasConfig
	if len(canvasIDs) > 0 {
		// Multi-canvas mode: use CANVAS_IDS
		for _, id := range canvasIDs {
			canvasConfigs = append(canvasConfigs, CanvasConfig{
				ID:        id,
				ServerURL: canvusServerURL,
				APIKey:    canvusAPIKey,
			})
		}
	} else if singleCanvasID != "" {
		// Single canvas mode: use CANVAS_ID (backward compatibility)
		canvasConfigs = append(canvasConfigs, CanvasConfig{
			ID:        singleCanvasID,
			Name:      os.Getenv("CANVAS_NAME"),
			ServerURL: canvusServerURL,
			APIKey:    canvusAPIKey,
		})
	}

	// Validate ONLY required Canvus credentials
	// OpenAI API key is NOT required - only needed for cloud fallback mode
	requiredVars := []string{
		"CANVUS_SERVER",
		"CANVUS_API_KEY",
		"WEBUI_PWD",
	}

	var missingVars []string
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			missingVars = append(missingVars, v)
		}
	}

	// Either CANVAS_ID or CANVAS_IDS must be set
	if singleCanvasID == "" && len(canvasIDs) == 0 {
		missingVars = append(missingVars, "CANVAS_ID or CANVAS_IDS")
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v. See .env.example for configuration template", missingVars)
	}

	return &Config{
		// API Keys (optional - cloud fallback only)
		OpenAIAPIKey:    openAIKey,
		GoogleVisionKey: os.Getenv("GOOGLE_VISION_API_KEY"),
		CanvusAPIKey:    canvusAPIKey,

		// Server Configuration
		CanvusServerURL:      canvusServerURL,
		CanvasName:           os.Getenv("CANVAS_NAME"),
		CanvasID:             singleCanvasID,
		CanvasConfigs:        canvasConfigs,
		WebUIPassword:        os.Getenv("WEBUI_PWD"),
		Port:                 parseIntEnv("PORT", 3000),
		AllowSelfSignedCerts: allowSelfSignedCerts,

		// LLM Configuration (defaults to local inference)
		BaseLLMURL:  baseLLMURL,
		TextLLMURL:  textLLMURL,
		ImageLLMURL: imageLLMURL,

		// Local LLM (llama.cpp) Configuration
		LlamaModelPath:    llamaModelPath,
		LlamaModelURL:     llamaModelURL,
		LlamaModelsDir:    llamaModelsDir,
		LlamaAutoDownload: llamaAutoDownload,

		// Stable Diffusion Configuration
		SDModelPath:      sdModelPath,
		SDImageSize:      sdImageSize,
		SDInferenceSteps: sdInferenceSteps,
		SDGuidanceScale:  sdGuidanceScale,
		SDNegativePrompt: sdNegativePrompt,
		SDTimeoutSeconds: sdTimeoutSeconds,
		SDMaxConcurrent:  sdMaxConcurrent,
		SDMaxImageSize:   sdMaxImageSize,

		// Azure OpenAI Configuration (optional cloud fallback)
		AzureOpenAIEndpoint:   azureOpenAIEndpoint,
		AzureOpenAIDeployment: azureOpenAIDeployment,
		AzureOpenAIApiVersion: azureOpenAIApiVersion,

		// Model Selection (optional - local models don't need identifiers)
		OpenAINoteModel:   noteModel,
		OpenAICanvasModel: canvasModel,
		OpenAIPDFModel:    pdfModel,
		OpenAIImageModel:  imageModel,

		// Token Limits (sensible defaults for local inference)
		PDFPrecisTokens:       pdfPrecisTokens,
		CanvasPrecisTokens:    canvasPrecisTokens,
		NoteResponseTokens:    noteResponseTokens,
		ImageAnalysisTokens:   imageAnalysisTokens,
		ErrorResponseTokens:   errorResponseTokens,
		PDFChunkSizeTokens:    pdfChunkSizeTokens,
		PDFMaxChunksTokens:    pdfMaxChunksTokens,
		PDFSummaryRatioTokens: pdfSummaryRatio,

		// Processing Configuration (optimized for local GPU)
		MaxRetries:        maxRetries,
		RetryDelay:        retryDelay,
		AITimeout:         aiTimeout,
		ProcessingTimeout: processingTimeout,
		MaxConcurrent:     maxConcurrent,
		MaxFileSize:       maxFileSize,
		DownloadsDir:      downloadsDir,
	}, nil
}

// GetHTTPClient returns an HTTP client configured with TLS settings based on AllowSelfSignedCerts
// This should be used for all HTTP requests to external APIs to ensure TLS configuration is respected
func GetHTTPClient(cfg *Config, timeout time.Duration) *http.Client {
	client := &http.Client{
		Timeout: timeout,
	}

	if cfg.AllowSelfSignedCerts {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return client
}

// GetDefaultHTTPClient returns an HTTP client with default timeout (30s) configured with TLS settings
func GetDefaultHTTPClient(cfg *Config) *http.Client {
	return GetHTTPClient(cfg, 30*time.Second)
}

// GetCanvasCount returns the number of configured canvases.
func (c *Config) GetCanvasCount() int {
	return len(c.CanvasConfigs)
}

// GetCanvasIDs returns a slice of all configured canvas IDs.
func (c *Config) GetCanvasIDs() []string {
	ids := make([]string, len(c.CanvasConfigs))
	for i, cfg := range c.CanvasConfigs {
		ids[i] = cfg.ID
	}
	return ids
}

// GetCanvasConfig returns the configuration for a specific canvas ID.
// Returns nil if the canvas ID is not found.
func (c *Config) GetCanvasConfig(canvasID string) *CanvasConfig {
	for i := range c.CanvasConfigs {
		if c.CanvasConfigs[i].ID == canvasID {
			return &c.CanvasConfigs[i]
		}
	}
	return nil
}

// GetPrimaryCanvasID returns the primary canvas ID.
// In multi-canvas mode, returns the first canvas ID.
// In single canvas mode, returns the CanvasID field.
func (c *Config) GetPrimaryCanvasID() string {
	if len(c.CanvasConfigs) > 0 {
		return c.CanvasConfigs[0].ID
	}
	return c.CanvasID
}

// IsMultiCanvasMode returns true if multiple canvases are configured.
func (c *Config) IsMultiCanvasMode() bool {
	return len(c.CanvasConfigs) > 1
}

// HasSDModel returns true if a Stable Diffusion model is configured.
func (c *Config) HasSDModel() bool {
	return c.SDModelPath != ""
}
