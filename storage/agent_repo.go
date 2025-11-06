package storage

import (
	"database/sql"
	"fmt"
	"strings"
)

// CreateAgent 创建Agent
func (db *DB) CreateAgent(agent *AgentConfig) error {
	// 插入Agent
	result, err := db.conn.Exec(`
		INSERT INTO agents (name, framework, description, system_prompt, max_steps, model, config, connection_config, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, agent.Name, agent.Framework, agent.Description, agent.SystemPrompt,
		agent.MaxSteps, agent.Model, agent.Config, agent.ConnectionConfig, agent.IsActive)

	if err != nil {
		return fmt.Errorf("创建Agent失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	agent.ID = int(id)

	// 绑定MCP服务
	if len(agent.MCPServices) > 0 {
		if err := db.bindMCPServices(agent.ID, agent.MCPServices); err != nil {
			return err
		}
	}

	return nil
}

// UpdateAgent 更新Agent
func (db *DB) UpdateAgent(agent *AgentConfig) error {
	// 更新Agent基本信息
	_, err := db.conn.Exec(`
		UPDATE agents 
		SET name = ?, framework = ?, description = ?, system_prompt = ?, 
		    max_steps = ?, model = ?, config = ?, connection_config = ?, is_active = ?
		WHERE id = ?
	`, agent.Name, agent.Framework, agent.Description, agent.SystemPrompt,
		agent.MaxSteps, agent.Model, agent.Config, agent.ConnectionConfig, agent.IsActive, agent.ID)

	if err != nil {
		return fmt.Errorf("更新Agent失败: %w", err)
	}

	// 重新绑定MCP服务
	// 先删除旧的绑定
	_, err = db.conn.Exec("DELETE FROM agent_mcp_bindings WHERE agent_id = ?", agent.ID)
	if err != nil {
		return err
	}

	// 添加新的绑定
	if len(agent.MCPServices) > 0 {
		if err := db.bindMCPServices(agent.ID, agent.MCPServices); err != nil {
			return err
		}
	}

	return nil
}

// DeleteAgent 删除Agent
func (db *DB) DeleteAgent(id int) error {
	_, err := db.conn.Exec("DELETE FROM agents WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除Agent失败: %w", err)
	}
	return nil
}

// GetAgent 获取Agent
func (db *DB) GetAgent(id int) (*AgentConfig, error) {
	agent := &AgentConfig{}

	err := db.conn.QueryRow(`
		SELECT id, name, framework, description, system_prompt, max_steps, model, 
		       COALESCE(config, '{}'), COALESCE(connection_config, '{}'), created_at, updated_at, is_active
		FROM agents WHERE id = ?
	`, id).Scan(&agent.ID, &agent.Name, &agent.Framework, &agent.Description,
		&agent.SystemPrompt, &agent.MaxSteps, &agent.Model, &agent.Config,
		&agent.ConnectionConfig, &agent.CreatedAt, &agent.UpdatedAt, &agent.IsActive)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Agent不存在: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询Agent失败: %w", err)
	}

	// 加载MCP服务绑定
	mcpServices, err := db.getAgentMCPServices(id)
	if err != nil {
		return nil, err
	}
	agent.MCPServices = mcpServices

	return agent, nil
}

// ListAgents 列出所有Agent
func (db *DB) ListAgents() ([]*AgentConfig, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, framework, description, system_prompt, max_steps, model,
		       COALESCE(config, '{}'), COALESCE(connection_config, '{}'), created_at, updated_at, is_active
		FROM agents
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询Agent列表失败: %w", err)
	}
	defer rows.Close()

	var agents []*AgentConfig
	for rows.Next() {
		agent := &AgentConfig{}
		err := rows.Scan(&agent.ID, &agent.Name, &agent.Framework, &agent.Description,
			&agent.SystemPrompt, &agent.MaxSteps, &agent.Model, &agent.Config,
			&agent.ConnectionConfig, &agent.CreatedAt, &agent.UpdatedAt, &agent.IsActive)
		if err != nil {
			return nil, err
		}

		// 加载MCP服务绑定
		mcpServices, err := db.getAgentMCPServices(agent.ID)
		if err != nil {
			return nil, err
		}
		agent.MCPServices = mcpServices

		agents = append(agents, agent)
	}

	return agents, nil
}

// 辅助方法

// bindMCPServices 绑定MCP服务
func (db *DB) bindMCPServices(agentID int, mcpServiceNames []string) error {
	if len(mcpServiceNames) == 0 {
		return nil
	}

	// 查询MCP服务ID
	placeholders := strings.Repeat("?,", len(mcpServiceNames))
	placeholders = placeholders[:len(placeholders)-1]

	query := fmt.Sprintf("SELECT id, name FROM mcp_services WHERE name IN (%s)", placeholders)

	args := make([]interface{}, len(mcpServiceNames))
	for i, name := range mcpServiceNames {
		args[i] = name
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	serviceIDs := make(map[string]int)
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		serviceIDs[name] = id
	}

	// 插入绑定
	for _, serviceName := range mcpServiceNames {
		if serviceID, ok := serviceIDs[serviceName]; ok {
			_, err := db.conn.Exec(`
				INSERT INTO agent_mcp_bindings (agent_id, mcp_service_id)
				VALUES (?, ?)
			`, agentID, serviceID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// getAgentMCPServices 获取Agent绑定的MCP服务
func (db *DB) getAgentMCPServices(agentID int) ([]string, error) {
	rows, err := db.conn.Query(`
		SELECT m.name
		FROM mcp_services m
		INNER JOIN agent_mcp_bindings b ON m.id = b.mcp_service_id
		WHERE b.agent_id = ?
	`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		services = append(services, name)
	}

	return services, nil
}
