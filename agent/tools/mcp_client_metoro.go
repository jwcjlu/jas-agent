package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	metoroMCP "github.com/metoro-io/mcp-golang"
	metoroHTTP "github.com/metoro-io/mcp-golang/transport/http"
)

// MetoroClient metoro-io/mcp-golang 客户端适配器
type MetoroClient struct {
	client *metoroMCP.Client
}

// NewMetoroClient 创建 metoro-io/mcp-golang 客户端适配器
func NewMetoroClient(endpoint string) (*MetoroClient, error) {
	// 创建 HTTP 传输层
	transport := metoroHTTP.NewHTTPClientTransport(endpoint)
	transport.WithHeader("Accept", "application/json, text/event-stream")

	// 创建 MCP 客户端
	mcpClient := metoroMCP.NewClient(transport)

	adapter := &MetoroClient{
		client: mcpClient,
	}

	return adapter, nil
}

// Initialize 初始化 MCP 会话
func (c *MetoroClient) Initialize(ctx context.Context) error {
	// metoro-io/mcp-golang 的 Initialize 会初始化客户端
	_, err := c.client.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}
	return nil
}

// ListTools 获取工具列表
func (c *MetoroClient) ListTools(ctx context.Context) ([]McpTool, error) {
	tools, err := c.client.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP tools: %w", err)
	}

	result := make([]McpTool, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		description := ""
		if tool.Description != nil {
			description = *tool.Description
		}

		result = append(result, &McpToolImpl{
			Name:        tool.Name,
			Description: description,
			InputSchema: tool.InputSchema,
		})
	}

	return result, nil
}

// CallTool 调用工具
func (c *MetoroClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	response, err := c.client.CallTool(ctx, name, args)
	if err != nil {
		return "", fmt.Errorf("failed to call MCP tool %s: %w", name, err)
	}

	if response == nil || len(response.Content) == 0 {
		return "工具执行完成", nil
	}

	// 提取文本内容
	var result strings.Builder
	for _, content := range response.Content {
		if content == nil {
			continue
		}

		// metoro-io/mcp-golang 的 Content 结构体
		// Content 有 Type 字段，根据类型访问相应的字段
		if content.Type == metoroMCP.ContentTypeText && content.TextContent != nil {
			// Text 类型的内容，访问 TextContent.Text 字段
			if content.TextContent.Text != "" {
				if result.Len() > 0 {
					result.WriteString("\n")
				}
				result.WriteString(content.TextContent.Text)
			}
		} else {
			// 对于其他类型的内容，尝试 JSON 序列化
			if jsonBytes, err := json.Marshal(content); err == nil {
				if result.Len() > 0 {
					result.WriteString("\n")
				}
				result.WriteString(string(jsonBytes))
			}
		}
	}

	if result.Len() > 0 {
		return result.String(), nil
	}

	return "工具执行完成", nil
}

// Close 关闭客户端
func (c *MetoroClient) Close() error {
	// metoro-io/mcp-golang 可能没有 Close 方法，或者需要不同的关闭方式
	// 这里暂时返回 nil，如果需要可以添加关闭逻辑
	return nil
}
