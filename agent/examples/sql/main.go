package main

import (
	"database/sql"
	"flag"
	"fmt"
	agent "jas-agent/agent/agent"
	"jas-agent/agent/llm"
	"jas-agent/agent/tools"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sashabaranov/go-openai"
)

func main() {
	fmt.Println("Starting SQL Agent example...")

	var apiKey string
	var baseUrl string
	var dsn string
	flag.StringVar(&apiKey, "apiKey", "", "OpenAI API Key")
	flag.StringVar(&baseUrl, "baseUrl", "", "OpenAI Base URL")
	flag.StringVar(&dsn, "dsn", "root:password@tcp(localhost:3306)/testdb", "MySQL DSN")
	flag.Parse()

	if apiKey == "" || baseUrl == "" {
		log.Fatal("Please provide -apiKey and -baseUrl flags")
	}

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("Database connected successfully")

	// 注册 SQL 工具
	sqlConn := &tools.SQLConnection{DB: db}
	tools.RegisterSQLTools(sqlConn)

	// 创建 LLM 客户端
	chat := llm.NewChat(&llm.Config{
		ApiKey:  apiKey,
		BaseURL: baseUrl,
	})

	// 创建上下文
	context := agent.NewContext(
		agent.WithModel(openai.GPT3Dot5Turbo),
		agent.WithChat(chat),
	)

	// 创建执行器（使用 SQL Agent）
	executor := agent.NewSQLAgentExecutor(context, fmt.Sprintf("MySQL Database: %s", dsn))

	// 示例查询
	fmt.Println("\n=== Example 1: List all tables ===")
	result := executor.Run("查询：列出数据库中的所有表")
	fmt.Printf("Result: %s\n\n", result)

	// 示例查询 2
	fmt.Println("=== Example 2: Query user count ===")
	result = executor.Run("查询：查询用户表有多少条记录")
	fmt.Printf("Result: %s\n\n", result)

	// 示例查询 3
	fmt.Println("=== Example 3: Complex query ===")
	result = executor.Run("查询：查询每个用户的订单总金额，按金额降序排列,请以表格的形式展示")
	fmt.Printf("Result: %s\n", result)
}
