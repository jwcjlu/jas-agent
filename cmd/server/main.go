package main

import (
	"flag"
	"fmt"
	"jas-agent/llm"
	"jas-agent/server"
	"jas-agent/storage"
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
		dbDSN    string
	)

	flag.StringVar(&httpAddr, "http", ":8080", "HTTP服务器地址")
	flag.StringVar(&apiKey, "apiKey", "", "OpenAI API Key")
	flag.StringVar(&baseURL, "baseUrl", "", "OpenAI Base URL")
	flag.StringVar(&model, "model", "gpt-3.5-turbo", "默认模型")
	flag.StringVar(&dbDSN, "dsn", "", "MySQL DSN (可选，格式: user:pass@tcp(host:port)/dbname)")
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

	// 连接数据库（如果提供了DSN）
	var db *storage.DB
	if dbDSN != "" {
		var err error
		db, err = storage.NewDB(dbDSN)
		if err != nil {
			log.Printf("⚠️ 数据库连接失败: %v (将在无数据库模式下运行)", err)
			db = nil
		} else {
			defer db.Close()
		}
	} else {
		fmt.Println("ℹ️ 未配置数据库，Agent管理功能将不可用")
	}

	// 创建gRPC服务
	grpcServer := server.NewAgentServer(chat, db)

	fmt.Println("✅ gRPC服务已创建")

	// 启动HTTP网关
	if err := server.StartHTTPServer(httpAddr, grpcServer); err != nil {
		log.Fatalf("❌ HTTP服务器启动失败: %v", err)
	}
}
