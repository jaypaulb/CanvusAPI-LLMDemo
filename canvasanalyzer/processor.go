package canvasanalyzer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// ErrAnalysisFailed is returned when AI analysis fails.
var ErrAnalysisFailed = errors.New("canvasanalyzer: analysis failed")

// ErrEmptyResponse is returned when the AI returns an empty response.
var ErrEmptyResponse = errors.New("canvasanalyzer: empty AI response")

// ErrInvalidResponse is returned when the AI response cannot be parsed.
var ErrInvalidResponse = errors.New("canvasanalyzer: invalid AI response format")

// DefaultSystemPrompt is the default prompt for canvas analysis.
const DefaultSystemPrompt = `You are an assistant analyzing a collaborative workspace.
Describe the content and relationships between items in a natural, narrative way.
Focus on the story the workspace is telling and how items relate to each other.
Avoid mentioning technical details like IDs or coordinates.
Format your response as text using markdown with three sections:
# Overview
Describe the main themes and content of the workspace.
# Insights
Share observations about relationships between items and suggest next steps.
# Recommendations
Provide actionable recommendations for improving the workspace.`

// ProcessorConfig holds configuration for the Analysis Processor.
type ProcessorConfig struct {
	// Model is the OpenAI model to use (default: gpt-4)
	Model string

	// SystemPrompt is the system message for AI analysis
	SystemPrompt string

	// MaxTokens is the maximum response tokens (default: 4096)
	MaxTokens int

	// Temperature controls randomness (default: 0.7)
	Temperature float32

	// Timeout is the maximum time for AI request (default: 2m)
	Timeout time.Duration

	// BaseURL is the OpenAI API base URL (empty = default OpenAI)
	BaseURL string
}

// DefaultProcessorConfig returns sensible default configuration.
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		Model:        "gpt-4",
		SystemPrompt: DefaultSystemPrompt,
		MaxTokens:    4096,
		Temperature:  0.7,
		Timeout:      2 * time.Minute,
		BaseURL:      "",
	}
}

// AnalysisResult contains the result of canvas analysis.
type AnalysisResult struct {
	// Content is the AI-generated analysis content
	Content string

	// RawResponse is the raw AI response before processing
	RawResponse string

	// PromptTokens is the estimated input token count
	PromptTokens int

	// CompletionTokens is the estimated output token count
	CompletionTokens int

	// Duration is the time taken for analysis
	Duration time.Duration

	// WidgetCount is the number of widgets analyzed
	WidgetCount int

	// Model is the model used for analysis
	Model string
}

// Processor generates AI-powered analysis from canvas widgets.
type Processor struct {
	config ProcessorConfig
	client *openai.Client
	logger *zap.Logger
}

// NewProcessor creates a new Processor with the given configuration and OpenAI client.
//
// Example:
//
//	client := openai.NewClient("api-key")
//	processor := NewProcessor(DefaultProcessorConfig(), client, logger)
//	result, err := processor.Analyze(ctx, widgets)
func NewProcessor(config ProcessorConfig, client *openai.Client, logger *zap.Logger) *Processor {
	if config.Model == "" {
		config.Model = "gpt-4"
	}
	if config.SystemPrompt == "" {
		config.SystemPrompt = DefaultSystemPrompt
	}
	if config.MaxTokens <= 0 {
		config.MaxTokens = 4096
	}
	if config.Temperature < 0 || config.Temperature > 2 {
		config.Temperature = 0.7
	}
	if config.Timeout <= 0 {
		config.Timeout = 2 * time.Minute
	}

	return &Processor{
		config: config,
		client: client,
		logger: logger,
	}
}

// Analyze generates an AI analysis of the provided widgets.
//
// The widgets are serialized to JSON and sent to the AI model along with
// the system prompt. The response is parsed to extract the content.
//
// Returns ErrAnalysisFailed if the AI request fails.
// Returns ErrEmptyResponse if the AI returns no content.
// Returns ErrInvalidResponse if the response cannot be parsed.
func (p *Processor) Analyze(ctx context.Context, widgets []Widget) (*AnalysisResult, error) {
	start := time.Now()

	// Serialize widgets to JSON
	widgetsJSON, err := WidgetsToJSON(widgets)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to serialize widgets: %v", ErrAnalysisFailed, err)
	}

	p.logger.Info("starting canvas analysis",
		zap.Int("widget_count", len(widgets)),
		zap.Int("json_length", len(widgetsJSON)),
		zap.String("model", p.config.Model))

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	// Build request
	request := openai.ChatCompletionRequest{
		Model: p.config.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: p.config.SystemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: widgetsJSON,
			},
		},
		MaxTokens:   p.config.MaxTokens,
		Temperature: p.config.Temperature,
	}

	// Call OpenAI
	response, err := p.client.CreateChatCompletion(timeoutCtx, request)
	if err != nil {
		p.logger.Error("AI request failed",
			zap.Error(err),
			zap.Duration("duration", time.Since(start)))
		return nil, fmt.Errorf("%w: %v", ErrAnalysisFailed, err)
	}

	// Extract response content
	if len(response.Choices) == 0 {
		p.logger.Error("AI returned no choices")
		return nil, ErrEmptyResponse
	}

	rawResponse := response.Choices[0].Message.Content
	if rawResponse == "" {
		p.logger.Error("AI returned empty content")
		return nil, ErrEmptyResponse
	}

	// Parse the response to extract content
	content := p.extractContent(rawResponse)

	// Calculate token estimates
	promptTokens := estimateTokens(p.config.SystemPrompt) + estimateTokens(widgetsJSON)
	completionTokens := estimateTokens(rawResponse)

	// Use actual usage if available
	if response.Usage.PromptTokens > 0 {
		promptTokens = response.Usage.PromptTokens
	}
	if response.Usage.CompletionTokens > 0 {
		completionTokens = response.Usage.CompletionTokens
	}

	p.logger.Info("canvas analysis completed",
		zap.Int("prompt_tokens", promptTokens),
		zap.Int("completion_tokens", completionTokens),
		zap.Duration("duration", time.Since(start)))

	return &AnalysisResult{
		Content:          content,
		RawResponse:      rawResponse,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		Duration:         time.Since(start),
		WidgetCount:      len(widgets),
		Model:            p.config.Model,
	}, nil
}

// AnalyzeWithPrompt generates analysis using a custom system prompt.
func (p *Processor) AnalyzeWithPrompt(ctx context.Context, widgets []Widget, systemPrompt string) (*AnalysisResult, error) {
	originalPrompt := p.config.SystemPrompt
	p.config.SystemPrompt = systemPrompt
	defer func() {
		p.config.SystemPrompt = originalPrompt
	}()

	return p.Analyze(ctx, widgets)
}

// extractContent parses the AI response and extracts the main content.
// It handles both JSON-wrapped responses and plain text responses.
func (p *Processor) extractContent(rawResponse string) string {
	// First, try to parse as JSON (the AI might wrap content in JSON)
	content := extractJSONContent(rawResponse)
	if content != "" {
		return content
	}

	// Otherwise, use the raw response with escaped newlines converted
	content = strings.ReplaceAll(rawResponse, "\\n", "\n")
	return strings.TrimSpace(content)
}

// extractJSONContent attempts to extract content from a JSON response.
// Returns empty string if not JSON or no content field found.
func extractJSONContent(response string) string {
	// Try to find JSON in the response
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")

	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return ""
	}

	jsonStr := response[startIdx : endIdx+1]

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return ""
	}

	// Look for common content fields
	for _, key := range []string{"content", "text", "response", "analysis"} {
		if content, ok := data[key].(string); ok && content != "" {
			return content
		}
	}

	return ""
}

// estimateTokens provides a rough token estimate (4 chars per token).
func estimateTokens(text string) int {
	return len(text) / 4
}

// GetConfig returns a copy of the current configuration.
func (p *Processor) GetConfig() ProcessorConfig {
	return p.config
}

// SetModel updates the model used for analysis.
func (p *Processor) SetModel(model string) {
	p.config.Model = model
}

// SetSystemPrompt updates the system prompt.
func (p *Processor) SetSystemPrompt(prompt string) {
	p.config.SystemPrompt = prompt
}

// SetMaxTokens updates the maximum response tokens.
func (p *Processor) SetMaxTokens(maxTokens int) {
	if maxTokens > 0 {
		p.config.MaxTokens = maxTokens
	}
}

// SetTemperature updates the temperature setting.
func (p *Processor) SetTemperature(temperature float32) {
	if temperature >= 0 && temperature <= 2 {
		p.config.Temperature = temperature
	}
}
