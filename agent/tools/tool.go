package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"jas-agent/agent/core"
	"jas-agent/pkg/algorithm"
	"strings"
	"time"
)

var tm = NewToolManager()

type ToolManager struct {
	tools             map[string]core.Tool
	toolsMiddleware   map[string][]core.DataHandlerFilter
	mcpToolManagers   map[string]*MCPToolManager
	mcpToolMiddleware map[string][]core.DataHandlerFilter
}

func NewToolManager() *ToolManager {
	return &ToolManager{
		tools:             map[string]core.Tool{},
		mcpToolManagers:   map[string]*MCPToolManager{},
		toolsMiddleware:   map[string][]core.DataHandlerFilter{},
		mcpToolMiddleware: map[string][]core.DataHandlerFilter{},
	}
}
func (tm *ToolManager) RegisterTool(tool core.Tool, dataHandlers ...core.DataHandlerFilter) {
	tm.tools[tool.Name()] = tool
	tm.toolsMiddleware[tool.Name()] = dataHandlers

}

func (tm *ToolManager) RegisterMCPToolManager(name string, mcpToolManager *MCPToolManager, dataHandlers ...core.DataHandlerFilter) {
	tm.mcpToolManagers[name] = mcpToolManager
	tm.mcpToolMiddleware[name] = dataHandlers
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
	startTime := time.Now()

	// 开始工具调用追踪
	tracer := core.NewAgentTracer()
	ctx, span := tracer.StartToolCall(ctx, tool.Name, tool.Input)
	defer span.End()

	// 获取事件总线并发布工具调用事件
	eventBus := core.GetGlobalEventBus()
	if eventBus != nil {
		eventBus.Publish(ctx, core.EventToolCalled, map[string]interface{}{
			"tool_name":  tool.Name,
			"tool_input": truncateString(tool.Input, 500),
		})
	}

	var result string
	var err error

	if fun, ok := tm.tools[tool.Name]; ok {
		dataHandlers := tm.toolsMiddleware[tool.Name]
		if len(dataHandlers) > 0 {
			result, err = core.DataHandlerChain(dataHandlers...)(fun.Handler)(ctx, tool.Input)
		} else {
			result, err = fun.Handler(ctx, tool.Input)
		}
	} else if strings.Index(tool.Name, MCP_SEP) == 0 {
		err = fmt.Errorf("not found function [%s]", tool.Name)
	} else {
		args := strings.Split(tool.Name, MCP_SEP)
		if len(args) != 2 {
			err = fmt.Errorf("not found function [%s]", tool.Name)
		} else {
			mcpToolManager, ok := tm.mcpToolManagers[args[0]]
			if !ok {
				err = fmt.Errorf("not found function [%s]", tool.Name)
			} else {
				dataHandlers := tm.mcpToolMiddleware[tool.Name]
				result, err = mcpToolManager.ExecTool(ctx, tool, dataHandlers...)
			}
		}
	}

	duration := time.Since(startTime)
	success := err == nil

	// 记录指标
	if m := core.GetMetrics(); m != nil {
		m.RecordToolCall(ctx, tool.Name, duration, success)
	}

	// 记录追踪
	if err != nil {
		tracer.RecordError(span, err)
	} else {
		tracer.RecordSuccess(span)
	}

	// 发布工具调用完成事件
	if eventBus != nil {
		eventBus.Publish(ctx, core.EventToolCompleted, map[string]interface{}{
			"tool_name":   tool.Name,
			"duration_ms": duration.Milliseconds(),
			"success":     success,
			"error":       err,
			"result_size": len(result),
		})
	}

	return result, err
}

// truncateString 截断字符串（工具包内部使用）
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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

// WithLogClustering 日志聚类，使用drain算法
func WithLogClustering() core.DataHandlerFilter {
	return func(next core.DataHandler) core.DataHandler {
		return func(ctx context.Context, data string) (string, error) {
			out, err := next(ctx, data)
			if err == nil {
				return logClustering(out)
			}
			return out, err
		}
	}
}

func logClustering(data string) (string, error) {
	var searchResp struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID     string                 `json:"_id"`
				Source map[string]interface{} `json:"_source"`
				Score  float64                `json:"_score"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(data), &searchResp); err != nil {
		return "json.Unmarshal failure ", err
	}
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d documents (showing %d):\n\n",
		searchResp.Hits.Total.Value, len(searchResp.Hits.Hits)))

	if len(searchResp.Hits.Hits) == 0 {
		return "No documents found matching the query", nil
	}

	// 提取文档用于聚类
	documents := make([]map[string]interface{}, 0, len(searchResp.Hits.Hits))
	for _, hit := range searchResp.Hits.Hits {
		documents = append(documents, hit.Source)
	}

	// 使用 Drain3 算法进行聚类
	clusters := algorithm.ClusterDocuments(documents)

	result.WriteString(fmt.Sprintf("Found %d documents (showing %d):\n\n",
		searchResp.Hits.Total.Value, len(searchResp.Hits.Hits)))

	// 如果有聚类结果，先展示聚类摘要
	if len(clusters) > 0 {
		result.WriteString("=== 聚类分析结果 ===\n")
		result.WriteString(fmt.Sprintf("共发现 %d 个聚类模式：\n\n", len(clusters)))

		for i, cluster := range clusters {
			if i >= 10 { // 只显示前5个聚类
				result.WriteString(fmt.Sprintf("... 还有 %d 个聚类未显示\n\n", len(clusters)-5))
				break
			}
			result.WriteString(fmt.Sprintf("聚类 %d (出现 %d 次):\n", cluster.ClusterID, cluster.Count))
			result.WriteString(fmt.Sprintf("  模板: %s\n", cluster.Template))
			if len(cluster.Logs) > 0 {
				// 显示一个示例日志
				exampleLog := cluster.Logs[0]
				if len(exampleLog) > 100 {
					exampleLog = exampleLog[:100] + "..."
				}
				result.WriteString(fmt.Sprintf("  示例: %s\n", exampleLog))
			}
			result.WriteString("\n")
		}

	}
	return result.String(), nil
}
