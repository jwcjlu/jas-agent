package biz

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	agent "jas-agent/agent/agent"
	"jas-agent/agent/core"
	"jas-agent/agent/llm"
	"jas-agent/agent/memory"
	tools "jas-agent/agent/tools"
	pb "jas-agent/api/agent/service/v1"

	_ "github.com/go-sql-driver/mysql"
)

// AgentUsecase 负责 Agent 相关业务逻辑
type AgentUsecase struct {
	chat         llm.Chat
	agentRepo    AgentRepo
	mcpRepo      MCPRepo
	sessions     map[string]*SessionContext
	sessionsLock sync.RWMutex
	mcpServices  map[string]*MCPServiceInfo
	mcpLock      sync.RWMutex
}

// MCPServiceInfo MCP服务信息
type MCPServiceInfo struct {
	Name        string
	Endpoint    string
	Manager     *tools.MCPToolManager
	Active      bool
	ToolCount   int
	CreatedAt   time.Time
	LastRefresh time.Time
}

// SessionContext 会话上下文
type SessionContext struct {
	Memory    core.Memory
	Context   *agent.Context
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewAgentUsecase 创建新的 AgentUsecase。
func NewAgentUsecase(chat llm.Chat, agentRepo AgentRepo, mcpRepo MCPRepo) *AgentUsecase {
	uc := &AgentUsecase{
		chat:        chat,
		sessions:    make(map[string]*SessionContext),
		mcpServices: make(map[string]*MCPServiceInfo),
		agentRepo:   agentRepo,
		mcpRepo:     mcpRepo,
	}

	if uc.mcpRepo != nil {
		uc.loadMCPServicesFromDB()
	}

	return uc
}

// Chat 实现单次对话
func (s *AgentUsecase) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	startTime := time.Now()

	// 创建或获取会话上下文
	agentCtx, err := s.getOrCreateSession(req)
	if err != nil {
		return &pb.ChatResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// 创建执行器
	executor, err := s.createExecutor(ctx, req, agentCtx)
	if err != nil {
		return &pb.ChatResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// 执行查询
	result := executor.Run(req.Query)

	// 收集元数据
	metadata := &pb.ExecutionMetadata{
		TotalSteps:      int32(executor.GetCurrentStep()),
		ExecutionTimeMs: time.Since(startTime).Milliseconds(),
		State:           string(executor.GetState()),
	}

	// 收集使用的工具
	toolNames := s.extractToolNames(agentCtx.GetMemory())
	metadata.ToolNames = toolNames
	metadata.ToolsCalled = int32(len(toolNames))

	return &pb.ChatResponse{
		Response:  result,
		AgentType: s.getAgentTypeName(req.AgentType),
		Metadata:  metadata,
		Success:   true,
	}, nil
}

// StreamChat 实现流式对话
func (s *AgentUsecase) StreamChat(req *pb.ChatRequest, stream pb.AgentService_StreamChatServer) error {
	return s.StreamChatWithSender(stream.Context(), req, stream.Send)
}

// StreamChatWithSender 使用自定义发送函数实现流式对话，可用于 WebSocket 等场景。
func (s *AgentUsecase) StreamChatWithSender(ctx context.Context, req *pb.ChatRequest, send func(*pb.ChatStreamResponse) error) error {
	startTime := time.Now()
	resultChan := make(chan string)

	agentCtx, err := s.getOrCreateSession(req)
	if err != nil {
		return send(&pb.ChatStreamResponse{
			Type:    pb.ChatStreamResponse_ERROR,
			Content: err.Error(),
		})
	}

	executor, err := s.createExecutor(ctx, req, agentCtx)
	if err != nil {
		return send(&pb.ChatStreamResponse{
			Type:    pb.ChatStreamResponse_ERROR,
			Content: err.Error(),
		})
	}

	messageChan := make(chan core.Message, 10)
	done := make(chan bool)

	go s.monitorExecution(agentCtx.GetMemory(), messageChan, done)

	go func() {
		result := executor.Run(req.Query)
		resultChan <- result
		close(done)
	}()

	step := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-messageChan:
			if !ok {
				goto SEND_FINAL
			}

			step++
			msgType, content := s.parseMessage(msg)

			if err = send(&pb.ChatStreamResponse{
				Type:    msgType,
				Content: content,
				Step:    int32(step),
			}); err != nil {
				return err
			}

		case result := <-resultChan:
			metadata := &pb.ExecutionMetadata{
				TotalSteps:      int32(executor.GetCurrentStep()),
				ExecutionTimeMs: time.Since(startTime).Milliseconds(),
				State:           string(executor.GetState()),
			}

			toolNames := s.extractToolNames(agentCtx.GetMemory())
			metadata.ToolNames = toolNames
			metadata.ToolsCalled = int32(len(toolNames))

			return send(&pb.ChatStreamResponse{
				Type:     pb.ChatStreamResponse_FINAL,
				Content:  result,
				Metadata: metadata,
			})
		}
	}

SEND_FINAL:
	result := <-resultChan
	metadata := &pb.ExecutionMetadata{
		TotalSteps:      int32(executor.GetCurrentStep()),
		ExecutionTimeMs: time.Since(startTime).Milliseconds(),
		State:           string(executor.GetState()),
	}

	return send(&pb.ChatStreamResponse{
		Type:     pb.ChatStreamResponse_FINAL,
		Content:  result,
		Metadata: metadata,
	})
}

// ListAgentTypes 列出可用的 Agent 类型
func (s *AgentUsecase) ListAgentTypes(ctx context.Context, req *pb.Empty) (*pb.AgentTypesResponse, error) {
	types := []*pb.AgentTypeInfo{
		{
			Type:        pb.AgentType_REACT,
			Name:        "ReAct Agent",
			Description: "通用推理代理，支持思考-行动-观察循环",
			Available:   true,
		},
		{
			Type:        pb.AgentType_CHAIN,
			Name:        "Chain Agent",
			Description: "链式代理，按预定义流程执行多个Agent",
			Available:   true,
		},
		{
			Type:        pb.AgentType_PLAN,
			Name:        "Plan Agent",
			Description: "计划代理，先规划后执行复杂任务",
			Available:   true,
		},
		{
			Type:        pb.AgentType_SQL,
			Name:        "SQL Agent",
			Description: "SQL查询专家，生成和执行数据库查询",
			Available:   false, // 需要数据库连接
		},
	}

	return &pb.AgentTypesResponse{
		Types: types,
	}, nil
}

// ListTools 列出可用的工具
func (s *AgentUsecase) ListTools(ctx context.Context, req *pb.Empty) (*pb.ToolsResponse, error) {
	toolManager := tools.GetToolManager()
	availableTools := toolManager.AvailableTools()

	toolInfos := make([]*pb.ToolInfo, 0, len(availableTools))
	for _, tool := range availableTools {
		toolType := "Normal"
		mcpService := ""

		if tool.Type() == core.Mcp {
			toolType = "MCP"
			// 提取MCP服务名称（格式：serviceName@toolName）
			if idx := strings.Index(tool.Name(), "@"); idx > 0 {
				mcpService = tool.Name()[:idx]
			}
		}

		toolInfos = append(toolInfos, &pb.ToolInfo{
			Name:        tool.Name(),
			Description: tool.Description(),
			Type:        toolType,
			McpService:  mcpService,
		})
	}

	return &pb.ToolsResponse{
		Tools: toolInfos,
	}, nil
}

func (s *AgentUsecase) CreateAgent(ctx context.Context, req *pb.AgentConfigRequest) (*pb.AgentConfigResponse, error) {
	if s.agentRepo == nil {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: "数据库未配置",
		}, nil
	}

	validFrameworks := map[string]struct{}{
		"react":         {},
		"plan":          {},
		"chain":         {},
		"sql":           {},
		"elasticsearch": {},
	}
	if _, ok := validFrameworks[req.Framework]; !ok {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: "无效的框架类型，必须是: react, plan, chain, sql, elasticsearch",
		}, nil
	}

	agentConfig := &Agent{
		Name:             req.Name,
		Framework:        req.Framework,
		Description:      req.Description,
		SystemPrompt:     req.SystemPrompt,
		MaxSteps:         int(req.MaxSteps),
		Model:            req.Model,
		MCPServices:      req.McpServices,
		ConnectionConfig: req.ConnectionConfig,
		ConfigJSON:       req.ConfigJson,
		IsActive:         true,
	}
	if agentConfig.ConfigJSON == "" {
		agentConfig.ConfigJSON = "{}"
	}
	if agentConfig.ConnectionConfig == "" {
		agentConfig.ConnectionConfig = "{}"
	}

	if err := s.agentRepo.CreateAgent(ctx, agentConfig); err != nil {
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

func (s *AgentUsecase) UpdateAgent(ctx context.Context, req *pb.AgentConfigRequest) (*pb.AgentConfigResponse, error) {
	if s.agentRepo == nil {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: "数据库未配置",
		}, nil
	}

	if req.Framework != "" {
		validFrameworks := map[string]struct{}{
			"react":         {},
			"plan":          {},
			"chain":         {},
			"sql":           {},
			"elasticsearch": {},
		}
		if _, ok := validFrameworks[req.Framework]; !ok {
			return &pb.AgentConfigResponse{
				Success: false,
				Message: "无效的框架类型，必须是: react, plan, chain, sql, elasticsearch",
			}, nil
		}
	}

	agentConfig := &Agent{
		ID:               int(req.Id),
		Name:             req.Name,
		Framework:        req.Framework,
		Description:      req.Description,
		SystemPrompt:     req.SystemPrompt,
		MaxSteps:         int(req.MaxSteps),
		Model:            req.Model,
		MCPServices:      req.McpServices,
		ConnectionConfig: req.ConnectionConfig,
		ConfigJSON:       req.ConfigJson,
		IsActive:         true,
	}

	if err := s.agentRepo.UpdateAgent(ctx, agentConfig); err != nil {
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

func (s *AgentUsecase) DeleteAgent(ctx context.Context, req *pb.AgentDeleteRequest) (*pb.AgentConfigResponse, error) {
	if s.agentRepo == nil {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: "数据库未配置",
		}, nil
	}

	if err := s.agentRepo.DeleteAgent(ctx, int(req.Id)); err != nil {
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

func (s *AgentUsecase) GetAgent(ctx context.Context, req *pb.AgentGetRequest) (*pb.AgentConfigResponse, error) {
	if s.agentRepo == nil {
		return &pb.AgentConfigResponse{
			Success: false,
			Message: "数据库未配置",
		}, nil
	}

	agentConfig, err := s.agentRepo.GetAgent(ctx, int(req.Id))
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

func (s *AgentUsecase) ListAgents(ctx context.Context, req *pb.Empty) (*pb.AgentListResponse, error) {
	if s.agentRepo == nil {
		return &pb.AgentListResponse{
			Agents: []*pb.AgentConfig{},
		}, nil
	}

	agentConfigs, err := s.agentRepo.ListAgents(ctx)
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

func (s *AgentUsecase) agentConfigToProto(config *Agent) *pb.AgentConfig {
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
		ConfigJson:       config.ConfigJSON,
	}
}

// AddMCPService 添加MCP服务
func (s *AgentUsecase) AddMCPService(ctx context.Context, req *pb.MCPServiceRequest) (*pb.MCPServiceResponse, error) {
	s.mcpLock.Lock()
	defer s.mcpLock.Unlock()

	// 检查是否已存在
	if _, exists := s.mcpServices[req.Name]; exists {
		return &pb.MCPServiceResponse{
			Success: false,
			Message: fmt.Sprintf("MCP服务 '%s' 已存在", req.Name),
		}, nil
	}

	// 创建MCP工具管理器
	mcpManager, err := tools.NewMCPToolManager(req.Name, req.Endpoint)
	if err != nil {
		return &pb.MCPServiceResponse{
			Success: false,
			Message: fmt.Sprintf("创建MCP服务失败: %v", err),
		}, nil
	}

	// 启动工具发现
	mcpManager.Start()

	// 注册到全局工具管理器
	tools.GetToolManager().RegisterMCPToolManager(req.Name, mcpManager)

	// 保存服务信息到内存
	serviceInfo := &MCPServiceInfo{
		Name:        req.Name,
		Endpoint:    req.Endpoint,
		Manager:     mcpManager,
		Active:      true,
		ToolCount:   len(mcpManager.GetTools()),
		CreatedAt:   time.Now(),
		LastRefresh: time.Now(),
	}
	s.mcpServices[req.Name] = serviceInfo

	// 保存到数据库
	if s.mcpRepo != nil {
		dbService := &MCPService{
			Name:        req.Name,
			Endpoint:    req.Endpoint,
			IsActive:    true,
			ToolCount:   serviceInfo.ToolCount,
			LastRefresh: time.Now(),
		}
		if err := s.mcpRepo.CreateMCPService(ctx, dbService); err != nil {
			fmt.Printf("⚠️ 保存MCP服务到数据库失败: %v\n", err)
		}
	}

	fmt.Printf("✅ MCP服务已添加: %s -> %s (%d个工具)\n", req.Name, req.Endpoint, serviceInfo.ToolCount)

	return &pb.MCPServiceResponse{
		Success: true,
		Message: fmt.Sprintf("成功添加MCP服务 '%s'", req.Name),
		Service: s.mcpServiceInfoToProto(serviceInfo),
	}, nil
}

// RemoveMCPService 移除MCP服务
func (s *AgentUsecase) RemoveMCPService(ctx context.Context, req *pb.MCPServiceRequest) (*pb.MCPServiceResponse, error) {
	s.mcpLock.Lock()
	defer s.mcpLock.Unlock()

	_, exists := s.mcpServices[req.Name]
	if !exists {
		return &pb.MCPServiceResponse{
			Success: false,
			Message: fmt.Sprintf("MCP服务 '%s' 不存在", req.Name),
		}, nil
	}

	// 从内存中删除
	delete(s.mcpServices, req.Name)

	// 从数据库中删除
	if s.mcpRepo != nil {
		if err := s.mcpRepo.DeleteMCPServiceByName(ctx, req.Name); err != nil {
			fmt.Printf("⚠️ 从数据库删除MCP服务失败: %v\n", err)
		}
	}

	fmt.Printf("🗑️ MCP服务已移除: %s\n", req.Name)

	return &pb.MCPServiceResponse{
		Success: true,
		Message: fmt.Sprintf("成功移除MCP服务 '%s'", req.Name),
	}, nil
}

// ListMCPServices 列出所有MCP服务
func (s *AgentUsecase) ListMCPServices(ctx context.Context, req *pb.Empty) (*pb.MCPServicesResponse, error) {
	s.mcpLock.RLock()
	defer s.mcpLock.RUnlock()

	services := make([]*pb.MCPServiceInfo, 0, len(s.mcpServices))
	for _, info := range s.mcpServices {
		// 更新工具数量
		if info.Manager != nil {
			info.ToolCount = len(info.Manager.GetTools())
			info.LastRefresh = time.Now()
		}

		services = append(services, s.mcpServiceInfoToProto(info))
	}

	return &pb.MCPServicesResponse{
		Services: services,
	}, nil
}

// mcpServiceInfoToProto 转换MCP服务信息为Proto格式
func (s *AgentUsecase) mcpServiceInfoToProto(info *MCPServiceInfo) *pb.MCPServiceInfo {
	return &pb.MCPServiceInfo{
		Name:        info.Name,
		Endpoint:    info.Endpoint,
		Active:      info.Active,
		ToolCount:   int32(info.ToolCount),
		CreatedAt:   info.CreatedAt.Format("2006-01-02 15:04:05"),
		LastRefresh: info.LastRefresh.Format("2006-01-02 15:04:05"),
	}
}

// 辅助方法

func (s *AgentUsecase) getOrCreateSession(req *pb.ChatRequest) (*agent.Context, error) {
	sessionID := req.SessionId
	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}

	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists || time.Since(session.UpdatedAt) > 30*time.Minute {
		// 创建新会话
		mem := memory.NewMemory()

		// 如果有自定义系统提示词，添加到内存
		if req.SystemPrompt != "" {
			mem.AddMessage(core.Message{
				Role:    core.MessageRoleSystem,
				Content: req.SystemPrompt,
			})
		}

		ctx := agent.NewContext(
			agent.WithModel(req.Model),
			agent.WithChat(s.chat),
			agent.WithMemory(mem),
		)

		session = &SessionContext{
			Memory:    mem,
			Context:   ctx,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		s.sessions[sessionID] = session
	}

	session.UpdatedAt = time.Now()
	return session.Context, nil
}

func (s *AgentUsecase) createExecutor(ctx context.Context, req *pb.ChatRequest, agentCtx *agent.Context) (*agent.AgentExecutor, error) {
	// 根据 agent_id 从数据库加载 Agent 配置
	if req.AgentId == 0 {
		return nil, fmt.Errorf("agent_id is required")
	}

	if s.agentRepo == nil {
		return nil, fmt.Errorf("database is not configured")
	}

	agentConfig, err := s.agentRepo.GetAgent(ctx, int(req.AgentId))
	if err != nil {
		return nil, fmt.Errorf("failed to load agent config: %w", err)
	}

	fmt.Printf("📋 加载Agent配置: ID=%d, 名称=%s, 框架=%s\n", agentConfig.ID, agentConfig.Name, agentConfig.Framework)

	// 使用配置中的参数（如果请求中没有覆盖）
	maxSteps := int(req.MaxSteps)
	if maxSteps == 0 {
		maxSteps = agentConfig.MaxSteps
		if maxSteps == 0 {
			maxSteps = 10
		}
	}

	systemPrompt := req.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = agentConfig.SystemPrompt
	}

	// 如果有系统提示词，添加到内存
	if systemPrompt != "" {
		agentCtx.GetMemory().AddMessage(core.Message{
			Role:    core.MessageRoleSystem,
			Content: systemPrompt,
		})
	}

	var executor *agent.AgentExecutor

	// 根据配置的框架类型创建 Agent
	switch agentConfig.Framework {
	case "react":
		executor = agent.NewAgentExecutor(agentCtx)
		executor.SetMaxSteps(maxSteps)

	case "chain":
		// Chain需要特殊配置，这里提供一个默认链
		builder := agent.NewChainBuilder(agentCtx)
		builder.AddNode("main", agent.ReactAgentType, maxSteps)
		chainAgent := builder.Build()
		executor = agent.NewChainAgentExecutor(agentCtx, chainAgent)

	case "plan":
		enableReplan := req.Config["enable_replan"] == "true"
		executor = agent.NewPlanAgentExecutor(agentCtx, enableReplan)
		executor.SetMaxSteps(maxSteps)

	case "sql":
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
		tools.RegisterSQLTools(sqlConn)

		// 创建 SQL Agent
		dbInfo := fmt.Sprintf("MySQL: %s@%s:%d/%s", connConfig.Username, connConfig.Host, connConfig.Port, connConfig.Database)
		executor = agent.NewSQLAgentExecutor(agentCtx, dbInfo)
		executor.SetMaxSteps(maxSteps)

		fmt.Printf("✅ SQL Agent已创建，连接到: %s\n", dbInfo)

	case "elasticsearch":
		// 解析 ES 连接配置
		esConfig, err := s.parseESConnectionConfig(agentConfig.ConnectionConfig)
		if err != nil {
			return nil, fmt.Errorf("invalid ES connection config: %w", err)
		}

		// 创建 ES 连接
		esConn := tools.NewESConnection(esConfig.Host, esConfig.Username, esConfig.Password)

		// 注册 ES 工具
		tools.RegisterESTools(esConn)

		// 创建 ES Agent
		clusterInfo := fmt.Sprintf("Elasticsearch: %s", esConfig.Host)
		executor = agent.NewESAgentExecutor(agentCtx, clusterInfo)
		executor.SetMaxSteps(maxSteps)

		fmt.Printf("✅ ES Agent已创建，连接到: %s\n", esConfig.Host)

	default:
		// 默认使用 ReAct
		executor = agent.NewAgentExecutor(agentCtx)
		executor.SetMaxSteps(maxSteps)
		fmt.Printf("⚠️ 未知框架类型 '%s'，使用默认 ReAct Agent\n", agentConfig.Framework)
	}

	return executor, nil
}

func (s *AgentUsecase) getAgentTypeName(agentType pb.AgentType) string {
	switch agentType {
	case pb.AgentType_REACT:
		return "ReAct"
	case pb.AgentType_CHAIN:
		return "Chain"
	case pb.AgentType_PLAN:
		return "Plan"
	case pb.AgentType_SQL:
		return "SQL"
	default:
		return "Unknown"
	}
}

func (s *AgentUsecase) extractToolNames(mem core.Memory) []string {
	toolNames := make(map[string]bool)
	messages := mem.GetMessages()

	for _, msg := range messages {
		if msg.Role == core.MessageRoleUser && strings.Contains(msg.Content, "Observation:") {
			// 简单提取工具名称
			content := msg.Content
			if idx := strings.Index(content, "["); idx != -1 {
				if endIdx := strings.Index(content[idx:], "]"); endIdx != -1 {
					toolName := content[idx+1 : idx+endIdx]
					toolNames[toolName] = true
				}
			}
		}
	}

	result := make([]string, 0, len(toolNames))
	for name := range toolNames {
		result = append(result, name)
	}
	return result
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

type esConnectionConfig struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *AgentUsecase) parseESConnectionConfig(raw string) (*esConnectionConfig, error) {
	if raw == "" {
		return nil, fmt.Errorf("Elasticsearch 连接配置为空")
	}

	cfg := &esConnectionConfig{}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("解析 Elasticsearch 连接配置失败: %w", err)
	}

	if cfg.Host == "" {
		return nil, fmt.Errorf("Elasticsearch 连接配置缺少 host")
	}
	return cfg, nil
}

func (s *AgentUsecase) loadMCPServicesFromDB() {
	if s.mcpRepo == nil {
		return
	}

	services, err := s.mcpRepo.ListMCPServices(context.Background())
	if err != nil {
		fmt.Printf("⚠️ 从数据库加载MCP服务失败: %v\n", err)
		return
	}

	for _, service := range services {
		if !service.IsActive {
			continue
		}

		mcpManager, err := tools.NewMCPToolManager(service.Name, service.Endpoint)
		if err != nil {
			fmt.Printf("⚠️ 创建MCP工具管理器失败 [%s]: %v\n", service.Name, err)
			continue
		}

		mcpManager.Start()
		tools.GetToolManager().RegisterMCPToolManager(service.Name, mcpManager)

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

func (s *AgentUsecase) monitorExecution(mem core.Memory, msgChan chan<- core.Message, done <-chan bool) {
	lastCount := 1
	ticker := time.NewTicker(50 * time.Millisecond) // 缩短轮询间隔到50ms
	defer ticker.Stop()

	fmt.Println("🔍 开始监听执行过程...")

	for {
		select {
		case <-done:
			fmt.Println("⏹️ 执行完成信号，检查剩余消息...")
			// 在关闭前，再检查一次是否有新消息
			messages := mem.GetMessages()
			if len(messages) > lastCount {
				fmt.Printf("📬 发送剩余 %d 条消息\n", len(messages)-lastCount)
				for i := lastCount; i < len(messages); i++ {
					msgChan <- messages[i]
				}
			}
			// 等待一小段时间确保消息都被发送
			time.Sleep(50 * time.Millisecond)
			fmt.Println("✅ 消息监听结束")
			close(msgChan)
			return
		case <-ticker.C:
			messages := mem.GetMessages()
			if len(messages) > lastCount {
				// 发送新消息
				newCount := len(messages) - lastCount
				fmt.Printf("📬 检测到 %d 条新消息 (总计: %d)\n", newCount, len(messages))
				for i := lastCount; i < len(messages); i++ {
					msg := messages[i]
					fmt.Printf("  → [%s] %s\n", msg.Role, msg.Content[:min(60, len(msg.Content))])
					msgChan <- msg
				}
				lastCount = len(messages)
			}
		}
	}
}

func (s *AgentUsecase) parseMessage(msg core.Message) (pb.ChatStreamResponse_MessageType, string) {
	content := msg.Content

	switch msg.Role {
	case core.MessageRoleAssistant:
		if strings.Contains(content, "Thought:") {
			return pb.ChatStreamResponse_THINKING, content
		} else if strings.Contains(content, "Action:") {
			return pb.ChatStreamResponse_ACTION, content
		}
		return pb.ChatStreamResponse_THINKING, content

	case core.MessageRoleUser:
		if strings.Contains(content, "Observation:") {
			return pb.ChatStreamResponse_OBSERVATION, content
		}
		return pb.ChatStreamResponse_OBSERVATION, content

	default:
		return pb.ChatStreamResponse_METADATA, content
	}
}

// CleanupSessions 清理过期会话
func (s *AgentUsecase) CleanupSessions() {
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()

	now := time.Now()
	for id, session := range s.sessions {
		if now.Sub(session.UpdatedAt) > 30*time.Minute {
			delete(s.sessions, id)
		}
	}
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
