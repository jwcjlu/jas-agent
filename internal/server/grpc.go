package server

import (
	"github.com/go-kratos/kratos/v2/log"
	grpctransport "github.com/go-kratos/kratos/v2/transport/grpc"
	"jas-agent/internal/service"

	pb "jas-agent/api/agent/service/v1"
	"jas-agent/internal/conf"
)

// NewGRPCServer 创建 Kratos gRPC 服务。
func NewGRPCServer(c *conf.Server, agentSvc *service.AgentService, logger log.Logger) *grpctransport.Server {
	addr := ":0"
	if c != nil && c.GRPC != nil && c.GRPC.Addr != "" {
		addr = c.GRPC.Addr
	}

	opts := []grpctransport.ServerOption{
		grpctransport.Address(addr),
	}
	if logger != nil {
		opts = append(opts, grpctransport.Logger(logger))
	}

	srv := grpctransport.NewServer(opts...)
	pb.RegisterAgentServiceServer(srv, agentSvc)
	return srv
}
