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
	// API Keys
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

	// LLM API Configuration
	BaseLLMURL  string // Default API endpoint for all LLM operations
	TextLLMURL  string // Optional override for text generation
	ImageLLMURL string // Optional override for image generation

	// Local LLM (llama.cpp) Configuration
	LlamaModelPath    string // Path to GGUF model file for local inference
	LlamaModelURL     string // Optional URL to download model if not found
	LlamaModelsDir    string // Directory for storing models (default: ./models)
	LlamaAutoDownload bool   // Enable auto-download of model if not found

	// Azure OpenAI Configuration
	AzureOpenAIEndpoint   string // Azure OpenAI endpoint (e.g., https://your-resource.openai.azure.com/)
	AzureOpenAIDeployment string // Azure deployment name for image generation
	AzureOpenAIApiVersion string // Azure API version (default: 2024-02-15-preview)

	// Model Selection
	OpenAINoteModel   string
	OpenAICanvasModel string
	OpenAIPDFModel    string
	OpenAIImageModel  string

	// Token Limits
	PDFPrecisTokens       int64
	CanvasPrecisTokens    int64
	NoteResponseTokens    int64
	ImageAnalysisTokens   int64
	ErrorResponseTokens   int64
	PDFChunkSizeTokens    int64
	PDFMaxChunksTokens    int64
	PDFSummaryRatioTokens float64

	// Processing Configuration
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

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load API keys
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		openAIKey = os.Getenv("OPENAI_KEY") // Legacy support
	}

	// Load LLM URLs
	baseLLMURL := getEnvOrDefault("BASE_LLM_URL", "http://127.0.0.1:1234/v1")
	textLLMURL := os.Getenv("TEXT_LLM_URL")                                      // Optional override
	imageLLMURL := getEnvOrDefault("IMAGE_LLM_URL", "https://api.openai.com/v1") // Default to OpenAI for image generation

	// Load Azure OpenAI configuration
	azureOpenAIEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	azureOpenAIDeployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")
	azureOpenAIApiVersion := getEnvOrDefault("AZURE_OPENAI_API_VERSION", "2024-02-15-preview")

	// Load Local LLM (llama.cpp) configuration
	llamaModelPath := os.Getenv("LLAMA_MODEL_PATH")
	llamaModelURL := os.Getenv("LLAMA_MODEL_URL")
	llamaModelsDir := getEnvOrDefault("LLAMA_MODELS_DIR", "./models")
	llamaAutoDownload := getEnvOrDefault("LLAMA_AUTO_DOWNLOAD", "false") == "true"

	// Load model names
	noteModel := getEnvOrDefault("OPENAI_NOTE_MODEL", "gpt-4")
	canvasModel := getEnvOrDefault("OPENAI_CANVAS_MODEL", "gpt-4")
	pdfModel := getEnvOrDefault("OPENAI_PDF_MODEL", "gpt-4")
	imageModel := getEnvOrDefault("IMAGE_GEN_MODEL", "dall-e-3")

	// Load token limits with standardized default values
	pdfPrecisTokens := parseInt64Env("OPENAI_PDF_PRECIS_TOKENS", 1000)
	canvasPrecisTokens := parseInt64Env("OPENAI_CANVAS_PRECIS_TOKENS", 600)
	noteResponseTokens := parseInt64Env("OPENAI_NOTE_RESPONSE_TOKENS", 400)
	imageAnalysisTokens := parseInt64Env("OPENAI_IMAGE_ANALYSIS_TOKENS", 16384)
	errorResponseTokens := parseInt64Env("OPENAI_ERROR_RESPONSE_TOKENS", 200)
	pdfChunkSizeTokens := parseInt64Env("OPENAI_PDF_CHUNK_SIZE_TOKENS", 20000)
	pdfMaxChunksTokens := parseInt64Env("OPENAI_PDF_MAX_CHUNKS_TOKENS", 10)
	pdfSummaryRatio := parseFloat64Env("OPENAI_PDF_SUMMARY_RATIO", 0.3)

	// Load processing configuration
	maxRetries := parseIntEnv("MAX_RETRIES", 3)
	retryDelay := time.Duration(parseIntEnv("RETRY_DELAY", 1)) * time.Second
	aiTimeout := time.Duration(parseIntEnv("AI_TIMEOUT", 60)) * time.Second
	processingTimeout := time.Duration(parseIntEnv("PROCESSING_TIMEOUT", 300)) * time.Second
	maxConcurrent := parseIntEnv("MAX_CONCURRENT", 5)
	maxFileSize := parseInt64Env("MAX_FILE_SIZE", 52428800) // 50MB
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

	// Validate required environment variables
	requiredVars := []string{
		"CANVUS_SERVER",
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

	// Either CANVAS_ID or CANVAS_IDS must be set
	if singleCanvasID == "" && len(canvasIDs) == 0 {
		missingVars = append(missingVars, "CANVAS_ID or CANVAS_IDS")
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	return &Config{
		// API Keys
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

		// LLM Configuration
		BaseLLMURL:  baseLLMURL,
		TextLLMURL:  textLLMURL,
		ImageLLMURL: imageLLMURL,

		// Local LLM (llama.cpp) Configuration
		LlamaModelPath:    llamaModelPath,
		LlamaModelURL:     llamaModelURL,
		LlamaModelsDir:    llamaModelsDir,
		LlamaAutoDownload: llamaAutoDownload,

		// Azure OpenAI Configuration
		AzureOpenAIEndpoint:   azureOpenAIEndpoint,
		AzureOpenAIDeployment: azureOpenAIDeployment,
		AzureOpenAIApiVersion: azureOpenAIApiVersion,

		// Model Selection
		OpenAINoteModel:   noteModel,
		OpenAICanvasModel: canvasModel,
		OpenAIPDFModel:    pdfModel,
		OpenAIImageModel:  imageModel,

		// Token Limits
		PDFPrecisTokens:       pdfPrecisTokens,
		CanvasPrecisTokens:    canvasPrecisTokens,
		NoteResponseTokens:    noteResponseTokens,
		ImageAnalysisTokens:   imageAnalysisTokens,
		ErrorResponseTokens:   errorResponseTokens,
		PDFChunkSizeTokens:    pdfChunkSizeTokens,
		PDFMaxChunksTokens:    pdfMaxChunksTokens,
		PDFSummaryRatioTokens: pdfSummaryRatio,

		// Processing Configuration
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
