package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"jas-agent/agent/core"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// McpTool MCP 工具接口
type McpTool interface {
	GetName() string
	GetDescription() string
	GetInputSchema() any
}

// McpToolImpl MCP 工具实现
type McpToolImpl struct {
	Name        string
	Description string
	InputSchema any
}

func (t *McpToolImpl) GetName() string {
	return t.Name
}

func (t *McpToolImpl) GetDescription() string {
	return t.Description
}

func (t *McpToolImpl) GetInputSchema() any {
	return t.InputSchema
}

// Client MCP 客户端接口（抽象层）
type Client interface {
	CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error)
	Initialize(ctx context.Context) error
	ListTools(ctx context.Context) ([]McpTool, error)
	Close() error
}

var MCP_SEP = "@"

// MCPToolWrapper 将 MCP 工具包装为 core.Tool
type MCPToolWrapper struct {
	name        string
	description string
	input       any
	client      Client
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

	// 使用抽象的 Client 接口调用工具
	response, err := w.client.CallTool(ctx, name, args)
	if err != nil {
		return "", fmt.Errorf("failed to call MCP tool %s: %w", w.name, err)
	}

	return response, nil
}

// MCPToolManager MCP 工具管理器
type MCPToolManager struct {
	client    Client
	tools     []map[string]core.Tool
	name      string
	lock      sync.Mutex
	isRunning atomic.Bool
	index     atomic.Int32
}

// MCPClientType MCP 客户端类型
type MCPClientType string

const (
	MCPClientTypeMark3Labs MCPClientType = "mark3labs" // github.com/mark3labs/mcp-go
	MCPClientTypeMetoro    MCPClientType = "metoro"    // github.com/metoro-io/mcp-golang
)

func TransferToMcpClientType(clientType string) MCPClientType {
	switch clientType {
	case "mark3labs":
		return MCPClientTypeMark3Labs
	}
	return MCPClientTypeMetoro
}

// NewMCPToolManager 创建新的 MCP 工具管理器
// clientType 指定使用的 MCP 客户端库类型，默认为 mark3labs
func NewMCPToolManager(name string, endpoint string, tm *ToolManager, clientType ...MCPClientType) (*MCPToolManager, error) {
	var mcpClient Client
	var err error

	// 确定客户端类型（默认使用 mark3labs）
	clientTypeStr := MCPClientTypeMetoro
	if len(clientType) > 0 {
		clientTypeStr = clientType[0]
	}

	// 根据类型创建相应的客户端
	switch clientTypeStr {
	case MCPClientTypeMetoro:
		mcpClient, err = NewMetoroClient(endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to create Metoro MCP client: %w", err)
		}
	default: // MCPClientTypeMark3Labs
		mcpClient, err = NewMark3LabsClient(endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to create Mark3Labs MCP client: %w", err)
		}
	}

	// 初始化客户端
	ctx := context.Background()
	if err := mcpClient.Initialize(ctx); err != nil {
		mcpClient.Close()
		return nil, fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	toolManager := &MCPToolManager{
		client: mcpClient,
		tools:  []map[string]core.Tool{map[string]core.Tool{}, map[string]core.Tool{}},
		name:   name,
	}
	if tm != nil {
		tm.RegisterMCPToolManager(name, toolManager)
	} else {
		GetToolManager().RegisterMCPToolManager(name, toolManager)
	}
	return toolManager, nil
}

// DiscoverAndRegisterTools 发现并注册 MCP 工具
func (mgr *MCPToolManager) DiscoverAndRegisterTools() error {
	// 使用抽象的 Client 接口获取工具列表
	tools, err := mgr.client.ListTools(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list MCP tools: %w", err)
	}

	index := mgr.index.Load() % 2
	mgr.tools[(index)] = make(map[string]core.Tool)
	for _, tool := range tools {
		wrapper := &MCPToolWrapper{
			name:        mgr.addToolPrefix(tool.GetName()),
			description: tool.GetDescription(),
			input:       tool.GetInputSchema(),
			client:      mgr.client,
			prefix:      mgr.addToolPrefix(""),
		}
		mgr.tools[(index)][mgr.addToolPrefix(tool.GetName())] = wrapper
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
func (mgr *MCPToolManager) Start(isFresh bool) {
	mgr.isRunning.Swap(true)
	mgr.DiscoverAndRegisterTools()
	if isFresh {
		go mgr.refresh()
	}

}
func (mgr *MCPToolManager) ExecTool(ctx context.Context, tool *ToolCall, dataHandlers ...core.DataHandlerFilter) (string, error) {

	if fun, ok := mgr.current()[tool.Name]; ok {
		return core.DataHandlerChain(dataHandlers...)(fun.Handler)(ctx, tool.Input)
	}
	return "", fmt.Errorf("not found function [%s]", tool.Name)
}
