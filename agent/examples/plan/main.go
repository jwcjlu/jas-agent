package main

import (
	"flag"
	"os"

	"github.com/sashabaranov/go-openai"

	"github.com/go-kratos/kratos/v2/log"
	agent "jas-agent/agent/agent"
	_ "jas-agent/agent/examples/react/tools"
	"jas-agent/agent/llm"
)

var logger = log.NewHelper(log.With(log.NewStdLogger(os.Stdout), "module", "examples/plan"))

func main() {
	logger.Info("📋 Starting PlanAgent example...")

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

	// 示例1: 基本计划执行
	logger.Info("\n=== 示例1: 基本计划执行 ===")
	basicPlanExample(context)

	// 示例2: 带依赖的复杂计划
	logger.Info("\n=== 示例2: 带依赖的复杂计划 ===")
	complexPlanExample(context)

	// 示例3: 启用重新规划
	logger.Info("\n=== 示例3: 启用重新规划 ===")
	replanExample(context)
}

// 示例1: 基本计划执行 - 简单的多步骤任务
func basicPlanExample(ctx *agent.Context) {
	// 创建Plan Agent执行器（不启用重新规划）
	executor := agent.NewPlanAgentExecutor(ctx, false)

	result := executor.Run("计算15 + 27的结果，然后乘以3")
	logger.Infof("📊 最终结果:\n%s", result)
}

// 示例2: 带依赖的复杂计划 - 多只狗的体重计算
func complexPlanExample(ctx *agent.Context) {
	// 创建Plan Agent执行器（不启用重新规划）
	executor := agent.NewPlanAgentExecutor(ctx, false)

	result := executor.Run("我有3只狗，分别是border collie、scottish terrier和toy poodle。请查询它们的平均体重，然后计算总重量")
	logger.Infof("📊 最终结果:\n%s", result)
}

// 示例3: 启用重新规划 - 遇到问题时自动调整计划
func replanExample(ctx *agent.Context) {
	// 创建Plan Agent执行器（启用重新规划）
	executor := agent.NewPlanAgentExecutor(ctx, true)

	result := executor.Run("查询拉布拉多和金毛的体重差异，并计算平均值")
	logger.Infof("📊 最终结果:\n%s", result)
}

// 示例4: 数学计算链
func mathPlanExample(ctx *agent.Context) {
	executor := agent.NewPlanAgentExecutor(ctx, false)

	result := executor.Run("计算(15 + 27) * 3 - 10，并说明计算过程")
	logger.Infof("📊 最终结果:\n%s", result)
}

// 示例5: 信息收集和分析
func analysisPlanExample(ctx *agent.Context) {
	executor := agent.NewPlanAgentExecutor(ctx, false)

	result := executor.Run("收集边境牧羊犬、德国牧羊犬、澳大利亚牧羊犬的体重信息，找出最重的品种")
	logger.Infof("📊 最终结果:\n%s", result)
}
