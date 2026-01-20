package agent

import (
	"context"
	"fmt"
	"time"

	"jas-agent/agent/core"
	"strings"
)

type Agent interface {
	Type() AgentType
	Step() string
}

type AgentExecutor struct {
	context      *Context
	maxSteps     int
	currentStep  int
	state        State
	agent        Agent
	summaryAgent Agent
	traceCtx     context.Context   // 存储追踪context，用于日志关联
	eventBus     core.EventBus     // 事件总线
	stateManager core.StateManager // 状态管理器
	agentID      string            // Agent实例ID（用于状态管理）
}

func NewAgentExecutor(context *Context) *AgentExecutor {
	executor := &AgentExecutor{
		context:      context,
		maxSteps:     10,
		currentStep:  0,
		state:        IdleState,
		eventBus:     core.GetGlobalEventBus(),
		stateManager: core.GetGlobalStateManager(),
		agentID:      fmt.Sprintf("agent_%d", time.Now().UnixNano()), // 生成唯一ID
	}
	executor.agent = NewReactAgent(context, executor)
	executor.summaryAgent = NewSummaryAgent(context, executor)
	return executor
}

func NewSQLAgentExecutor(context *Context, dbInfo string) *AgentExecutor {
	executor := &AgentExecutor{
		context:      context,
		maxSteps:     15, // SQL查询可能需要更多步骤
		currentStep:  0,
		state:        IdleState,
		eventBus:     core.GetGlobalEventBus(),
		stateManager: core.GetGlobalStateManager(),
		agentID:      fmt.Sprintf("agent_%d", time.Now().UnixNano()), // 生成唯一ID
	}
	executor.agent = NewSQLAgent(context, executor, dbInfo)
	executor.summaryAgent = NewSummaryAgent(context, executor)
	return executor
}
func (agent *AgentExecutor) GetMemory() core.Memory {
	return agent.context.memory
}
func (agent *AgentExecutor) UpdateState(state State) {
	agent.state = state
}

// GetCurrentStep 获取当前步骤数
func (agent *AgentExecutor) GetCurrentStep() int {
	return agent.currentStep
}

// GetState 获取当前状态
func (agent *AgentExecutor) GetState() State {
	return agent.state
}

// GetMaxSteps 获取最大步骤数
func (agent *AgentExecutor) GetMaxSteps() int {
	return agent.maxSteps
}

// SetMaxSteps 设置最大步骤数
func (agent *AgentExecutor) SetMaxSteps(steps int) {
	agent.maxSteps = steps
}

func (agent *AgentExecutor) Run(query string) string {
	ctx := context.Background()
	startTime := time.Now()

	// 获取Agent类型
	agentType := string(agent.agent.Type())

	// 开始Agent执行追踪
	tracer := core.NewAgentTracer()
	ctx, span := tracer.StartAgentExecution(ctx, agentType, query)
	defer span.End()

	// 存储追踪context以便后续使用
	agent.traceCtx = ctx

	if len(query) > 0 {
		agent.context.memory.AddMessage(core.Message{
			Role:    core.MessageRoleUser,
			Content: query,
		})
	}

	agent.state = RunningState

	// 发布Agent开始事件
	if agent.eventBus != nil {
		agent.eventBus.Publish(ctx, core.EventAgentStarted, map[string]interface{}{
			"agent_id":   agent.agentID,
			"agent_type": agentType,
			"query":      query,
			"max_steps":  agent.maxSteps,
		})
	}

	// 保存初始状态快照
	agent.saveStateSnapshot(ctx, agentType, query)
	var results []string
	var lastErr error

	// 执行主要的 ReAct 循环
	for agent.currentStep < agent.maxSteps && agent.state != FinishState {
		agent.currentStep++

		// 记录步骤追踪
		stepStart := time.Now()
		stepCtx, stepSpan := tracer.StartAgentStep(ctx, agentType, "step")

		// 更新追踪context
		agent.traceCtx = stepCtx

		// 发布步骤开始事件
		if agent.eventBus != nil {
			agent.eventBus.Publish(stepCtx, core.EventAgentStepStarted, map[string]interface{}{
				"agent_id":    agent.agentID,
				"agent_type":  agentType,
				"step":        agent.currentStep,
				"total_steps": agent.maxSteps,
			})
		}

		result := agent.agent.Step()
		results = append(results, result)

		stepDuration := time.Since(stepStart)
		stepSpan.End()

		// 记录步骤指标
		if m := core.GetMetrics(); m != nil {
			m.RecordAgentStep(stepCtx, agentType, "step", stepDuration)
		}

		// 发布步骤完成事件
		if agent.eventBus != nil {
			agent.eventBus.Publish(stepCtx, core.EventAgentStepDone, map[string]interface{}{
				"agent_id":    agent.agentID,
				"agent_type":  agentType,
				"step":        agent.currentStep,
				"duration_ms": stepDuration.Milliseconds(),
				"result":      truncateString(result, 200),
				"success":     true,
			})
		}

		// 保存状态快照
		agent.saveStateSnapshot(stepCtx, agentType, query)

		// 检查是否完成
		if strings.Contains(strings.ToLower(result), "final answer") {
			agent.state = FinishState
			break
		}
	}

	if agent.currentStep >= agent.maxSteps {
		agent.state = ErrorState
		lastErr = context.DeadlineExceeded
	}

	// 如果启用了总结功能且执行完成，使用 SummarAgent 进行总结
	if agent.state == FinishState {
		summary := agent.summaryAgent.Step()

		// 记录成功指标
		duration := time.Since(startTime)
		if m := core.GetMetrics(); m != nil {
			m.RecordAgentExecution(ctx, agentType, duration, true)
		}
		tracer.RecordSuccess(span)

		// 发布Agent完成事件
		if agent.eventBus != nil {
			agent.eventBus.Publish(ctx, core.EventAgentFinished, map[string]interface{}{
				"agent_id":    agent.agentID,
				"agent_type":  agentType,
				"duration_ms": duration.Milliseconds(),
				"total_steps": agent.currentStep,
				"success":     true,
				"result":      truncateString(summary, 500),
			})
		}

		// 保存最终状态快照
		agent.saveStateSnapshot(ctx, agentType, query)

		return summary
	}

	// 记录执行指标
	duration := time.Since(startTime)
	success := agent.state == FinishState
	if m := core.GetMetrics(); m != nil {
		m.RecordAgentExecution(ctx, agentType, duration, success)
	}

	if lastErr != nil {
		tracer.RecordError(span, lastErr)
		// 发布错误事件
		if agent.eventBus != nil {
			agent.eventBus.Publish(ctx, core.EventAgentError, map[string]interface{}{
				"agent_id":     agent.agentID,
				"agent_type":   agentType,
				"duration_ms":  duration.Milliseconds(),
				"error":        lastErr.Error(),
				"current_step": agent.currentStep,
			})
		}
	} else if !success {
		tracer.RecordError(span, context.DeadlineExceeded)
		// 发布超时事件
		if agent.eventBus != nil {
			agent.eventBus.Publish(ctx, core.EventAgentError, map[string]interface{}{
				"agent_id":     agent.agentID,
				"agent_type":   agentType,
				"duration_ms":  duration.Milliseconds(),
				"error":        "max steps reached",
				"current_step": agent.currentStep,
			})
		}
	} else {
		tracer.RecordSuccess(span)
	}

	// 发布Agent完成事件（失败情况）
	if agent.state == ErrorState && agent.eventBus != nil {
		agent.eventBus.Publish(ctx, core.EventAgentFinished, map[string]interface{}{
			"agent_id":    agent.agentID,
			"agent_type":  agentType,
			"duration_ms": duration.Milliseconds(),
			"total_steps": agent.currentStep,
			"success":     false,
			"error":       lastErr,
		})
	}

	// 保存最终状态快照
	agent.saveStateSnapshot(ctx, agentType, query)

	if len(results) == 0 {
		return "No results generated"
	}
	return results[len(results)-1]
}

// saveStateSnapshot 保存状态快照
func (agent *AgentExecutor) saveStateSnapshot(ctx context.Context, agentType, query string) {
	if agent.stateManager == nil {
		return
	}

	// 构建状态数据
	stateData := map[string]interface{}{
		"agent_id":     agent.agentID,
		"agent_type":   agentType,
		"state":        string(agent.state),
		"current_step": agent.currentStep,
		"max_steps":    agent.maxSteps,
		"query":        query,
	}

	// 获取内存消息（限制数量以避免快照过大）
	messages := agent.context.memory.GetMessages()
	maxMessages := 20
	if len(messages) > maxMessages {
		messages = messages[len(messages)-maxMessages:]
	}
	stateData["memory"] = messages

	// 创建并保存快照
	snapshot, err := agent.stateManager.CreateSnapshot(ctx, agent.agentID, agentType, stateData)
	if err == nil && snapshot != nil {
		snapshot.State = string(agent.state)
		snapshot.CurrentStep = agent.currentStep
		snapshot.MaxSteps = agent.maxSteps
		snapshot.Query = query
		snapshot.Memory = messages
		_ = agent.stateManager.Save(ctx, snapshot)
	}
}

type State string

const (
	IdleState    State = "Idle"
	RunningState State = "Running"
	FinishState  State = "Finish"
	ErrorState   State = "Error"
)

type AgentType string

const (
	ReactAgentType     AgentType = "ReactAgent"
	PlanAgentType      AgentType = "PlanAgent"
	ChainAgentType     AgentType = "ChainAgent"
	SummaryAgentType   AgentType = "SummaryAgent"
	SQLAgentType       AgentType = "SQLAgent"
	ESAgentType        AgentType = "ESAgent"
	RootCauseAgentType AgentType = "RootCauseAgent"
	VMLogAgentType     AgentType = "VMLogAgent"
)
