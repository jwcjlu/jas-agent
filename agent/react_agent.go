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
	tools        []*tools.ToolCall
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
		if tool.Type() == core.Normal {
			datas = append(datas, core.ToolData{
				Name:        tool.Name(),
				Description: tool.Description(),
			})
		}
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
		// 检查是否包含完成标记
		if strings.Contains(strings.ToLower(lastMessage.Content), "action: finish") ||
			strings.Contains(strings.ToLower(lastMessage.Content), "final answer") {
			agent.executor.UpdateState(FinishState)
			return false // 思考完成
		}
	}
	tools := agent.context.toolManager.AvailableTools()
	var ts []core.Tool
	for _, tool := range tools {
		if tool.Type() == core.Mcp {
			ts = append(ts, tool)
		}
	}
	// 调用LLM进行思考
	resp, err := agent.context.chat.Completions(context.TODO(), llm.NewChatRequest(agent.context.model, agent.context.memory.GetMessages(), ts...))
	if err != nil {
		// 添加错误消息
		agent.context.memory.AddMessage(core.Message{
			Role:    core.MessageRoleAssistant,
			Content: fmt.Sprintf("Error during thinking: %s", err.Error()),
		})
		return false
	}
	fmt.Println(resp.Content())
	// 添加助手的思考结果
	agent.context.memory.AddMessage(core.Message{
		Role:    core.MessageRoleAssistant,
		Content: resp.Content(),
	})

	// 检查是否是Finish命令
	if strings.Contains(strings.ToLower(resp.Content()), "action: finish") ||
		strings.Contains(strings.ToLower(resp.Content()), "final answer") {
		agent.executor.UpdateState(FinishState)
		return true // Finish也需要执行Action
	}
	agent.tools = resp.GetToolCalls()
	return agent.tools != nil
}

func (agent *ReactAgent) Action() string {
	lastMessage := agent.context.memory.GetLastMessage()
	if lastMessage.Role != core.MessageRoleAssistant {
		return "No action needed - waiting for assistant response"
	}
	// 检查是否是Finish命令
	if strings.Contains(strings.ToLower(lastMessage.Content), "action: finish") {
		// 提取最终答案
		re := regexp.MustCompile(`Action:\s*Finish\[([^\]]+)\]`)
		matches := re.FindStringSubmatch(lastMessage.Content)
		if len(matches) >= 2 {
			finalAnswer := strings.TrimSpace(matches[1])
			return fmt.Sprintf("Final answer: %s", finalAnswer)
		}
		return "Task completed"
	}

	toolCalls := agent.tools
	if len(toolCalls) == 0 {
		return "No tool call found in assistant response"
	}
	exeResult := ""
	for _, toolCall := range toolCalls {
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
		exeResult = fmt.Sprintf("Executed %s with result: %s", toolCall.Name, result)
	}

	return exeResult
}
