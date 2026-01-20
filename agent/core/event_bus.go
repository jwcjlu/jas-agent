package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// EventType 事件类型
type EventType string

const (
	// EventAgentStarted Agent执行开始
	EventAgentStarted EventType = "agent.started"
	// EventAgentStepDone Agent步骤完成
	EventAgentStepDone EventType = "agent.step.done"
	// EventAgentStepStarted Agent步骤开始
	EventAgentStepStarted EventType = "agent.step.started"
	// EventToolCalled 工具调用
	EventToolCalled EventType = "tool.called"
	// EventToolCompleted 工具调用完成
	EventToolCompleted EventType = "tool.completed"
	// EventLLMCalled LLM调用
	EventLLMCalled EventType = "llm.called"
	// EventLLMCompleted LLM调用完成
	EventLLMCompleted EventType = "llm.completed"
	// EventAgentFinished Agent执行完成
	EventAgentFinished EventType = "agent.finished"
	// EventAgentError Agent执行错误
	EventAgentError EventType = "agent.error"
	// EventAgentStateChanged Agent状态变更
	EventAgentStateChanged EventType = "agent.state.changed"
)

// Event 事件结构
type Event struct {
	Type      EventType   `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
	TraceID   string      `json:"trace_id,omitempty"`
	SpanID    string      `json:"span_id,omitempty"`
}

// EventHandler 事件处理器
type EventHandler func(ctx context.Context, event *Event) error

// EventBus 事件总线接口
type EventBus interface {
	// Publish 发布事件
	Publish(ctx context.Context, eventType EventType, payload interface{})
	// Subscribe 订阅事件
	Subscribe(eventType EventType, handler EventHandler)
	// Unsubscribe 取消订阅
	Unsubscribe(eventType EventType, handler EventHandler)
	// SubscribeAll 订阅所有事件
	SubscribeAll(handler EventHandler)
	// Close 关闭事件总线
	Close() error
}

// DefaultEventBus 默认事件总线实现
type DefaultEventBus struct {
	mu             sync.RWMutex
	handlers       map[EventType][]EventHandler
	allHandlers    []EventHandler
	eventChan      chan *Event
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	bufferSize     int
	maxConcurrency int
}

// EventBusOption 事件总线选项
type EventBusOption func(*DefaultEventBus)

// WithBufferSize 设置事件缓冲区大小
func WithBufferSize(size int) EventBusOption {
	return func(eb *DefaultEventBus) {
		eb.bufferSize = size
	}
}

// WithMaxConcurrency 设置最大并发处理数
func WithMaxConcurrency(n int) EventBusOption {
	return func(eb *DefaultEventBus) {
		eb.maxConcurrency = n
	}
}

// NewEventBus 创建新的事件总线
func NewEventBus(opts ...EventBusOption) *DefaultEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	eb := &DefaultEventBus{
		handlers:       make(map[EventType][]EventHandler),
		allHandlers:    make([]EventHandler, 0),
		eventChan:      make(chan *Event, 100),
		ctx:            ctx,
		cancel:         cancel,
		bufferSize:     100,
		maxConcurrency: 10,
	}

	for _, opt := range opts {
		opt(eb)
	}

	// 启动事件处理协程
	eb.startEventProcessor()

	return eb
}

// startEventProcessor 启动事件处理器
func (eb *DefaultEventBus) startEventProcessor() {
	eb.wg.Add(1)
	go func() {
		defer eb.wg.Done()
		semaphore := make(chan struct{}, eb.maxConcurrency)
		for {
			select {
			case <-eb.ctx.Done():
				return
			case event := <-eb.eventChan:
				semaphore <- struct{}{}
				eb.wg.Add(1)
				go func(e *Event) {
					defer func() {
						<-semaphore
						eb.wg.Done()
					}()
					eb.processEvent(e)
				}(event)
			}
		}
	}()
}

// processEvent 处理事件
func (eb *DefaultEventBus) processEvent(event *Event) {
	ctx := context.Background()
	if event.TraceID != "" {
		// 可以在这里注入TraceID到context
		// 简化实现，直接使用background context
	}

	// 调用特定事件类型的处理器
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	allHandlers := eb.allHandlers
	eb.mu.RUnlock()

	// 执行特定类型的处理器
	for _, handler := range handlers {
		func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					// 记录panic但不影响其他处理器
					_ = r
				}
			}()
			_ = h(ctx, event)
		}(handler)
	}

	// 执行全局处理器
	for _, handler := range allHandlers {
		func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					// 记录panic但不影响其他处理器
					_ = r
				}
			}()
			_ = h(ctx, event)
		}(handler)
	}
}

// Publish 发布事件
func (eb *DefaultEventBus) Publish(ctx context.Context, eventType EventType, payload interface{}) {
	event := &Event{
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	// 尝试从context提取TraceID和SpanID
	if span := trace.SpanFromContext(ctx); span != nil {
		spanCtx := span.SpanContext()
		if spanCtx.IsValid() {
			event.TraceID = spanCtx.TraceID().String()
			event.SpanID = spanCtx.SpanID().String()
		}
	}

	select {
	case eb.eventChan <- event:
		// 事件已发送
	case <-eb.ctx.Done():
		// 事件总线已关闭
	default:
		// 缓冲区满，丢弃事件（可以根据需要改为阻塞或记录警告）
	}
}

// Subscribe 订阅事件
func (eb *DefaultEventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// Unsubscribe 取消订阅
func (eb *DefaultEventBus) Unsubscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers := eb.handlers[eventType]
	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			eb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// SubscribeAll 订阅所有事件
func (eb *DefaultEventBus) SubscribeAll(handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.allHandlers = append(eb.allHandlers, handler)
}

// Close 关闭事件总线
func (eb *DefaultEventBus) Close() error {
	eb.cancel()
	close(eb.eventChan)
	eb.wg.Wait()
	return nil
}

var (
	globalEventBus EventBus
	eventBusOnce   sync.Once
)

// GetGlobalEventBus 获取全局事件总线
func GetGlobalEventBus() EventBus {
	eventBusOnce.Do(func() {
		globalEventBus = NewEventBus(
			WithBufferSize(100),
			WithMaxConcurrency(10),
		)
	})
	return globalEventBus
}

// SetGlobalEventBus 设置全局事件总线
func SetGlobalEventBus(eb EventBus) {
	globalEventBus = eb
}
