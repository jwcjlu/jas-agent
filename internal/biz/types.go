package biz

import (
	"context"
	"jas-agent/agent/agent"
	"jas-agent/agent/core"
	pb "jas-agent/api/agent/service/v1"
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
	MCPServers       []*MCPService
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

type MCPServiceDetail struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	Description string `json:"description,omitempty"`
	Active      bool   `json:"active"`
	ToolCount   int    `json:"tool_count"`
	CreatedAt   string `json:"created_at"`
	LastRefresh string `json:"last_refresh"`
}

type MCPToolDetail struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
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

type IAgent interface {
	Validate() bool
	CreateAgentExecutor(ctx context.Context,
		req *pb.ChatRequest,
		send func(c context.Context, msg core.Message) error) (*agent.AgentExecutor, error)
	AgentType() agent.AgentType
}
