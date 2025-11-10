package biz

import (
	"context"
	"fmt"
	"strings"
	"time"

	agent "jas-agent/agent/agent"
	"jas-agent/agent/core"
	"jas-agent/agent/llm"
	"jas-agent/agent/memory"
	tools "jas-agent/agent/tools"
	pb "jas-agent/api/agent/service/v1"

	"github.com/go-kratos/kratos/v2/log"
	_ "github.com/go-sql-driver/mysql"
)

// AgentUsecase 负责 Agent 相关业务逻辑
type AgentUsecase struct {
	chat      llm.Chat
	agentRepo AgentRepo
	logger    *log.Helper
	factory   *AgentFactory
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

// NewAgentUsecase 创建新的 AgentUsecase。
func NewAgentUsecase(chat llm.Chat, agentRepo AgentRepo, factory *AgentFactory, logger log.Logger) *AgentUsecase {
	uc := &AgentUsecase{
		chat:      chat,
		agentRepo: agentRepo,
		logger:    log.NewHelper(log.With(logger, "module", "biz/agent")),
		factory:   factory,
	}
	return uc
}

// Chat 实现单次对话
func (s *AgentUsecase) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	return nil, nil
}

// StreamChat 实现流式对话
func (s *AgentUsecase) StreamChat(req *pb.ChatRequest, stream pb.AgentService_StreamChatServer) error {
	return s.StreamChatWithSender(stream.Context(), req, stream.Send)
}

// StreamChatWithSender 使用自定义发送函数实现流式对话，可用于 WebSocket 等场景。
func (s *AgentUsecase) StreamChatWithSender(ctx context.Context, req *pb.ChatRequest, send func(*pb.ChatStreamResponse) error) error {
	startTime := time.Now()
	resultChan := make(chan string, 1)
	messageChan := make(chan core.Message, 10)
	executor, err := s.createExecutor(ctx, req, func(c context.Context, msg core.Message) error {
		messageChan <- msg
		return nil
	})
	if err != nil {
		return send(&pb.ChatStreamResponse{
			Type:    pb.ChatStreamResponse_ERROR,
			Content: err.Error(),
		})
	}

	go func() {
		defer close(resultChan)
		resultChan <- executor.Run(req.Query)
	}()

	buildMetadata := func() *pb.ExecutionMetadata {
		metadata := &pb.ExecutionMetadata{
			TotalSteps:      int32(executor.GetCurrentStep()),
			ExecutionTimeMs: time.Since(startTime).Milliseconds(),
			State:           string(executor.GetState()),
		}

		toolNames := s.extractToolNames(executor.GetMemory())
		metadata.ToolNames = toolNames
		metadata.ToolsCalled = int32(len(toolNames))

		return metadata
	}

	sendFinal := func(result string) error {
		return send(&pb.ChatStreamResponse{
			Type:     pb.ChatStreamResponse_FINAL,
			Content:  result,
			Metadata: buildMetadata(),
		})
	}

	step := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case msg, ok := <-messageChan:
			if !ok {
				if result, ok := <-resultChan; ok {
					return sendFinal(result)
				}
				return nil
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

		case result, ok := <-resultChan:
			if !ok {
				return nil
			}
			return sendFinal(result)
		}
	}
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

func (s *AgentUsecase) CreateAgent(ctx context.Context, req *pb.AgentConfigRequest) error {
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
	return s.agentRepo.CreateAgent(ctx, agentConfig)
}

func (s *AgentUsecase) UpdateAgent(ctx context.Context, req *pb.AgentConfigRequest) error {
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

	return s.agentRepo.UpdateAgent(ctx, agentConfig)
}

func (s *AgentUsecase) DeleteAgent(ctx context.Context, req *pb.AgentDeleteRequest) error {
	return s.agentRepo.DeleteAgent(ctx, int(req.Id))
}

func (s *AgentUsecase) GetAgent(ctx context.Context, req *pb.AgentGetRequest) (*Agent, error) {
	return s.agentRepo.GetAgent(ctx, int(req.Id))

}
func (s *AgentUsecase) ListAgents(ctx context.Context, req *pb.Empty) ([]*Agent, error) {
	return s.agentRepo.ListAgents(ctx)
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

func (s *AgentUsecase) createExecutor(ctx context.Context,
	req *pb.ChatRequest,
	send func(c context.Context, msg core.Message) error) (*agent.AgentExecutor, error) {

	agentConfig, err := s.agentRepo.GetAgent(ctx, int(req.AgentId))
	if err != nil {
		return nil, fmt.Errorf("failed to load agent config: %w", err)
	}
	s.logger.Infof("Loaded agent config: id=%d name=%s framework=%s", agentConfig.ID, agentConfig.Name, agentConfig.Framework)
	tm := tools.NewToolManager()
	tm.Inherit(tools.GetToolManager())
	for _, server := range agentConfig.MCPServers {
		mcpManager, err := tools.NewMCPToolManager(server.Name, server.Endpoint, tm)
		if err != nil {
			return nil, err
		}
		mcpManager.DiscoverAndRegisterTools()
	}
	mem := memory.NewMemory()
	agentCtx := agent.NewContext(agent.WithModel(req.Model),
		agent.WithChat(s.chat),
		agent.WithMemory(mem),
		agent.WithToolManager(tm),
		agent.WithSend(send),
	)
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
	executor, err := s.factory.CreateAgentExecutor(ctx, agentConfig, agentCtx)
	if err != nil {
		return nil, err
	}
	return executor, nil
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
