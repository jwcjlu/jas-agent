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
	context *Context

	maxSteps    int
	currentStep int
	state       State
	agent       Agent
}

func NewAgentExecutor(context *Context) *AgentExecutor {
	executor := &AgentExecutor{
		context:     context,
		maxSteps:    10,
		currentStep: 0,
		state:       IdleState,
	}
	executor.agent = NewReactAgent(context, executor)
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
	ReactAgentType AgentType = "ReactAgent"
	PlanAgentType  AgentType = "PlanAgent"
)
