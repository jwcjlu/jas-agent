package server

import (
	"encoding/json"
	"fmt"
)

// SQLConnectionConfig SQL连接配置
type SQLConnectionConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// ESConnectionConfig ES连接配置
type ESConnectionConfig struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// parseSQLConnectionConfig 解析 SQL 连接配置
func (s *AgentServer) parseSQLConnectionConfig(configJSON string) (*SQLConnectionConfig, error) {
	if configJSON == "" || configJSON == "{}" {
		return nil, fmt.Errorf("SQL connection config is required")
	}

	var config SQLConnectionConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("failed to parse SQL config: %w", err)
	}

	// 验证必填字段
	if config.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if config.Port == 0 {
		config.Port = 3306 // 默认端口
	}
	if config.Database == "" {
		return nil, fmt.Errorf("database is required")
	}
	if config.Username == "" {
		return nil, fmt.Errorf("username is required")
	}

	return &config, nil
}

// parseESConnectionConfig 解析 ES 连接配置
func (s *AgentServer) parseESConnectionConfig(configJSON string) (*ESConnectionConfig, error) {
	if configJSON == "" || configJSON == "{}" {
		return nil, fmt.Errorf("Elasticsearch connection config is required")
	}

	var config ESConnectionConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("failed to parse ES config: %w", err)
	}

	// 验证必填字段
	if config.Host == "" {
		return nil, fmt.Errorf("host is required")
	}

	return &config, nil
}
