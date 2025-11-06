package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DB 数据库连接
type DB struct {
	conn *sql.DB
}

// NewDB 创建数据库连接
func NewDB(dsn string) (*DB, error) {
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 测试连接
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 设置连接池
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	fmt.Println("✅ 数据库连接成功")

	return &DB{conn: conn}, nil
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// AgentConfig Agent配置
type AgentConfig struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Framework        string    `json:"framework"`
	Description      string    `json:"description"`
	SystemPrompt     string    `json:"system_prompt"`
	MaxSteps         int       `json:"max_steps"`
	Model            string    `json:"model"`
	MCPServices      []string  `json:"mcp_services"`
	Config           string    `json:"config"`            // JSON string
	ConnectionConfig string    `json:"connection_config"` // JSON string for DB/ES connection
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	IsActive         bool      `json:"is_active"`
}

// MCPService MCP服务
type MCPService struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Endpoint    string    `json:"endpoint"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	ToolCount   int       `json:"tool_count"`
	LastRefresh time.Time `json:"last_refresh"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
