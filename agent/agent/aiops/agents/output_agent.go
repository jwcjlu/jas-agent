package agents

import (
	"context"
	"fmt"
	"jas-agent/agent/agent/aiops/framework"
	"sort"
	"strings"
	"time"

	"jas-agent/agent/core"
	"jas-agent/agent/llm"
)

// OutputAgent 最终输出智能体
// 负责将分析结果格式化为标准化的诊断报告
type OutputAgent struct {
	*BaseAgent
}

// NewOutputAgent 创建最终输出智能体
func NewOutputAgent(ctx *framework.CollaborationContext) *OutputAgent {
	base := NewBaseAgent(
		framework.RoleOutput,
		"最终输出智能体",
		`你是一个专业的运维报告撰写专家。你的职责是：
1. 将分析结果格式化为标准化的诊断报告
2. 生成易于理解的摘要
3. 构建清晰的时间线
4. 提供可执行的修复建议
5. 确保报告结构清晰、逻辑严谨

报告应包含：
- 摘要：简要概述问题和根因
- 影响范围：受影响的服务和用户
- 根因：明确的根因描述
- 证据链：支持根因的证据
- 行动建议：具体的修复建议
- 时间线：关键事件的时间序列`,
		ctx.Chat(),
		ctx.Memory(),
	)

	return &OutputAgent{BaseAgent: base}
}

// Execute 执行报告生成
func (a *OutputAgent) Execute(ctx context.Context, task *framework.Task) (*framework.TaskResult, error) {
	result := &framework.TaskResult{
		TaskID:    task.ID,
		AgentRole: framework.RoleOutput,
		Success:   true,
		Evidence:  make([]framework.Evidence, 0),
		Findings:  make([]framework.Finding, 0),
		Metadata:  make(map[string]interface{}),
	}

	// 1. 提取决策结果
	decisionResult, ok := task.Context["decision_result"].(*framework.TaskResult)
	if !ok {
		result.Success = false
		result.Metadata["error"] = "missing decision result"
		return result, nil
	}

	// 2. 提取任务结果
	taskResults, ok := task.Context["task_results"].(map[string]*framework.TaskResult)
	if !ok {
		result.Success = false
		result.Metadata["error"] = "missing task results"
		return result, nil
	}

	// 3. 生成摘要
	summary := a.generateSummary(ctx, task, decisionResult, taskResults)

	// 4. 提取根因
	rootCause := a.extractRootCause(decisionResult)

	// 5. 构建证据链
	evidenceChain := a.buildEvidenceChain(taskResults)

	// 6. 生成时间线
	timeline := a.buildTimeline(taskResults)

	// 7. 生成修复建议
	recommendations := a.generateRecommendations(ctx, task, decisionResult, taskResults)

	// 8. 构建报告
	report := &Report{
		Summary:          summary,
		RootCause:        rootCause,
		EvidenceChain:    evidenceChain,
		Timeline:         timeline,
		Recommendations:  recommendations,
		AffectedServices: a.extractAffectedServices(taskResults),
		Confidence:       a.calculateOverallConfidence(decisionResult, taskResults),
	}

	// 9. 使用 LLM 优化报告
	optimizedReport := a.optimizeReportWithLLM(ctx, task, report)

	result.Metadata["summary"] = optimizedReport.Summary
	result.Metadata["root_cause"] = optimizedReport.RootCause
	result.Metadata["evidence_chain"] = optimizedReport.EvidenceChain
	result.Metadata["timeline"] = optimizedReport.Timeline
	result.Metadata["recommendations"] = optimizedReport.Recommendations
	result.Metadata["affected_services"] = optimizedReport.AffectedServices
	result.Metadata["confidence"] = optimizedReport.Confidence

	return result, nil
}

// Report 报告结构
type Report struct {
	Summary          string
	RootCause        string
	EvidenceChain    []framework.Evidence
	Timeline         []framework.TimelineEvent
	Recommendations  []string
	AffectedServices []string
	Confidence       float64
}

// generateSummary 生成摘要
func (a *OutputAgent) generateSummary(
	ctx context.Context,
	task *framework.Task,
	decisionResult *framework.TaskResult,
	taskResults map[string]*framework.TaskResult,
) string {
	// 使用 LLM 生成摘要
	prompt := fmt.Sprintf(`基于以下分析结果生成故障诊断摘要：

查询: %s
时间范围: %d - %d

决策结果：
- 根因: %s
- 置信度: %.2f

任务结果数: %d

请生成一个简洁、清晰的摘要（2-3句话），包括：
1. 问题概述
2. 主要根因
3. 影响范围`,
		task.Query,
		task.TimeRange.StartTime,
		task.TimeRange.EndTime,
		a.getRootCauseFromDecision(decisionResult),
		a.getConfidenceFromDecision(decisionResult),
		len(taskResults))

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
		return fmt.Sprintf("故障分析摘要：在 %s 时间范围内，系统出现异常。根因：%s", time.Unix(task.TimeRange.StartTime, 0).Format("2006-01-02 15:04:05"), a.getRootCauseFromDecision(decisionResult))
	}

	return resp.Content()
}

// extractRootCause 提取根因
func (a *OutputAgent) extractRootCause(decisionResult *framework.TaskResult) string {
	if rootCause, ok := decisionResult.Metadata["root_cause"].(string); ok {
		return rootCause
	}

	// 从发现中提取
	if len(decisionResult.Findings) > 0 {
		for _, finding := range decisionResult.Findings {
			if finding.Type == "root_cause" {
				return finding.Description
			}
		}
	}

	return "未找到明确的根因"
}

// buildEvidenceChain 构建证据链
func (a *OutputAgent) buildEvidenceChain(taskResults map[string]*framework.TaskResult) []framework.Evidence {
	evidenceChain := make([]framework.Evidence, 0)

	// 收集所有证据
	for _, result := range taskResults {
		if result.Success {
			evidenceChain = append(evidenceChain, result.Evidence...)
		}
	}

	// 按时间戳排序
	sort.Slice(evidenceChain, func(i, j int) bool {
		return evidenceChain[i].Timestamp < evidenceChain[j].Timestamp
	})

	return evidenceChain
}

// buildTimeline 构建时间线
func (a *OutputAgent) buildTimeline(taskResults map[string]*framework.TaskResult) []framework.TimelineEvent {
	timeline := make([]framework.TimelineEvent, 0)

	// 从任务结果中提取时间线事件
	for _, result := range taskResults {
		if !result.Success {
			continue
		}

		// 从证据中提取时间线事件
		for _, evidence := range result.Evidence {
			severity := "INFO"
			if evidence.Score > 0.8 {
				severity = "CRITICAL"
			} else if evidence.Score > 0.6 {
				severity = "HIGH"
			} else if evidence.Score > 0.4 {
				severity = "MEDIUM"
			}

			timeline = append(timeline, framework.TimelineEvent{
				Timestamp:   evidence.Timestamp,
				Service:     evidence.Service,
				EventType:   evidence.Type,
				Description: evidence.Description,
				Severity:    severity,
			})
		}

		// 从发现中提取时间线事件
		for _, finding := range result.Findings {
			timeline = append(timeline, framework.TimelineEvent{
				Timestamp:   time.Now().Unix(), // 发现可能没有时间戳，使用当前时间
				Service:     finding.Service,
				EventType:   finding.Type,
				Description: finding.Description,
				Severity:    finding.Severity,
			})
		}
	}

	// 按时间戳排序
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Timestamp < timeline[j].Timestamp
	})

	return timeline
}

// generateRecommendations 生成修复建议
func (a *OutputAgent) generateRecommendations(
	ctx context.Context,
	task *framework.Task,
	decisionResult *framework.TaskResult,
	taskResults map[string]*framework.TaskResult,
) []string {
	// 使用 LLM 生成修复建议
	prompt := fmt.Sprintf(`基于以下根因分析结果生成具体的修复建议：

根因: %s
受影响服务: %v

证据链（前5条）：
%s

请生成3-5条具体的、可执行的修复建议，包括：
1. 立即采取的紧急措施
2. 短期修复方案
3. 长期预防措施`,
		a.extractRootCause(decisionResult),
		a.extractAffectedServices(taskResults),
		a.formatEvidenceChain(a.buildEvidenceChain(taskResults)[:minInt(5, len(a.buildEvidenceChain(taskResults)))]))

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
		return a.generateDefaultRecommendations(decisionResult)
	}

	// 解析 LLM 返回的建议（可能是列表格式）
	recommendations := a.parseRecommendations(resp.Content())
	if len(recommendations) == 0 {
		return a.generateDefaultRecommendations(decisionResult)
	}

	return recommendations
}

// parseRecommendations 解析建议列表
func (a *OutputAgent) parseRecommendations(content string) []string {
	recommendations := make([]string, 0)

	// 按行分割
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 移除编号（如 "1. ", "- "）
		line = strings.TrimPrefix(line, "1. ")
		line = strings.TrimPrefix(line, "2. ")
		line = strings.TrimPrefix(line, "3. ")
		line = strings.TrimPrefix(line, "4. ")
		line = strings.TrimPrefix(line, "5. ")
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")

		if len(line) > 10 { // 过滤太短的行
			recommendations = append(recommendations, line)
		}
	}

	return recommendations
}

// generateDefaultRecommendations 生成默认建议
func (a *OutputAgent) generateDefaultRecommendations(decisionResult *framework.TaskResult) []string {
	recommendations := []string{
		"1. 立即检查并重启受影响的服务",
		"2. 检查相关服务的日志和监控指标",
		"3. 验证依赖服务的健康状态",
		"4. 执行回滚操作（如果适用）",
		"5. 加强监控和告警规则",
	}

	// 根据根因调整建议
	rootCause := a.extractRootCause(decisionResult)
	if strings.Contains(rootCause, "数据库") || strings.Contains(rootCause, "database") {
		recommendations = append([]string{"1. 检查数据库连接和性能"}, recommendations...)
	} else if strings.Contains(rootCause, "内存") || strings.Contains(rootCause, "memory") {
		recommendations = append([]string{"1. 检查内存使用情况，考虑扩容或优化"}, recommendations...)
	} else if strings.Contains(rootCause, "网络") || strings.Contains(rootCause, "network") {
		recommendations = append([]string{"1. 检查网络连接和带宽"}, recommendations...)
	}

	return recommendations
}

// extractAffectedServices 提取受影响的服务
func (a *OutputAgent) extractAffectedServices(taskResults map[string]*framework.TaskResult) []string {
	services := make(map[string]bool)

	for _, result := range taskResults {
		if !result.Success {
			continue
		}

		for _, evidence := range result.Evidence {
			services[evidence.Service] = true
		}

		for _, finding := range result.Findings {
			services[finding.Service] = true
		}
	}

	result := make([]string, 0, len(services))
	for service := range services {
		result = append(result, service)
	}

	return result
}

// calculateOverallConfidence 计算整体置信度
func (a *OutputAgent) calculateOverallConfidence(decisionResult *framework.TaskResult, taskResults map[string]*framework.TaskResult) float64 {
	// 使用决策结果的置信度作为基础
	confidence := a.getConfidenceFromDecision(decisionResult)

	// 根据任务结果数量调整
	taskCount := len(taskResults)
	successCount := 0
	for _, result := range taskResults {
		if result.Success {
			successCount++
		}
	}

	// 成功率影响置信度
	successRate := float64(successCount) / float64(taskCount)
	confidence = confidence * (0.7 + successRate*0.3)

	return confidence
}

// optimizeReportWithLLM 使用 LLM 优化报告
func (a *OutputAgent) optimizeReportWithLLM(ctx context.Context, task *framework.Task, report *Report) *Report {
	// 可以进一步使用 LLM 优化报告的格式和可读性
	// 这里暂时直接返回原报告
	return report
}

// formatEvidenceChain 格式化证据链
func (a *OutputAgent) formatEvidenceChain(evidence []framework.Evidence) string {
	result := ""
	for i, e := range evidence {
		result += fmt.Sprintf("%d. [%s] %s: %s\n", i+1, e.Type, e.Service, e.Description)
	}
	return result
}

// getRootCauseFromDecision 从决策结果获取根因
func (a *OutputAgent) getRootCauseFromDecision(decisionResult *framework.TaskResult) string {
	if rootCause, ok := decisionResult.Metadata["root_cause"].(string); ok {
		return rootCause
	}
	if len(decisionResult.Findings) > 0 {
		return decisionResult.Findings[0].Description
	}
	return "未找到明确的根因"
}

// getConfidenceFromDecision 从决策结果获取置信度
func (a *OutputAgent) getConfidenceFromDecision(decisionResult *framework.TaskResult) float64 {
	if confidence, ok := decisionResult.Metadata["confidence"].(float64); ok {
		return confidence
	}
	return decisionResult.Confidence
}
