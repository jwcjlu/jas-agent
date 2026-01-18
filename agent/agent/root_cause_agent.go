package agent

import (
	"os"
	"strings"
	"time"

	"jas-agent/agent/core"

	"github.com/go-kratos/kratos/v2/log"
)

var rootCauseLogger = log.NewHelper(log.With(log.NewStdLogger(os.Stdout), "module", "agent/root_cause_agent"))

// RootCauseAgent 根因分析Agent
type RootCauseAgent struct {
	*BaseReact
	systemPrompt string
}

func (agent *RootCauseAgent) Type() AgentType {
	return RootCauseAgentType
}
func (agent *RootCauseAgent) IncludeMcpType(toolType core.ToolType) bool {
	return toolType == core.Normal
}
func NewRootCauseAgent(context *Context, executor *AgentExecutor, traceConfig string, logConfig string) Agent {
	// 获取所有相关工具
	allTools := context.toolManager.AvailableTools()
	var datas []core.ToolData
	for _, tool := range allTools {
		toolName := tool.Name()
		// 只添加Trace和日志相关工具
		if tool.Type() == core.Normal &&
			(strings.Contains(toolName, "trace") ||
				strings.Contains(toolName, "query_trace") ||
				strings.Contains(toolName, "search_documents") ||
				strings.Contains(toolName, "get_index_mapping") ||
				strings.Contains(toolName, "search_indices")) {
			datas = append(datas, core.ToolData{
				Name:        tool.Name(),
				Description: tool.Description(),
				Input:       tool.Input(),
			})
		}
	}

	rootCauseLogger.Infof("📋 根因分析Agent加载了 %d 个工具", len(datas))
	for _, tool := range datas {
		rootCauseLogger.Infof("  - %s", tool.Name)
	}

	systemPrompt := core.GetRootCauseSystemPrompt(core.RootCauseSystemPrompt{
		Date:        time.Now().Format("2006-01-02 15:04:05"),
		Tools:       datas,
		TraceConfig: traceConfig,
		LogConfig:   logConfig,
	})

	context.memory.AddMessage(core.Message{
		Role:    core.MessageRoleSystem,
		Content: systemPrompt,
	})
	baseReact := NewBaseReact(context, executor)
	baseReact.includeMcpType = func(toolType core.ToolType) bool {
		return toolType == core.Normal
	}
	return &RootCauseAgent{
		systemPrompt: systemPrompt,
		BaseReact:    baseReact,
	}
}

// NewRootCauseAgentExecutor 创建根因分析Agent执行器
func NewRootCauseAgentExecutor(context *Context, traceConfig string, logConfig string) *AgentExecutor {
	executor := NewAgentExecutor(context)
	rootCauseAgent := NewRootCauseAgent(context, executor, traceConfig, logConfig)
	executor.agent = rootCauseAgent
	return executor
}
