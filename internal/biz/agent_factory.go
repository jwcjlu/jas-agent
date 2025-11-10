package biz

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"jas-agent/agent/agent"
	"jas-agent/agent/tools"
)

type AgentFactory struct {
	factory map[agent.AgentType]IAgent
}

func NewAgentFactory() *AgentFactory {
	af := &AgentFactory{factory: make(map[agent.AgentType]IAgent)}
	af.RegisterAgent(&reactAgent{})
	af.RegisterAgent(&planAgent{})
	af.RegisterAgent(&chainAgent{})
	af.RegisterAgent(&sqlAgent{})
	af.RegisterAgent(&esAgent{})
	return af
}

func (factory *AgentFactory) RegisterAgent(agent IAgent) {
	factory.factory[agent.AgentType()] = agent
}

func (factory *AgentFactory) GetAgent(agentType agent.AgentType) (IAgent, error) {
	agent, ok := factory.factory[agentType]
	if !ok {
		return nil, fmt.Errorf("not found agent type")
	}
	return agent, nil
}

func (factory *AgentFactory) CreateAgentExecutor(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context) (*agent.AgentExecutor, error) {
	iAgent := factory.findIAgent(agentConfig.Framework)
	if iAgent == nil {
		return agent.NewAgentExecutor(agentCtx), nil
	}
	return iAgent.CreateAgentExecutor(ctx, agentConfig, agentCtx)
}

func (factory *AgentFactory) findIAgent(framework string) IAgent {
	iAgent, ok := factory.factory[agent.AgentType(framework)]
	if ok {
		return iAgent
	}
	for _, v := range factory.factory {
		if v.Alias() == framework {
			return v
		}
	}
	return nil
}

type planAgent struct {
}

func (s *planAgent) Validate() bool {
	return true
}
func (s *planAgent) CreateAgentExecutor(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context) (*agent.AgentExecutor, error) {

	return agent.NewPlanAgentExecutor(agentCtx, true), nil
}
func (s *planAgent) AgentType() agent.AgentType {
	return agent.PlanAgentType
}
func (s *planAgent) Description() string {
	return "计划代理，先规划后执行复杂任务"
}
func (s *planAgent) Alias() string {
	return "plan"
}

type reactAgent struct {
}

func (s *reactAgent) Validate() bool {
	return true
}
func (s *reactAgent) CreateAgentExecutor(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context) (*agent.AgentExecutor, error) {
	return agent.NewAgentExecutor(agentCtx), nil
}
func (s *reactAgent) AgentType() agent.AgentType {
	return agent.ReactAgentType
}
func (s *reactAgent) Description() string {
	return "通用推理代理，支持思考-行动-观察循环"
}
func (s *reactAgent) Alias() string {
	return "react"
}

type chainAgent struct {
}

func (s *chainAgent) Validate() bool {
	return true
}
func (s *chainAgent) CreateAgentExecutor(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context) (*agent.AgentExecutor, error) {
	builder := agent.NewChainBuilder(agentCtx)
	builder.AddNode("main", agent.ReactAgentType, 100)
	ca := builder.Build()
	return agent.NewChainAgentExecutor(agentCtx, ca), nil
}
func (s *chainAgent) AgentType() agent.AgentType {
	return agent.ChainAgentType
}
func (s *chainAgent) Description() string {
	return "链式代理，按预定义流程执行多个Agent"
}
func (s *chainAgent) Alias() string {
	return "chain"
}

type sqlAgent struct {
}

func (s *sqlAgent) Validate() bool {
	return true
}

type sqlConnectionConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

func (s *sqlAgent) parseSQLConnectionConfig(raw string) (*sqlConnectionConfig, error) {
	if raw == "" {
		return nil, fmt.Errorf("SQL 连接配置为空")
	}

	cfg := &sqlConnectionConfig{
		Port: 3306,
	}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("解析 SQL 连接配置失败: %w", err)
	}

	if cfg.Host == "" || cfg.Username == "" || cfg.Database == "" {
		return nil, fmt.Errorf("SQL 连接配置缺少必要字段")
	}
	return cfg, nil
}
func (s *sqlAgent) CreateAgentExecutor(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context) (*agent.AgentExecutor, error) {
	// 解析 SQL 连接配置
	connConfig, err := s.parseSQLConnectionConfig(agentConfig.ConnectionConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid SQL connection config: %w", err)
	}

	// 创建 SQL 连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		connConfig.Username, connConfig.Password,
		connConfig.Host, connConfig.Port, connConfig.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	// 注册 SQL 工具
	sqlConn := &tools.SQLConnection{DB: db}
	tools.RegisterSQLTools(sqlConn, agentCtx.GetToolManager())

	// 创建 SQL Agent
	dbInfo := fmt.Sprintf("MySQL: %s@%s:%d/%s", connConfig.Username, connConfig.Host, connConfig.Port, connConfig.Database)
	return agent.NewSQLAgentExecutor(agentCtx, dbInfo), nil

}
func (s *sqlAgent) AgentType() agent.AgentType {
	return agent.SQLAgentType
}
func (s *sqlAgent) Description() string {
	return "SQL查询专家，生成和执行数据库查询"
}
func (s *sqlAgent) Alias() string {
	return "sql"
}

type esAgent struct {
}

func (s *esAgent) Validate() bool {
	return true
}
func (s *esAgent) CreateAgentExecutor(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context) (*agent.AgentExecutor, error) {
	// 解析 ES 连接配置
	esConfig, err := s.parseESConnectionConfig(agentConfig.ConnectionConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid ES connection config: %w", err)
	}

	// 创建 ES 连接
	esConn := tools.NewESConnection(esConfig.Host, esConfig.Username, esConfig.Password)

	// 注册 ES 工具
	tools.RegisterESTools(esConn, agentCtx.GetToolManager())

	// 创建 ES Agent
	clusterInfo := fmt.Sprintf("Elasticsearch: %s", esConfig.Host)
	executor := agent.NewESAgentExecutor(agentCtx, clusterInfo)
	return executor, nil

}

type esConnectionConfig struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *esAgent) parseESConnectionConfig(raw string) (*esConnectionConfig, error) {
	if raw == "" {
		return nil, fmt.Errorf("elasticsearch 连接配置为空")
	}

	cfg := &esConnectionConfig{}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("解析 Elasticsearch 连接配置失败: %w", err)
	}

	if cfg.Host == "" {
		return nil, fmt.Errorf("elasticsearch 连接配置缺少 host")
	}
	return cfg, nil
}

func (s *esAgent) AgentType() agent.AgentType {
	return agent.ESAgentType
}
func (s *esAgent) Description() string {
	return "计划代理，先规划后执行复杂任务"
}

func (s *esAgent) Alias() string {
	return "elasticsearch"
}
