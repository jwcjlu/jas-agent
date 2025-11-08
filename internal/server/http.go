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

	// 提供静态文件（前端构建产物）
	/*if root := detectWebDist(); root != "" {
		srv.HandlePrefix("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api") {
				http.NotFound(w, r)
				return
			}

			path := filepath.Join(root, filepath.Clean(r.URL.Path))
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				http.ServeFile(w, r, path)
				return
			}

			http.ServeFile(w, r, filepath.Join(root, "index.html"))
		}))
	} else if logger != nil {
		log.NewHelper(logger).Warn("web/dist 目录不存在，静态资源将不被提供")
	}
	*/
	return srv
}

/*
	func detectWebDist() string {
		candidates := []string{
			"web/dist",
			"./web/dist",
		}
		for _, dir := range candidates {
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				return dir
			}
		}
		return ""
	}
*/

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

		/*	ctx, cancel := context.WithCancel(context.Background())
			defer cancel()*/

		// 监听原始请求上下文，若被取消则关闭流式处理
		/*	go func() {
			select {
			case <-r.Context().Done():
				cancel()
			case <-ctx.Done():
			}
		}()*/

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
