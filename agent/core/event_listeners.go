package core

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

// LoggingEventListener 日志事件监听器
type LoggingEventListener struct {
	logger log.Logger
	helper *log.Helper
}

// NewLoggingEventListener 创建日志事件监听器
func NewLoggingEventListener(logger log.Logger) *LoggingEventListener {
	return &LoggingEventListener{
		logger: logger,
		helper: log.NewHelper(logger),
	}
}

// Handle 处理事件
func (l *LoggingEventListener) Handle(ctx context.Context, event *Event) error {
	// 构建日志字段
	fields := []interface{}{
		"event_type", string(event.Type),
		"timestamp", event.Timestamp,
	}

	if event.TraceID != "" {
		fields = append(fields, "trace_id", event.TraceID)
	}
	if event.SpanID != "" {
		fields = append(fields, "span_id", event.SpanID)
	}

	// 序列化payload
	if event.Payload != nil {
		payloadJSON, err := json.Marshal(event.Payload)
		if err == nil {
			fields = append(fields, "payload", string(payloadJSON))
		} else {
			fields = append(fields, "payload", fmt.Sprintf("%v", event.Payload))
		}
	}

	// 构建消息字符串
	msg := fmt.Sprintf("Event: %s", event.Type)
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			msg += fmt.Sprintf(" %v=%v", fields[i], fields[i+1])
		}
	}

	// 根据事件类型选择日志级别
	switch event.Type {
	case EventAgentError:
		l.helper.Error(msg)
	case EventAgentFinished, EventAgentStepDone, EventToolCompleted, EventLLMCompleted:
		l.helper.Info(msg)
	default:
		l.helper.Debug(msg)
	}

	return nil
}

// MetricsEventListener 指标事件监听器
type MetricsEventListener struct {
	metrics *Metrics
}

// NewMetricsEventListener 创建指标事件监听器
func NewMetricsEventListener(metrics *Metrics) *MetricsEventListener {
	return &MetricsEventListener{
		metrics: metrics,
	}
}

// Handle 处理事件
func (m *MetricsEventListener) Handle(ctx context.Context, event *Event) error {
	if m.metrics == nil {
		return nil
	}

	// 根据事件类型更新指标
	switch event.Type {
	case EventAgentStarted:
		// Agent开始执行已在指标中记录
	case EventAgentFinished:
		// Agent完成执行已在指标中记录
	case EventAgentError:
		// 错误已在指标中记录
	case EventToolCalled:
		// 工具调用已在指标中记录
	case EventLLMCalled:
		// LLM调用已在指标中记录
	}

	return nil
}

// StateSnapshotEventListener 状态快照事件监听器
type StateSnapshotEventListener struct {
	stateManager StateManager
}

// NewStateSnapshotEventListener 创建状态快照事件监听器
func NewStateSnapshotEventListener(stateManager StateManager) *StateSnapshotEventListener {
	return &StateSnapshotEventListener{
		stateManager: stateManager,
	}
}

// Handle 处理事件
func (s *StateSnapshotEventListener) Handle(ctx context.Context, event *Event) error {
	if s.stateManager == nil {
		return nil
	}

	// 只在关键事件时创建快照
	switch event.Type {
	case EventAgentStepDone, EventAgentFinished, EventAgentError:
		if payload, ok := event.Payload.(map[string]interface{}); ok {
			if agentID, ok := payload["agent_id"].(string); ok {
				agentType, _ := payload["agent_type"].(string)
				snapshot, err := s.stateManager.CreateSnapshot(ctx, agentID, agentType, payload)
				if err == nil {
					_ = s.stateManager.Save(ctx, snapshot)
				}
			}
		}
	}

	return nil
}

// SetupDefaultEventListeners 设置默认事件监听器
func SetupDefaultEventListeners(logger log.Logger, metrics *Metrics, stateManager StateManager) {
	bus := GetGlobalEventBus()

	// 日志监听器
	if logger != nil {
		loggingListener := NewLoggingEventListener(logger)
		bus.SubscribeAll(func(ctx context.Context, event *Event) error {
			return loggingListener.Handle(ctx, event)
		})
	}

	// 指标监听器
	if metrics != nil {
		metricsListener := NewMetricsEventListener(metrics)
		bus.SubscribeAll(func(ctx context.Context, event *Event) error {
			return metricsListener.Handle(ctx, event)
		})
	}

	// 状态快照监听器
	if stateManager != nil {
		stateListener := NewStateSnapshotEventListener(stateManager)
		bus.SubscribeAll(func(ctx context.Context, event *Event) error {
			return stateListener.Handle(ctx, event)
		})
	}
}
