package tools

import (
	"context"
	"fmt"
	"jas-agent/core"
)

var tm = NewToolManager()

type ToolManager struct {
	tools map[string]core.Tool
}

func NewToolManager() *ToolManager {
	return &ToolManager{
		tools: map[string]core.Tool{},
	}
}
func (tm *ToolManager) RegisterTool(tool core.Tool) {
	tm.tools[tool.Name()] = tool
}
func (tm *ToolManager) AvailableTools(filters ...core.FilterFunc) []core.Tool {
	var tools []core.Tool
	for _, v := range tm.tools {
		for _, filter := range filters {
			if filter(v) {
				break
			}
		}
		tools = append(tools, v)
	}
	return tools
}

func (tm *ToolManager) ExecTool(ctx context.Context, tool *ToolCall) (string, error) {
	if fun, ok := tm.tools[tool.Name]; ok {
		return fun.Handler(ctx, tool.Input)
	} else {
		return "", fmt.Errorf("not found function [%s]", tool.Name)
	}
}

func GetToolManager() *ToolManager {
	return tm
}

type ToolCall struct {
	Name  string
	Input string
}
