package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	agent "jas-agent/agent/agent"
	"jas-agent/agent/llm"
	"jas-agent/agent/tools"

	"github.com/go-kratos/kratos/v2/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sashabaranov/go-openai"
)

var logger = log.NewHelper(log.With(log.NewStdLogger(os.Stdout), "module", "examples/sql"))

func main() {
	logger.Info("Starting SQL Agent example...")

	var apiKey string
	var baseUrl string
	var dsn string
	flag.StringVar(&apiKey, "apiKey", "", "OpenAI API Key")
	flag.StringVar(&baseUrl, "baseUrl", "", "OpenAI Base URL")
	flag.StringVar(&dsn, "dsn", "root:password@tcp(localhost:3306)/testdb", "MySQL DSN")
	flag.Parse()

	if apiKey == "" || baseUrl == "" {
		logger.Fatal("Please provide -apiKey and -baseUrl flags")
	}

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err = db.Ping(); err != nil {
		logger.Fatalf("Failed to ping database: %v", err)
	}
	logger.Info("Database connected successfully")

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
	logger.Info("\n=== Example 1: List all tables ===")
	result := executor.Run("查询：列出数据库中的所有表")
	logger.Infof("Result: %s", result)
	logger.Info("")

	// 示例查询 2
	logger.Info("=== Example 2: Query user count ===")
	result = executor.Run("查询：查询用户表有多少条记录")
	logger.Infof("Result: %s", result)
	logger.Info("")

	// 示例查询 3
	logger.Info("=== Example 3: Complex query ===")
	result = executor.Run("查询：查询每个用户的订单总金额，按金额降序排列,请以表格的形式展示")
	logger.Infof("Result: %s", result)
}
