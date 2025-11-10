package agent

import (
	"context"
	"fmt"
	"jas-agent/agent/core"
	"jas-agent/agent/llm"
	"strings"
)

type SummaryAgent struct {
	context      *Context
	systemPrompt string
	executor     *AgentExecutor
}

func (agent *SummaryAgent) Type() AgentType {
	return SummaryAgentType
}

func (agent *SummaryAgent) Step() string {
	// 获取所有执行历史
	messages := agent.context.memory.GetMessages()

	// 构建总结上下文
	var executionLog strings.Builder
	executionLog.WriteString("执行过程总结:\n\n")

	for i, msg := range messages {
		switch msg.Role {
		case core.MessageRoleSystem:
			// 跳过系统提示词
			continue
		case core.MessageRoleUser:
			if strings.Contains(msg.Content, "Observation:") {
				executionLog.WriteString(fmt.Sprintf("观察 %d: %s\n", i, msg.Content))
			} else {
				executionLog.WriteString(fmt.Sprintf("用户查询: %s\n", msg.Content))
			}
		case core.MessageRoleAssistant:
			executionLog.WriteString(fmt.Sprintf("思考 %d: %s\n", i, msg.Content))
		}
	}

	// 添加总结提示
	executionLog.WriteString("\n请基于以上执行过程，提供简洁明了的最终答案。")

	// 创建总结请求
	summaryMessages := []core.Message{
		{
			Role:    core.MessageRoleSystem,
			Content: agent.systemPrompt,
		},
		{
			Role:    core.MessageRoleUser,
			Content: executionLog.String(),
		},
	}

	// 调用LLM进行总结
	resp, err := agent.context.chat.Completions(context.TODO(), llm.NewChatRequest(agent.context.model, summaryMessages))
	if err != nil {
		return fmt.Sprintf("总结生成失败: %s", err.Error())
	}

	// 更新状态为完成
	agent.executor.UpdateState(FinishState)

	return resp.Content()
}

func NewSummaryAgent(context *Context, executor *AgentExecutor) Agent {
	systemPrompt := core.GetSummarySystemPrompt()

	return &SummaryAgent{
		context:      context,
		systemPrompt: systemPrompt,
		executor:     executor,
	}
}
