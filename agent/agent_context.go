package agent

import (
	"jas-agent/core"
	"jas-agent/llm"
	"jas-agent/memory"
	"jas-agent/tools"
)

type Context struct {
	agentType   AgentType
	model       string
	chat        llm.Chat
	toolManager *tools.ToolManager
	memory      core.Memory
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

func WithToolManager(tm *tools.ToolManager) Option {
	return func(context *Context) {
		context.toolManager = tm
	}
}

// GetToolManager 获取工具管理器
func (ctx *Context) GetToolManager() *tools.ToolManager {
	return ctx.toolManager
}
