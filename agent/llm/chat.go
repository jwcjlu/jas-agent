package llm

import (
	"context"
	"time"

	"jas-agent/agent/core"

	"github.com/sashabaranov/go-openai"
)

type Chat interface {
	Completions(ctx context.Context, chatReq ChatRequest) (*ChatResponse, error)
}

type openaiChat struct {
	client *openai.Client
}

type Config struct {
	ApiKey  string
	BaseURL string
}

func (config *Config) Enable() bool {
	return len(config.ApiKey) > 0 && len(config.BaseURL) > 0
}
func NewChat(chatConfig *Config) Chat {
	config := openai.DefaultConfig(chatConfig.ApiKey)
	if chatConfig.BaseURL != "" {
		config.BaseURL = chatConfig.BaseURL
	}
	return &openaiChat{client: openai.NewClientWithConfig(config)}
}

func (chat *openaiChat) Completions(ctx context.Context, chatReq ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()
	req := chatReq.Request()
	model := req.Model

	// 开始LLM请求追踪
	tracer := core.NewAgentTracer()
	ctx, span := tracer.StartLLMRequest(ctx, model)
	defer span.End()

	// 获取事件总线并发布LLM调用事件
	eventBus := core.GetGlobalEventBus()
	if eventBus != nil {
		eventBus.Publish(ctx, core.EventLLMCalled, map[string]interface{}{
			"model":      model,
			"prompt_len": len(req.Messages),
		})
	}

	// 调用API
	resp, err := chat.client.CreateChatCompletion(
		context.WithoutCancel(ctx),
		req,
	)

	duration := time.Since(startTime)

	if err != nil {
		// 记录错误
		tracer.RecordError(span, err)
		if m := core.GetMetrics(); m != nil {
			m.RecordLLMRequest(ctx, model, duration, false, 0, 0)
		}

		// 发布LLM调用失败事件
		eventBus := core.GetGlobalEventBus()
		if eventBus != nil {
			eventBus.Publish(ctx, core.EventLLMCompleted, map[string]interface{}{
				"model":       model,
				"duration_ms": duration.Milliseconds(),
				"success":     false,
				"error":       err.Error(),
			})
		}

		return nil, err
	}

	// 记录Token使用量
	promptTokens := resp.Usage.PromptTokens
	completionTokens := resp.Usage.CompletionTokens
	tracer.RecordLLMTokenUsage(span, promptTokens, completionTokens, resp.Usage.TotalTokens)
	tracer.RecordSuccess(span)

	// 记录成功指标
	if m := core.GetMetrics(); m != nil {
		m.RecordLLMRequest(ctx, model, duration, true, promptTokens, completionTokens)
	}

	// 发布LLM调用完成事件
	if eventBus != nil {
		eventBus.Publish(ctx, core.EventLLMCompleted, map[string]interface{}{
			"model":             model,
			"duration_ms":       duration.Milliseconds(),
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
			"success":           true,
		})
	}

	return &ChatResponse{resp}, nil
}
