package core

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("jas-agent")
)

// Metrics 指标收集器
type Metrics struct {
	// Agent指标
	agentExecutions        metric.Int64Counter
	agentExecutionDuration metric.Float64Histogram
	agentStepDuration      metric.Float64Histogram
	agentErrors            metric.Int64Counter

	// 工具指标
	toolCalls        metric.Int64Counter
	toolCallDuration metric.Float64Histogram
	toolErrors       metric.Int64Counter

	// LLM指标
	llmRequests        metric.Int64Counter
	llmRequestDuration metric.Float64Histogram
	llmErrors          metric.Int64Counter
	llmTokens          metric.Int64Counter

	// 数据库指标
	dbQueries       metric.Int64Counter
	dbQueryDuration metric.Float64Histogram
	dbErrors        metric.Int64Counter
}

var (
	globalMetrics *Metrics
)

// NewMetrics 创建指标收集器
func NewMetrics() (*Metrics, error) {
	agentExecutions, err := meter.Int64Counter(
		"agent_executions_total",
		metric.WithDescription("Total number of agent executions"),
	)
	if err != nil {
		return nil, err
	}

	agentExecutionDuration, err := meter.Float64Histogram(
		"agent_execution_duration_seconds",
		metric.WithDescription("Duration of agent executions in seconds"),
	)
	if err != nil {
		return nil, err
	}

	agentStepDuration, err := meter.Float64Histogram(
		"agent_step_duration_seconds",
		metric.WithDescription("Duration of agent steps in seconds"),
	)
	if err != nil {
		return nil, err
	}

	agentErrors, err := meter.Int64Counter(
		"agent_errors_total",
		metric.WithDescription("Total number of agent errors"),
	)
	if err != nil {
		return nil, err
	}

	toolCalls, err := meter.Int64Counter(
		"tool_calls_total",
		metric.WithDescription("Total number of tool calls"),
	)
	if err != nil {
		return nil, err
	}

	toolCallDuration, err := meter.Float64Histogram(
		"tool_call_duration_seconds",
		metric.WithDescription("Duration of tool calls in seconds"),
	)
	if err != nil {
		return nil, err
	}

	toolErrors, err := meter.Int64Counter(
		"tool_errors_total",
		metric.WithDescription("Total number of tool errors"),
	)
	if err != nil {
		return nil, err
	}

	llmRequests, err := meter.Int64Counter(
		"llm_requests_total",
		metric.WithDescription("Total number of LLM requests"),
	)
	if err != nil {
		return nil, err
	}

	llmRequestDuration, err := meter.Float64Histogram(
		"llm_request_duration_seconds",
		metric.WithDescription("Duration of LLM requests in seconds"),
	)
	if err != nil {
		return nil, err
	}

	llmErrors, err := meter.Int64Counter(
		"llm_errors_total",
		metric.WithDescription("Total number of LLM errors"),
	)
	if err != nil {
		return nil, err
	}

	llmTokens, err := meter.Int64Counter(
		"llm_tokens_total",
		metric.WithDescription("Total number of LLM tokens used"),
	)
	if err != nil {
		return nil, err
	}

	dbQueries, err := meter.Int64Counter(
		"db_queries_total",
		metric.WithDescription("Total number of database queries"),
	)
	if err != nil {
		return nil, err
	}

	dbQueryDuration, err := meter.Float64Histogram(
		"db_query_duration_seconds",
		metric.WithDescription("Duration of database queries in seconds"),
	)
	if err != nil {
		return nil, err
	}

	dbErrors, err := meter.Int64Counter(
		"db_errors_total",
		metric.WithDescription("Total number of database errors"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		agentExecutions:        agentExecutions,
		agentExecutionDuration: agentExecutionDuration,
		agentStepDuration:      agentStepDuration,
		agentErrors:            agentErrors,
		toolCalls:              toolCalls,
		toolCallDuration:       toolCallDuration,
		toolErrors:             toolErrors,
		llmRequests:            llmRequests,
		llmRequestDuration:     llmRequestDuration,
		llmErrors:              llmErrors,
		llmTokens:              llmTokens,
		dbQueries:              dbQueries,
		dbQueryDuration:        dbQueryDuration,
		dbErrors:               dbErrors,
	}, nil
}

// RecordAgentExecution 记录Agent执行
func (m *Metrics) RecordAgentExecution(ctx context.Context, agentType string, duration time.Duration, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("agent.type", agentType),
		attribute.Bool("success", success),
	}

	m.agentExecutions.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.agentExecutionDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if !success {
		m.agentErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordAgentStep 记录Agent步骤
func (m *Metrics) RecordAgentStep(ctx context.Context, agentType string, step string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("agent.type", agentType),
		attribute.String("agent.step", step),
	}

	m.agentStepDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordToolCall 记录工具调用
func (m *Metrics) RecordToolCall(ctx context.Context, toolName string, duration time.Duration, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("tool.name", toolName),
		attribute.Bool("success", success),
	}

	m.toolCalls.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.toolCallDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if !success {
		m.toolErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordLLMRequest 记录LLM请求
func (m *Metrics) RecordLLMRequest(ctx context.Context, model string, duration time.Duration, success bool, promptTokens, completionTokens int) {
	attrs := []attribute.KeyValue{
		attribute.String("llm.model", model),
		attribute.Bool("success", success),
	}

	m.llmRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.llmRequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if !success {
		m.llmErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	} else {
		totalTokens := promptTokens + completionTokens
		m.llmTokens.Add(ctx, int64(totalTokens), metric.WithAttributes(attrs...))
	}
}

// RecordDatabaseQuery 记录数据库查询
func (m *Metrics) RecordDatabaseQuery(ctx context.Context, operation string, table string, duration time.Duration, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("db.operation", operation),
		attribute.String("db.table", table),
		attribute.Bool("success", success),
	}

	m.dbQueries.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.dbQueryDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if !success {
		m.dbErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// InitGlobalMetrics 初始化全局指标
func InitGlobalMetrics() error {
	m, err := NewMetrics()
	if err != nil {
		return err
	}
	globalMetrics = m
	return nil
}

// GetMetrics 获取全局指标实例
func GetMetrics() *Metrics {
	if globalMetrics == nil {
		// 如果未初始化，尝试初始化
		if err := InitGlobalMetrics(); err != nil {
			// 如果初始化失败，返回nil（允许降级）
			return nil
		}
	}
	return globalMetrics
}
