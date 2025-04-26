package core

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// GenerateAIResponse generates a response using the OpenAI API
func GenerateAIResponse(ctx context.Context, cfg *Config, prompt string) (string, error) {
	client := createOpenAIClient(cfg)

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: cfg.OpenAINoteModel,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens: int(cfg.NoteResponseTokens),
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to generate AI response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

func createOpenAIClient(cfg *Config) *openai.Client {
	// Create client with configuration
	clientConfig := openai.DefaultConfig(cfg.OpenAIAPIKey)

	// Use TextLLMURL if set, otherwise fall back to BaseLLMURL
	if cfg.TextLLMURL != "" {
		clientConfig.BaseURL = cfg.TextLLMURL
	} else if cfg.BaseLLMURL != "" {
		clientConfig.BaseURL = cfg.BaseLLMURL
	}

	return openai.NewClientWithConfig(clientConfig)
}
