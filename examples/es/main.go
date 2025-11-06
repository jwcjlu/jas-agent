package main

import (
	"fmt"
	"jas-agent/agent"
	"jas-agent/llm"
	"jas-agent/tools"
	"os"
)

func main() {
	// 从环境变量获取配置
	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	esHost := os.Getenv("ES_HOST") // 例如: http://localhost:9200
	esUser := os.Getenv("ES_USER")
	esPass := os.Getenv("ES_PASS")

	if apiKey == "" || baseURL == "" {
		fmt.Println("❌ 请设置 OPENAI_API_KEY 和 OPENAI_BASE_URL 环境变量")
		os.Exit(1)
	}

	if esHost == "" {
		esHost = "http://localhost:9200"
		fmt.Printf("ℹ️ 使用默认ES地址: %s\n", esHost)
	}

	fmt.Println("🚀 启动 Elasticsearch Agent 示例...")
	fmt.Println("=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=")

	// 创建ES连接
	esConn := tools.NewESConnection(esHost, esUser, esPass)

	// 注册ES工具
	tools.RegisterESTools(esConn)

	// 创建LLM
	chat := llm.NewChat(&llm.Config{
		ApiKey:  apiKey,
		BaseURL: baseURL,
	})

	// 创建Agent上下文
	context := agent.NewContext(
		agent.WithChat(chat),
	)

	// 创建ES Agent执行器
	clusterInfo := fmt.Sprintf("Elasticsearch cluster at %s", esHost)
	executor := agent.NewESAgentExecutor(context, clusterInfo)

	// 示例查询
	queries := []string{
		"列出所有索引及其文档数量",
		"查看 logs 索引的结构",
		"搜索最近的10条错误日志",
		"统计每小时的日志数量",
	}

	for i, query := range queries {
		fmt.Printf("\n\n🔍 查询 %d: %s\n", i+1, query)
		fmt.Println("-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-" + "-")

		result := executor.Run(query)
		fmt.Printf("\n✅ 结果:\n%s\n", result)
	}

	fmt.Println("\n" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=")
	fmt.Println("🎉 Elasticsearch Agent 示例完成!")
}
