package agent

import (
	"context"
	"fmt"
	"jas-agent/agent/agent/aiops/framework"
	"jas-agent/agent/core"
	"strings"
	"time"
)

// AIOPSAgent AIOPS 智能运维代理
type AIOPSAgent struct {
	*BaseReact
	collaborator *framework.Collaborator
	systemPrompt string
	query        string
	timeRange    framework.TimeRange
	services     []string
	alerts       []framework.Alert
}

// Type 返回 Agent 类型
func (agent *AIOPSAgent) Type() AgentType {
	return AIOPSAgentType
}

// NewAIOPSAgent 创建 AIOPS Agent
func NewAIOPSAgent(
	agentCtx *Context,
	executor *AgentExecutor,
	collaborator *framework.Collaborator,
	systemPrompt string,
	configuredServices []string, // 从配置中获取的服务列表
) Agent {
	agentCtx.memory.AddMessage(core.Message{
		Role:    core.MessageRoleSystem,
		Content: systemPrompt,
	})

	return &AIOPSAgent{
		BaseReact:    NewBaseReact(agentCtx, executor),
		collaborator: collaborator,
		systemPrompt: systemPrompt,
		timeRange: framework.TimeRange{
			StartTime: time.Now().Add(-1 * time.Hour).Unix(), // 默认查询最近1小时
			EndTime:   time.Now().Unix(),
		},
		services: configuredServices, // 使用配置的服务列表
		alerts:   make([]framework.Alert, 0),
	}
}

// Step 执行 AIOPS 分析步骤
func (agent *AIOPSAgent) Step() string {
	// 如果是第一步，解析用户查询，提取时间范围和服务信息
	if agent.query == "" {
		return agent.parseQuery()
	}

	// 检查是否已经完成分析
	lastMessage := agent.context.memory.GetLastMessage()
	if lastMessage.Role == core.MessageRoleAssistant &&
		strings.Contains(strings.ToLower(lastMessage.Content), "final answer") {
		agent.executor.UpdateState(FinishState)
		return "Analysis completed"
	}

	// 执行协作分析
	return agent.runFullCollaboration()
}

// parseQuery 解析用户查询，提取关键信息
func (agent *AIOPSAgent) parseQuery() string {
	messages := agent.context.memory.GetMessages()
	if len(messages) == 0 {
		return "No query found"
	}

	// 获取用户的最后一个消息
	var userQuery string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == core.MessageRoleUser {
			userQuery = messages[i].Content
			break
		}
	}

	if userQuery == "" {
		return "No user query found"
	}

	agent.query = userQuery

	// 如果没有配置服务列表，尝试从查询中提取
	if len(agent.services) == 0 {
		agent.services = extractServiceNames(userQuery)
	}

	// 发送解析信息
	servicesInfo := "未指定"
	if len(agent.services) > 0 {
		servicesInfo = strings.Join(agent.services, ", ")
	}
	msg := core.Message{
		Role:    core.MessageRoleAssistant,
		Content: fmt.Sprintf("📋 任务规划：正在解析查询并准备 AIOPS 根因分析\n查询内容: %s\n配置的服务: %s", userQuery, servicesInfo),
	}
	agent.context.Send(context.TODO(), msg)
	agent.context.memory.AddMessage(msg)

	return fmt.Sprintf("Query parsed. Services: %v", agent.services)
}

// executeCollaboration 执行多智能体协作分析
// 这个方法会被多次调用，每次执行一个阶段
func (agent *AIOPSAgent) executeCollaboration() string {
	// 检查执行阶段（通过 agent.executor.currentStep 判断）
	currentStep := agent.executor.GetCurrentStep()

	switch currentStep {
	case 1:
		// 第一阶段：发送开始分析的消息
		startMsg := core.Message{
			Role: core.MessageRoleAssistant,
			Content: fmt.Sprintf("🚀 开始 AIOPS 根因分析\n时间范围: %s - %s\n分析服务: %v",
				time.Unix(agent.timeRange.StartTime, 0).Format("2006-01-02 15:04:05"),
				time.Unix(agent.timeRange.EndTime, 0).Format("2006-01-02 15:04:05"),
				agent.services),
		}
		agent.context.Send(context.TODO(), startMsg)
		agent.context.memory.AddMessage(startMsg)
		return "Starting AIOPS analysis"

	case 2:
		// 第二阶段：任务规划
		planMsg := core.Message{
			Role:    core.MessageRoleAssistant,
			Content: "📋 任务规划：正在制定根因分析计划...",
		}
		agent.context.Send(context.TODO(), planMsg)
		return "Planning phase"

	default:
		// 后续阶段：执行完整的协作分析
		// 发送各阶段进度信息
		progressMsg := core.Message{
			Role:    core.MessageRoleAssistant,
			Content: fmt.Sprintf("⚙️ 执行分析（步骤 %d/%d）...", currentStep-2, agent.executor.GetMaxSteps()-2),
		}
		agent.context.Send(context.TODO(), progressMsg)

		// 只在最后阶段执行完整的协作分析
		if currentStep >= agent.executor.GetMaxSteps()-1 {
			return agent.runFullCollaboration()
		}

		return fmt.Sprintf("Analysis in progress (step %d)", currentStep)
	}
}

// runFullCollaboration 执行完整的协作分析
func (agent *AIOPSAgent) runFullCollaboration() string {
	// 发送正在分析的提示
	analyzingMsg := core.Message{
		Role:    core.MessageRoleAssistant,
		Content: "🔍 正在协同多个智能体进行分析：\n- 📊 指标分析智能体\n- 📝 日志分析智能体\n- 🗺️ 拓扑分析智能体\n- 🧠 决策分析智能体",
	}
	agent.context.Send(context.TODO(), analyzingMsg)

	// 执行协作分析
	report, err := agent.collaborator.Collaborate(
		context.TODO(),
		agent.query,
		agent.timeRange,
		agent.services,
		agent.alerts,
	)

	if err != nil {
		errorMsg := core.Message{
			Role:    core.MessageRoleAssistant,
			Content: fmt.Sprintf("❌ 分析失败: %s", err.Error()),
		}
		agent.context.Send(context.TODO(), errorMsg)
		agent.context.memory.AddMessage(errorMsg)
		agent.executor.UpdateState(ErrorState)
		return fmt.Sprintf("Collaboration failed: %s", err.Error())
	}

	// 发送分析结果
	result := agent.formatReport(report)
	finalMsg := core.Message{
		Role:    core.MessageRoleAssistant,
		Content: fmt.Sprintf("✅ 分析完成\n\n%s", result),
	}
	agent.context.Send(context.TODO(), finalMsg)
	agent.context.memory.AddMessage(finalMsg)

	// 标记完成
	agent.executor.UpdateState(FinishState)

	return fmt.Sprintf("Final answer: %s", result)
}

// formatReport 格式化分析报告
func (agent *AIOPSAgent) formatReport(report *framework.AnalysisReport) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 📊 根因分析报告\n\n"))

	if report.Summary != "" {
		sb.WriteString(fmt.Sprintf("### 📝 摘要\n%s\n\n", report.Summary))
	}

	if report.RootCause != "" {
		sb.WriteString(fmt.Sprintf("### 🎯 根因\n%s\n\n", report.RootCause))
	}

	if len(report.AffectedServices) > 0 {
		sb.WriteString(fmt.Sprintf("### 🔍 受影响服务\n%s\n\n", strings.Join(report.AffectedServices, ", ")))
	}

	if len(report.EvidenceChain) > 0 {
		sb.WriteString(fmt.Sprintf("### 🔗 证据链 (%d 条)\n", len(report.EvidenceChain)))
		for i, evidence := range report.EvidenceChain {
			if i >= 5 { // 只显示前5条
				sb.WriteString(fmt.Sprintf("... 还有 %d 条证据\n", len(report.EvidenceChain)-5))
				break
			}
			sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n", evidence.Type, evidence.Service, evidence.Description))
		}
		sb.WriteString("\n")
	}

	if len(report.Findings) > 0 {
		sb.WriteString(fmt.Sprintf("### 🔎 发现 (%d 项)\n", len(report.Findings)))
		for i, finding := range report.Findings {
			if i >= 5 { // 只显示前5项
				sb.WriteString(fmt.Sprintf("... 还有 %d 项发现\n", len(report.Findings)-5))
				break
			}
			sb.WriteString(fmt.Sprintf("- [%s] %s: %s (严重程度: %s)\n",
				finding.Type, finding.Service, finding.Description, finding.Severity))
		}
		sb.WriteString("\n")
	}

	if len(report.Recommendations) > 0 {
		sb.WriteString(fmt.Sprintf("### 💡 修复建议\n"))
		for i, rec := range report.Recommendations {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
		}
		sb.WriteString("\n")
	}

	if report.Confidence > 0 {
		sb.WriteString(fmt.Sprintf("### 📈 置信度: %.1f%%\n", report.Confidence*100))
	}

	return sb.String()
}

// extractServiceNames 从查询中提取服务名（简单实现）
func extractServiceNames(query string) []string {
	// 简单的关键字匹配，实际应该使用更智能的方式
	services := make([]string, 0)
	lowerQuery := strings.ToLower(query)

	// 常见的服务名模式
	commonServices := []string{}
	for _, service := range commonServices {
		if strings.Contains(lowerQuery, service) {
			services = append(services, service)
		}
	}

	// 如果没有找到，返回空数组（后续可以从配置中获取）
	return services
}

// NewAIOPSAgentExecutor 创建 AIOPS Agent 执行器
func NewAIOPSAgentExecutor(context *Context, collaborator *framework.Collaborator, systemPrompt string, configuredServices []string) *AgentExecutor {
	executor := &AgentExecutor{
		context:     context,
		maxSteps:    50, // AIOPS 分析可能需要更多步骤
		currentStep: 0,
		state:       IdleState,
	}
	executor.agent = NewAIOPSAgent(context, executor, collaborator, systemPrompt, configuredServices)
	executor.summaryAgent = NewSummaryAgent(context, executor)
	return executor
}
