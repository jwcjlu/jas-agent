package server

import (
	"github.com/go-kratos/aegis/ratelimit/bbr"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	middleware "jas-agent/internal/server/ middleware"
	"jas-agent/internal/service"
	"net/http"

	"github.com/go-kratos/kratos/v2/log"
	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	pb "jas-agent/api/agent/service/v1"
	"jas-agent/internal/conf"
)

// NewHTTPServer 创建 Kratos HTTP 服务。
func NewHTTPServer(c *conf.Server, agentSvc *service.AgentService, logger log.Logger) *httptransport.Server {
	addr := ":0"
	if c != nil && c.HTTP != nil && c.HTTP.Addr != "" {
		addr = c.HTTP.Addr
	}

	opts := []httptransport.ServerOption{
		httptransport.Address(addr),
	}
	opts = append(opts, httptransport.Middleware(
		recovery.Recovery(),
		tracing.Server(
			tracing.WithTracerProvider(otel.GetTracerProvider()),
			tracing.WithPropagator(
				propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{}),
			),
		),
		logging.Server(logger),
		middleware.Validator(),
		middleware.Metric(),
		middleware.TraceparentMiddleware(),
		middleware.MetaData(),
		ratelimit.Server(ratelimit.WithLimiter(bbr.NewLimiter())),
	))

	if logger != nil {
		opts = append(opts, httptransport.Logger(logger))
	}

	srv := httptransport.NewServer(opts...)
	pb.RegisterAgentServiceHTTPServer(srv, agentSvc)
	srv.Handle("/api/chat/stream", http.HandlerFunc(agentSvc.WebSocket))
	return srv
}
