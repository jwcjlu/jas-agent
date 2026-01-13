package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// Mark3LabsClient mark3labs/mcp-go 客户端适配器
type Mark3LabsClient struct {
	client *client.Client
}

// NewMark3LabsClient 创建 mark3labs/mcp-go 客户端适配器
func NewMark3LabsClient(endpoint string) (*Mark3LabsClient, error) {
	// 创建 HTTP 传输层，设置 Accept 头
	httpTransport, err := transport.NewStreamableHTTP(
		endpoint,
		transport.WithHTTPHeaders(map[string]string{
			"Accept": "application/json, text/event-stream",
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP transport: %w", err)
	}

	// 创建 MCP 客户端
	mcpClient := client.NewClient(httpTransport)

	// 启动客户端（启动传输层）
	ctx := context.Background()
	if err := mcpClient.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start MCP client: %w", err)
	}

	adapter := &Mark3LabsClient{
		client: mcpClient,
	}

	return adapter, nil
}

// Initialize 初始化 MCP 会话
func (c *Mark3LabsClient) Initialize(ctx context.Context) error {
	// 初始化 MCP 会话（必须调用，否则客户端未初始化）
	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo: mcp.Implementation{
				Name:    "jas-agent",
				Version: "1.0.0",
			},
		},
	}
	_, err := c.client.Initialize(ctx, initRequest)
	if err != nil {
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}
	return nil
}

// ListTools 获取工具列表
func (c *Mark3LabsClient) ListTools(ctx context.Context) ([]McpTool, error) {
	listRequest := mcp.ListToolsRequest{}
	result, err := c.client.ListTools(ctx, listRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP tools: %w", err)
	}

	tools := make([]McpTool, 0, len(result.Tools))
	for _, tool := range result.Tools {
		tools = append(tools, &McpToolImpl{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}

	return tools, nil
}

// CallTool 调用工具
func (c *Mark3LabsClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	// 转换参数为 mcp.CallToolRequest
	callRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}

	response, err := c.client.CallTool(ctx, callRequest)
	if err != nil {
		return "", fmt.Errorf("failed to call MCP tool %s: %w", name, err)
	}

	if response == nil || len(response.Content) == 0 {
		return "工具执行完成", nil
	}

	// 提取文本内容
	var result strings.Builder
	for _, content := range response.Content {
		// 检查是否是 TextContent
		if textContent, ok := content.(mcp.TextContent); ok {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(textContent.Text)
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
func (c *Mark3LabsClient) Close() error {
	return c.client.Close()
}
