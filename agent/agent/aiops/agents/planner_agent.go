package agents

import (
	"context"
	"fmt"
	framework2 "jas-agent/agent/agent/aiops/framework"

	"jas-agent/agent/core"
	"jas-agent/agent/llm"
)

// PlannerAgent 任务规划智能体
// 作为"总指挥"，接收异常告警后，生成根因分析计划
type PlannerAgent struct {
	*BaseAgent
}

// NewPlannerAgent 创建任务规划智能体
func NewPlannerAgent(ctx *framework2.CollaborationContext) *PlannerAgent {
	base := NewBaseAgent(
		framework2.RolePlanner,
		"任务规划智能体",
		`你是一个专业的运维故障分析规划专家。你的职责是：
1. 接收异常告警或故障查询
2. 制定根因分析计划（如"先确认指标异常范围→下钻追踪链路→查询错误日志"）
3. 将复杂任务分解为多个子任务
4. 调度其他智能体工作

分析计划应该遵循以下原则：
- 先分析指标异常，确定影响范围
- 然后追踪调用链路，找出关键路径
- 接着分析错误日志，提取错误模式
- 最后进行拓扑分析，确定根因服务
- 综合所有证据，生成根因假设`,
		ctx.Chat(),
		ctx.Memory(),
	)

	agent := &PlannerAgent{BaseAgent: base}
	return agent
}

// Execute 执行任务规划
func (a *PlannerAgent) Execute(ctx context.Context, task *framework2.Task) (*framework2.TaskResult, error) {
	// 1. 分析任务和上下文
	analysis := a.analyzeTask(task)

	// 2. 生成分析计划
	plan := a.generatePlan(ctx, task, analysis)

	// 3. 分解为子任务
	subTasks := a.decomposeTasks(task, plan)

	// 4. 构建结果
	result := &framework2.TaskResult{
		TaskID:     task.ID,
		AgentRole:  framework2.RolePlanner,
		Success:    true,
		Evidence:   make([]framework2.Evidence, 0),
		Findings:   make([]framework2.Finding, 0),
		Confidence: 0.9, // 规划阶段的置信度较高
		Metadata: map[string]interface{}{
			"analysis":  analysis,
			"plan":      plan,
			"sub_tasks": subTasks,
		},
		NextActions: []string{
			"执行指标分析任务",
			"执行日志分析任务",
			"执行拓扑分析任务",
		},
	}

	return result, nil
}

// analyzeTask 分析任务
func (a *PlannerAgent) analyzeTask(task *framework2.Task) map[string]interface{} {
	analysis := make(map[string]interface{})

	// 分析告警信息
	if len(task.Alerts) > 0 {
		criticalCount := 0
		highCount := 0
		services := make(map[string]bool)

		for _, alert := range task.Alerts {
			if alert.Severity == "CRITICAL" {
				criticalCount++
			} else if alert.Severity == "HIGH" {
				highCount++
			}
			services[alert.Service] = true
		}

		analysis["alert_count"] = len(task.Alerts)
		analysis["critical_count"] = criticalCount
		analysis["high_count"] = highCount
		analysis["affected_services"] = keys(services)
	}

	// 分析时间范围
	timeWindow := task.TimeRange.EndTime - task.TimeRange.StartTime
	analysis["time_window_seconds"] = timeWindow
	analysis["is_recent"] = timeWindow < 3600 // 1小时内

	// 分析服务数量
	analysis["service_count"] = len(task.Services)

	return analysis
}

// generatePlan 生成分析计划
func (a *PlannerAgent) generatePlan(ctx context.Context, task *framework2.Task, analysis map[string]interface{}) string {
	// 使用 LLM 生成分析计划
	prompt := fmt.Sprintf(`基于以下信息生成根因分析计划：

查询: %s
时间范围: %d - %d
相关服务: %v
告警数量: %v
关键告警数量: %v

请生成详细的分析计划，包括：
1. 第一步：指标分析（确认异常范围和指标相关性）
2. 第二步：日志分析（提取错误模式和异常堆栈）
3. 第三步：拓扑分析（分析服务依赖和故障传播路径）
4. 第四步：综合决策（综合所有证据生成根因假设）

计划应该清晰、可执行。`,
		task.Query,
		task.TimeRange.StartTime,
		task.TimeRange.EndTime,
		task.Services,
		analysis["alert_count"],
		analysis["critical_count"])

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
		// 如果 LLM 调用失败，返回默认计划
		return "默认分析计划：1. 指标分析 2. 日志分析 3. 拓扑分析 4. 综合决策"
	}

	return resp.Content()
}

// decomposeTasks 分解为子任务
func (a *PlannerAgent) decomposeTasks(rootTask *framework2.Task, plan string) []*framework2.Task {
	subTasks := make([]*framework2.Task, 0)

	// 根据任务类型和服务生成子任务
	if len(rootTask.Services) == 0 {
		// 如果没有指定服务，从告警中提取
		services := make(map[string]bool)
		for _, alert := range rootTask.Alerts {
			services[alert.Service] = true
		}
		rootTask.Services = keys(services)
	}

	// 生成指标分析任务
	subTasks = append(subTasks, &framework2.Task{
		ID:         framework2.GenerateTaskID(),
		Type:       framework2.TaskTypeMetricsAnalysis,
		Query:      fmt.Sprintf("分析服务 %v 的指标异常", rootTask.Services),
		TimeRange:  rootTask.TimeRange,
		Services:   rootTask.Services,
		Alerts:     rootTask.Alerts,
		ParentTask: rootTask,
	})

	// 生成日志分析任务
	subTasks = append(subTasks, &framework2.Task{
		ID:         framework2.GenerateTaskID(),
		Type:       framework2.TaskTypeLogsAnalysis,
		Query:      fmt.Sprintf("分析服务 %v 的错误日志", rootTask.Services),
		TimeRange:  rootTask.TimeRange,
		Services:   rootTask.Services,
		Alerts:     rootTask.Alerts,
		ParentTask: rootTask,
	})

	// 生成拓扑分析任务
	subTasks = append(subTasks, &framework2.Task{
		ID:         framework2.GenerateTaskID(),
		Type:       framework2.TaskTypeTopologyAnalysis,
		Query:      fmt.Sprintf("分析服务 %v 的依赖关系和故障传播路径", rootTask.Services),
		TimeRange:  rootTask.TimeRange,
		Services:   rootTask.Services,
		Alerts:     rootTask.Alerts,
		ParentTask: rootTask,
	})

	return subTasks
}

// keys 提取 map 的键
func keys(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}
