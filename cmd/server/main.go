package main

import (
	"flag"
	"fmt"
	"jas-agent/llm"
	"jas-agent/server"
	"jas-agent/tools"
	"log"

	_ "jas-agent/examples/react/tools" // 注册工具
)

func main() {
	// 命令行参数
	var (
		httpAddr string
		apiKey   string
		baseURL  string
		model    string
	)

	flag.StringVar(&httpAddr, "http", ":8080", "HTTP服务器地址")
	flag.StringVar(&apiKey, "apiKey", "", "OpenAI API Key")
	flag.StringVar(&baseURL, "baseUrl", "", "OpenAI Base URL")
	flag.StringVar(&model, "model", "gpt-3.5-turbo", "默认模型")
	flag.Parse()

	if apiKey == "" {
		log.Fatal("❌ 请提供 API Key: -apiKey YOUR_API_KEY")
	}

	if baseURL == "" {
		log.Fatal("❌ 请提供 Base URL: -baseUrl YOUR_BASE_URL")
	}

	fmt.Println("🚀 启动 JAS Agent 服务器...")

	// 创建LLM客户端
	chat := llm.NewChat(&llm.Config{
		ApiKey:  apiKey,
		BaseURL: baseURL,
	})

	// 创建gRPC服务
	grpcServer := server.NewAgentServer(chat)

	fmt.Println("✅ gRPC服务已创建")
	mcpManager, _ := tools.NewMCPToolManager("my-mcp", "http://localhost:8082/mcp")
	mcpManager.Start()
	// 启动HTTP网关
	if err := server.StartHTTPServer(httpAddr, grpcServer); err != nil {
		log.Fatalf("❌ HTTP服务器启动失败: %v", err)
	}
}
