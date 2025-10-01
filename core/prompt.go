package core

import (
	"fmt"
	"strings"
)

type ToolData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ReactSystemPrompt struct {
	Date  string     `json:"date"`
	Tools []ToolData `json:"tools"`
}

// GetReactSystemPrompt 生成React系统提示词
func GetReactSystemPrompt(prompt ReactSystemPrompt) string {
	var toolsDesc strings.Builder
	for _, tool := range prompt.Tools {
		toolsDesc.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
	}

	return fmt.Sprintf(`你是一个基于ReAct框架的AI代理。你必须严格按照思考-行动-观察的循环解决问题。

当前时间: %s

可用工具:
%s

重要约束条件:
1. 每次只能执行一个步骤，不要一次性生成所有步骤
2. 每次响应必须只包含一个思考和一个行动
3. 思考格式: Thought: [你的思考过程]
4. 行动格式: Action: toolName[input] 或 Action: Finish[final answer]
5. 等待观察结果后再进行下一步思考
6. 不要预测工具执行结果，等待实际观察

请开始第一步思考。`, prompt.Date, toolsDesc.String())
}

// GetSummarySystemPrompt 生成总结系统提示词
func GetSummarySystemPrompt() string {
	return `你是一个专业的总结助手。你的任务是对执行过程进行总结，提供清晰、准确的最终答案。

总结要求:
1. 分析整个执行过程
2. 提取关键信息和结果
3. 提供简洁明了的最终答案
4. 确保答案准确无误
5. 严禁使用未列出的工具名称（如 search、browse 等）。只能从“可用工具”列表中选择。

请基于执行过程提供总结。`
}
