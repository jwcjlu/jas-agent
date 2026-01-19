package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"jas-agent/agent/core"
)

// HTTPRequestInput 通用 HTTP 请求输入结构
type HTTPRequestInput struct {
	URL           string            `json:"url"`                       // 请求地址，必须是完整 URL
	Method        string            `json:"method"`                    // HTTP 方法，默认 GET
	Headers       map[string]string `json:"headers,omitempty"`         // 自定义请求头
	Query         map[string]string `json:"query,omitempty"`           // 查询参数，会追加到 URL 上
	Body          interface{}       `json:"body,omitempty"`            // 请求体，可以是字符串（文本）或对象（JSON）
	BodyType      string            `json:"body_type,omitempty"`       // 请求体类型："text" 或 "json"，如果不指定则自动判断
	TimeoutSecond int               `json:"timeout_seconds,omitempty"` // 超时时间（秒），默认 30s
}

// HTTPRequestTool 通用 HTTP 请求工具（可被 MCP 或 Agent 使用）
type HTTPRequestTool struct {
	client *http.Client
}

// NewHTTPRequestTool 创建一个新的 HTTPRequestTool
func NewHTTPRequestTool() *HTTPRequestTool {
	return &HTTPRequestTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (t *HTTPRequestTool) Name() string {
	return "http_request"
}

func (t *HTTPRequestTool) Description() string {
	return "通用的 HTTP 请求工具。支持自定义方法、查询参数、请求头和请求体（支持文本或JSON格式），适合通过 MCP 或 Agent 发起外部 HTTP 请求。输入为 JSON，包含 url、method、headers、query、body、body_type、timeout_seconds 等字段。body可以是字符串（文本格式）或对象（JSON格式）。如果指定body_type为\"text\"，body将作为纯文本发送；如果为\"json\"或未指定，body将序列化为JSON。"
}

// Input 返回 JSON Schema，用于 LLM 了解如何构造参数
func (t *HTTPRequestTool) Input() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "完整的请求 URL，例如：https://api.example.com/v1/resource",
			},
			"method": map[string]any{
				"type":        "string",
				"description": "HTTP 方法，支持 GET、POST、PUT、DELETE、PATCH 等，默认 GET",
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "可选的 HTTP 请求头，例如 {\"Authorization\": \"Bearer xxx\"}",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"query": map[string]any{
				"type":        "object",
				"description": "可选的查询参数，将自动追加到 URL 上",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"body": map[string]any{
				"description": "可选的请求体，可以是字符串（文本格式）或对象（JSON格式），只在非 GET/DELETE 请求时使用",
				"oneOf": []map[string]any{
					{
						"type": "string",
					},
					{
						"type": "object",
					},
				},
			},
			"body_type": map[string]any{
				"type":        "string",
				"description": "请求体类型：\"text\"（文本）或 \"json\"（JSON），如果不指定则根据body类型自动判断",
				"enum":        []string{"text", "json"},
			},
			"timeout_seconds": map[string]any{
				"type":        "integer",
				"description": "请求超时时间（秒），默认 30 秒",
			},
		},
		"required": []string{"url"},
	}
}

func (t *HTTPRequestTool) Type() core.ToolType {
	return core.Normal
}

// Handler 执行 HTTP 请求
func (t *HTTPRequestTool) Handler(ctx context.Context, input string) (string, error) {
	var reqInput HTTPRequestInput
	if err := json.Unmarshal([]byte(input), &reqInput); err != nil {
		return "", fmt.Errorf("解析输入 JSON 失败: %w\n\n输入内容:\n%s", err, input)
	}

	if strings.TrimSpace(reqInput.URL) == "" {
		return "", fmt.Errorf("url 字段不能为空")
	}

	method := strings.ToUpper(strings.TrimSpace(reqInput.Method))
	if method == "" {
		method = http.MethodGet
	}

	// 解析 URL
	parsedURL, err := url.Parse(reqInput.URL)
	if err != nil {
		return "", fmt.Errorf("无效的 URL: %w", err)
	}

	// 合并查询参数
	if len(reqInput.Query) > 0 {
		q := parsedURL.Query()
		for k, v := range reqInput.Query {
			q.Set(k, v)
		}
		parsedURL.RawQuery = q.Encode()
	}

	// 构造请求体
	var bodyReader io.Reader
	var contentType string
	if method != http.MethodGet && method != http.MethodDelete && reqInput.Body != nil {
		var bodyBytes []byte
		var err error

		// 判断body类型
		bodyType := strings.ToLower(strings.TrimSpace(reqInput.BodyType))

		// 如果body是字符串，直接使用字符串
		if bodyStr, ok := reqInput.Body.(string); ok {
			if bodyType == "json" {
				// 即使body是字符串，如果指定了json类型，尝试解析为JSON
				var jsonObj interface{}
				if err := json.Unmarshal([]byte(bodyStr), &jsonObj); err != nil {
					return "", fmt.Errorf("body_type指定为json，但body字符串不是有效的JSON: %w", err)
				}
				bodyBytes, err = json.Marshal(jsonObj)
				if err != nil {
					return "", fmt.Errorf("序列化JSON请求体失败: %w", err)
				}
				contentType = "application/json"
			} else {
				// 文本格式
				bodyBytes = []byte(bodyStr)
				contentType = "text/plain; charset=utf-8"
			}
		} else {
			// body是对象，序列化为JSON
			if bodyType == "text" {
				// 如果指定了text类型但body是对象，转换为JSON字符串作为文本发送
				jsonBytes, err := json.Marshal(reqInput.Body)
				if err != nil {
					return "", fmt.Errorf("序列化请求体失败: %w", err)
				}
				bodyBytes = jsonBytes
				contentType = "text/plain; charset=utf-8"
			} else {
				// JSON格式
				bodyBytes, err = json.Marshal(reqInput.Body)
				if err != nil {
					return "", fmt.Errorf("序列化JSON请求体失败: %w", err)
				}
				contentType = "application/json"
			}
		}

		bodyReader = bytes.NewReader(bodyBytes)
	}

	// 使用带超时的 context
	timeout := 30 * time.Second
	if reqInput.TimeoutSecond > 0 {
		timeout = time.Duration(reqInput.TimeoutSecond) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, parsedURL.String(), bodyReader)
	if err != nil {
		return "", fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	if len(reqInput.Headers) > 0 {
		for k, v := range reqInput.Headers {
			req.Header.Set(k, v)
		}
	}
	// 如果未指定Content-Type且存在请求体，使用自动判断的contentType
	if req.Body != nil && req.Header.Get("Content-Type") == "" && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	start := time.Now()
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("执行 HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 限制 1MB，避免过大响应
	if err != nil {
		return "", fmt.Errorf("读取响应体失败: %w", err)
	}

	duration := time.Since(start)

	// 构造统一的响应 JSON
	result := map[string]any{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"duration_ms": duration.Milliseconds(),
		"headers":     resp.Header,
		"body":        string(respBody),
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		// 如果序列化失败，退化为纯文本
		return fmt.Sprintf("status=%d, duration=%dms, body=%s", resp.StatusCode, duration.Milliseconds(), string(respBody)), nil
	}

	return string(out), nil
}

// RegisterHTTPTool 注册通用 HTTP 请求工具
func RegisterHTTPTool(tm *ToolManager) {
	tm.RegisterTool(NewHTTPRequestTool())
}
