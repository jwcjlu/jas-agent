package agent

import (
	"context"
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
	send               func(ctx context.Context, msg core.Message) error
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
func WithSend(send func(ctx context.Context, msg core.Message) error) Option {
	return func(context *Context) {
		context.send = send
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
func (ctx *Context) Send(c context.Context, message core.Message) {
	if ctx.send == nil {
		return
	}
	ctx.send(c, message)
}
