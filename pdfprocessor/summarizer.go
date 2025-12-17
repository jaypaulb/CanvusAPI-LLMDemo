// Package pdfprocessor provides PDF processing functionality for CanvusLocalLLM.
//
// summarizer.go implements the Summarizer molecule that generates AI summaries
// from chunked PDF text. It composes:
//   - chunker.go: ChunkerResult for receiving chunked text
//   - Uses OpenAI client for AI summarization
package pdfprocessor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// ErrNoChunks is returned when no chunks are provided for summarization.
var ErrNoChunks = errors.New("no chunks provided for summarization")

// ErrEmptyResponse is returned when the AI returns an empty response.
var ErrEmptyResponse = errors.New("AI returned empty response")

// ErrInvalidJSON is returned when the AI response doesn't contain valid JSON.
var ErrInvalidJSON = errors.New("AI response does not contain valid JSON")

// SummarizerConfig holds configuration for AI summarization.
type SummarizerConfig struct {
	// Model is the OpenAI model to use (e.g., "gpt-4", "gpt-3.5-turbo")
	Model string

	// MaxTokens is the maximum tokens for the response
	MaxTokens int

	// Temperature controls response randomness (0.0-1.0)
	Temperature float32

	// SystemPromptTemplate is the template for the system message
	// %d placeholder will be replaced with total chunk count
	SystemPromptTemplate string

	// ChunkTemplate is the template for each chunk message
	// First %d is chunk number, second %d is total chunks, %s is chunk content, fourth %d is chunk number
	ChunkTemplate string

	// FinalPrompt is the prompt sent after all chunks
	FinalPrompt string
}

// DefaultSummarizerConfig returns sensible default configuration for PDF summarization.
func DefaultSummarizerConfig() SummarizerConfig {
	return SummarizerConfig{
		Model:                "gpt-4",
		MaxTokens:            2000,
		Temperature:          0.3,
		SystemPromptTemplate: `You will receive %d chunks of a document. Do not respond until you receive the final chunk. After the last chunk, I will prompt you for your analysis of the entire document.`,
		ChunkTemplate:        "#--- chunk %d of %d ---#\n%s\n#--- end of chunk %d ---#",
		FinalPrompt: `You have now received all chunks. Please analyze the entire document and provide a summary in the following JSON format:
{"type": "text", "content": "..."}
The content field must be a Markdown-formatted summary with the following sections:
# Overview
# Key Points
# Details
# Conclusions

Respond ONLY with valid JSON as shown above, and ensure the content is Markdown.`,
	}
}

// SummaryResult contains the result of AI summarization.
type SummaryResult struct {
	// Content is the markdown-formatted summary content
	Content string

	// RawResponse is the raw AI response before JSON extraction
	RawResponse string

	// PromptTokens is the estimated prompt token count
	PromptTokens int

	// CompletionTokens is the estimated completion token count
	CompletionTokens int

	// ChunksProcessed is the number of chunks that were sent to the AI
	ChunksProcessed int
}

// AIResponse represents the JSON structure expected from the AI.
type AIResponse struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Summarizer generates AI summaries from chunked text.
type Summarizer struct {
	config SummarizerConfig
	client *openai.Client
}

// NewSummarizer creates a new Summarizer with the given configuration and OpenAI client.
func NewSummarizer(config SummarizerConfig, client *openai.Client) *Summarizer {
	return &Summarizer{
		config: config,
		client: client,
	}
}

// Summarize generates a summary from the provided chunks.
// It sends all chunks to the AI and requests a final summary.
//
// Example:
//
//	client := openai.NewClient("api-key")
//	summarizer := NewSummarizer(DefaultSummarizerConfig(), client)
//	result, err := summarizer.Summarize(ctx, []string{"chunk1", "chunk2"})
func (s *Summarizer) Summarize(ctx context.Context, chunks []string) (*SummaryResult, error) {
	if len(chunks) == 0 {
		return nil, ErrNoChunks
	}

	// Build messages array
	messages := s.buildMessages(chunks)

	// Call OpenAI API
	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       s.config.Model,
		Messages:    messages,
		MaxTokens:   s.config.MaxTokens,
		Temperature: s.config.Temperature,
	})

	if err != nil {
		return nil, fmt.Errorf("AI summarization failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, ErrEmptyResponse
	}

	rawResponse := resp.Choices[0].Message.Content

	// Parse JSON response
	content, err := extractJSONContent(rawResponse)
	if err != nil {
		return nil, err
	}

	// Estimate tokens
	promptTokens := s.estimatePromptTokens(chunks)
	completionTokens := EstimateTokenCount(rawResponse)

	return &SummaryResult{
		Content:          content,
		RawResponse:      rawResponse,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		ChunksProcessed:  len(chunks),
	}, nil
}

// SummarizeChunkerResult is a convenience method that takes a ChunkerResult directly.
//
// Example:
//
//	chunker := NewDefaultChunker()
//	chunkerResult := chunker.SplitIntoChunks(text)
//	result, err := summarizer.SummarizeChunkerResult(ctx, chunkerResult)
func (s *Summarizer) SummarizeChunkerResult(ctx context.Context, chunkerResult *ChunkerResult) (*SummaryResult, error) {
	if chunkerResult == nil || len(chunkerResult.Chunks) == 0 {
		return nil, ErrNoChunks
	}

	// Convert ChunkResults to string slice
	chunks := ChunksToStrings(chunkerResult)
	return s.Summarize(ctx, chunks)
}

// buildMessages constructs the OpenAI message array for multi-chunk summarization.
func (s *Summarizer) buildMessages(chunks []string) []openai.ChatCompletionMessage {
	totalChunks := len(chunks)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: fmt.Sprintf(s.config.SystemPromptTemplate, totalChunks),
		},
	}

	// Add each chunk as a user message
	for i, chunk := range chunks {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf(s.config.ChunkTemplate, i+1, totalChunks, chunk, i+1),
		})
	}

	// Add final analysis prompt
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: s.config.FinalPrompt,
	})

	return messages
}

// estimatePromptTokens estimates the total prompt tokens for the request.
func (s *Summarizer) estimatePromptTokens(chunks []string) int {
	total := 0

	// System prompt tokens
	total += EstimateTokenCount(fmt.Sprintf(s.config.SystemPromptTemplate, len(chunks)))

	// Chunk message tokens
	for i, chunk := range chunks {
		formatted := fmt.Sprintf(s.config.ChunkTemplate, i+1, len(chunks), chunk, i+1)
		total += EstimateTokenCount(formatted)
	}

	// Final prompt tokens
	total += EstimateTokenCount(s.config.FinalPrompt)

	return total
}

// extractJSONContent extracts the content field from an AI JSON response.
func extractJSONContent(rawResponse string) (string, error) {
	// Find JSON boundaries
	startIdx := strings.Index(rawResponse, "{")
	endIdx := strings.LastIndex(rawResponse, "}")

	if startIdx == -1 || endIdx == -1 || startIdx > endIdx {
		return "", ErrInvalidJSON
	}

	jsonStr := rawResponse[startIdx : endIdx+1]

	// Parse JSON
	var aiResp AIResponse
	if err := json.Unmarshal([]byte(jsonStr), &aiResp); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	if aiResp.Content == "" {
		return "", fmt.Errorf("%w: content field is empty", ErrInvalidJSON)
	}

	return aiResp.Content, nil
}

// SummarizeText is a convenience function that chunks text and summarizes it.
// It uses the provided chunker to split text, then summarizes all chunks.
//
// Example:
//
//	client := openai.NewClient("api-key")
//	summarizer := NewSummarizer(DefaultSummarizerConfig(), client)
//	chunker := NewChunker(DefaultChunkerConfig())
//	result, err := SummarizeText(ctx, summarizer, chunker, longText)
func SummarizeText(ctx context.Context, summarizer *Summarizer, chunker *Chunker, text string) (*SummaryResult, error) {
	chunkerResult := chunker.SplitIntoChunks(text)
	return summarizer.SummarizeChunkerResult(ctx, chunkerResult)
}
