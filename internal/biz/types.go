package biz

import (
	"context"
	"time"
)

// Agent 领域模型
type Agent struct {
	ID               int
	Name             string
	Framework        string
	Description      string
	SystemPrompt     string
	MaxSteps         int
	Model            string
	MCPServices      []string
	ConnectionConfig string
	ConfigJSON       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	IsActive         bool
}

// MCPService 领域模型
type MCPService struct {
	ID          int
	Name        string
	Endpoint    string
	Description string
	IsActive    bool
	ToolCount   int
	LastRefresh time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AgentRepo 定义 Agent 数据访问接口
type AgentRepo interface {
	CreateAgent(ctx context.Context, agent *Agent) error
	UpdateAgent(ctx context.Context, agent *Agent) error
	DeleteAgent(ctx context.Context, id int) error
	GetAgent(ctx context.Context, id int) (*Agent, error)
	ListAgents(ctx context.Context) ([]*Agent, error)
}

// MCPRepo 定义 MCP 服务数据访问接口
type MCPRepo interface {
	CreateMCPService(ctx context.Context, service *MCPService) error
	UpdateMCPService(ctx context.Context, service *MCPService) error
	DeleteMCPService(ctx context.Context, id int) error
	DeleteMCPServiceByName(ctx context.Context, name string) error
	GetMCPService(ctx context.Context, id int) (*MCPService, error)
	GetMCPServiceByName(ctx context.Context, name string) (*MCPService, error)
	ListMCPServices(ctx context.Context) ([]*MCPService, error)
	UpdateMCPToolCount(ctx context.Context, name string, count int) error
}
