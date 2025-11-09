package agent

import (
	"jas-agent/agent/core"
	"jas-agent/agent/llm"
	"jas-agent/agent/memory"
	"jas-agent/agent/tools"
)

type Context struct {
	agentType          AgentType
	model              string
	chat               llm.Chat
	toolManager        *tools.ToolManager
	memory             core.Memory
	msg                chan string
	allowedMCPServices []string
}

type Option func(*Context)

func NewContext(opts ...Option) *Context {
	ctx := &Context{
		agentType:   ReactAgentType,
		toolManager: tools.GetToolManager(),
		memory:      memory.NewMemory(),
	}
	for _, opt := range opts {
		opt(ctx)
	}
	return ctx
}

func WithModel(model string) Option {
	return func(context *Context) {
		context.model = model
	}
}

func WithChat(chat llm.Chat) Option {
	return func(context *Context) {
		context.chat = chat
	}
}

func WithMemory(memory core.Memory) Option {
	return func(context *Context) {
		context.memory = memory
	}
}
func WithMsg(msg chan string) Option {
	return func(context *Context) {
		context.msg = msg
	}
}
func WithToolManager(tm *tools.ToolManager) Option {
	return func(context *Context) {
		context.toolManager = tm
	}
}

// WithAllowedMCPServices 限制当前上下文可用的 MCP 服务前缀（service@tool）
func WithAllowedMCPServices(services []string) Option {
	return func(context *Context) {
		context.allowedMCPServices = append([]string(nil), services...)
	}
}

// GetToolManager 获取工具管理器
func (ctx *Context) GetToolManager() *tools.ToolManager {
	return ctx.toolManager
}

// GetMemory 获取内存
func (ctx *Context) GetMemory() core.Memory {
	return ctx.memory
}

// Send 发送消息
func (ctx *Context) Send(message string) {
	if ctx.msg == nil {
		return
	}
	ctx.msg <- message
}

// GetMemory 获取内存

// SetAllowedMCPServices 运行时更新允许的 MCP 服务列表
func (ctx *Context) SetAllowedMCPServices(services []string) {
	ctx.allowedMCPServices = append([]string(nil), services...)
}

func (ctx *Context) ResetToolManager(tm *tools.ToolManager) {
	ctx.toolManager = tm
}

// MCPFilter 返回一个用于 ToolManager.AvailableTools 的过滤器：
// - 当工具为 MCP 类型时，若其 service 前缀不在允许列表中，则排除（返回 true 表示排除）
// - 普通工具不过滤
func (ctx *Context) MCPFilter() core.FilterFunc {
	allowed := map[string]struct{}{}
	for _, s := range ctx.allowedMCPServices {
		allowed[s] = struct{}{}
	}
	return func(t core.Tool) bool {
		if t.Type() != core.Mcp {
			return false
		}
		name := t.Name()
		// name 形如: service@tool
		sep := "@"
		for i := 0; i < len(name); i++ {
			if string(name[i]) == sep {
				service := name[:i]
				if _, ok := allowed[service]; ok {
					return false // 不排除
				}
				return true // 排除不在 allow 列表的 MCP 工具
			}
		}
		// 没有前缀的不应出现，但为安全起见不过滤
		return false
	}
}
