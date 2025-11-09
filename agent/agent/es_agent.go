package agent

import (
	"fmt"
	"jas-agent/agent/core"
	"strings"
	"time"
)

type ESAgent struct {
	*BaseReact
	systemPrompt string
}

func (agent *ESAgent) Type() AgentType {
	return ESAgentType
}

func NewESAgent(context *Context, executor *AgentExecutor, clusterInfo string) Agent {
	// 获取 Elasticsearch 相关工具
	allTools := context.toolManager.AvailableTools()
	var datas []core.ToolData
	for _, tool := range allTools {
		// 只添加 ES 相关工具到提示词
		toolName := tool.Name()
		if tool.Type() == core.Normal &&
			(strings.Contains(toolName, "indice") ||
				strings.Contains(toolName, "index") ||
				strings.Contains(toolName, "document") ||
				strings.Contains(toolName, "search") ||
				strings.Contains(toolName, "aggregate") ||
				toolName == "list_indices" ||
				toolName == "search_indices" ||
				toolName == "get_index_mapping" ||
				toolName == "search_documents" ||
				toolName == "get_document" ||
				toolName == "aggregate_data") {
			datas = append(datas, core.ToolData{
				Name:        tool.Name(),
				Description: tool.Description(),
			})
		}
	}

	// 打印调试信息
	fmt.Printf("📋 ES Agent 加载了 %d 个工具：\n", len(datas))
	for _, tool := range datas {
		fmt.Printf("  - %s\n", tool.Name)
	}

	systemPrompt := core.GetESSystemPrompt(core.ESSystemPrompt{
		Date:        time.Now().Format("2006-01-02 15:04:05"),
		Tools:       datas,
		ClusterInfo: clusterInfo,
	})

	context.memory.AddMessage(core.Message{
		Role:    core.MessageRoleSystem,
		Content: systemPrompt,
	})

	return &ESAgent{
		systemPrompt: systemPrompt,
		BaseReact:    NewBaseReact(context, executor),
	}
}
