package llm

import (
	"github.com/sashabaranov/go-openai"
	"jas-agent/core"
	"jas-agent/tools"
	"regexp"
	"strings"
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
	var tools []openai.Tool
	for _, tool := range req.tools {
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name(),
				Description: tool.Description(),
				Strict:      false,
				Parameters:  tool.Input(),
			},
		})
	}
	return openai.ChatCompletionRequest{
		Model:    req.model,
		Messages: messages,
		Tools:    tools,
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
func (resp ChatResponse) CallTools() []openai.ToolCall {
	if len(resp.Choices) == 0 {
		return nil
	}
	return resp.Choices[0].Message.ToolCalls
}

// GetToolCalls 解析助手响应中的工具调用
func (resp ChatResponse) GetToolCalls() []*tools.ToolCall {
	var callsTools []*tools.ToolCall
	for _, callTool := range resp.CallTools() {
		callsTools = append(callsTools, &tools.ToolCall{
			Name:  callTool.Function.Name,
			Input: callTool.Function.Arguments,
		})
	}
	callsTools = append(callsTools, parseToolCall(resp.Content())...)
	return callsTools
}

func parseToolCall(content string) []*tools.ToolCall {
	// 匹配格式: Action: toolName[input]
	var toolCalls []*tools.ToolCall
	re := regexp.MustCompile(`Action:\s*(\w+@?\w*)\[([^]]+)\]`)
	matches := re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 3 {
			toolCalls = append(toolCalls, &tools.ToolCall{
				Name:  match[1],
				Input: match[2],
			})
		}
		if len(match) == 2 {
			if strings.ToLower(match[1]) == "finish" {
				continue
			}
			toolCalls = append(toolCalls, &tools.ToolCall{
				Name: match[1],
			})
		}
	}
	return toolCalls
}
