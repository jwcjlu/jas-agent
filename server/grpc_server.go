package server

import (
	"context"
	"fmt"
	"jas-agent/agent"
	"jas-agent/core"
	"jas-agent/llm"
	"jas-agent/memory"
	"jas-agent/tools"
	"strings"
	"sync"
	"time"

	pb "jas-agent/api/proto"
)

// AgentServer 实现 AgentService gRPC 服务
type AgentServer struct {
	pb.UnimplementedAgentServiceServer
	chat         llm.Chat
	sessions     map[string]*SessionContext
	sessionsLock sync.RWMutex
}

// SessionContext 会话上下文
type SessionContext struct {
	Memory    core.Memory
	Context   *agent.Context
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewAgentServer 创建新的 Agent 服务
func NewAgentServer(chat llm.Chat) *AgentServer {
	return &AgentServer{
		chat:     chat,
		sessions: make(map[string]*SessionContext),
	}
}

// Chat 实现单次对话
func (s *AgentServer) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
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
	executor, err := s.createExecutor(req, agentCtx)
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
func (s *AgentServer) StreamChat(req *pb.ChatRequest, stream pb.AgentService_StreamChatServer) error {
	startTime := time.Now()
	// 在新的goroutine中执行
	resultChan := make(chan string)

	// 创建或获取会话上下文
	agentCtx, err := s.getOrCreateSession(req)
	if err != nil {
		return stream.Send(&pb.ChatStreamResponse{
			Type:    pb.ChatStreamResponse_ERROR,
			Content: err.Error(),
		})
	}

	// 创建执行器
	executor, err := s.createExecutor(req, agentCtx)
	if err != nil {
		return stream.Send(&pb.ChatStreamResponse{
			Type:    pb.ChatStreamResponse_ERROR,
			Content: err.Error(),
		})
	}

	// 创建消息监听通道
	messageChan := make(chan core.Message, 10)
	done := make(chan bool)

	// 启动消息监听器
	go s.monitorExecution(agentCtx.GetMemory(), messageChan, done)

	go func() {
		result := executor.Run(req.Query)
		resultChan <- result
		close(done)
	}()

	// 流式发送执行过程
	step := 0
	for {
		select {
		case msg, ok := <-messageChan:
			if !ok {
				goto SEND_FINAL
			}

			step++
			msgType, content := s.parseMessage(msg)

			if err := stream.Send(&pb.ChatStreamResponse{
				Type:    msgType,
				Content: content,
				Step:    int32(step),
			}); err != nil {
				return err
			}

		case result := <-resultChan:
			// 发送最终结果
			metadata := &pb.ExecutionMetadata{
				TotalSteps:      int32(executor.GetCurrentStep()),
				ExecutionTimeMs: time.Since(startTime).Milliseconds(),
				State:           string(executor.GetState()),
			}

			toolNames := s.extractToolNames(agentCtx.GetMemory())
			metadata.ToolNames = toolNames
			metadata.ToolsCalled = int32(len(toolNames))

			return stream.Send(&pb.ChatStreamResponse{
				Type:     pb.ChatStreamResponse_FINAL,
				Content:  result,
				Metadata: metadata,
			})
		}
	}

SEND_FINAL:
	// 等待最终结果
	result := <-resultChan
	metadata := &pb.ExecutionMetadata{
		TotalSteps:      int32(executor.GetCurrentStep()),
		ExecutionTimeMs: time.Since(startTime).Milliseconds(),
		State:           string(executor.GetState()),
	}

	return stream.Send(&pb.ChatStreamResponse{
		Type:     pb.ChatStreamResponse_FINAL,
		Content:  result,
		Metadata: metadata,
	})
}

// ListAgentTypes 列出可用的 Agent 类型
func (s *AgentServer) ListAgentTypes(ctx context.Context, req *pb.Empty) (*pb.AgentTypesResponse, error) {
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
func (s *AgentServer) ListTools(ctx context.Context, req *pb.Empty) (*pb.ToolsResponse, error) {
	toolManager := tools.GetToolManager()
	availableTools := toolManager.AvailableTools()

	toolInfos := make([]*pb.ToolInfo, 0, len(availableTools))
	for _, tool := range availableTools {
		toolType := "Normal"
		if tool.Type() == core.Mcp {
			toolType = "MCP"
		}

		toolInfos = append(toolInfos, &pb.ToolInfo{
			Name:        tool.Name(),
			Description: tool.Description(),
			Type:        toolType,
		})
	}

	return &pb.ToolsResponse{
		Tools: toolInfos,
	}, nil
}

// 辅助方法

func (s *AgentServer) getOrCreateSession(req *pb.ChatRequest) (*agent.Context, error) {
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

func (s *AgentServer) createExecutor(req *pb.ChatRequest, ctx *agent.Context) (*agent.AgentExecutor, error) {
	maxSteps := int(req.MaxSteps)
	if maxSteps == 0 {
		maxSteps = 10
	}

	var executor *agent.AgentExecutor

	switch req.AgentType {
	case pb.AgentType_REACT:
		executor = agent.NewAgentExecutor(ctx)
		executor.SetMaxSteps(maxSteps)

	case pb.AgentType_CHAIN:
		// Chain需要特殊配置，这里提供一个默认链
		builder := agent.NewChainBuilder(ctx)
		builder.
			AddNode("main", agent.ReactAgentType, maxSteps)
		chainAgent := builder.Build()
		executor = agent.NewChainAgentExecutor(ctx, chainAgent)

	case pb.AgentType_PLAN:
		enableReplan := req.Config["enable_replan"] == "true"
		executor = agent.NewPlanAgentExecutor(ctx, enableReplan)
		executor.SetMaxSteps(maxSteps)

	case pb.AgentType_SQL:
		dbInfo := req.Config["db_info"]
		if dbInfo == "" {
			dbInfo = "Database"
		}
		executor = agent.NewSQLAgentExecutor(ctx, dbInfo)
		executor.SetMaxSteps(maxSteps)

	default:
		return nil, fmt.Errorf("unsupported agent type: %v", req.AgentType)
	}

	return executor, nil
}

func (s *AgentServer) getAgentTypeName(agentType pb.AgentType) string {
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

func (s *AgentServer) extractToolNames(mem core.Memory) []string {
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

func (s *AgentServer) monitorExecution(mem core.Memory, msgChan chan<- core.Message, done <-chan bool) {
	lastCount := 0
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

func (s *AgentServer) parseMessage(msg core.Message) (pb.ChatStreamResponse_MessageType, string) {
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
func (s *AgentServer) CleanupSessions() {
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
