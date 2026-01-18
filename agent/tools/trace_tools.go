package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"jas-agent/agent/core"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TraceConnection Trace连接配置
type TraceConnection struct {
	Type     string // "jaeger" 或 "skywalking"
	BaseURL  string
	Username string
	Password string
	Client   *http.Client
}

// NewTraceConnection 创建Trace连接
func NewTraceConnection(traceType, baseURL, username, password string) *TraceConnection {
	return &TraceConnection{
		Type:     traceType,
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		Client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// doRequest 执行HTTP请求
func (conn *TraceConnection) doRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	url := fmt.Sprintf("%s%s", conn.BaseURL, path)

	var reqBody io.Reader
	if body != nil {
		reqBody = strings.NewReader(string(body))
	}
	fmt.Println(url)

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if conn.Username != "" && conn.Password != "" {
		req.SetBasicAuth(conn.Username, conn.Password)
	}

	resp, err := conn.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("trace service error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Span 表示调用链中的一个Span
type Span struct {
	TraceID      string                 `json:"traceId"`
	SpanID       string                 `json:"spanId"`
	ParentSpanID string                 `json:"parentSpanId,omitempty"` // 父Span ID
	ServiceName  string                 `json:"serviceName"`
	Operation    string                 `json:"operation"`
	StartTime    int64                  `json:"startTime"` // 微秒
	Duration     int64                  `json:"duration"`  // 微秒
	Tags         map[string]interface{} `json:"tags"`
	Logs         []SpanLog              `json:"logs"`
	Error        bool                   `json:"error"`
	Children     []*Span                `json:"children,omitempty"` // 子Span列表（被调用的服务）
	ProcessID    string                 `json:"processID"`
}

func (span *Span) GetServerName(trace *Trace) string {

	for _, s := range span.Children {
		if s.Operation == span.Operation {
			process, ok := trace.Processes[s.ProcessID]
			if ok {
				return process.ServiceName
			}
		}
	}
	process, ok := trace.Processes[span.ProcessID]
	if ok {
		return process.ServiceName
	}
	return ""

}

// SpanLog Span中的日志记录
type SpanLog struct {
	Timestamp int64                  `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields"`
}

// Trace 完整的调用链
type Trace struct {
	TraceID   string             `json:"traceId"`
	Spans     []*Span            `json:"spans"`
	StartTime int64              `json:"startTime"`
	Duration  int64              `json:"duration"`
	Error     bool               `json:"error"`
	Processes map[string]Process `json:"processes"`
}

type Process struct {
	ServiceName string `json:"serviceName"`
}

// QueryTrace 查询Trace工具
type QueryTrace struct {
	conn *TraceConnection
}

func NewQueryTrace(conn *TraceConnection) *QueryTrace {
	return &QueryTrace{conn: conn}
}

func (t *QueryTrace) Name() string {
	return "query_trace"
}

func (t *QueryTrace) Description() string {
	return "根据Trace ID查询完整的调用链信息。输入：JSON格式包含traceId。返回：调用链的完整信息，包括所有Span的详细信息（服务名、操作名、开始时间、持续时间、错误标记、标签等）。"
}

func (t *QueryTrace) Input() any {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"traceId": map[string]interface{}{
				"type":        "string",
				"description": "Trace ID",
			},
		},
		"required": []string{"traceId"},
	}
}

func (t *QueryTrace) Type() core.ToolType {
	return core.Normal
}

func (t *QueryTrace) Handler(ctx context.Context, input string) (string, error) {
	var req struct {
		TraceID string `json:"traceId"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("JSON解析失败: %w\n\n输入内容:\n%s\n\n请确保JSON格式正确", err, input)
	}

	if req.TraceID == "" {
		return "", fmt.Errorf("traceId is required")
	}

	var trace *Trace
	var err error

	switch strings.ToLower(t.conn.Type) {
	case "jaeger":
		trace, err = t.queryJaegerTrace(ctx, req.TraceID)
	case "skywalking":
		trace = nil
	default:
		return "", fmt.Errorf("unsupported trace type: %s, supported types: jaeger, skywalking", t.conn.Type)
	}

	if err != nil {
		return "", err
	}

	// 分析Trace，找出问题Span
	problemSpans := t.analyzeProblemSpans(trace)

	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Trace查询结果 ===\n\n"))
	result.WriteString(fmt.Sprintf("Trace ID: %s\n", trace.TraceID))
	result.WriteString(fmt.Sprintf("总Span数: %d\n", len(trace.Spans)))
	result.WriteString(fmt.Sprintf("开始时间: %s\n", time.Unix(0, trace.StartTime*1000).Format("2006-01-02 15:04:05.000")))
	result.WriteString(fmt.Sprintf("总耗时: %d ms\n", trace.Duration/1000))
	result.WriteString(fmt.Sprintf("是否有错误: %v\n\n", trace.Error))

	// 列出所有被调用的服务
	if len(problemSpans) > 0 {
		result.WriteString("=== 问题Span识别 ===\n")
		for i, span := range problemSpans {
			result.WriteString(fmt.Sprintf("\n问题Span #%d:\n", i+1))
			result.WriteString(fmt.Sprintf("  - 服务名: %s\n", span.GetServerName(trace)))
			result.WriteString(fmt.Sprintf("  - 操作名: %s\n", span.Operation))
			result.WriteString(fmt.Sprintf("  - Span ID: %s\n", span.SpanID))
			result.WriteString(fmt.Sprintf("  - 开始时间: %s\n", time.Unix(0, span.StartTime*1000).Format("2006-01-02 15:04:05.000")))
			result.WriteString(fmt.Sprintf("  - 持续时间: %d ms\n", span.Duration/1000))
			result.WriteString(fmt.Sprintf("  - 错误标记: %v\n", span.Error))

			// 显示被调服务（子Span）
			if len(span.Children) > 0 {
				result.WriteString("  - 被调服务:\n")
				for j, child := range span.Children {
					result.WriteString(fmt.Sprintf("    %d. 服务名: %s, 操作名: %s, Span ID: %s",
						j+1, child.ServiceName, child.Operation, child.SpanID))
					if child.Error {
						result.WriteString(" ⚠️ [有错误]")
					}
					result.WriteString("\n")
				}
			} else {
				// 如果没有在Children中，尝试从所有Span中查找parentSpanID匹配的子Span
				childSpans := t.findChildSpans(trace, span.SpanID)
				if len(childSpans) > 0 {
					result.WriteString("  - 被调服务:\n")
					for j, child := range childSpans {
						result.WriteString(fmt.Sprintf("    %d. 服务名: %s, 操作名: %s, Span ID: %s",
							j+1, child.ServiceName, child.Operation, child.SpanID))
						if child.Error {
							result.WriteString(" ⚠️ [有错误]")
						}
						result.WriteString("\n")
					}
				}
			}

			if len(span.Tags) > 0 {
				result.WriteString("  - 标签:\n")
				for k, v := range span.Tags {
					result.WriteString(fmt.Sprintf("    * %s: %v\n", k, v))
				}
			}
		}
		result.WriteString("\n")
	}

	/*// 输出完整的Span列表

	traceJSON, _ := json.MarshalIndent(trace, "", "  ")
	result.WriteString(fmt.Sprintf("\n=== JSON格式数据 ===\n%s\n", string(traceJSON)))*/

	return result.String(), nil
}

// queryJaegerTrace 查询Jaeger Trace
func (t *QueryTrace) queryJaegerTrace(ctx context.Context, traceID string) (*Trace, error) {
	// Jaeger API: GET /api/traces/{traceId}
	path := fmt.Sprintf("/api/traces/%s", traceID)
	respBody, err := t.conn.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("查询Jaeger Trace失败: %w", err)
	}

	var jaegerResp struct {
		Data []struct {
			TraceID string `json:"traceID"`
			Spans   []struct {
				TraceID    string `json:"traceID"`
				SpanID     string `json:"spanID"`
				References []struct {
					TraceID string `json:"traceID"`
					SpanID  string `json:"spanID"`
					RefType string `json:"refType"` // "CHILD_OF" 或 "FOLLOWS_FROM"
				} `json:"references"`
				Operation string `json:"operationName"`
				StartTime int64  `json:"startTime"` // 微秒
				Duration  int64  `json:"duration"`  // 微秒
				Tags      []struct {
					Key   string      `json:"key"`
					Value interface{} `json:"value"`
				} `json:"tags"`
				Logs []struct {
					Timestamp int64 `json:"timestamp"`
					Fields    []struct {
						Key   string      `json:"key"`
						Value interface{} `json:"value"`
					} `json:"fields"`
				} `json:"logs"`
				ProcessID string `json:"processID"`
			} `json:"spans"`
			Processes map[string]Process `json:"processes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &jaegerResp); err != nil {
		return nil, fmt.Errorf("解析Jaeger响应失败: %w", err)
	}

	if len(jaegerResp.Data) == 0 {
		return nil, fmt.Errorf("未找到Trace ID: %s", traceID)
	}

	traceData := jaegerResp.Data[0]
	trace := &Trace{
		TraceID: traceData.TraceID,
		Spans:   make([]*Span, 0, len(traceData.Spans)),
	}

	var minStartTime int64 = 0
	var maxEndTime int64 = 0

	for _, jaegerSpan := range traceData.Spans {
		span := &Span{
			TraceID:   jaegerSpan.TraceID,
			SpanID:    jaegerSpan.SpanID,
			Operation: "",
			StartTime: jaegerSpan.StartTime,
			Duration:  jaegerSpan.Duration,
			Tags:      make(map[string]interface{}),
			Logs:      make([]SpanLog, 0),
			Error:     false,
			Children:  make([]*Span, 0),
			ProcessID: jaegerSpan.ProcessID,
		}

		// 解析Operation（可能是对象或字符串）
		if jaegerSpan.Operation != "" {
			span.Operation = jaegerSpan.Operation
		}

		// 如果没有references但有parentSpanID，使用parentSpanID
		if span.ParentSpanID == "" && len(jaegerSpan.References) > 0 {
			for _, ref := range jaegerSpan.References {
				if ref.RefType == "CHILD_OF" {
					span.ParentSpanID = ref.SpanID
					break
				}
			}
		}

		// 提取服务名和错误标记
		for _, tag := range jaegerSpan.Tags {
			span.Tags[tag.Key] = tag.Value
			if tag.Key == "service.name" || tag.Key == "component" {
				span.ServiceName = fmt.Sprintf("%v", tag.Value)
			}
			if tag.Key == "error" || tag.Key == "http.status_code" {
				errValue := fmt.Sprintf("%v", tag.Value)
				if tag.Key == "error" && (errValue == "true" || errValue == "True") {
					span.Error = true
					trace.Error = true
				}
				// 检查HTTP状态码
				if tag.Key == "http.status_code" {
					if code, err := strconv.Atoi(errValue); err == nil && code >= 500 {
						span.Error = true
						trace.Error = true
					}
				}
				if reply, ok := span.Tags["reply"]; ok {
					if !strings.HasPrefix(parseCodeFromJSON(fmt.Sprintf("%v", reply)), "0") {
						span.Error = true
						trace.Error = true
					}

				}
			}
		}

		// 提取日志
		for _, log := range jaegerSpan.Logs {
			logFields := make(map[string]interface{})
			for _, field := range log.Fields {
				logFields[field.Key] = field.Value
			}
			span.Logs = append(span.Logs, SpanLog{
				Timestamp: log.Timestamp,
				Fields:    logFields,
			})
		}

		trace.Spans = append(trace.Spans, span)

		// 计算时间范围
		if minStartTime == 0 || span.StartTime < minStartTime {
			minStartTime = span.StartTime
		}
		if spanEndTime := span.StartTime + span.Duration; spanEndTime > maxEndTime {
			maxEndTime = spanEndTime
		}
	}
	trace.StartTime = minStartTime
	trace.Duration = maxEndTime - minStartTime
	trace.Processes = traceData.Processes
	// 建立父子关系
	t.buildSpanRelationships(trace)

	return trace, nil
}

// buildSpanRelationships 建立Span之间的父子关系
func (t *QueryTrace) buildSpanRelationships(trace *Trace) {
	// 创建Span ID到Span的映射
	spanMap := make(map[string]*Span)
	for _, span := range trace.Spans {
		spanMap[span.SpanID] = span
	}

	// 建立父子关系
	for _, span := range trace.Spans {
		if span.ParentSpanID != "" {
			if parent, ok := spanMap[span.ParentSpanID]; ok {
				parent.Children = append(parent.Children, span)
			}
		}
	}
}

// findChildSpans 查找指定Span ID的子Span（被调服务）
func (t *QueryTrace) findChildSpans(trace *Trace, spanID string) []*Span {
	children := make([]*Span, 0)
	for _, span := range trace.Spans {
		if span.ParentSpanID == spanID {
			children = append(children, span)
		}
	}
	return children
}

// CalledServiceInfo 被调服务信息
type CalledServiceInfo struct {
	ServiceName    string
	OperationCount int
	Operations     []string
	HasError       bool
}

// getCalledServices 获取Trace中所有被调用的服务
func (t *QueryTrace) getCalledServices(trace *Trace) []CalledServiceInfo {
	serviceMap := make(map[string]*CalledServiceInfo)

	// 遍历所有Span，找出有父Span的（即被调用的服务）
	for _, span := range trace.Spans {
		// 如果有父Span，说明这是一个被调用的服务
		if span.ParentSpanID != "" && span.ServiceName != "" {
			if service, exists := serviceMap[span.ServiceName]; exists {
				// 检查操作是否已存在
				opExists := false
				for _, op := range service.Operations {
					if op == span.Operation {
						opExists = true
						break
					}
				}
				if !opExists && span.Operation != "" {
					service.Operations = append(service.Operations, span.Operation)
					service.OperationCount++
				}
				if span.Error {
					service.HasError = true
				}
			} else {
				// 创建新的服务信息
				operations := make([]string, 0)
				if span.Operation != "" {
					operations = append(operations, span.Operation)
				}
				serviceMap[span.ServiceName] = &CalledServiceInfo{
					ServiceName:    span.ServiceName,
					OperationCount: 1,
					Operations:     operations,
					HasError:       span.Error,
				}
			}
		}
	}

	// 转换为切片并排序
	result := make([]CalledServiceInfo, 0, len(serviceMap))
	for _, service := range serviceMap {
		result = append(result, *service)
	}

	// 按服务名排序
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].ServiceName > result[j].ServiceName {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// analyzeProblemSpans 分析Trace，找出有问题的Span
func (t *QueryTrace) analyzeProblemSpans(trace *Trace) []*Span {
	problemSpans := make([]*Span, 0)
	const slowThresholdMs = 1000 // 慢查询阈值：1000ms

	for _, span := range trace.Spans {
		// 检查是否有错误标记
		if span.Error {
			problemSpans = append(problemSpans, span)
			continue
		}

		// 检查是否耗时过长（超过阈值）
		if span.Duration > slowThresholdMs*1000 { // 转换为微秒
			problemSpans = append(problemSpans, span)
			continue
		}

		// 检查HTTP状态码
		if httpStatus, ok := span.Tags["http.status_code"]; ok {
			if code, err := strconv.Atoi(fmt.Sprintf("%v", httpStatus)); err == nil {
				if code >= 500 {
					problemSpans = append(problemSpans, span)
					continue
				}
			}
		}
		if reply, ok := span.Tags["reply"]; ok {
			if !strings.HasPrefix(parseCodeFromJSON(fmt.Sprintf("%v", reply)), "0") {
				problemSpans = append(problemSpans, span)
			}

		}
	}
	return problemSpans
}

// RegisterTraceTools 注册所有Trace工具
func RegisterTraceTools(conn *TraceConnection, toolManager *ToolManager) {
	toolManager.RegisterTool(NewQueryTrace(conn))
}

// parseCodeFromJSON 使用正则表达式从JSON格式字符串中解析code字段的值
func parseCodeFromJSON(jsonStr string) string {
	// 正则表达式匹配 "code":数字 ，数字部分用捕获组提取
	re := regexp.MustCompile(`"code":(\d+)`)
	matches := re.FindStringSubmatch(jsonStr)
	if matches == nil || len(matches) < 2 {
		return ""
	}
	return matches[1]
}
