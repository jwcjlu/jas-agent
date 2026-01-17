package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"jas-agent/agent/agent/aiops/agents"
	"jas-agent/agent/agent/aiops/framework"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PrometheusDataSource Prometheus 指标数据源适配器
type PrometheusDataSource struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewPrometheusDataSource 创建 Prometheus 数据源
func NewPrometheusDataSource(baseURL string, timeout time.Duration) *PrometheusDataSource {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &PrometheusDataSource{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// FetchMetrics 获取指标数据
func (p *PrometheusDataSource) FetchMetrics(
	ctx context.Context,
	services []string,
	timeRange framework.TimeRange,
) ([]agents.MetricsData, error) {
	allMetrics := make([]agents.MetricsData, 0)

	// 定义要查询的指标列表
	metrics := []string{
		"cpu_usage",
		"memory_usage",
		"qps",
		"error_rate",
		"latency",
	}

	// 为每个服务和每个指标查询数据
	for _, service := range services {
		for _, metric := range metrics {
			data, err := p.queryMetric(ctx, service, metric, timeRange)
			if err != nil {
				// 记录错误但继续查询其他指标
				continue
			}
			allMetrics = append(allMetrics, data...)
		}
	}

	return allMetrics, nil
}

// queryMetric 查询单个指标
func (p *PrometheusDataSource) queryMetric(
	ctx context.Context,
	service, metric string,
	timeRange framework.TimeRange,
) ([]agents.MetricsData, error) {
	// 构建 Prometheus 查询
	query := p.buildQuery(service, metric)

	// 设置查询参数
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", strconv.FormatInt(timeRange.StartTime, 10))
	params.Set("end", strconv.FormatInt(timeRange.EndTime, 10))
	params.Set("step", "60s") // 1分钟步长

	// 构建请求URL
	queryURL := fmt.Sprintf("%s/api/v1/query_range?%s", p.baseURL, params.Encode())

	// 发送HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus query failed (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result prometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 转换为 MetricsData
	metricsData := make([]agents.MetricsData, 0)
	for _, series := range result.Data.Result {
		serviceName := p.extractServiceFromLabels(series.Metric)
		if serviceName == "" {
			serviceName = service
		}

		for _, value := range series.Values {
			timestamp := int64(value.(float64))
			val := value.(string)
			floatVal, err := strconv.ParseFloat(val, 64)
			if err != nil {
				continue
			}

			metricsData = append(metricsData, agents.MetricsData{
				Service:   serviceName,
				Metric:    metric,
				Timestamp: timestamp,
				Value:     floatVal,
				Labels:    series.Metric,
			})
		}
	}

	return metricsData, nil
}

// buildQuery 构建 Prometheus 查询语句
func (p *PrometheusDataSource) buildQuery(service, metric string) string {
	// 根据指标类型构建不同的查询
	switch metric {
	case "cpu_usage":
		return fmt.Sprintf(`rate(container_cpu_usage_seconds_total{pod=~"%s.*"}[5m]) * 100`, service)
	case "memory_usage":
		return fmt.Sprintf(`container_memory_usage_bytes{pod=~"%s.*"} / container_spec_memory_limit_bytes{pod=~"%s.*"} * 100`, service, service)
	case "qps":
		return fmt.Sprintf(`rate(http_requests_total{service="%s"}[5m])`, service)
	case "error_rate":
		return fmt.Sprintf(`rate(http_requests_total{service="%s",status=~"5.."}[5m]) / rate(http_requests_total{service="%s"}[5m]) * 100`, service, service)
	case "latency":
		return fmt.Sprintf(`histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{service="%s"}[5m]))`, service)
	default:
		// 通用查询模式
		return fmt.Sprintf(`%s{service="%s"}`, metric, service)
	}
}

// extractServiceFromLabels 从标签中提取服务名
func (p *PrometheusDataSource) extractServiceFromLabels(labels map[string]interface{}) string {
	// 尝试多个可能的标签键
	keys := []string{"service", "service_name", "app", "job"}
	for _, key := range keys {
		if val, ok := labels[key].(string); ok && val != "" {
			return val
		}
	}
	return ""
}

// prometheusResponse Prometheus API 响应结构
type prometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string             `json:"resultType"`
		Result     []prometheusSeries `json:"result"`
	} `json:"data"`
}

// prometheusSeries Prometheus 时间序列数据
type prometheusSeries struct {
	Metric map[string]interface{} `json:"metric"`
	Values []interface{}          `json:"values"` // [[timestamp, value], ...]
}
