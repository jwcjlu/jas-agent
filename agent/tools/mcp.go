package tools

import (
	"context"
	"encoding/json"
	"fmt"
	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/http"
	"jas-agent/agent/core"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var MCP_SEP = "@"

// MCPToolWrapper 将 MCP 工具包装为 core.Tool
type MCPToolWrapper struct {
	name        string
	description string
	input       any
	client      *mcp.Client
	prefix      string
}

func (w *MCPToolWrapper) Name() string {
	return w.name
}

func (w *MCPToolWrapper) Description() string {
	return w.description
}

func (w *MCPToolWrapper) Input() any {
	return w.input
}
func (w *MCPToolWrapper) Type() core.ToolType {
	return core.Mcp
}
func (w *MCPToolWrapper) Handler(ctx context.Context, input string) (string, error) {
	// 解析参数为通用 Map
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		log.Printf("Error parsing arguments: %v\n", err)
	}
	// 调用 MCP 工具
	name := strings.TrimPrefix(w.name, w.prefix)
	response, err := w.client.CallTool(ctx, name, args)
	if err != nil {
		return "", fmt.Errorf("failed to call MCP tool %s: %w", w.name, err)
	}

	if response == nil || len(response.Content) == 0 {
		return "工具执行完成", nil
	}
	// 提取文本内容
	if response.Content[0].TextContent != nil {
		return response.Content[0].TextContent.Text, nil
	}

	return "工具执行完成", nil
}

// MCPToolManager MCP 工具管理器
type MCPToolManager struct {
	client    *mcp.Client
	tools     []map[string]core.Tool
	name      string
	lock      sync.Mutex
	isRunning atomic.Bool
	index     atomic.Int32
}

// NewMCPToolManager 创建新的 MCP 工具管理器
func NewMCPToolManager(name string, endpoint string) (*MCPToolManager, error) {
	// 创建 MCP 客户端
	transport := http.NewHTTPClientTransport(endpoint)
	client := mcp.NewClient(transport)

	// 初始化客户端
	if _, err := client.Initialize(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	toolManager := &MCPToolManager{
		client: client,
		tools:  []map[string]core.Tool{map[string]core.Tool{}, map[string]core.Tool{}},
		name:   name,
	}
	GetToolManager().RegisterMCPToolManager(name, toolManager)

	return toolManager, nil
}

// DiscoverAndRegisterTools 发现并注册 MCP 工具
func (mgr *MCPToolManager) DiscoverAndRegisterTools() error {
	// 获取工具列表
	tools, err := mgr.client.ListTools(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to list MCP tools: %w", err)
	}
	index := mgr.index.Load() % 2
	mgr.tools[(index)] = make(map[string]core.Tool)
	for _, tool := range tools.Tools {
		description := ""
		if tool.Description != nil {
			description = *tool.Description
		}
		wrapper := &MCPToolWrapper{
			name:        mgr.addToolPrefix(tool.Name),
			description: description,
			input:       tool.InputSchema,
			client:      mgr.client,
			prefix:      mgr.addToolPrefix(""),
		}
		mgr.tools[(index)][mgr.addToolPrefix(tool.Name)] = wrapper
	}
	mgr.index.Swap(index + 1)
	return nil
}
func (mgr *MCPToolManager) Switch() {
	for !mgr.index.CompareAndSwap(mgr.index.Load(), mgr.index.Load()+1) {
	}
}

func (mgr *MCPToolManager) current() map[string]core.Tool {
	return mgr.tools[(mgr.index.Load()+1)%2]
}
func (mgr *MCPToolManager) addToolPrefix(name string) string {
	return fmt.Sprintf("%s%s%s", mgr.name, MCP_SEP, name)
}
func (mgr *MCPToolManager) GetTools(filters ...core.FilterFunc) []core.Tool {
	var tools []core.Tool
	for _, v := range mgr.current() {
		if !filter(v, filters...) {
			continue
		}
		tools = append(tools, v)
	}
	return tools
}

func (mgr *MCPToolManager) refresh() {
	for mgr.isRunning.Load() {
		if err := mgr.DiscoverAndRegisterTools(); err != nil {

		}
		time.Sleep(5 * time.Second)
	}
}
func (mgr *MCPToolManager) Start() {
	mgr.isRunning.Swap(true)
	mgr.DiscoverAndRegisterTools()
	go mgr.refresh()
}
func (mgr *MCPToolManager) ExecTool(ctx context.Context, tool *ToolCall) (string, error) {

	if fun, ok := mgr.current()[tool.Name]; ok {
		return fun.Handler(ctx, tool.Input)
	}
	return "", fmt.Errorf("not found function [%s]", tool.Name)
}
