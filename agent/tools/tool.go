package tools

import (
	"context"
	"fmt"
	"jas-agent/agent/core"
	"strings"
)

var tm = NewToolManager()

type ToolManager struct {
	tools           map[string]core.Tool
	mcpToolManagers map[string]*MCPToolManager
}

func NewToolManager() *ToolManager {
	return &ToolManager{
		tools:           map[string]core.Tool{},
		mcpToolManagers: map[string]*MCPToolManager{},
	}
}
func (tm *ToolManager) RegisterTool(tool core.Tool) {
	tm.tools[tool.Name()] = tool
}

func (tm *ToolManager) RegisterMCPToolManager(name string, mcpToolManager *MCPToolManager) {
	tm.mcpToolManagers[name] = mcpToolManager
}
func (tm *ToolManager) AvailableTools(filters ...core.FilterFunc) []core.Tool {
	var tools []core.Tool
	for _, v := range tm.tools {
		if !filter(v, filters...) {
			continue
		}
		tools = append(tools, v)
	}
	for _, v := range tm.mcpToolManagers {
		tools = append(tools, v.GetTools(filters...)...)
	}
	return tools
}

func (tm *ToolManager) ExecTool(ctx context.Context, tool *ToolCall) (string, error) {
	if fun, ok := tm.tools[tool.Name]; ok {
		return fun.Handler(ctx, tool.Input)
	}
	if strings.Index(tool.Name, MCP_SEP) == 0 {
		return "", fmt.Errorf("not found function [%s]", tool.Name)
	}
	args := strings.Split(tool.Name, MCP_SEP)
	if len(args) != 2 {
		return "", fmt.Errorf("not found function [%s]", tool.Name)
	}
	mcpToolManager, ok := tm.mcpToolManagers[args[0]]
	if !ok {
		return "", fmt.Errorf("not found function [%s]", tool.Name)
	}
	return mcpToolManager.ExecTool(ctx, tool)
}

func GetToolManager() *ToolManager {
	return tm
}

type ToolCall struct {
	Name  string
	Input string
}

func filter(tool core.Tool, filters ...core.FilterFunc) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if f(tool) {
			return false
		}
	}
	return true
}

func (tm *ToolManager) Inherit(baseToolManager *ToolManager) {
	for k, v := range baseToolManager.tools {
		tm.tools[k] = v
	}
	for k, v := range baseToolManager.mcpToolManagers {
		tm.mcpToolManagers[k] = v
	}
}
