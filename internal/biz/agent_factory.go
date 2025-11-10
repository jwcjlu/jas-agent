package biz

import (
	"encoding/json"
	"fmt"
	"jas-agent/agent/agent"
)

type AgentFactory struct {
	factory map[agent.AgentType]IAgent
}

func NewAgentFactory() *AgentFactory {
	return &AgentFactory{factory: make(map[agent.AgentType]IAgent)}
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

type esConnectionConfig struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *AgentUsecase) parseESConnectionConfig(raw string) (*esConnectionConfig, error) {
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

type sqlConnectionConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

func (s *AgentUsecase) parseSQLConnectionConfig(raw string) (*sqlConnectionConfig, error) {
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
