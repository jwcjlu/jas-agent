package agent

import (
	"jas-agent/agent/core"
	"strings"
	"time"
)

type SQLAgent struct {
	*BaseReact
	systemPrompt string
}

func (agent *SQLAgent) Type() AgentType {
	return SQLAgentType
}

func NewSQLAgent(context *Context, executor *AgentExecutor, dbInfo string) Agent {
	// 获取 SQL 相关工具
	allTools := context.toolManager.AvailableTools()
	var datas []core.ToolData
	for _, tool := range allTools {
		// 只添加 SQL 相关工具到提示词
		if tool.Type() == core.Normal &&
			(strings.Contains(tool.Name(), "sql") ||
				strings.Contains(tool.Name(), "table") ||
				strings.Contains(tool.Name(), "schema")) {
			datas = append(datas, core.ToolData{
				Name:        tool.Name(),
				Description: tool.Description(),
			})
		}
	}

	systemPrompt := core.GetSQLSystemPrompt(core.SQLSystemPrompt{
		Date:         time.Now().Format("2006-01-02 15:04:05"),
		Tools:        datas,
		DatabaseInfo: dbInfo,
	})

	context.memory.AddMessage(core.Message{
		Role:    core.MessageRoleSystem,
		Content: systemPrompt,
	})

	return &SQLAgent{
		systemPrompt: systemPrompt,
		BaseReact:    NewBaseReact(context, executor),
	}
}
