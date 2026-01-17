package agents

import (
	"context"
	"fmt"
	"jas-agent/agent/agent/aiops/framework"
	"jas-agent/pkg/algorithm"
	"math"
	"regexp"
	"strings"

	"jas-agent/agent/core"
	"jas-agent/agent/llm"
)

// LogsAgent 日志分析智能体
// 具备自然语言处理能力，能从海量日志中快速提取错误模式、异常堆栈和关键事件
type LogsAgent struct {
	*BaseAgent
	dataSource LogDataSource
}

// NewLogsAgent 创建日志分析智能体
func NewLogsAgent(ctx *framework.CollaborationContext, dataSource LogDataSource) *LogsAgent {
	base := NewBaseAgent(
		framework.RoleLogs,
		"日志分析智能体",
		`你是一个专业的日志分析专家。你的职责是：
1. 从海量日志中快速提取错误模式、异常堆栈和关键事件
2. 识别ERROR、WARN级别的日志
3. 使用TF-IDF向量化与K-Means聚类等技术对日志进行自动归类
4. 提取关键信息：错误类型、堆栈跟踪、时间戳、服务名、TraceID等

分析时应该：
- 关注ERROR和WARN级别的日志
- 识别重复出现的错误模式
- 提取异常堆栈信息
- 关联TraceID进行链路追踪
- 识别关键时间点的事件序列`,
		ctx.Chat(),
		ctx.Memory(),
	)

	agent := &LogsAgent{
		BaseAgent:  base,
		dataSource: dataSource,
	}

	return agent
}

// Execute 执行日志分析
func (a *LogsAgent) Execute(ctx context.Context, task *framework.Task) (*framework.TaskResult, error) {
	result := &framework.TaskResult{
		TaskID:    task.ID,
		AgentRole: framework.RoleLogs,
		Success:   true,
		Evidence:  make([]framework.Evidence, 0),
		Findings:  make([]framework.Finding, 0),
		Metadata:  make(map[string]interface{}),
	}

	// 1. 获取日志数据
	logs, err := a.fetchLogs(ctx, task)
	if err != nil {
		result.Success = false
		result.Metadata["error"] = err.Error()
		return result, nil
	}

	// 2. 过滤错误日志
	errorLogs := a.filterErrorLogs(logs)

	// 3. 提取错误模式
	patterns := a.extractPatterns(errorLogs)

	// 4. 聚类分析
	clusters := a.clusterLogs(errorLogs)

	// 5. 提取堆栈信息
	stackTraces := a.extractStackTraces(errorLogs)

	// 6. 构建证据和发现
	for _, log := range errorLogs {
		result.Evidence = append(result.Evidence, framework.Evidence{
			Type:        "logs",
			Service:     log.Service,
			Timestamp:   log.Timestamp,
			Description: log.Message,
			Data:        log,
			Score:       a.calculateLogScore(log),
		})
	}

	for _, pattern := range patterns {
		result.Findings = append(result.Findings, framework.Finding{
			Type:        "error_pattern",
			Service:     pattern.Service,
			Description: fmt.Sprintf("错误模式: %s (出现 %d 次)", pattern.Pattern, pattern.Count),
			Severity:    pattern.Severity,
			Score:       pattern.Score,
		})
	}

	// 7. 使用 LLM 进行高级分析
	analysis := a.analyzeWithLLM(ctx, task, errorLogs, patterns, clusters)

	result.Metadata["total_logs"] = len(logs)
	result.Metadata["error_logs"] = len(errorLogs)
	result.Metadata["patterns"] = patterns
	result.Metadata["clusters"] = len(clusters)
	result.Metadata["stack_traces"] = stackTraces
	result.Metadata["analysis"] = analysis
	result.Confidence = a.calculateConfidence(errorLogs, patterns)

	return result, nil
}

// LogEntry 日志条目
type LogEntry struct {
	Service   string
	Level     string // ERROR, WARN, INFO, DEBUG
	Timestamp int64
	Message   string
	TraceID   string
	Labels    map[string]string
	Raw       string
}

// LogPattern 日志模式
type LogPattern struct {
	Service  string
	Pattern  string
	Count    int
	Severity string
	Score    float64
	Examples []LogEntry
}

// fetchLogs 获取日志数据
func (a *LogsAgent) fetchLogs(ctx context.Context, task *framework.Task) ([]LogEntry, error) {
	if a.dataSource == nil {
		return a.generateMockLogs(task), nil
	}

	return a.dataSource.FetchLogs(ctx, task.Services, task.TimeRange)
}

// generateMockLogs 生成模拟日志（用于测试）
func (a *LogsAgent) generateMockLogs(task *framework.Task) []LogEntry {
	logs := make([]LogEntry, 0)
	baseTime := task.TimeRange.StartTime

	for _, service := range task.Services {
		// 生成一些错误日志
		for i := 0; i < 10; i++ {
			logs = append(logs, LogEntry{
				Service:   service,
				Level:     "ERROR",
				Timestamp: baseTime + int64(i*60),
				Message:   fmt.Sprintf("Failed to connect to database: connection timeout"),
				TraceID:   fmt.Sprintf("trace_%d", i),
			})
		}

		// 生成一些警告日志
		for i := 0; i < 5; i++ {
			logs = append(logs, LogEntry{
				Service:   service,
				Level:     "WARN",
				Timestamp: baseTime + int64(i*120),
				Message:   fmt.Sprintf("High memory usage detected: 85%%"),
				TraceID:   fmt.Sprintf("trace_%d", i),
			})
		}
	}

	return logs
}

// filterErrorLogs 过滤错误日志
func (a *LogsAgent) filterErrorLogs(logs []LogEntry) []LogEntry {
	errorLogs := make([]LogEntry, 0)
	for _, log := range logs {
		if log.Level == "ERROR" || log.Level == "WARN" {
			errorLogs = append(errorLogs, log)
		}
	}
	return errorLogs
}

// extractPatterns 提取错误模式
func (a *LogsAgent) extractPatterns(logs []LogEntry) []LogPattern {
	// 按服务分组
	byService := make(map[string][]LogEntry)
	for _, log := range logs {
		byService[log.Service] = append(byService[log.Service], log)
	}

	patterns := make([]LogPattern, 0)

	// 对每个服务的日志提取模式
	for service, serviceLogs := range byService {
		// 简单的模式提取：归一化消息（移除动态内容）
		patternCount := make(map[string]int)
		patternExamples := make(map[string][]LogEntry)

		for _, log := range serviceLogs {
			pattern := a.normalizeLogMessage(log.Message)
			patternCount[pattern]++
			if len(patternExamples[pattern]) < 3 {
				patternExamples[pattern] = append(patternExamples[pattern], log)
			}
		}

		// 生成模式
		for pattern, count := range patternCount {
			if count >= 2 { // 至少出现2次才算模式
				severity := "MEDIUM"
				if count >= 10 {
					severity = "CRITICAL"
				} else if count >= 5 {
					severity = "HIGH"
				}

				patterns = append(patterns, LogPattern{
					Service:  service,
					Pattern:  pattern,
					Count:    count,
					Severity: severity,
					Score:    math.Min(float64(count)/10.0, 1.0),
					Examples: patternExamples[pattern],
				})
			}
		}
	}

	return patterns
}

// normalizeLogMessage 归一化日志消息（移除动态内容）
func (a *LogsAgent) normalizeLogMessage(message string) string {
	// 移除时间戳
	message = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[\sT]\d{2}:\d{2}:\d{2}`).ReplaceAllString(message, "")

	// 移除数字
	message = regexp.MustCompile(`\d+`).ReplaceAllString(message, "N")

	// 移除引号内容（通常是动态参数）
	message = regexp.MustCompile(`"[^"]*"`).ReplaceAllString(message, "\"*\"")

	// 移除单引号内容
	message = regexp.MustCompile(`'[^']*'`).ReplaceAllString(message, "'*'")

	// 移除UUID
	message = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).ReplaceAllString(message, "UUID")

	// 移除IP地址
	message = regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`).ReplaceAllString(message, "IP")

	return strings.TrimSpace(message)
}

// clusterLogs 日志聚类（使用 Drain3 算法）
func (a *LogsAgent) clusterLogs(logs []LogEntry) map[string][]LogEntry {
	// 创建 Drain3 聚类器
	// maxDepth: 4 (前缀树最大深度)
	// simThreshold: 0.5 (相似度阈值，0.5 表示至少50%的token相同才归为一类)
	drain3 := algorithm.NewDrain3(4, 0.5)

	// 将日志消息添加到聚类器
	logMessages := make([]string, 0, len(logs))
	for _, log := range logs {
		if log.Message != "" {
			logMessages = append(logMessages, log.Message)
			drain3.AddLog(log.Message)
		}
	}

	// 获取聚类结果
	clusters := drain3.GetClusters()

	// 将聚类结果转换为 map[string][]LogEntry 格式
	result := make(map[string][]LogEntry)

	// 为每个聚类创建映射
	for _, cluster := range clusters {
		// 使用模板作为key
		template := cluster.Template
		if template == "" {
			template = fmt.Sprintf("cluster_%d", cluster.ClusterID)
		}

		// 将属于该聚类的日志添加到结果中
		// 由于 Drain3 返回的是原始日志字符串，我们需要匹配日志消息
		for _, log := range logs {
			// 检查日志是否属于当前聚类
			// 通过比较日志消息的模板来判断
			logTemplate := a.createTemplateFromLog(log.Message)
			if a.templatesMatch(template, logTemplate) {
				result[template] = append(result[template], log)
			}
		}

		// 如果聚类中没有匹配的日志，至少创建一个条目
		if len(result[template]) == 0 && len(cluster.Logs) > 0 {
			// 尝试从原始日志中找到匹配的
			for _, logMsg := range cluster.Logs {
				for i, log := range logs {
					if log.Message == logMsg {
						result[template] = append(result[template], logs[i])
						break
					}
				}
			}
		}
	}

	return result
}

// createTemplateFromLog 从日志消息创建模板（使用与 Drain3 相同的逻辑）
func (a *LogsAgent) createTemplateFromLog(log string) string {
	// 使用 Drain3 的 tokenize 和 replaceNumbers 逻辑
	tokens := strings.Fields(strings.TrimSpace(log))
	templateTokens := make([]string, 0, len(tokens))

	for _, token := range tokens {
		// 替换数字为 <*>
		matched, _ := regexp.MatchString(`^\d+$`, token)
		if matched {
			templateTokens = append(templateTokens, "<*>")
			continue
		}
		// 替换IP地址
		ipPattern := regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
		token = ipPattern.ReplaceAllString(token, "<*>")
		// 替换十六进制
		hexPattern := regexp.MustCompile(`0x[0-9a-fA-F]+`)
		token = hexPattern.ReplaceAllString(token, "<*>")
		templateTokens = append(templateTokens, token)
	}

	return strings.Join(templateTokens, " ")
}

// templatesMatch 检查两个模板是否匹配（简化版本）
func (a *LogsAgent) templatesMatch(template1, template2 string) bool {
	// 如果模板完全相同，直接返回 true
	if template1 == template2 {
		return true
	}

	// 计算相似度（使用简单的token匹配）
	tokens1 := strings.Fields(template1)
	tokens2 := strings.Fields(template2)

	if len(tokens1) == 0 || len(tokens2) == 0 {
		return false
	}

	// 计算相同位置的相同token数量
	common := 0
	minLen := len(tokens1)
	if len(tokens2) < minLen {
		minLen = len(tokens2)
	}

	for i := 0; i < minLen; i++ {
		if tokens1[i] == tokens2[i] {
			common++
		}
	}

	// 如果至少50%的token相同，认为匹配
	similarity := float64(common) / float64(len(tokens1))
	return similarity >= 0.5
}

// extractStackTraces 提取堆栈跟踪
func (a *LogsAgent) extractStackTraces(logs []LogEntry) []string {
	stackTraces := make([]string, 0)
	stackRegex := regexp.MustCompile(`(?m)^\s+at\s+.+`)

	for _, log := range logs {
		if matches := stackRegex.FindAllString(log.Message, -1); len(matches) > 0 {
			stackTraces = append(stackTraces, strings.Join(matches, "\n"))
		}
	}

	return stackTraces
}

// calculateLogScore 计算日志重要性评分
func (a *LogsAgent) calculateLogScore(log LogEntry) float64 {
	score := 0.5

	if log.Level == "ERROR" {
		score += 0.4
	} else if log.Level == "WARN" {
		score += 0.2
	}

	// 包含关键字增加分数
	keywords := []string{"exception", "failed", "error", "timeout", "deadlock"}
	lowerMsg := strings.ToLower(log.Message)
	for _, keyword := range keywords {
		if strings.Contains(lowerMsg, keyword) {
			score += 0.1
			break
		}
	}

	return math.Min(score, 1.0)
}

// analyzeWithLLM 使用 LLM 进行高级分析
func (a *LogsAgent) analyzeWithLLM(
	ctx context.Context,
	task *framework.Task,
	logs []LogEntry,
	patterns []LogPattern,
	clusters map[string][]LogEntry,
) string {
	// 格式化聚类信息
	clustersInfo := a.formatClusters(clusters)

	prompt := fmt.Sprintf(`分析以下日志数据和错误模式：

日志总数: %d
错误日志数: %d
错误模式数: %d
聚类数量: %d

主要错误模式：
%s

日志聚类结果（使用 Drain3 算法）：
%s

请分析：
1. 主要错误类型和原因
2. 错误发生的时间分布
3. 错误之间的关联性
4. 聚类结果反映的日志模式特征
5. 高频出现的日志模板及其可能原因
6. 可能导致的业务影响`,
		len(logs),
		len(logs),
		len(patterns),
		len(clusters),
		a.formatPatterns(patterns[:minInt(5, len(patterns))]),
		clustersInfo)

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
		return "LLM 分析失败: " + err.Error()
	}

	return resp.Content()
}

func (a *LogsAgent) formatPatterns(patterns []LogPattern) string {
	result := ""
	for _, p := range patterns {
		result += fmt.Sprintf("- %s: %s (出现 %d 次)\n", p.Service, p.Pattern, p.Count)
	}
	return result
}

// formatClusters 格式化聚类信息用于 LLM 分析
func (a *LogsAgent) formatClusters(clusters map[string][]LogEntry) string {
	if len(clusters) == 0 {
		return "无聚类结果"
	}

	var result strings.Builder

	// 按聚类大小排序（从大到小）
	type clusterInfo struct {
		template string
		count    int
		logs     []LogEntry
	}

	clusterList := make([]clusterInfo, 0, len(clusters))
	for template, logs := range clusters {
		clusterList = append(clusterList, clusterInfo{
			template: template,
			count:    len(logs),
			logs:     logs,
		})
	}

	// 排序
	for i := 0; i < len(clusterList)-1; i++ {
		for j := i + 1; j < len(clusterList); j++ {
			if clusterList[i].count < clusterList[j].count {
				clusterList[i], clusterList[j] = clusterList[j], clusterList[i]
			}
		}
	}

	// 只显示前10个最大的聚类，避免 prompt 过长
	maxClusters := 10
	if len(clusterList) < maxClusters {
		maxClusters = len(clusterList)
	}

	for i := 0; i < maxClusters; i++ {
		cluster := clusterList[i]
		result.WriteString(fmt.Sprintf("\n聚类 #%d (共 %d 条日志):\n", i+1, cluster.count))
		result.WriteString(fmt.Sprintf("  模板: %s\n", cluster.template))

		// 显示该聚类的服务分布
		serviceCount := make(map[string]int)
		for _, log := range cluster.logs {
			if log.Service != "" {
				serviceCount[log.Service]++
			}
		}
		if len(serviceCount) > 0 {
			result.WriteString("  涉及服务: ")
			services := make([]string, 0, len(serviceCount))
			for svc, count := range serviceCount {
				services = append(services, fmt.Sprintf("%s(%d)", svc, count))
			}
			result.WriteString(strings.Join(services, ", "))
			result.WriteString("\n")
		}

		// 显示该聚类的示例日志（最多3条）
		exampleCount := 3
		if len(cluster.logs) < exampleCount {
			exampleCount = len(cluster.logs)
		}
		result.WriteString("  示例日志:\n")
		for j := 0; j < exampleCount; j++ {
			log := cluster.logs[j]
			// 截断过长的日志
			msg := log.Message
			if len(msg) > 200 {
				msg = msg[:200] + "..."
			}
			result.WriteString(fmt.Sprintf("    - [%s] %s\n", log.Level, msg))
		}
	}

	if len(clusterList) > maxClusters {
		result.WriteString(fmt.Sprintf("\n... 还有 %d 个聚类未显示\n", len(clusterList)-maxClusters))
	}

	return result.String()
}

func (a *LogsAgent) calculateConfidence(logs []LogEntry, patterns []LogPattern) float64 {
	if len(logs) == 0 {
		return 0.2
	}

	baseConfidence := 0.5
	logBonus := math.Min(float64(len(logs))/50.0, 0.3)
	patternBonus := math.Min(float64(len(patterns))*0.1, 0.2)

	return math.Min(baseConfidence+logBonus+patternBonus, 1.0)
}

// LogDataSource 日志数据源接口
type LogDataSource interface {
	FetchLogs(ctx context.Context, services []string, timeRange framework.TimeRange) ([]LogEntry, error)
}
