package llm

import (
	"jas-agent/agent/core"
	"jas-agent/agent/tools"
	"regexp"
	"strings"

	"github.com/sashabaranov/go-openai"
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
	// 匹配格式:
	// - Action: toolName[input]  (有括号有输入)
	// - Action: toolName[]       (有括号无输入)
	// - Action: toolName         (无括号)
	var toolCalls []*tools.ToolCall

	// 匹配 Action: toolName（可能有或没有括号）

	actionPattern := regexp.MustCompile(`Action:\s*([a-zA-Z0-9_-]+@?[a-zA-Z0-9_-]*)`)
	actionMatches := actionPattern.FindAllStringSubmatchIndex(content, -1)

	for _, match := range actionMatches {
		if len(match) < 4 {
			continue
		}

		// 提取工具名称
		toolName := content[match[2]:match[3]]

		// 跳过 Finish
		if strings.ToLower(toolName) == "finish" {
			continue
		}

		// 查找工具名称后面是否有括号
		afterToolName := match[3] // 工具名称结束位置
		remainingContent := content[afterToolName:]

		// 跳过空白字符
		trimmed := strings.TrimLeft(remainingContent, " \t\n\r")

		var input string
		if strings.HasPrefix(trimmed, "[") {
			// 有括号，提取括号内容
			bracketStart := afterToolName + (len(remainingContent) - len(trimmed))
			extracted, found := extractBracketContent(content, bracketStart)
			if found {
				input = extracted
			}
		}
		// 如果没有括号，input 保持为空字符串

		toolCalls = append(toolCalls, &tools.ToolCall{
			Name:  toolName,
			Input: input,
		})
	}

	return toolCalls
}

// extractBracketContent 提取括号内的内容（处理嵌套，支持复杂JSON）
func extractBracketContent(content string, startPos int) (string, bool) {
	if startPos >= len(content) || content[startPos] != '[' {
		return "", false
	}

	depth := 0
	inString := false
	escapeNext := false

	for i := startPos; i < len(content); i++ {
		char := content[i]

		// 处理转义字符
		if escapeNext {
			escapeNext = false
			continue
		}

		// 处理字符串内的字符
		if inString {
			if char == '\\' {
				escapeNext = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}

		// 检测字符串开始
		if char == '"' {
			inString = true
			continue
		}

		// 处理括号匹配（只在非字符串内处理）
		switch char {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				// 找到匹配的右括号
				extracted := content[startPos+1 : i]
				// 去除首尾空白字符
				extracted = strings.TrimSpace(extracted)
				return extracted, true
			}
		}
	}

	return "", false
}
