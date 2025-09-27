package agent

import (
	"context"
	"fmt"
	"jas-agent/core"
	"jas-agent/llm"
	"jas-agent/tools"
	"regexp"
	"strings"
	"time"
)

type ReactAgent struct {
	context      *Context
	systemPrompt string
	tools        *tools.ToolCall
	executor     *AgentExecutor
}

func (agent *ReactAgent) Type() AgentType {
	return ReactAgentType
}

func (agent *ReactAgent) Step() string {
	shouldAct := agent.Thought()
	if !shouldAct {
		return "Thinking complete - no action needed"
	}
	return agent.Action()
}

func NewReactAgent(context *Context, executor *AgentExecutor) Agent {
	tools := context.toolManager.AvailableTools()
	var datas []core.ToolData
	for _, tool := range tools {
		datas = append(datas, core.ToolData{
			Name:        tool.Name(),
			Description: tool.Description(),
		})
	}
	systemPrompt := core.GetReactSystemPrompt(core.ReactSystemPrompt{
		Date:  time.Now().Format("2025-09-11 12:23:23"),
		Tools: datas,
	})
	context.memory.AddMessage(core.Message{
		Role:    core.MessageRoleSystem,
		Content: systemPrompt,
	})

	return &ReactAgent{
		context:      context,
		systemPrompt: systemPrompt,
		executor:     executor,
	}
}
func (agent *ReactAgent) Thought() bool {
	// 检查是否需要继续思考
	lastMessage := agent.context.memory.GetLastMessage()
	if lastMessage.Role == core.MessageRoleAssistant {
		// 如果最后一条消息是助手的，检查是否包含工具调用
		if agent.parseToolCall(lastMessage.Content) != nil {
			return true // 需要执行工具
		}
		// 检查是否包含完成标记
		if strings.Contains(strings.ToLower(lastMessage.Content), "action: finish") ||
			strings.Contains(strings.ToLower(lastMessage.Content), "final answer") {
			agent.executor.UpdateState(FinishState)
			return false // 思考完成
		}
	}

	// 调用LLM进行思考
	resp, err := agent.context.chat.Completions(context.TODO(), llm.NewChatRequest(agent.context.model, agent.context.memory.GetMessages()))
	if err != nil {
		// 添加错误消息
		agent.context.memory.AddMessage(core.Message{
			Role:    core.MessageRoleAssistant,
			Content: fmt.Sprintf("Error during thinking: %s", err.Error()),
		})
		return false
	}

	// 添加助手的思考结果
	agent.context.memory.AddMessage(core.Message{
		Role:    core.MessageRoleAssistant,
		Content: resp.Content(),
	})

	// 检查是否需要执行工具
	toolCall := agent.parseToolCall(resp.Content())

	// 检查是否是Finish命令
	if strings.Contains(strings.ToLower(resp.Content()), "action: Finish") {
		return true // Finish也需要执行Action
	}
	agent.tools = toolCall
	return toolCall != nil
}

func (agent *ReactAgent) Action() string {
	lastMessage := agent.context.memory.GetLastMessage()
	if lastMessage.Role != core.MessageRoleAssistant {
		return "No action needed - waiting for assistant response"
	}
	toolCall := agent.tools
	// 执行工具
	result, err := agent.context.toolManager.ExecTool(context.Background(), toolCall)
	if err != nil {
		// 添加错误观察
		agent.context.memory.AddMessage(core.Message{
			Role:    core.MessageRoleUser,
			Content: fmt.Sprintf("Tool execution error: %s", err.Error()),
		})
		return fmt.Sprintf("Tool execution failed: %s", err.Error())
	}

	// 添加观察结果
	agent.context.memory.AddMessage(core.Message{
		Role:    core.MessageRoleUser,
		Content: fmt.Sprintf("Observation: %s", result),
	})

	return fmt.Sprintf("Executed %s with result: %s", toolCall.Name, result)
}

// parseToolCall 解析助手响应中的工具调用
func (agent *ReactAgent) parseToolCall(content string) *tools.ToolCall {
	// 匹配格式: Action: toolName[input]
	re := regexp.MustCompile(`Action:\s*(\w+)\[([^\]]+)\]`)
	matches := re.FindStringSubmatch(content)

	if len(matches) < 3 {
		// 尝试匹配其他格式: toolName(input) 或 toolName: input
		re2 := regexp.MustCompile(`(\w+)\s*[\[\(]\s*([^\]\)]+)\s*[\]\)]`)
		matches2 := re2.FindStringSubmatch(content)
		if len(matches2) >= 3 {
			toolName := strings.TrimSpace(matches2[1])
			// 检查是否是Finish命令
			if strings.ToLower(toolName) == "finish" {
				return nil // Finish不是工具调用
			}
			return &tools.ToolCall{
				Name:  toolName,
				Input: strings.TrimSpace(matches2[2]),
			}
		}
		return nil
	}

	toolName := strings.TrimSpace(matches[1])
	// 检查是否是Finish命令
	if strings.ToLower(toolName) == "finish" {
		return nil // Finish不是工具调用
	}

	return &tools.ToolCall{
		Name:  toolName,
		Input: strings.TrimSpace(matches[2]),
	}
}
