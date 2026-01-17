package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"jas-agent/agent/agent/aiops/framework"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// JaegerTraceDataSource Jaeger Trace 数据源适配器
type JaegerTraceDataSource struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewJaegerTraceDataSource 创建 Jaeger Trace 数据源
func NewJaegerTraceDataSource(baseURL string, timeout time.Duration) *JaegerTraceDataSource {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &JaegerTraceDataSource{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
		timeout:    timeout,
	}
}

// FetchTraces 获取 Trace 数据
func (j *JaegerTraceDataSource) FetchTraces(
	ctx context.Context,
	services []string,
	timeRange framework.TimeRange,
) ([]TraceData, error) {
	allTraces := make([]TraceData, 0)

	// 为每个服务查询 Trace
	for _, service := range services {
		traces, err := j.queryTraces(ctx, service, timeRange)
		if err != nil {
			// 记录错误但继续查询其他服务
			continue
		}
		allTraces = append(allTraces, traces...)
	}

	return allTraces, nil
}

// queryTraces 查询单个服务的 Trace
func (j *JaegerTraceDataSource) queryTraces(
	ctx context.Context,
	service string,
	timeRange framework.TimeRange,
) ([]TraceData, error) {
	// 构建 Jaeger API 查询参数
	params := url.Values{}
	params.Set("service", service)
	params.Set("start", strconv.FormatInt(timeRange.StartTime*1000000, 10)) // Jaeger 使用微秒
	params.Set("end", strconv.FormatInt(timeRange.EndTime*1000000, 10))
	params.Set("limit", "100") // 最多返回100个trace

	// 确保 baseURL 不以 / 结尾
	baseURL := j.baseURL
	if baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}

	// 构建请求URL
	queryURL := fmt.Sprintf("%s/api/traces?%s", baseURL, params.Encode())

	// 发送HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jaeger query failed (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result jaegerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 转换为 TraceData
	traces := make([]TraceData, 0)
	for _, trace := range result.Data {
		traceData := j.convertTrace(trace, service)
		traces = append(traces, traceData)
	}

	return traces, nil
}

// convertTrace 将 Jaeger trace 转换为 TraceData
func (j *JaegerTraceDataSource) convertTrace(trace jaegerTrace, defaultService string) TraceData {
	// 查找根 span
	var rootSpan *jaegerSpan
	for i := range trace.Spans {
		if trace.Spans[i].References == nil || len(trace.Spans[i].References) == 0 {
			rootSpan = &trace.Spans[i]
			break
		}
	}

	if rootSpan == nil && len(trace.Spans) > 0 {
		rootSpan = &trace.Spans[0]
	}

	// 提取服务信息
	services := make(map[string]bool)
	var startTime, endTime int64

	if rootSpan != nil {
		startTime = rootSpan.StartTime / 1000 // 转换为秒
		endTime = startTime + int64(rootSpan.Duration/1000)
	}

	// 提取所有涉及的服务
	for _, span := range trace.Spans {
		if process, ok := trace.Processes[span.ProcessID]; ok {
			services[process.ServiceName] = true
		}
	}

	serviceList := make([]string, 0, len(services))
	for svc := range services {
		serviceList = append(serviceList, svc)
	}

	// 提取错误信息
	hasError := false
	errorMessage := ""
	for _, span := range trace.Spans {
		for _, tag := range span.Tags {
			if tag.Key == "error" && tag.Value == true {
				hasError = true
			}
			if tag.Key == "error.message" {
				errorMessage = tag.Value.(string)
			}
		}
	}

	// 构建调用链
	callChain := j.buildCallChain(trace.Spans, trace.Processes)

	return TraceData{
		TraceID:      trace.TraceID,
		Services:     serviceList,
		StartTime:    startTime,
		EndTime:      endTime,
		Duration:     endTime - startTime,
		HasError:     hasError,
		ErrorMessage: errorMessage,
		CallChain:    callChain,
	}
}

// buildCallChain 构建调用链
func (j *JaegerTraceDataSource) buildCallChain(spans []jaegerSpan, processes map[string]jaegerProcess) []CallChainNode {
	nodes := make([]CallChainNode, 0)

	for _, span := range spans {
		process, ok := processes[span.ProcessID]
		if !ok {
			continue
		}

		// 提取操作名和方法
		operation := span.OperationName
		method := ""
		for _, tag := range span.Tags {
			if tag.Key == "http.method" {
				method = tag.Value.(string)
			}
		}

		nodes = append(nodes, CallChainNode{
			Service:   process.ServiceName,
			Operation: operation,
			Method:    method,
			StartTime: span.StartTime / 1000,
			Duration:  span.Duration / 1000,
			Status:    j.extractSpanStatus(span),
		})
	}

	return nodes
}

// extractSpanStatus 提取 span 状态
func (j *JaegerTraceDataSource) extractSpanStatus(span jaegerSpan) string {
	for _, tag := range span.Tags {
		if tag.Key == "error" && tag.Value == true {
			return "error"
		}
		if tag.Key == "http.status_code" {
			statusCode := int(tag.Value.(float64))
			if statusCode >= 400 {
				return "error"
			}
		}
	}
	return "success"
}

// TraceData Trace 数据
type TraceData struct {
	TraceID      string
	Services     []string
	StartTime    int64
	EndTime      int64
	Duration     int64
	HasError     bool
	ErrorMessage string
	CallChain    []CallChainNode
}

// CallChainNode 调用链节点
type CallChainNode struct {
	Service   string
	Operation string
	Method    string
	StartTime int64
	Duration  int64
	Status    string
}

// jaegerResponse Jaeger API 响应
type jaegerResponse struct {
	Data []jaegerTrace `json:"data"`
}

// jaegerTrace Jaeger Trace 结构
type jaegerTrace struct {
	TraceID   string                   `json:"traceID"`
	Spans     []jaegerSpan             `json:"spans"`
	Processes map[string]jaegerProcess `json:"processes"`
}

// jaegerSpan Jaeger Span 结构
type jaegerSpan struct {
	TraceID       string      `json:"traceID"`
	SpanID        string      `json:"spanID"`
	OperationName string      `json:"operationName"`
	StartTime     int64       `json:"startTime"`
	Duration      int64       `json:"duration"`
	Tags          []jaegerTag `json:"tags"`
	References    []jaegerRef `json:"references"`
	ProcessID     string      `json:"processID"`
}

// jaegerTag Jaeger Tag 结构
type jaegerTag struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// jaegerRef Jaeger Reference 结构
type jaegerRef struct {
	RefType string `json:"refType"`
	TraceID string `json:"traceID"`
	SpanID  string `json:"spanID"`
}

// jaegerProcess Jaeger Process 结构
type jaegerProcess struct {
	ServiceName string      `json:"serviceName"`
	Tags        []jaegerTag `json:"tags"`
}
