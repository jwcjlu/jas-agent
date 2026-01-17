package framework

import (
	"fmt"
	"time"

	"jas-agent/agent/core"
	"jas-agent/agent/llm"
)

// Chat 获取 LLM 客户端
func (ctx *CollaborationContext) Chat() llm.Chat {
	return ctx.chat
}

// Memory 获取内存
func (ctx *CollaborationContext) Memory() core.Memory {
	return ctx.memory
}

// TraceID 获取追踪ID
func (ctx *CollaborationContext) TraceID() string {
	return ctx.traceID
}

// TenantID 获取租户ID
func (ctx *CollaborationContext) TenantID() string {
	return ctx.tenantID
}

// GetSharedData 获取共享数据
func (ctx *CollaborationContext) GetSharedData(key string) interface{} {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.sharedData[key]
}

// SetSharedData 设置共享数据
func (ctx *CollaborationContext) SetSharedData(key string, value interface{}) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.sharedData[key] = value
}

// GenerateTaskID 生成任务ID（包级别函数）
func GenerateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}
