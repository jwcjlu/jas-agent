package main

import (
	"flag"
	"fmt"
	agent "jas-agent/agent/agent"
	"jas-agent/agent/llm"
	"strings"

	_ "jas-agent/agent/examples/react/tools"

	"github.com/sashabaranov/go-openai"
)

func main() {
	fmt.Println("🔗 Starting ChainAgent example...")

	var apiKey string
	var baseUrl string
	flag.StringVar(&apiKey, "apiKey", "apiKey", "apiKey")
	flag.StringVar(&baseUrl, "baseUrl", "baseUrl", "baseUrl")
	flag.Parse()

	// 创建LLM客户端
	chat := llm.NewChat(&llm.Config{
		ApiKey:  apiKey,
		BaseURL: baseUrl,
	})

	// 创建上下文
	context := agent.NewContext(
		agent.WithModel(openai.GPT3Dot5Turbo),
		agent.WithChat(chat),
	)

	// 示例1: 简单线性链
	fmt.Println("\n=== 示例1: 简单线性链 ===")
	simpleChainExample(context)

	// 示例2: 带转换的链
	fmt.Println("\n=== 示例2: 带转换的链 ===")
	transformChainExample(context)

	// 示例3: 条件分支链
	fmt.Println("\n=== 示例3: 条件分支链 ===")
	conditionalChainExample(context)
}

// 示例1: 简单线性链 - 查询狗狗体重然后计算总和
func simpleChainExample(ctx *agent.Context) {
	// 构建链式Agent
	builder := agent.NewChainBuilder(ctx)

	// 添加节点：查询狗狗信息 -> 计算总和
	builder.
		AddNode("query_weights", agent.ReactAgentType, 5).
		AddNode("calculate_total", agent.ReactAgentType, 3).
		Link("query_weights", "calculate_total")

	chainAgent := builder.Build()
	executor := agent.NewChainAgentExecutor(ctx, chainAgent)

	result := executor.Run("我有一只边境牧羊犬和一只苏格兰梗，它们的总体重是多少？")
	fmt.Printf("📊 最终结果: %s\n", result)
}

// 示例2: 带转换的链 - 提取关键信息
func transformChainExample(ctx *agent.Context) {
	// 构建链式Agent
	builder := agent.NewChainBuilder(ctx)

	// 添加节点并设置转换函数
	builder.
		AddNode("query_info", agent.ReactAgentType, 5).
		AddNode("summarize", agent.ReactAgentType, 3).
		Link("query_info", "summarize")

	// 为第一个节点设置转换函数：提取数值
	builder.SetTransform("query_info", func(output string) string {
		// 简单的数值提取逻辑
		if strings.Contains(output, "磅") || strings.Contains(output, "lbs") {
			return "已获取体重信息: " + output
		}
		return output
	})

	chainAgent := builder.Build()
	executor := agent.NewChainAgentExecutor(ctx, chainAgent)

	result := executor.Run("玩具贵宾犬的平均体重是多少？")
	fmt.Printf("📊 最终结果: %s\n", result)
}

// 示例3: 条件分支链 - 根据结果选择不同的处理路径
func conditionalChainExample(ctx *agent.Context) {
	// 构建链式Agent
	builder := agent.NewChainBuilder(ctx)

	// 添加节点：检查 -> 分支A/分支B
	builder.
		AddNode("check_value", agent.ReactAgentType, 3).
		AddNode("large_process", agent.ReactAgentType, 3).
		AddNode("small_process", agent.ReactAgentType, 3).
		Link("check_value", "large_process").
		Link("check_value", "small_process")

	// 设置条件：如果结果包含"大"或数值>50，走large_process
	builder.SetCondition("large_process", func(input string) bool {
		return strings.Contains(input, "大") ||
			strings.Contains(input, "50") ||
			strings.Contains(input, "60") ||
			strings.Contains(input, "70")
	})

	// 设置条件：如果结果包含"小"或数值<20，走small_process
	builder.SetCondition("small_process", func(input string) bool {
		return strings.Contains(input, "小") ||
			strings.Contains(input, "10") ||
			strings.Contains(input, "15") ||
			strings.Contains(input, "7")
	})

	chainAgent := builder.Build()
	executor := agent.NewChainAgentExecutor(ctx, chainAgent)

	result := executor.Run("玩具贵宾犬的平均体重是多少？")
	fmt.Printf("📊 最终结果: %s\n", result)
}

// 示例4: 多步骤数据处理链
func dataProcessingChainExample(ctx *agent.Context) {
	// 构建链式Agent
	builder := agent.NewChainBuilder(ctx)

	// 数据处理流程：收集 -> 清洗 -> 分析 -> 报告
	builder.
		AddNode("collect", agent.ReactAgentType, 5).
		AddNode("clean", agent.ReactAgentType, 3).
		AddNode("analyze", agent.ReactAgentType, 5).
		AddNode("report", agent.ReactAgentType, 3).
		Link("collect", "clean").
		Link("clean", "analyze").
		Link("analyze", "report")

	// 设置每个节点的转换函数
	builder.SetTransform("collect", func(output string) string {
		return fmt.Sprintf("[已收集] %s", output)
	})

	builder.SetTransform("clean", func(output string) string {
		return fmt.Sprintf("[已清洗] %s", output)
	})

	builder.SetTransform("analyze", func(output string) string {
		return fmt.Sprintf("[已分析] %s", output)
	})

	chainAgent := builder.Build()
	executor := agent.NewChainAgentExecutor(ctx, chainAgent)

	result := executor.Run("收集边境牧羊犬、苏格兰梗、玩具贵宾犬的体重数据并进行分析")
	fmt.Printf("📊 最终结果: %s\n", result)
}
