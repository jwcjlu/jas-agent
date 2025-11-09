package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-kratos/kratos/v2/log"
	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/gorilla/websocket"

	pb "jas-agent/api/agent/service/v1"
	"jas-agent/internal/biz"
	"jas-agent/internal/conf"
)

// NewHTTPServer 创建 Kratos HTTP 服务。
func NewHTTPServer(c *conf.Server, agentSvc pb.AgentServiceHTTPServer, uc *biz.AgentUsecase, logger log.Logger) *httptransport.Server {
	addr := ":0"
	if c != nil && c.HTTP != nil && c.HTTP.Addr != "" {
		addr = c.HTTP.Addr
	}

	opts := []httptransport.ServerOption{
		httptransport.Address(addr),
	}
	if logger != nil {
		opts = append(opts, httptransport.Logger(logger))
	}

	srv := httptransport.NewServer(opts...)
	pb.RegisterAgentServiceHTTPServer(srv, agentSvc)

	if uc != nil {
		registerChatWebsocket(srv, uc, logger)
	}

	return srv
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func registerChatWebsocket(srv *httptransport.Server, uc *biz.AgentUsecase, logger log.Logger) {
	helper := log.NewHelper(logger)

	srv.Handle("/api/chat/stream", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			helper.Errorf("WebSocket upgrade failed: %v", err)
			http.Error(w, "websocket upgrade failed", http.StatusBadRequest)
			return
		}
		defer conn.Close()

		var req pb.ChatRequest
		if err = conn.ReadJSON(&req); err != nil {
			helper.Errorf("WebSocket read request failed: %v", err)
			_ = conn.WriteJSON(&pb.ChatStreamResponse{
				Type:    pb.ChatStreamResponse_ERROR,
				Content: "invalid request body",
			})
			return
		}

		conn.SetCloseHandler(func(code int, text string) error {
			/*cancel()*/
			return nil
		})

		if err = uc.StreamChatWithSender(context.TODO(), &req, func(resp *pb.ChatStreamResponse) error {
			return conn.WriteJSON(resp)
		}); err != nil && !errors.Is(err, context.Canceled) {
			fmt.Println("报错", err)
			helper.Errorf("WebSocket stream chat failed: %v", err)
			_ = conn.WriteJSON(&pb.ChatStreamResponse{
				Type:    pb.ChatStreamResponse_ERROR,
				Content: err.Error(),
			})
		}
	}))
}
