package agent

import (
	"jas-agent/agent/core"
	"jas-agent/agent/llm"
	"jas-agent/agent/memory"
	"jas-agent/agent/tools"
)

type Context struct {
	agentType   AgentType
	model       string
	chat        llm.Chat
	toolManager *tools.ToolManager
	memory      core.Memory
	msg         chan string
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
