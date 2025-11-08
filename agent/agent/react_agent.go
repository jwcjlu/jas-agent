package agent

import (
	"jas-agent/agent/core"
	"time"
)

type ReactAgent struct {
	systemPrompt string
	*BaseReact
}

func (agent *ReactAgent) Type() AgentType {
	return ReactAgentType
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
		systemPrompt: systemPrompt,
		BaseReact:    NewBaseReact(context, executor),
	}
}
