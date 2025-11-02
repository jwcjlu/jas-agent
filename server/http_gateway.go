package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	pb "jas-agent/api/proto"
	"jas-agent/core"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// HTTPGateway HTTP网关，将HTTP请求转换为gRPC调用
type HTTPGateway struct {
	grpcServer *AgentServer
	upgrader   websocket.Upgrader
}

// NewHTTPGateway 创建新的HTTP网关
func NewHTTPGateway(grpcServer *AgentServer) *HTTPGateway {
	return &HTTPGateway{
		grpcServer: grpcServer,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源（生产环境应该限制）
			},
		},
	}
}

// SetupRoutes 设置HTTP路由
func (gw *HTTPGateway) SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// API路由
	api := router.PathPrefix("/api").Subrouter()

	// CORS中间件
	router.Use(gw.corsMiddleware)

	// 对话接口
	api.HandleFunc("/chat", gw.handleChat).Methods("POST", "OPTIONS")

	// WebSocket流式对话
	api.HandleFunc("/chat/stream", gw.handleStreamChat)

	// 获取Agent类型
	api.HandleFunc("/agents", gw.handleListAgents).Methods("GET", "OPTIONS")

	// 获取工具列表
	api.HandleFunc("/tools", gw.handleListTools).Methods("GET", "OPTIONS")

	// 静态文件服务（前端）
	// 生产环境使用构建后的文件，开发环境可以直接服务 web 目录
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/dist")))

	return router
}

// handleChat 处理单次对话请求
func (gw *HTTPGateway) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req ChatRequestHTTP
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		gw.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 转换为gRPC请求
	grpcReq := &pb.ChatRequest{
		Query:        req.Query,
		AgentType:    gw.parseAgentType(req.AgentType),
		Model:        req.Model,
		SystemPrompt: req.SystemPrompt,
		MaxSteps:     int32(req.MaxSteps),
		Config:       req.Config,
		SessionId:    req.SessionID,
	}

	// 调用gRPC服务
	resp, err := gw.grpcServer.Chat(r.Context(), grpcReq)
	if err != nil {
		gw.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 转换响应
	httpResp := ChatResponseHTTP{
		Response:  resp.Response,
		AgentType: resp.AgentType,
		Success:   resp.Success,
		Error:     resp.Error,
	}

	if resp.Metadata != nil {
		httpResp.Metadata = &ExecutionMetadataHTTP{
			TotalSteps:      int(resp.Metadata.TotalSteps),
			ToolsCalled:     int(resp.Metadata.ToolsCalled),
			ToolNames:       resp.Metadata.ToolNames,
			ExecutionTimeMs: resp.Metadata.ExecutionTimeMs,
			State:           resp.Metadata.State,
		}
	}

	gw.sendJSON(w, httpResp)
}

// handleStreamChat 处理流式对话（WebSocket）
func (gw *HTTPGateway) handleStreamChat(w http.ResponseWriter, r *http.Request) {
	// 升级到WebSocket
	conn, err := gw.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer conn.Close()

	// 读取请求
	var req ChatRequestHTTP
	if err := conn.ReadJSON(&req); err != nil {
		fmt.Printf("❌ WebSocket读取请求失败: %v\n", err)
		conn.WriteJSON(map[string]interface{}{
			"type":  "error",
			"error": err.Error(),
		})
		return
	}

	fmt.Printf("📨 收到WebSocket请求: query=%s, agent=%s, stream=true\n", req.Query, req.AgentType)

	// 直接在这里实现流式逻辑，不通过 gRPC
	startTime := time.Now()

	// 转换为gRPC请求格式
	grpcReq := &pb.ChatRequest{
		Query:        req.Query,
		AgentType:    gw.parseAgentType(req.AgentType),
		Model:        req.Model,
		SystemPrompt: req.SystemPrompt,
		MaxSteps:     int32(req.MaxSteps),
		Config:       req.Config,
		SessionId:    req.SessionID,
	}

	// 创建或获取会话
	agentCtx, err := gw.grpcServer.getOrCreateSession(grpcReq)
	if err != nil {
		conn.WriteJSON(map[string]interface{}{
			"type":  "error",
			"error": err.Error(),
		})
		return
	}

	// 创建执行器
	executor, err := gw.grpcServer.createExecutor(grpcReq, agentCtx)
	if err != nil {
		conn.WriteJSON(map[string]interface{}{
			"type":  "error",
			"error": err.Error(),
		})
		return
	}

	// 创建消息监听通道
	messageChan := make(chan core.Message, 100)
	resultChan := make(chan string, 1)
	done := make(chan bool, 1)

	// 启动消息监听器
	go gw.grpcServer.monitorExecution(agentCtx.GetMemory(), messageChan, done)

	// 在新的goroutine中执行
	go func() {
		defer close(done)
		result := executor.Run(grpcReq.Query)

		// 等待一小段时间确保所有消息都被监听到
		time.Sleep(200 * time.Millisecond)
		resultChan <- result
	}()

	// 流式发送执行过程
	step := 0
	var finalResult string
	executing := true

	for executing {
		select {
		case msg, ok := <-messageChan:
			if !ok {
				// 消息通道关闭，准备发送最终结果
				executing = false
				break
			}

			step++
			msgType, content := gw.grpcServer.parseMessage(msg)
			typeStr := gw.getMessageTypeString(msgType)

			fmt.Printf("📤 发送消息 [步骤%d, 类型:%s]: %s\n", step, typeStr, content[:min(50, len(content))])

			// 发送消息到前端
			if err := conn.WriteJSON(map[string]interface{}{
				"type":    typeStr,
				"content": content,
				"step":    step,
			}); err != nil {
				fmt.Printf("❌ 发送消息失败: %v\n", err)
				return
			}

		case result := <-resultChan:
			// 收到最终结果
			finalResult = result

			// 继续等待剩余消息
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 等待最终结果（如果还没收到）
	if finalResult == "" {
		finalResult = <-resultChan
	}

	// 发送最终结果
	toolNames := gw.grpcServer.extractToolNames(agentCtx.GetMemory())

	fmt.Printf("✅ 发送最终结果: %s (总步骤: %d, 工具: %v)\n",
		finalResult[:min(50, len(finalResult))], executor.GetCurrentStep(), toolNames)

	conn.WriteJSON(map[string]interface{}{
		"type":    "final",
		"content": finalResult,
		"metadata": map[string]interface{}{
			"total_steps":       executor.GetCurrentStep(),
			"tools_called":      len(toolNames),
			"tool_names":        toolNames,
			"execution_time_ms": time.Since(startTime).Milliseconds(),
			"state":             string(executor.GetState()),
		},
	})
}

// handleListAgents 获取Agent类型列表
func (gw *HTTPGateway) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	resp, err := gw.grpcServer.ListAgentTypes(r.Context(), &pb.Empty{})
	if err != nil {
		gw.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	types := make([]map[string]interface{}, len(resp.Types))
	for i, t := range resp.Types {
		types[i] = map[string]interface{}{
			"type":        gw.getAgentTypeString(t.Type),
			"name":        t.Name,
			"description": t.Description,
			"available":   t.Available,
		}
	}

	gw.sendJSON(w, map[string]interface{}{
		"agents": types,
	})
}

// handleListTools 获取工具列表
func (gw *HTTPGateway) handleListTools(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	resp, err := gw.grpcServer.ListTools(r.Context(), &pb.Empty{})
	if err != nil {
		gw.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tools := make([]map[string]interface{}, len(resp.Tools))
	for i, t := range resp.Tools {
		tools[i] = map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"type":        t.Type,
		}
	}

	gw.sendJSON(w, map[string]interface{}{
		"tools": tools,
	})
}

// 辅助方法

func (gw *HTTPGateway) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (gw *HTTPGateway) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (gw *HTTPGateway) sendError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func (gw *HTTPGateway) parseAgentType(typeStr string) pb.AgentType {
	switch typeStr {
	case "react", "REACT":
		return pb.AgentType_REACT
	case "chain", "CHAIN":
		return pb.AgentType_CHAIN
	case "plan", "PLAN":
		return pb.AgentType_PLAN
	case "sql", "SQL":
		return pb.AgentType_SQL
	default:
		return pb.AgentType_REACT
	}
}

func (gw *HTTPGateway) getAgentTypeString(t pb.AgentType) string {
	switch t {
	case pb.AgentType_REACT:
		return "react"
	case pb.AgentType_CHAIN:
		return "chain"
	case pb.AgentType_PLAN:
		return "plan"
	case pb.AgentType_SQL:
		return "sql"
	default:
		return "unknown"
	}
}

// HTTP 请求/响应类型

type ChatRequestHTTP struct {
	Query        string            `json:"query"`
	AgentType    string            `json:"agent_type"`
	Model        string            `json:"model"`
	SystemPrompt string            `json:"system_prompt,omitempty"`
	MaxSteps     int               `json:"max_steps,omitempty"`
	Config       map[string]string `json:"config,omitempty"`
	SessionID    string            `json:"session_id,omitempty"`
}

type ChatResponseHTTP struct {
	Response  string                 `json:"response"`
	AgentType string                 `json:"agent_type"`
	Metadata  *ExecutionMetadataHTTP `json:"metadata,omitempty"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
}

type ExecutionMetadataHTTP struct {
	TotalSteps      int      `json:"total_steps"`
	ToolsCalled     int      `json:"tools_called"`
	ToolNames       []string `json:"tool_names"`
	ExecutionTimeMs int64    `json:"execution_time_ms"`
	State           string   `json:"state"`
}

// getMessageTypeString 将 gRPC 消息类型转换为字符串
func (gw *HTTPGateway) getMessageTypeString(t pb.ChatStreamResponse_MessageType) string {
	switch t {
	case pb.ChatStreamResponse_THINKING:
		return "thinking"
	case pb.ChatStreamResponse_ACTION:
		return "action"
	case pb.ChatStreamResponse_OBSERVATION:
		return "observation"
	case pb.ChatStreamResponse_FINAL:
		return "final"
	case pb.ChatStreamResponse_ERROR:
		return "error"
	case pb.ChatStreamResponse_METADATA:
		return "metadata"
	default:
		return "unknown"
	}
}

// StartHTTPServer 启动HTTP服务器
func StartHTTPServer(addr string, grpcServer *AgentServer) error {
	gateway := NewHTTPGateway(grpcServer)
	router := gateway.SetupRoutes()

	// 启动会话清理定时器
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			grpcServer.CleanupSessions()
		}
	}()

	fmt.Printf("🌐 HTTP服务器启动在 %s\n", addr)
	fmt.Printf("📡 API端点: http://%s/api\n", addr)
	fmt.Printf("🌍 前端界面: http://%s\n", addr)

	return http.ListenAndServe(addr, router)
}
