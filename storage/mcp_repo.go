package storage

import (
	"database/sql"
	"fmt"
)

// CreateMCPService 创建MCP服务
func (db *DB) CreateMCPService(service *MCPService) error {
	result, err := db.conn.Exec(`
		INSERT INTO mcp_services (name, endpoint, description, is_active, tool_count)
		VALUES (?, ?, ?, ?, ?)
	`, service.Name, service.Endpoint, service.Description, service.IsActive, service.ToolCount)

	if err != nil {
		return fmt.Errorf("创建MCP服务失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	service.ID = int(id)

	return nil
}

// UpdateMCPService 更新MCP服务
func (db *DB) UpdateMCPService(service *MCPService) error {
	_, err := db.conn.Exec(`
		UPDATE mcp_services
		SET endpoint = ?, description = ?, is_active = ?, tool_count = ?, last_refresh = ?
		WHERE id = ?
	`, service.Endpoint, service.Description, service.IsActive, service.ToolCount,
		service.LastRefresh, service.ID)

	if err != nil {
		return fmt.Errorf("更新MCP服务失败: %w", err)
	}

	return nil
}

// DeleteMCPService 删除MCP服务
func (db *DB) DeleteMCPService(id int) error {
	_, err := db.conn.Exec("DELETE FROM mcp_services WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除MCP服务失败: %w", err)
	}
	return nil
}

// DeleteMCPServiceByName 根据名称删除MCP服务
func (db *DB) DeleteMCPServiceByName(name string) error {
	_, err := db.conn.Exec("DELETE FROM mcp_services WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("删除MCP服务失败: %w", err)
	}
	return nil
}

// GetMCPService 获取MCP服务
func (db *DB) GetMCPService(id int) (*MCPService, error) {
	service := &MCPService{}

	err := db.conn.QueryRow(`
		SELECT id, name, endpoint, COALESCE(description, ''), is_active, tool_count,
		       COALESCE(last_refresh, NOW()), created_at, updated_at
		FROM mcp_services WHERE id = ?
	`, id).Scan(&service.ID, &service.Name, &service.Endpoint, &service.Description,
		&service.IsActive, &service.ToolCount, &service.LastRefresh,
		&service.CreatedAt, &service.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("MCP服务不存在: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询MCP服务失败: %w", err)
	}

	return service, nil
}

// GetMCPServiceByName 根据名称获取MCP服务
func (db *DB) GetMCPServiceByName(name string) (*MCPService, error) {
	service := &MCPService{}

	err := db.conn.QueryRow(`
		SELECT id, name, endpoint, COALESCE(description, ''), is_active, tool_count,
		       COALESCE(last_refresh, NOW()), created_at, updated_at
		FROM mcp_services WHERE name = ?
	`, name).Scan(&service.ID, &service.Name, &service.Endpoint, &service.Description,
		&service.IsActive, &service.ToolCount, &service.LastRefresh,
		&service.CreatedAt, &service.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil // 不存在返回nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询MCP服务失败: %w", err)
	}

	return service, nil
}

// ListMCPServices 列出所有MCP服务
func (db *DB) ListMCPServices() ([]*MCPService, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, endpoint, COALESCE(description, ''), is_active, tool_count,
		       COALESCE(last_refresh, NOW()), created_at, updated_at
		FROM mcp_services
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询MCP服务列表失败: %w", err)
	}
	defer rows.Close()

	var services []*MCPService
	for rows.Next() {
		service := &MCPService{}
		err := rows.Scan(&service.ID, &service.Name, &service.Endpoint, &service.Description,
			&service.IsActive, &service.ToolCount, &service.LastRefresh,
			&service.CreatedAt, &service.UpdatedAt)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}

	return services, nil
}

// UpdateMCPToolCount 更新MCP服务的工具数量
func (db *DB) UpdateMCPToolCount(name string, count int) error {
	_, err := db.conn.Exec(`
		UPDATE mcp_services
		SET tool_count = ?, last_refresh = NOW()
		WHERE name = ?
	`, count, name)
	return err
}
