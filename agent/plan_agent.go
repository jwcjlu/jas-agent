package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"jas-agent/core"
	"jas-agent/llm"
	"jas-agent/tools"
	"strings"
	"time"
)

// PlanStep 计划步骤
type PlanStep struct {
	ID           int    `json:"id"`
	Description  string `json:"description"`
	Tool         string `json:"tool"`
	Input        string `json:"input"`
	Status       string `json:"status"` // pending, executing, completed, failed, skipped
	Result       string `json:"result"`
	Dependencies []int  `json:"dependencies"` // 依赖的步骤ID
}

// Plan 执行计划
type Plan struct {
	Goal    string      `json:"goal"`
	Steps   []*PlanStep `json:"steps"`
	Created time.Time   `json:"created"`
	Updated time.Time   `json:"updated"`
	Status  string      `json:"status"` // planning, executing, completed, failed
}

// PlanAgent 计划Agent
type PlanAgent struct {
	*BaseReact
	context      *Context
	executor     *AgentExecutor
	plan         *Plan
	currentStep  int
	systemPrompt string
	enableReplan bool // 是否允许重新规划
}

func (a *PlanAgent) Type() AgentType {
	return PlanAgentType
}

func (a *PlanAgent) Step() string {
	if a.plan == nil {
		// 第一步：生成计划
		return a.generatePlan()
	}

	if a.plan.Status == "completed" || a.plan.Status == "failed" {
		a.executor.UpdateState(FinishState)
		return fmt.Sprintf("Plan execution %s", a.plan.Status)
	}

	// 执行计划中的下一步
	return a.executeNextStep()
}

// generatePlan 生成执行计划
func (a *PlanAgent) generatePlan() string {
	fmt.Println("📋 Generating execution plan...")

	// 获取用户查询
	messages := a.context.memory.GetMessages()
	var userQuery string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == core.MessageRoleUser {
			userQuery = messages[i].Content
			break
		}
	}

	// 构建计划生成提示
	tools := a.context.toolManager.AvailableTools()
	var toolsDesc strings.Builder
	toolsDesc.WriteString("可用工具:\n")
	for _, tool := range tools {
		toolsDesc.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
	}

	planPrompt := fmt.Sprintf(`请为以下任务生成详细的执行计划。

用户任务: %s

%s

请生成一个JSON格式的执行计划，包含以下结构:
{
  "goal": "任务目标",
  "steps": [
    {
      "id": 1,
      "description": "步骤描述",
      "tool": "工具名称",
      "input": "工具输入",
      "dependencies": []
    }
  ]
}

注意事项:
1. 步骤要有逻辑顺序
2. 如果某步骤依赖其他步骤的结果，在dependencies中标注
3. 工具名称必须从可用工具列表中选择
4. 每个步骤要具体、可执行
5. 最后一步应该是总结或返回答案

请只返回JSON格式的计划，不要包含其他内容。`, userQuery, toolsDesc.String())

	// 调用LLM生成计划
	planMessages := []core.Message{
		{
			Role:    core.MessageRoleSystem,
			Content: a.systemPrompt,
		},
		{
			Role:    core.MessageRoleUser,
			Content: planPrompt,
		},
	}

	resp, err := a.context.chat.Completions(context.TODO(), llm.NewChatRequest(a.context.model, planMessages))
	if err != nil {
		return fmt.Sprintf("Plan generation failed: %s", err.Error())
	}

	// 解析计划
	planJSON := extractJSON(resp.Content())
	var planData struct {
		Goal  string      `json:"goal"`
		Steps []*PlanStep `json:"steps"`
	}

	err = json.Unmarshal([]byte(planJSON), &planData)
	if err != nil {
		fmt.Printf("Failed to parse plan JSON: %s\n", err.Error())
		fmt.Printf("Response: %s\n", resp.Content())
		return fmt.Sprintf("Failed to parse plan: %s", err.Error())
	}

	// 初始化计划
	a.plan = &Plan{
		Goal:    planData.Goal,
		Steps:   planData.Steps,
		Created: time.Now(),
		Updated: time.Now(),
		Status:  "executing",
	}

	// 设置所有步骤状态为pending
	for _, step := range a.plan.Steps {
		step.Status = "pending"
	}

	// 显示计划
	fmt.Println("\n📝 Generated Plan:")
	fmt.Printf("Goal: %s\n", a.plan.Goal)
	fmt.Println("Steps:")
	for _, step := range a.plan.Steps {
		deps := ""
		if len(step.Dependencies) > 0 {
			deps = fmt.Sprintf(" (depends on: %v)", step.Dependencies)
		}
		fmt.Printf("  %d. %s%s\n", step.ID, step.Description, deps)
	}
	fmt.Println()

	return "Plan generated successfully"
}

// executeNextStep 执行下一步
func (a *PlanAgent) executeNextStep() string {
	// 查找下一个待执行的步骤
	var nextStep *PlanStep
	for _, step := range a.plan.Steps {
		if step.Status == "pending" {
			// 检查依赖是否已完成
			canExecute := true
			for _, depID := range step.Dependencies {
				depStep := a.findStepByID(depID)
				if depStep == nil || depStep.Status != "completed" {
					canExecute = false
					break
				}
			}

			if canExecute {
				nextStep = step
				break
			}
		}
	}

	if nextStep == nil {
		// 检查是否所有步骤都完成
		allCompleted := true
		for _, step := range a.plan.Steps {
			if step.Status != "completed" && step.Status != "skipped" {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			a.plan.Status = "completed"
			a.executor.UpdateState(FinishState)
			return a.generateSummary()
		}

		// 如果有失败的步骤且启用重新规划
		if a.enableReplan {
			return a.replan()
		}

		a.plan.Status = "failed"
		return "Plan execution blocked: dependencies not met"
	}

	// 执行步骤
	return a.executeStep(nextStep)
}

// executeStep 执行具体步骤
func (a *PlanAgent) executeStep(step *PlanStep) string {
	step.Status = "executing"
	fmt.Printf("⚙️  Executing step %d: %s\n", step.ID, step.Description)

	// 替换输入中的依赖引用 ${step.X}
	input := a.resolveDependencies(step.Input, step.Dependencies)

	// 创建工具调用
	ctx := context.Background()
	toolCall := &tools.ToolCall{
		Name:  step.Tool,
		Input: input,
	}

	// 执行工具调用
	result, err := a.context.toolManager.ExecTool(ctx, toolCall)

	if err != nil {
		step.Status = "failed"
		step.Result = fmt.Sprintf("Error: %s", err.Error())
		fmt.Printf("❌ Step %d failed: %s\n", step.ID, err.Error())
		return fmt.Sprintf("Step %d execution failed: %s", step.ID, err.Error())
	}

	step.Status = "completed"
	step.Result = result
	a.plan.Updated = time.Now()

	fmt.Printf("✅ Step %d completed: %s\n", step.ID, truncateString(result, 100))

	// 添加到内存
	a.context.memory.AddMessage(core.Message{
		Role:    core.MessageRoleAssistant,
		Content: fmt.Sprintf("Completed step %d: %s", step.ID, step.Description),
	})
	a.context.memory.AddMessage(core.Message{
		Role:    core.MessageRoleUser,
		Content: fmt.Sprintf("Result: %s", result),
	})

	return fmt.Sprintf("Step %d completed", step.ID)
}

// findStepByID 根据ID查找步骤
func (a *PlanAgent) findStepByID(id int) *PlanStep {
	for _, step := range a.plan.Steps {
		if step.ID == id {
			return step
		}
	}
	return nil
}

// resolveDependencies 解析依赖引用
func (a *PlanAgent) resolveDependencies(input string, dependencies []int) string {
	result := input
	for _, depID := range dependencies {
		step := a.findStepByID(depID)
		if step != nil && step.Status == "completed" {
			placeholder := fmt.Sprintf("${step.%d}", depID)
			result = strings.ReplaceAll(result, placeholder, step.Result)
		}
	}
	return result
}

// replan 重新规划
func (a *PlanAgent) replan() string {
	fmt.Println("🔄 Replanning...")

	// 收集当前执行状态
	var statusReport strings.Builder
	statusReport.WriteString("当前执行状态:\n")
	for _, step := range a.plan.Steps {
		statusReport.WriteString(fmt.Sprintf("Step %d (%s): %s\n", step.ID, step.Status, step.Description))
		if step.Status == "failed" {
			statusReport.WriteString(fmt.Sprintf("  Error: %s\n", step.Result))
		}
	}

	// 请求重新规划
	replanPrompt := fmt.Sprintf(`任务执行遇到问题，需要重新规划。

原始目标: %s

%s

请生成一个新的执行计划，避免之前失败的问题。返回JSON格式。`, a.plan.Goal, statusReport.String())

	replanMessages := []core.Message{
		{
			Role:    core.MessageRoleSystem,
			Content: a.systemPrompt,
		},
		{
			Role:    core.MessageRoleUser,
			Content: replanPrompt,
		},
	}

	resp, err := a.context.chat.Completions(context.TODO(), llm.NewChatRequest(a.context.model, replanMessages))
	if err != nil {
		return fmt.Sprintf("Replan failed: %s", err.Error())
	}

	// 解析新计划
	planJSON := extractJSON(resp.Content())
	var planData struct {
		Goal  string      `json:"goal"`
		Steps []*PlanStep `json:"steps"`
	}

	err = json.Unmarshal([]byte(planJSON), &planData)
	if err != nil {
		return fmt.Sprintf("Failed to parse new plan: %s", err.Error())
	}

	// 更新计划
	a.plan.Steps = planData.Steps
	a.plan.Updated = time.Now()
	for _, step := range a.plan.Steps {
		step.Status = "pending"
	}

	fmt.Println("✨ Plan updated successfully")
	return "Replanned successfully"
}

// generateSummary 生成总结
func (a *PlanAgent) generateSummary() string {
	fmt.Println("📊 Generating summary...")

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("任务: %s\n\n", a.plan.Goal))
	summary.WriteString("执行结果:\n")

	for _, step := range a.plan.Steps {
		if step.Status == "completed" {
			summary.WriteString(fmt.Sprintf("✓ %s\n", step.Description))
			if step.Result != "" {
				summary.WriteString(fmt.Sprintf("  结果: %s\n", truncateString(step.Result, 200)))
			}
		}
	}

	// 使用LLM生成最终总结
	summaryMessages := []core.Message{
		{
			Role:    core.MessageRoleSystem,
			Content: "你是一个专业的总结助手，请基于任务执行结果提供清晰的总结。",
		},
		{
			Role:    core.MessageRoleUser,
			Content: summary.String() + "\n\n请提供简洁明了的最终答案。",
		},
	}

	resp, err := a.context.chat.Completions(context.TODO(), llm.NewChatRequest(a.context.model, summaryMessages))
	if err != nil {
		return summary.String()
	}

	return resp.Content()
}

// NewPlanAgent 创建计划Agent
func NewPlanAgent(context *Context, executor *AgentExecutor, enableReplan bool) Agent {
	systemPrompt := core.GetPlanSystemPrompt()

	return &PlanAgent{
		BaseReact:    NewBaseReact(context, executor),
		context:      context,
		executor:     executor,
		plan:         nil,
		currentStep:  0,
		systemPrompt: systemPrompt,
		enableReplan: enableReplan,
	}
}

// NewPlanAgentExecutor 创建计划Agent执行器
func NewPlanAgentExecutor(context *Context, enableReplan bool) *AgentExecutor {
	executor := &AgentExecutor{
		context:     context,
		maxSteps:    50, // 计划执行可能需要较多步骤
		currentStep: 0,
		state:       IdleState,
	}
	executor.agent = NewPlanAgent(context, executor, enableReplan)
	executor.summaryAgent = NewSummaryAgent(context, executor)
	return executor
}

// extractJSON 从文本中提取JSON
func extractJSON(text string) string {
	// 尝试找到JSON块
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")

	if start == -1 || end == -1 || start >= end {
		return text
	}

	return text[start : end+1]
}
