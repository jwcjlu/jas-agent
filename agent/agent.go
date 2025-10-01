package agent

import (
	"jas-agent/core"
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
}

func NewAgentExecutor(context *Context) *AgentExecutor {
	executor := &AgentExecutor{
		context:     context,
		maxSteps:    10,
		currentStep: 0,
		state:       IdleState,
	}
	executor.agent = NewReactAgent(context, executor)
	executor.summaryAgent = NewSummaryAgent(context, executor)
	return executor
}

func NewSQLAgentExecutor(context *Context, dbInfo string) *AgentExecutor {
	executor := &AgentExecutor{
		context:     context,
		maxSteps:    15, // SQL查询可能需要更多步骤
		currentStep: 0,
		state:       IdleState,
	}
	executor.agent = NewSQLAgent(context, executor, dbInfo)
	executor.summaryAgent = NewSummaryAgent(context, executor)
	return executor
}

func (agent *AgentExecutor) UpdateState(state State) {
	agent.state = state
}

func (agent *AgentExecutor) Run(query string) string {
	if len(query) > 0 {
		agent.context.memory.AddMessage(core.Message{
			Role:    core.MessageRoleUser,
			Content: query,
		})
	}

	agent.state = RunningState
	var results []string

	// 执行主要的 ReAct 循环
	for agent.currentStep < agent.maxSteps && agent.state != FinishState {
		agent.currentStep++
		result := agent.agent.Step()
		results = append(results, result)

		// 检查是否完成
		if strings.Contains(strings.ToLower(result), "final answer") {
			agent.state = FinishState
			break
		}
	}

	if agent.currentStep >= agent.maxSteps {
		agent.state = ErrorState
	}

	// 如果启用了总结功能且执行完成，使用 SummarAgent 进行总结
	if agent.state == FinishState {
		summary := agent.summaryAgent.Step()
		return summary
	}

	if len(results) == 0 {
		return "No results generated"
	}
	return results[len(results)-1]
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
	ReactAgentType   AgentType = "ReactAgent"
	PlanAgentType    AgentType = "PlanAgent"
	SummaryAgentType AgentType = "SummarAgent"
	SQLAgentType     AgentType = "SQLAgent"
)
