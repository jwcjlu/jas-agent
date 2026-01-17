package agents

import (
	"context"
	"fmt"
	"jas-agent/agent/agent/aiops/framework"
	"math"

	"jas-agent/agent/core"
	"jas-agent/agent/llm"
)

// DecisionAgent 分析决策智能体
// 扮演"专家会诊"角色，综合各智能体上报的证据，进行结构化推理，生成根因假设
type DecisionAgent struct {
	*BaseAgent
}

// NewDecisionAgent 创建分析决策智能体
func NewDecisionAgent(ctx *framework.CollaborationContext) *DecisionAgent {
	base := NewBaseAgent(
		framework.RoleDecision,
		"分析决策智能体",
		`你是一个专业的根因分析决策专家。你的职责是：
1. 综合各智能体上报的证据（指标异常、日志错误、拓扑问题等）
2. 进行结构化推理
3. 在信息冲突时做出判断
4. 生成根因假设
5. 评估根因假设的置信度

分析时应该：
- 综合考虑所有证据
- 识别证据之间的关联性
- 排除干扰信息
- 生成最可能的根因假设
- 评估假设的可信度`,
		ctx.Chat(),
		ctx.Memory(),
	)

	return &DecisionAgent{BaseAgent: base}
}

// Execute 执行决策分析
func (a *DecisionAgent) Execute(ctx context.Context, task *framework.Task) (*framework.TaskResult, error) {
	result := &framework.TaskResult{
		TaskID:    task.ID,
		AgentRole: framework.RoleDecision,
		Success:   true,
		Evidence:  make([]framework.Evidence, 0),
		Findings:  make([]framework.Finding, 0),
		Metadata:  make(map[string]interface{}),
	}

	// 1. 提取所有任务结果
	taskResults, ok := task.Context["task_results"].(map[string]*framework.TaskResult)
	if !ok {
		result.Success = false
		result.Metadata["error"] = "missing task results"
		return result, nil
	}

	// 2. 收集所有证据
	allEvidence := a.collectEvidence(taskResults)

	// 3. 收集所有发现
	allFindings := a.collectFindings(taskResults)

	// 4. 证据关联分析
	correlations := a.analyzeCorrelations(allEvidence, allFindings)

	// 5. 冲突检测与解决
	conflicts := a.detectConflicts(allEvidence, allFindings)
	resolvedConflicts := a.resolveConflicts(conflicts)

	// 6. 生成根因假设
	rootCauses := a.generateRootCauses(allEvidence, allFindings, correlations)

	// 7. 使用 LLM 进行高级推理
	reasoning := a.reasonWithLLM(ctx, task, allEvidence, allFindings, correlations, rootCauses)

	// 8. 选择最佳根因假设
	bestRootCause := a.selectBestRootCause(rootCauses)

	result.Findings = []framework.Finding{
		{
			Type:        "root_cause",
			Service:     bestRootCause.Service,
			Description: bestRootCause.Description,
			Severity:    bestRootCause.Severity,
			Score:       bestRootCause.Confidence,
		},
	}

	result.Metadata["all_evidence"] = allEvidence
	result.Metadata["all_findings"] = allFindings
	result.Metadata["correlations"] = correlations
	result.Metadata["conflicts"] = resolvedConflicts
	result.Metadata["root_causes"] = rootCauses
	result.Metadata["reasoning"] = reasoning
	result.Metadata["root_cause"] = bestRootCause.Description
	result.Metadata["confidence"] = bestRootCause.Confidence
	result.Confidence = bestRootCause.Confidence

	return result, nil
}

// RootCause 根因假设
type RootCause struct {
	Service     string
	Description string
	Evidence    []framework.Evidence
	Findings    []framework.Finding
	Confidence  float64
	Severity    string
	Reasoning   string
}

// EvidenceCorrelation 证据关联
type EvidenceCorrelation struct {
	Evidence1   framework.Evidence
	Evidence2   framework.Evidence
	Coefficient float64
	Type        string // temporal, spatial, causal
}

// Conflict 冲突
type Conflict struct {
	Type        string
	Evidence1   framework.Evidence
	Evidence2   framework.Evidence
	Description string
	Resolution  string
}

// collectEvidence 收集所有证据
func (a *DecisionAgent) collectEvidence(taskResults map[string]*framework.TaskResult) []framework.Evidence {
	allEvidence := make([]framework.Evidence, 0)

	for _, result := range taskResults {
		if result.Success {
			allEvidence = append(allEvidence, result.Evidence...)
		}
	}

	return allEvidence
}

// collectFindings 收集所有发现
func (a *DecisionAgent) collectFindings(taskResults map[string]*framework.TaskResult) []framework.Finding {
	allFindings := make([]framework.Finding, 0)

	for _, result := range taskResults {
		if result.Success {
			allFindings = append(allFindings, result.Findings...)
		}
	}

	return allFindings
}

// analyzeCorrelations 证据关联分析
func (a *DecisionAgent) analyzeCorrelations(evidence []framework.Evidence, findings []framework.Finding) []EvidenceCorrelation {
	correlations := make([]EvidenceCorrelation, 0)

	// 时间关联：在相近时间发生的证据
	for i := 0; i < len(evidence); i++ {
		for j := i + 1; j < len(evidence); j++ {
			temporalScore := a.calculateTemporalCorrelation(evidence[i], evidence[j])
			if temporalScore > 0.7 {
				correlations = append(correlations, EvidenceCorrelation{
					Evidence1:   evidence[i],
					Evidence2:   evidence[j],
					Coefficient: temporalScore,
					Type:        "temporal",
				})
			}
		}
	}

	// 空间关联：同一服务的证据
	for i := 0; i < len(evidence); i++ {
		for j := i + 1; j < len(evidence); j++ {
			if evidence[i].Service == evidence[j].Service {
				spatialScore := a.calculateSpatialCorrelation(evidence[i], evidence[j])
				if spatialScore > 0.6 {
					correlations = append(correlations, EvidenceCorrelation{
						Evidence1:   evidence[i],
						Evidence2:   evidence[j],
						Coefficient: spatialScore,
						Type:        "spatial",
					})
				}
			}
		}
	}

	return correlations
}

// calculateTemporalCorrelation 计算时间关联性
func (a *DecisionAgent) calculateTemporalCorrelation(e1, e2 framework.Evidence) float64 {
	timeDiff := math.Abs(float64(e1.Timestamp - e2.Timestamp))
	// 1小时内视为时间相关
	if timeDiff < 3600 {
		return 1.0 - (timeDiff / 3600.0)
	}
	return 0
}

// calculateSpatialCorrelation 计算空间关联性
func (a *DecisionAgent) calculateSpatialCorrelation(e1, e2 framework.Evidence) float64 {
	// 同一服务且类型相关
	if e1.Type == e2.Type {
		return 0.9
	}
	// 不同类型的证据（如指标+日志）也有一定关联
	if (e1.Type == "metrics" && e2.Type == "logs") || (e1.Type == "logs" && e2.Type == "metrics") {
		return 0.7
	}
	return 0.5
}

// detectConflicts 检测冲突
func (a *DecisionAgent) detectConflicts(evidence []framework.Evidence, findings []framework.Finding) []Conflict {
	conflicts := make([]Conflict, 0)

	// 检测证据冲突：同一服务在同一时间有矛盾的证据
	for i := 0; i < len(evidence); i++ {
		for j := i + 1; j < len(evidence); j++ {
			if evidence[i].Service == evidence[j].Service {
				timeDiff := math.Abs(float64(evidence[i].Timestamp - evidence[j].Timestamp))
				if timeDiff < 300 { // 5分钟内
					// 检查是否有矛盾（例如：一个显示正常，一个显示异常）
					if a.isContradictory(evidence[i], evidence[j]) {
						conflicts = append(conflicts, Conflict{
							Type:        "evidence_contradiction",
							Evidence1:   evidence[i],
							Evidence2:   evidence[j],
							Description: fmt.Sprintf("服务 %s 在同一时间有矛盾的证据", evidence[i].Service),
						})
					}
				}
			}
		}
	}

	return conflicts
}

// isContradictory 判断证据是否矛盾
func (a *DecisionAgent) isContradictory(e1, e2 framework.Evidence) bool {
	// 简单的矛盾检测：一个高分（异常），一个低分（正常）
	scoreDiff := math.Abs(e1.Score - e2.Score)
	return scoreDiff > 0.7 && (e1.Score > 0.7 && e2.Score < 0.3 || e1.Score < 0.3 && e2.Score > 0.7)
}

// resolveConflicts 解决冲突
func (a *DecisionAgent) resolveConflicts(conflicts []Conflict) []Conflict {
	resolved := make([]Conflict, 0, len(conflicts))

	for _, conflict := range conflicts {
		// 优先选择分数更高的证据
		if conflict.Evidence1.Score > conflict.Evidence2.Score {
			conflict.Resolution = fmt.Sprintf("选择证据1（分数更高: %.2f > %.2f）", conflict.Evidence1.Score, conflict.Evidence2.Score)
		} else {
			conflict.Resolution = fmt.Sprintf("选择证据2（分数更高: %.2f > %.2f）", conflict.Evidence2.Score, conflict.Evidence1.Score)
		}
		resolved = append(resolved, conflict)
	}

	return resolved
}

// generateRootCauses 生成根因假设
func (a *DecisionAgent) generateRootCauses(
	evidence []framework.Evidence,
	findings []framework.Finding,
	correlations []EvidenceCorrelation,
) []RootCause {
	rootCauses := make([]RootCause, 0)

	// 按服务分组证据和发现
	serviceGroups := make(map[string]struct {
		Evidence []framework.Evidence
		Findings []framework.Finding
	})

	for _, e := range evidence {
		group := serviceGroups[e.Service]
		group.Evidence = append(group.Evidence, e)
		serviceGroups[e.Service] = group
	}

	for _, f := range findings {
		group := serviceGroups[f.Service]
		group.Findings = append(group.Findings, f)
		serviceGroups[f.Service] = group
	}

	// 为每个服务生成根因假设
	for service, group := range serviceGroups {
		if len(group.Evidence) == 0 && len(group.Findings) == 0 {
			continue
		}

		// 计算置信度
		confidence := a.calculateRootCauseConfidence(group.Evidence, group.Findings)

		// 生成描述
		description := a.generateRootCauseDescription(service, group.Evidence, group.Findings)

		// 确定严重程度
		severity := a.determineSeverity(group.Evidence, group.Findings)

		rootCauses = append(rootCauses, RootCause{
			Service:     service,
			Description: description,
			Evidence:    group.Evidence,
			Findings:    group.Findings,
			Confidence:  confidence,
			Severity:    severity,
		})
	}

	return rootCauses
}

// calculateRootCauseConfidence 计算根因置信度
func (a *DecisionAgent) calculateRootCauseConfidence(evidence []framework.Evidence, findings []framework.Finding) float64 {
	if len(evidence) == 0 && len(findings) == 0 {
		return 0.3
	}

	baseConfidence := 0.5

	// 证据数量加分
	evidenceBonus := math.Min(float64(len(evidence))*0.05, 0.2)

	// 发现数量加分
	findingsBonus := math.Min(float64(len(findings))*0.05, 0.2)

	// 证据平均分数
	avgScore := 0.0
	for _, e := range evidence {
		avgScore += e.Score
	}
	if len(evidence) > 0 {
		avgScore /= float64(len(evidence))
	}
	scoreBonus := avgScore * 0.1

	return math.Min(baseConfidence+evidenceBonus+findingsBonus+scoreBonus, 1.0)
}

// generateRootCauseDescription 生成根因描述
func (a *DecisionAgent) generateRootCauseDescription(service string, evidence []framework.Evidence, findings []framework.Finding) string {
	if len(findings) > 0 {
		// 优先使用发现的描述
		return fmt.Sprintf("服务 %s 的 %s: %s", service, findings[0].Type, findings[0].Description)
	}

	if len(evidence) > 0 {
		return fmt.Sprintf("服务 %s 的 %s 异常: %s", service, evidence[0].Type, evidence[0].Description)
	}

	return fmt.Sprintf("服务 %s 可能存在异常", service)
}

// determineSeverity 确定严重程度
func (a *DecisionAgent) determineSeverity(evidence []framework.Evidence, findings []framework.Finding) string {
	// 检查发现中的严重程度
	for _, f := range findings {
		if f.Severity == "CRITICAL" {
			return "CRITICAL"
		}
		if f.Severity == "HIGH" {
			return "HIGH"
		}
	}

	// 基于证据分数判断
	avgScore := 0.0
	for _, e := range evidence {
		avgScore += e.Score
	}
	if len(evidence) > 0 {
		avgScore /= float64(len(evidence))
	}

	if avgScore > 0.8 {
		return "CRITICAL"
	} else if avgScore > 0.6 {
		return "HIGH"
	} else if avgScore > 0.4 {
		return "MEDIUM"
	}
	return "LOW"
}

// reasonWithLLM 使用 LLM 进行高级推理
func (a *DecisionAgent) reasonWithLLM(
	ctx context.Context,
	task *framework.Task,
	evidence []framework.Evidence,
	findings []framework.Finding,
	correlations []EvidenceCorrelation,
	rootCauses []RootCause,
) string {
	prompt := fmt.Sprintf(`综合以下证据和发现进行根因分析：

证据数量: %d
发现数量: %d
关联数量: %d
根因假设数: %d

主要证据：
%s

主要发现：
%s

主要关联：
%s

根因假设：
%s

请进行结构化推理：
1. 综合分析所有证据
2. 识别证据之间的关联性
3. 排除干扰信息
4. 生成最可能的根因假设
5. 评估假设的可信度`,
		len(evidence),
		len(findings),
		len(correlations),
		len(rootCauses),
		a.formatEvidence(evidence[:minInt(10, len(evidence))]),
		a.formatFindings(findings[:minInt(10, len(findings))]),
		a.formatCorrelations(correlations[:minInt(5, len(correlations))]),
		a.formatRootCauses(rootCauses[:minInt(5, len(rootCauses))]))

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
		return "LLM 推理失败: " + err.Error()
	}

	return resp.Content()
}

func (a *DecisionAgent) formatEvidence(evidence []framework.Evidence) string {
	result := ""
	for i, e := range evidence {
		result += fmt.Sprintf("%d. [%s] %s: %s (分数: %.2f)\n", i+1, e.Type, e.Service, e.Description, e.Score)
	}
	return result
}

func (a *DecisionAgent) formatFindings(findings []framework.Finding) string {
	result := ""
	for i, f := range findings {
		result += fmt.Sprintf("%d. [%s] %s: %s (严重程度: %s, 分数: %.2f)\n", i+1, f.Type, f.Service, f.Description, f.Severity, f.Score)
	}
	return result
}

func (a *DecisionAgent) formatCorrelations(correlations []EvidenceCorrelation) string {
	result := ""
	for i, c := range correlations {
		result += fmt.Sprintf("%d. %s 与 %s (%s关联, 系数: %.2f)\n", i+1, c.Evidence1.Service, c.Evidence2.Service, c.Type, c.Coefficient)
	}
	return result
}

func (a *DecisionAgent) formatRootCauses(rootCauses []RootCause) string {
	result := ""
	for i, rc := range rootCauses {
		result += fmt.Sprintf("%d. %s: %s (置信度: %.2f, 严重程度: %s)\n", i+1, rc.Service, rc.Description, rc.Confidence, rc.Severity)
	}
	return result
}

// selectBestRootCause 选择最佳根因假设
func (a *DecisionAgent) selectBestRootCause(rootCauses []RootCause) RootCause {
	if len(rootCauses) == 0 {
		return RootCause{
			Description: "未找到明确的根因",
			Confidence:  0.3,
			Severity:    "LOW",
		}
	}

	best := rootCauses[0]
	for _, rc := range rootCauses[1:] {
		if rc.Confidence > best.Confidence {
			best = rc
		}
	}

	return best
}
