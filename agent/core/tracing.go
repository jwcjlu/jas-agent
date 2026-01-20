package core

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer = otel.Tracer("jas-agent")
)

// AgentTracer Agent追踪器
type AgentTracer struct {
	tracer trace.Tracer
}

// NewAgentTracer 创建Agent追踪器
func NewAgentTracer() *AgentTracer {
	return &AgentTracer{
		tracer: tracer,
	}
}

// StartAgentStep 开始Agent步骤追踪
func (at *AgentTracer) StartAgentStep(ctx context.Context, agentType string, step string) (context.Context, trace.Span) {
	ctx, span := at.tracer.Start(ctx, fmt.Sprintf("agent.%s.%s", agentType, step))
	span.SetAttributes(
		attribute.String("agent.type", agentType),
		attribute.String("agent.step", step),
	)
	return ctx, span
}

// StartAgentExecution 开始Agent执行追踪
func (at *AgentTracer) StartAgentExecution(ctx context.Context, agentType string, query string) (context.Context, trace.Span) {
	ctx, span := at.tracer.Start(ctx, fmt.Sprintf("agent.%s.execute", agentType))
	span.SetAttributes(
		attribute.String("agent.type", agentType),
		attribute.String("agent.query", truncateString(query, 200)), // 限制长度避免过长
	)
	return ctx, span
}

// StartToolCall 开始工具调用追踪
func (at *AgentTracer) StartToolCall(ctx context.Context, toolName string, input string) (context.Context, trace.Span) {
	ctx, span := at.tracer.Start(ctx, fmt.Sprintf("tool.%s.call", toolName))
	span.SetAttributes(
		attribute.String("tool.name", toolName),
		attribute.String("tool.input", truncateString(input, 500)), // 限制长度
	)
	return ctx, span
}

// StartLLMRequest 开始LLM请求追踪
func (at *AgentTracer) StartLLMRequest(ctx context.Context, model string) (context.Context, trace.Span) {
	ctx, span := at.tracer.Start(ctx, "llm.request")
	span.SetAttributes(
		attribute.String("llm.model", model),
	)
	return ctx, span
}

// RecordLLMTokenUsage 记录LLM Token使用量
func (at *AgentTracer) RecordLLMTokenUsage(span trace.Span, promptTokens, completionTokens, totalTokens int) {
	if span != nil {
		span.SetAttributes(
			attribute.Int("llm.tokens.prompt", promptTokens),
			attribute.Int("llm.tokens.completion", completionTokens),
			attribute.Int("llm.tokens.total", totalTokens),
		)
	}
}

// StartDatabaseQuery 开始数据库查询追踪
func (at *AgentTracer) StartDatabaseQuery(ctx context.Context, operation string, table string) (context.Context, trace.Span) {
	ctx, span := at.tracer.Start(ctx, fmt.Sprintf("db.%s", operation))
	span.SetAttributes(
		attribute.String("db.operation", operation),
		attribute.String("db.table", table),
	)
	return ctx, span
}

// RecordError 记录错误
func (at *AgentTracer) RecordError(span trace.Span, err error) {
	if span != nil && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// RecordSuccess 记录成功
func (at *AgentTracer) RecordSuccess(span trace.Span) {
	if span != nil {
		span.SetStatus(codes.Ok, "success")
	}
}

// SetAttribute 设置Span属性
func (at *AgentTracer) SetAttribute(span trace.Span, key string, value interface{}) {
	if span != nil {
		switch v := value.(type) {
		case string:
			span.SetAttributes(attribute.String(key, v))
		case int:
			span.SetAttributes(attribute.Int(key, v))
		case int64:
			span.SetAttributes(attribute.Int64(key, v))
		case bool:
			span.SetAttributes(attribute.Bool(key, v))
		case float64:
			span.SetAttributes(attribute.Float64(key, v))
		default:
			span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
		}
	}
}

// truncateString 截断字符串以避免属性过长
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
