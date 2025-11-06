package server

import (
	"context"
	"fmt"
	"jas-agent/storage"
	"jas-agent/tools"

	pb "jas-agent/api/proto"
)

// CreateAgent 创建Agent
func (s *AgentServer) CreateAgent(ctx context.Context, req *pb.AgentConfigRequest) (*pb.AgentConfigResponse, error) {
	if s.db == nil {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: "数据库未配置",
		}, nil
	}

	// 验证框架类型
	validFrameworks := []string{"react", "plan", "chain", "sql", "elasticsearch"}
	isValid := false
	for _, f := range validFrameworks {
		if req.Framework == f {
			isValid = true
			break
		}
	}
	if !isValid {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: "无效的框架类型，必须是: react, plan, chain, sql, elasticsearch",
		}, nil
	}

	// 创建Agent配置
	agentConfig := &storage.AgentConfig{
		Name:             req.Name,
		Framework:        req.Framework,
		Description:      req.Description,
		SystemPrompt:     req.SystemPrompt,
		MaxSteps:         int(req.MaxSteps),
		Model:            req.Model,
		MCPServices:      req.McpServices,
		ConnectionConfig: req.ConnectionConfig,
		IsActive:         true,
	}
	if len(agentConfig.Config) == 0 {
		agentConfig.Config = "{}"
	}
	if len(agentConfig.ConnectionConfig) == 0 {
		agentConfig.ConnectionConfig = "{}"
	}

	// 保存到数据库
	if err := s.db.CreateAgent(agentConfig); err != nil {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: fmt.Sprintf("创建失败: %v", err),
		}, nil
	}

	fmt.Printf("✅ Agent已创建: ID=%d, 名称=%s, 框架=%s\n", agentConfig.ID, agentConfig.Name, agentConfig.Framework)

	return &pb.AgentConfigResponse{
		Success: true,
		Message: fmt.Sprintf("成功创建Agent '%s'", req.Name),
		Agent:   s.agentConfigToProto(agentConfig),
	}, nil
}

// UpdateAgent 更新Agent
func (s *AgentServer) UpdateAgent(ctx context.Context, req *pb.AgentConfigRequest) (*pb.AgentConfigResponse, error) {
	if s.db == nil {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: "数据库未配置",
		}, nil
	}

	// 验证框架类型（如果提供）
	if req.Framework != "" {
		validFrameworks := []string{"react", "plan", "chain", "sql", "elasticsearch"}
		isValid := false
		for _, f := range validFrameworks {
			if req.Framework == f {
				isValid = true
				break
			}
		}
		if !isValid {
			return &pb.AgentConfigResponse{
				Success: false,
				Message: "无效的框架类型，必须是: react, plan, chain, sql, elasticsearch",
			}, nil
		}
	}

	// 更新Agent配置
	agentConfig := &storage.AgentConfig{
		ID:               int(req.Id),
		Name:             req.Name,
		Framework:        req.Framework,
		Description:      req.Description,
		SystemPrompt:     req.SystemPrompt,
		MaxSteps:         int(req.MaxSteps),
		Model:            req.Model,
		MCPServices:      req.McpServices,
		ConnectionConfig: req.ConnectionConfig,
		IsActive:         true,
	}

	if err := s.db.UpdateAgent(agentConfig); err != nil {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: fmt.Sprintf("更新失败: %v", err),
		}, nil
	}

	fmt.Printf("✅ Agent已更新: ID=%d, 名称=%s\n", agentConfig.ID, agentConfig.Name)

	return &pb.AgentConfigResponse{
		Success: true,
		Message: fmt.Sprintf("成功更新Agent '%s'", req.Name),
		Agent:   s.agentConfigToProto(agentConfig),
	}, nil
}

// DeleteAgent 删除Agent
func (s *AgentServer) DeleteAgent(ctx context.Context, req *pb.AgentDeleteRequest) (*pb.AgentConfigResponse, error) {
	if s.db == nil {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: "数据库未配置",
		}, nil
	}

	if err := s.db.DeleteAgent(int(req.Id)); err != nil {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: fmt.Sprintf("删除失败: %v", err),
		}, nil
	}

	fmt.Printf("🗑️ Agent已删除: ID=%d\n", req.Id)

	return &pb.AgentConfigResponse{
		Success: true,
		Message: "成功删除Agent",
	}, nil
}

// GetAgent 获取Agent
func (s *AgentServer) GetAgent(ctx context.Context, req *pb.AgentGetRequest) (*pb.AgentConfigResponse, error) {
	if s.db == nil {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: "数据库未配置",
		}, nil
	}

	agentConfig, err := s.db.GetAgent(int(req.Id))
	if err != nil {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: fmt.Sprintf("查询失败: %v", err),
		}, nil
	}

	return &pb.AgentConfigResponse{
		Success: true,
		Message: "查询成功",
		Agent:   s.agentConfigToProto(agentConfig),
	}, nil
}

// ListAgents 列出所有Agent
func (s *AgentServer) ListAgents(ctx context.Context, req *pb.Empty) (*pb.AgentListResponse, error) {
	if s.db == nil {
		return &pb.AgentListResponse{
			Agents: []*pb.AgentConfig{},
		}, nil
	}

	agentConfigs, err := s.db.ListAgents()
	if err != nil {
		fmt.Printf("❌ 查询Agent列表失败: %v\n", err)
		return &pb.AgentListResponse{
			Agents: []*pb.AgentConfig{},
		}, nil
	}

	agents := make([]*pb.AgentConfig, len(agentConfigs))
	for i, config := range agentConfigs {
		agents[i] = s.agentConfigToProto(config)
	}

	return &pb.AgentListResponse{
		Agents: agents,
	}, nil
}

// agentConfigToProto 转换Agent配置为Proto格式
func (s *AgentServer) agentConfigToProto(config *storage.AgentConfig) *pb.AgentConfig {
	return &pb.AgentConfig{
		Id:               int32(config.ID),
		Name:             config.Name,
		Framework:        config.Framework,
		Description:      config.Description,
		SystemPrompt:     config.SystemPrompt,
		MaxSteps:         int32(config.MaxSteps),
		Model:            config.Model,
		McpServices:      config.MCPServices,
		CreatedAt:        config.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:        config.UpdatedAt.Format("2006-01-02 15:04:05"),
		IsActive:         config.IsActive,
		ConnectionConfig: config.ConnectionConfig,
	}
}

// loadMCPServicesFromDB 从数据库加载MCP服务
func (s *AgentServer) loadMCPServicesFromDB() {
	if s.db == nil {
		return
	}

	services, err := s.db.ListMCPServices()
	if err != nil {
		fmt.Printf("⚠️ 从数据库加载MCP服务失败: %v\n", err)
		return
	}

	for _, service := range services {
		if !service.IsActive {
			continue
		}

		// 创建MCP工具管理器
		mcpManager, err := tools.NewMCPToolManager(service.Name, service.Endpoint)
		if err != nil {
			fmt.Printf("⚠️ 创建MCP工具管理器失败 [%s]: %v\n", service.Name, err)
			continue
		}

		// 启动工具发现
		mcpManager.Start()

		// 注册到全局工具管理器
		tools.GetToolManager().RegisterMCPToolManager(service.Name, mcpManager)

		// 保存到内存
		s.mcpLock.Lock()
		s.mcpServices[service.Name] = &MCPServiceInfo{
			Name:        service.Name,
			Endpoint:    service.Endpoint,
			Manager:     mcpManager,
			Active:      true,
			ToolCount:   service.ToolCount,
			CreatedAt:   service.CreatedAt,
			LastRefresh: service.LastRefresh,
		}
		s.mcpLock.Unlock()

		fmt.Printf("📋 已加载MCP服务: %s (%d个工具)\n", service.Name, service.ToolCount)
	}
}
