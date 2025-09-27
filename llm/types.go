package llm

import (
	"github.com/sashabaranov/go-openai"
	"jas-agent/core"
)

type ChatRequest struct {
	model    string
	messages []core.Message
	stream   bool
	tools    []core.Tool
}

func NewChatRequest(model string, messages []core.Message, tools ...core.Tool) ChatRequest {
	return ChatRequest{
		model:    model,
		messages: messages,
		stream:   false,
		tools:    tools,
	}
}
func (req ChatRequest) Request() openai.ChatCompletionRequest {
	var messages []openai.ChatCompletionMessage
	for _, message := range req.messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    string(message.Role),
			Content: message.Content,
			Name:    message.Name,
		})
	}
	return openai.ChatCompletionRequest{
		Model:    req.model,
		Messages: messages,
		Stream:   req.stream,
	}
}

type ChatResponse struct {
	openai.ChatCompletionResponse
}

func (resp ChatResponse) Content() string {
	if len(resp.Choices) == 0 {
		return ""
	}
	return resp.Choices[0].Message.Content
}
