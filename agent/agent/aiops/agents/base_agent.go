package agents

import (
	"context"
	"fmt"
	"jas-agent/agent/agent/aiops/framework"

	"jas-agent/agent/core"
	"jas-agent/agent/llm"
)

// BaseAgent 智能体基类
// 提供通用的 ReAct 模式实现
type BaseAgent struct {
	role        framework.AgentRole
	name        string
	chat        llm.Chat
	memory      core.Memory
	description string
	tools       []Tool
}

// NewBaseAgent 创建基础智能体
func NewBaseAgent(
	role framework.AgentRole,
	name, description string,
	chat llm.Chat,
	memory core.Memory,
) *BaseAgent {
	return &BaseAgent{
		role:        role,
		name:        name,
		chat:        chat,
		memory:      memory,
		description: description,
		tools:       make([]Tool, 0),
	}
}

// Role 返回智能体角色
func (a *BaseAgent) Role() framework.AgentRole {
	return a.role
}

// Name 返回智能体名称
func (a *BaseAgent) Name() string {
	return a.name
}

// CanHandle 检查是否可以处理任务（子类可覆盖）
func (a *BaseAgent) CanHandle(task *framework.Task) bool {
	// 默认实现：根据任务类型匹配
	switch a.role {
	case framework.RoleMetrics:
		return task.Type == framework.TaskTypeMetricsAnalysis
	case framework.RoleLogs:
		return task.Type == framework.TaskTypeLogsAnalysis
	case framework.RoleTopology:
		return task.Type == framework.TaskTypeTopologyAnalysis
	case framework.RoleDecision:
		return task.Type == framework.TaskTypeDecision
	case framework.RoleOutput:
		return task.Type == framework.TaskTypeOutput
	case framework.RolePlanner:
		return task.Type == framework.TaskTypeRootCauseAnalysis
	default:
		return false
	}
}

// RegisterTool 注册工具
func (a *BaseAgent) RegisterTool(tool Tool) {
	a.tools = append(a.tools, tool)
}

// Execute 执行任务（ReAct 模式）
func (a *BaseAgent) Execute(ctx context.Context, task *framework.Task) (*framework.TaskResult, error) {
	// 1. Thought: 思考阶段
	thought, err := a.think(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("think failed: %w", err)
	}

	// 2. Action: 行动阶段
	action, err := a.act(ctx, task, thought)
	if err != nil {
		return nil, fmt.Errorf("act failed: %w", err)
	}

	// 3. Observation: 观察阶段
	observation, err := a.observe(ctx, task, action)
	if err != nil {
		return nil, fmt.Errorf("observe failed: %w", err)
	}

	// 4. 构建结果
	result := a.buildResult(task, thought, action, observation)

	return result, nil
}

// think 思考阶段（子类可覆盖）
func (a *BaseAgent) think(ctx context.Context, task *framework.Task) (map[string]interface{}, error) {
	// 默认实现：使用 LLM 进行思考
	prompt := a.buildThoughtPrompt(task)

	resp, err := a.chat.Completions(ctx, llm.NewChatRequest(
		"gpt-3.5-turbo",
		[]core.Message{
			{
				Role:    core.MessageRoleSystem,
				Content: a.description,
			},
			{
				Role:    core.MessageRoleUser,
				Content: prompt,
			},
		},
	))
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"thought": resp.Content(),
	}, nil
}

// act 行动阶段（子类可覆盖）
func (a *BaseAgent) act(ctx context.Context, task *framework.Task, thought map[string]interface{}) (map[string]interface{}, error) {
	// 默认实现：不执行具体行动，由子类实现
	return map[string]interface{}{}, nil
}

// observe 观察阶段（子类可覆盖）
func (a *BaseAgent) observe(ctx context.Context, task *framework.Task, action map[string]interface{}) (map[string]interface{}, error) {
	// 默认实现：返回空观察
	return map[string]interface{}{}, nil
}

// buildThoughtPrompt 构建思考提示词（子类可覆盖）
func (a *BaseAgent) buildThoughtPrompt(task *framework.Task) string {
	return fmt.Sprintf("任务类型: %s\n查询: %s\n时间范围: %d - %d\n相关服务: %v\n请分析此任务并制定执行计划。",
		task.Type, task.Query, task.TimeRange.StartTime, task.TimeRange.EndTime, task.Services)
}

// buildResult 构建任务结果
func (a *BaseAgent) buildResult(
	task *framework.Task,
	thought map[string]interface{},
	action map[string]interface{},
	observation map[string]interface{},
) *framework.TaskResult {
	return &framework.TaskResult{
		TaskID:    task.ID,
		AgentRole: a.role,
		Success:   true,
		Evidence:  make([]framework.Evidence, 0),
		Findings:  make([]framework.Finding, 0),
		Metadata: map[string]interface{}{
			"thought":     thought,
			"action":      action,
			"observation": observation,
		},
	}
}

// Tool 工具接口
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}
