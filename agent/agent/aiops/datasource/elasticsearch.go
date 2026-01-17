package datasource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"jas-agent/agent/agent/aiops/agents"
	"jas-agent/agent/agent/aiops/framework"
	"net/http"
	"time"
)

// ElasticsearchLogDataSource Elasticsearch 日志数据源适配器
type ElasticsearchLogDataSource struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	username   string
	password   string
}

// NewElasticsearchLogDataSource 创建 Elasticsearch 日志数据源
func NewElasticsearchLogDataSource(baseURL, username, password string, timeout time.Duration) *ElasticsearchLogDataSource {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &ElasticsearchLogDataSource{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
		timeout:    timeout,
		username:   username,
		password:   password,
	}
}

// FetchLogs 获取日志数据
func (e *ElasticsearchLogDataSource) FetchLogs(
	ctx context.Context,
	services []string,
	timeRange framework.TimeRange,
) ([]agents.LogEntry, error) {
	allLogs := make([]agents.LogEntry, 0)

	// 为每个服务查询日志
	for _, service := range services {
		logs, err := e.queryLogs(ctx, service, timeRange)
		if err != nil {
			// 记录错误但继续查询其他服务
			continue
		}
		allLogs = append(allLogs, logs...)
	}

	return allLogs, nil
}

// queryLogs 查询单个服务的日志
func (e *ElasticsearchLogDataSource) queryLogs(
	ctx context.Context,
	service string,
	timeRange framework.TimeRange,
) ([]agents.LogEntry, error) {
	// 构建索引名称（支持多种命名模式）
	indices := []string{
		fmt.Sprintf("backend-%s-*", service),
	}
	// 尝试每个索引模式
	var logs []agents.LogEntry
	var lastErr error

	for _, index := range indices {
		result, err := e.searchLogs(ctx, index, service, timeRange)
		if err == nil && len(result) > 0 {
			logs = result
			break
		}
		if err != nil {
			lastErr = err
		}
	}

	if len(logs) == 0 && lastErr != nil {
		return nil, fmt.Errorf("failed to query logs for service %s: %w", service, lastErr)
	}

	return logs, nil
}

// QueryLogsWithIndex 使用指定的索引查询日志
func (e *ElasticsearchLogDataSource) QueryLogsWithIndex(
	ctx context.Context,
	index string,
	service string,
	timeRange framework.TimeRange,
) ([]agents.LogEntry, error) {
	return e.searchLogs(ctx, index, service, timeRange)
}

// searchLogs 搜索日志
func (e *ElasticsearchLogDataSource) searchLogs(
	ctx context.Context,
	index, service string,
	timeRange framework.TimeRange,
) ([]agents.LogEntry, error) {
	// 构建 Elasticsearch 查询
	query := e.buildQuery(service, timeRange)

	// 构建请求体
	searchBody := map[string]interface{}{
		"query": query,
		"size":  1000, // 最多返回1000条日志
		"sort": []map[string]interface{}{
			{
				"@timestamp": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// 确保 baseURL 不以 / 结尾
	baseURL := e.baseURL
	if len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}

	// 构建请求URL
	searchURL := fmt.Sprintf("%s/%s/_search", baseURL, index)

	// 创建HTTP请求（Elasticsearch _search API 使用 POST）
	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 添加认证（如果提供）
	if e.username != "" && e.password != "" {
		req.SetBasicAuth(e.username, e.password)
	}

	// 发送请求
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elasticsearch query failed (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result elasticsearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 转换为 LogEntry
	logs := make([]agents.LogEntry, 0)
	for _, hit := range result.Hits.Hits {
		logEntry := e.convertHitToLogEntry(hit, service)
		if logEntry != nil {
			logs = append(logs, *logEntry)
		}
	}

	return logs, nil
}

// buildQuery 构建 Elasticsearch 查询
func (e *ElasticsearchLogDataSource) buildQuery(service string, timeRange framework.TimeRange) map[string]interface{} {
	startTime := time.Unix(timeRange.StartTime, 0).Format(time.RFC3339)
	endTime := time.Unix(timeRange.EndTime, 0).Format(time.RFC3339)

	must := []map[string]interface{}{
		{
			"range": map[string]interface{}{
				"@timestamp": map[string]interface{}{
					"gte": startTime,
					"lte": endTime,
				},
			},
		},
	}

	// 添加日志级别过滤（尝试多个可能的字段名）
	// 使用 should 子句匹配任意一个字段，然后用 bool 包装确保至少匹配一个
	shouldLevel := []map[string]interface{}{

		{
			"terms": map[string]interface{}{
				"L": []string{"ERROR", "WARN", "error", "warn", "Error", "Warn"},
			},
		},
	}

	// 将级别过滤包装在 bool 查询中，确保至少匹配一个字段
	levelFilter := map[string]interface{}{
		"bool": map[string]interface{}{
			"should":               shouldLevel,
			"minimum_should_match": 1,
		},
	}

	must = append(must, levelFilter)

	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"must": must,
		},
	}

	return query
}

// convertHitToLogEntry 将 Elasticsearch hit 转换为 LogEntry
func (e *ElasticsearchLogDataSource) convertHitToLogEntry(hit elasticsearchHit, defaultService string) *agents.LogEntry {
	source := hit.Source

	// 提取字段
	service := e.extractString(source, "service", "app", "service_name", "pod")
	if service == "" {
		service = defaultService
	}

	level := e.extractString(source, "level", "L", "log.level", "severity", "level_name")
	if level == "" {
		level = "INFO"
	}

	message := e.extractString(source, "message", "log.message", "msg", "M", "message_text")
	if message == "" {
		message = e.extractString(source, "@message", "text", "content")
	}

	traceID := e.extractString(source, "TRACE_ID", "trace_id", "traceId", "traceid", "x-trace-id", "request_id")

	// 解析时间戳
	var timestamp int64
	if tsStr := e.extractString(source, "@timestamp", "timestamp", "time", "log_time"); tsStr != "" {
		if t, err := time.Parse(time.RFC3339, tsStr); err == nil {
			timestamp = t.Unix()
		}
	}
	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}

	// 提取标签
	labels := make(map[string]string)
	labelKeys := []string{"host", "pod", "namespace", "container", "environment"}
	for _, key := range labelKeys {
		if val := e.extractString(source, key); val != "" {
			labels[key] = val
		}
	}

	// 构建原始日志字符串
	rawBytes, _ := json.Marshal(source)
	raw := string(rawBytes)

	return &agents.LogEntry{
		Service:   service,
		Level:     level,
		Timestamp: timestamp,
		Message:   message,
		TraceID:   traceID,
		Labels:    labels,
		Raw:       raw,
	}
}

// extractString 从 map 中提取字符串值（尝试多个键）
func (e *ElasticsearchLogDataSource) extractString(source map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := source[key]; ok {
			switch v := val.(type) {
			case string:
				if v != "" {
					return v
				}
			case float64:
				return fmt.Sprintf("%.0f", v)
			case int:
				return fmt.Sprintf("%d", v)
			}
		}
	}
	return ""
}

// elasticsearchResponse Elasticsearch 查询响应
type elasticsearchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []elasticsearchHit `json:"hits"`
	} `json:"hits"`
}

// elasticsearchHit Elasticsearch 命中结果
type elasticsearchHit struct {
	Source map[string]interface{} `json:"_source"`
}
