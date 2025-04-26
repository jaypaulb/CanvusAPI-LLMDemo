package llm

import (
	"context"
	"fmt"

	"go_backend/core"

	"github.com/sashabaranov/go-openai"
)

// GenerateAIResponse generates a response from the AI model
func GenerateAIResponse(ctx context.Context, config *core.Config, prompt string) (string, error) {
	clientConfig := openai.DefaultConfig(config.OpenAIAPIKey)
	if config.OpenAIAPIBaseURL != "" {
		clientConfig.BaseURL = config.OpenAIAPIBaseURL
	}
	client := openai.NewClientWithConfig(clientConfig)

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: config.OpenAINoteModel,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens: int(config.NoteResponseTokens),
		},
	)

	if err != nil {
		return "", fmt.Errorf("error generating AI response: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}
