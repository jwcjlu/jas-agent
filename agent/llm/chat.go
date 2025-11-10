package llm

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

type Chat interface {
	Completions(ctx context.Context, chatReq ChatRequest) (*ChatResponse, error)
}

type openaiChat struct {
	client *openai.Client
}

type Config struct {
	ApiKey  string
	BaseURL string
}

func (config *Config) Enable() bool {
	return len(config.ApiKey) > 0 && len(config.BaseURL) > 0
}
func NewChat(chatConfig *Config) Chat {
	config := openai.DefaultConfig(chatConfig.ApiKey)
	if chatConfig.BaseURL != "" {
		config.BaseURL = chatConfig.BaseURL
	}
	return &openaiChat{client: openai.NewClientWithConfig(config)}
}

func (chat *openaiChat) Completions(ctx context.Context, chatReq ChatRequest) (*ChatResponse, error) {
	// 调用API
	resp, err := chat.client.CreateChatCompletion(
		context.WithoutCancel(ctx),
		chatReq.Request(),
	)
	if err != nil {
		return nil, err
	}
	return &ChatResponse{resp}, nil
}
