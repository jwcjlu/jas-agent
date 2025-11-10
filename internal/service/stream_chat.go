package service

import (
	"context"
	"errors"
	"github.com/gorilla/websocket"
	pb "jas-agent/api/agent/service/v1"
	"net/http"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *AgentService) WebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "websocket upgrade failed", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	var req pb.ChatRequest
	if err = conn.ReadJSON(&req); err != nil {
		_ = conn.WriteJSON(&pb.ChatStreamResponse{
			Type:    pb.ChatStreamResponse_ERROR,
			Content: "invalid request body",
		})
		return
	}
	// 根据 agent_id 从数据库加载 Agent 配置
	if req.AgentId == 0 {
		_ = conn.WriteJSON(&pb.ChatStreamResponse{
			Type:    pb.ChatStreamResponse_ERROR,
			Content: "agent_id is required",
		})
		return
	}
	if err = s.delegate.StreamChatWithSender(context.TODO(), &req, func(resp *pb.ChatStreamResponse) error {
		return conn.WriteJSON(resp)
	}); err != nil && !errors.Is(err, context.Canceled) {
		_ = conn.WriteJSON(&pb.ChatStreamResponse{
			Type:    pb.ChatStreamResponse_ERROR,
			Content: err.Error(),
		})
	}
}
