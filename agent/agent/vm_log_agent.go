package agent

import (
	"os"
	"strings"
	"time"

	"jas-agent/agent/core"

	"github.com/go-kratos/kratos/v2/log"
)

var vmLogLogger = log.NewHelper(log.With(log.NewStdLogger(os.Stdout), "module", "agent/vm_log_agent"))

// VMLogAgent VM日志查询Agent
type VMLogAgent struct {
	*BaseReact
	systemPrompt string
}

func (agent *VMLogAgent) Type() AgentType {
	return VMLogAgentType
}

func NewVMLogAgent(context *Context, executor *AgentExecutor, dbInfo string) Agent {
	// 获取所有相关工具
	allTools := context.toolManager.AvailableTools()
	var datas []core.ToolData
	for _, tool := range allTools {
		toolName := tool.Name()
		// 添加SQL、HTTP和VM相关工具
		if tool.Type() == core.Normal &&
			(strings.Contains(toolName, "list_tables") ||
				strings.Contains(toolName, "tables_schema") ||
				strings.Contains(toolName, "execute_sql") ||
				strings.Contains(toolName, "http_request")) {
			datas = append(datas, core.ToolData{
				Name:        tool.Name(),
				Description: tool.Description(),
				Input:       tool.Input(),
			})
		}

	}

	vmLogLogger.Infof("📋 VM日志查询Agent加载了 %d 个工具", len(datas))
	for _, tool := range datas {
		vmLogLogger.Infof("  - %s", tool.Name)
	}

	systemPrompt := core.GetVMLogSystemPrompt(core.VMLogSystemPrompt{
		Date:         time.Now().Format("2006-01-02 15:04:05"),
		Tools:        datas,
		DatabaseInfo: dbInfo,
	})

	context.memory.AddMessage(core.Message{
		Role:    core.MessageRoleSystem,
		Content: systemPrompt,
	})

	return &VMLogAgent{
		systemPrompt: systemPrompt,
		BaseReact:    NewBaseReact(context, executor),
	}
}

// NewVMLogAgentExecutor 创建VM日志查询Agent执行器
func NewVMLogAgentExecutor(context *Context, dbInfo string) *AgentExecutor {
	executor := NewAgentExecutor(context)
	vmLogAgent := NewVMLogAgent(context, executor, dbInfo)
	executor.agent = vmLogAgent
	return executor
}
