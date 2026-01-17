package agents

import (
	"context"
	"fmt"
	"jas-agent/agent/agent/aiops/framework"
	"math"

	"jas-agent/agent/core"
	"jas-agent/agent/llm"
)

// MetricsAgent 指标分析智能体
// 擅长时序数据异常检测，能发现指标间的相关性
type MetricsAgent struct {
	*BaseAgent
	dataSource DataSource
}

// NewMetricsAgent 创建指标分析智能体
func NewMetricsAgent(ctx *framework.CollaborationContext, dataSource DataSource) *MetricsAgent {
	base := NewBaseAgent(
		framework.RoleMetrics,
		"指标分析智能体",
		`你是一个专业的指标分析专家。你的职责是：
1. 分析服务指标的时序异常（如CPU使用率、内存使用率、QPS、错误率、延迟等）
2. 发现指标间的相关性（如CPU使用率升高与数据库慢查询增长同步发生）
3. 使用动态基线、3-sigma等算法检测异常
4. 识别指标异常的模式和趋势

分析时应该：
- 关注系统、应用、业务三级黄金指标
- 识别异常的时间点和持续时间
- 分析异常指标之间的因果关系
- 评估异常对业务的影响程度`,
		ctx.Chat(),
		ctx.Memory(),
	)

	agent := &MetricsAgent{
		BaseAgent:  base,
		dataSource: dataSource,
	}

	return agent
}

// Execute 执行指标分析
func (a *MetricsAgent) Execute(ctx context.Context, task *framework.Task) (*framework.TaskResult, error) {
	result := &framework.TaskResult{
		TaskID:    task.ID,
		AgentRole: framework.RoleMetrics,
		Success:   true,
		Evidence:  make([]framework.Evidence, 0),
		Findings:  make([]framework.Finding, 0),
		Metadata:  make(map[string]interface{}),
	}

	// 1. 获取指标数据
	metricsData, err := a.fetchMetrics(ctx, task)
	if err != nil {
		result.Success = false
		result.Metadata["error"] = err.Error()
		return result, nil
	}

	// 2. 异常检测
	anomalies := a.detectAnomalies(metricsData)

	// 3. 相关性分析
	correlations := a.analyzeCorrelations(metricsData)

	// 4. 构建证据和发现
	for _, anomaly := range anomalies {
		result.Evidence = append(result.Evidence, framework.Evidence{
			Type:        "metrics",
			Service:     anomaly.Service,
			Timestamp:   anomaly.Timestamp,
			Description: anomaly.Description,
			Data:        anomaly.Data,
			Score:       anomaly.Score,
		})

		result.Findings = append(result.Findings, framework.Finding{
			Type:        "anomaly",
			Service:     anomaly.Service,
			Description: anomaly.Description,
			Severity:    anomaly.Severity,
			Score:       anomaly.Score,
		})
	}

	// 5. 使用 LLM 进行高级分析
	analysis := a.analyzeWithLLM(ctx, task, metricsData, anomalies, correlations)

	result.Metadata["metrics_data"] = metricsData
	result.Metadata["anomalies"] = anomalies
	result.Metadata["correlations"] = correlations
	result.Metadata["analysis"] = analysis
	result.Confidence = a.calculateConfidence(anomalies, correlations)

	return result, nil
}

// MetricsData 指标数据
type MetricsData struct {
	Service   string
	Metric    string // cpu_usage, memory_usage, qps, error_rate, latency
	Timestamp int64
	Value     float64
	Labels    map[string]interface{}
}

// Anomaly 异常点
type Anomaly struct {
	Service     string
	Metric      string
	Timestamp   int64
	Value       float64
	Expected    float64
	Deviation   float64
	Description string
	Severity    string
	Score       float64
	Data        interface{}
}

// fetchMetrics 获取指标数据
func (a *MetricsAgent) fetchMetrics(ctx context.Context, task *framework.Task) ([]MetricsData, error) {
	if a.dataSource == nil {
		// 如果没有数据源，返回模拟数据
		return a.generateMockMetrics(task), nil
	}

	// 从数据源获取指标
	return a.dataSource.FetchMetrics(ctx, task.Services, task.TimeRange)
}

// generateMockMetrics 生成模拟指标数据（用于测试）
func (a *MetricsAgent) generateMockMetrics(task *framework.Task) []MetricsData {
	data := make([]MetricsData, 0)
	baseTime := task.TimeRange.StartTime

	for _, service := range task.Services {
		// 生成 CPU 使用率
		for i := 0; i < 60; i++ {
			data = append(data, MetricsData{
				Service:   service,
				Metric:    "cpu_usage",
				Timestamp: baseTime + int64(i*60),
				Value:     30 + float64(i%20)*2, // 模拟波动
			})
		}

		// 生成错误率
		for i := 0; i < 60; i++ {
			data = append(data, MetricsData{
				Service:   service,
				Metric:    "error_rate",
				Timestamp: baseTime + int64(i*60),
				Value:     0.01 + float64(i%10)*0.01, // 模拟波动
			})
		}
	}

	return data
}

// detectAnomalies 异常检测（使用 3-sigma 方法）
func (a *MetricsAgent) detectAnomalies(data []MetricsData) []Anomaly {
	anomalies := make([]Anomaly, 0)

	// 按服务+指标分组
	groups := make(map[string][]MetricsData)
	for _, d := range data {
		key := fmt.Sprintf("%s:%s", d.Service, d.Metric)
		groups[key] = append(groups[key], d)
	}

	// 对每组数据进行异常检测
	for _, group := range groups {
		if len(group) < 3 {
			continue
		}

		// 计算均值和标准差
		mean := a.calculateMean(group)
		stdDev := a.calculateStdDev(group, mean)

		// 3-sigma 规则
		threshold := mean + 3*stdDev
		lowerThreshold := mean - 3*stdDev

		for _, d := range group {
			if d.Value > threshold || d.Value < lowerThreshold {
				severity := "MEDIUM"
				deviation := math.Abs(d.Value - mean)
				score := math.Min(deviation/(3*stdDev), 1.0)

				if score > 0.8 {
					severity = "CRITICAL"
				} else if score > 0.6 {
					severity = "HIGH"
				}

				anomalies = append(anomalies, Anomaly{
					Service:     d.Service,
					Metric:      d.Metric,
					Timestamp:   d.Timestamp,
					Value:       d.Value,
					Expected:    mean,
					Deviation:   deviation,
					Description: fmt.Sprintf("%s 指标异常：实际值 %.2f，期望值 %.2f，偏差 %.2f", d.Metric, d.Value, mean, deviation),
					Severity:    severity,
					Score:       score,
					Data:        d,
				})
			}
		}
	}

	return anomalies
}

// analyzeCorrelations 相关性分析
func (a *MetricsAgent) analyzeCorrelations(data []MetricsData) []Correlation {
	correlations := make([]Correlation, 0)

	// 按服务分组
	services := make(map[string][]MetricsData)
	for _, d := range data {
		services[d.Service] = append(services[d.Service], d)
	}

	// 分析每个服务内不同指标的相关性
	for service, metrics := range services {
		// 按指标类型分组
		byMetric := make(map[string][]MetricsData)
		for _, m := range metrics {
			byMetric[m.Metric] = append(byMetric[m.Metric], m)
		}

		// 计算常见指标对的相关性
		pairs := [][]string{
			{"cpu_usage", "qps"},
			{"error_rate", "latency"},
			{"memory_usage", "cpu_usage"},
		}

		for _, pair := range pairs {
			if len(byMetric[pair[0]]) > 0 && len(byMetric[pair[1]]) > 0 {
				correlation := a.calculateCorrelation(
					byMetric[pair[0]],
					byMetric[pair[1]],
				)
				if math.Abs(correlation) > 0.7 {
					correlations = append(correlations, Correlation{
						Service:     service,
						Metric1:     pair[0],
						Metric2:     pair[1],
						Coefficient: correlation,
					})
				}
			}
		}
	}

	return correlations
}

// Correlation 相关性
type Correlation struct {
	Service     string
	Metric1     string
	Metric2     string
	Coefficient float64
}

// calculateCorrelation 计算皮尔逊相关系数
func (a *MetricsAgent) calculateCorrelation(x, y []MetricsData) float64 {
	// 按时间戳对齐
	aligned := a.alignByTimestamp(x, y)
	if len(aligned) < 2 {
		return 0
	}

	// 计算均值
	var sumX, sumY float64
	for _, p := range aligned {
		sumX += p.x
		sumY += p.y
	}
	meanX := sumX / float64(len(aligned))
	meanY := sumY / float64(len(aligned))

	// 计算协方差和方差
	var cov, varX, varY float64
	for _, p := range aligned {
		dx := p.x - meanX
		dy := p.y - meanY
		cov += dx * dy
		varX += dx * dx
		varY += dy * dy
	}

	if varX == 0 || varY == 0 {
		return 0
	}

	return cov / math.Sqrt(varX*varY)
}

type point struct {
	x, y float64
}

// alignByTimestamp 按时间戳对齐两个指标序列
func (a *MetricsAgent) alignByTimestamp(x, y []MetricsData) []point {
	// 创建时间戳到值的映射
	xMap := make(map[int64]float64)
	yMap := make(map[int64]float64)

	for _, d := range x {
		xMap[d.Timestamp] = d.Value
	}
	for _, d := range y {
		yMap[d.Timestamp] = d.Value
	}

	// 找到共同的时间戳
	aligned := make([]point, 0)
	for ts := range xMap {
		if yVal, ok := yMap[ts]; ok {
			aligned = append(aligned, point{
				x: xMap[ts],
				y: yVal,
			})
		}
	}

	return aligned
}

// calculateMean 计算均值
func (a *MetricsAgent) calculateMean(data []MetricsData) float64 {
	if len(data) == 0 {
		return 0
	}
	var sum float64
	for _, d := range data {
		sum += d.Value
	}
	return sum / float64(len(data))
}

// calculateStdDev 计算标准差
func (a *MetricsAgent) calculateStdDev(data []MetricsData, mean float64) float64 {
	if len(data) == 0 {
		return 0
	}
	var sum float64
	for _, d := range data {
		diff := d.Value - mean
		sum += diff * diff
	}
	variance := sum / float64(len(data))
	return math.Sqrt(variance)
}

// analyzeWithLLM 使用 LLM 进行高级分析
func (a *MetricsAgent) analyzeWithLLM(
	ctx context.Context,
	task *framework.Task,
	data []MetricsData,
	anomalies []Anomaly,
	correlations []Correlation,
) string {
	prompt := fmt.Sprintf(`分析以下指标数据和异常：

指标数量: %d
异常数量: %d
相关性数量: %d

主要异常：
%s

主要相关性：
%s

请分析：
1. 指标异常的潜在原因
2. 异常指标之间的关联性
3. 对业务的影响评估`,
		len(data),
		len(anomalies),
		len(correlations),
		a.formatAnomalies(anomalies[:minInt(5, len(anomalies))]),
		a.formatCorrelations(correlations[:minInt(5, len(correlations))]),
	)

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

func (a *MetricsAgent) formatAnomalies(anomalies []Anomaly) string {
	result := ""
	for _, a := range anomalies {
		result += fmt.Sprintf("- %s: %s\n", a.Service, a.Description)
	}
	return result
}

func (a *MetricsAgent) formatCorrelations(correlations []Correlation) string {
	result := ""
	for _, c := range correlations {
		result += fmt.Sprintf("- %s: %s 与 %s 相关系数 %.2f\n", c.Service, c.Metric1, c.Metric2, c.Coefficient)
	}
	return result
}

func (a *MetricsAgent) calculateConfidence(anomalies []Anomaly, correlations []Correlation) float64 {
	if len(anomalies) == 0 {
		return 0.3
	}

	// 基于异常数量和相关性计算置信度
	baseConfidence := 0.5
	anomalyBonus := math.Min(float64(len(anomalies))*0.1, 0.3)
	correlationBonus := math.Min(float64(len(correlations))*0.05, 0.2)

	return math.Min(baseConfidence+anomalyBonus+correlationBonus, 1.0)
}

// DataSource 数据源接口
type DataSource interface {
	FetchMetrics(ctx context.Context, services []string, timeRange framework.TimeRange) ([]MetricsData, error)
}
