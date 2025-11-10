package server

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	http2 "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/wire"
)

// ProviderSet server provider.
var ProviderSet = wire.NewSet(NewHTTPServer, NewGRPCServer, NewApp)

// NewApp 构造 Kratos 应用。
func NewApp(logger log.Logger, gs *grpc.Server, hs *http2.Server) *kratos.App {
	var opts []kratos.Option
	if logger != nil {
		opts = append(opts, kratos.Logger(logger))
	}
	if gs != nil {
		opts = append(opts, kratos.Server(gs))
	}
	if hs != nil {
		opts = append(opts, kratos.Server(hs))
	}
	return kratos.New(opts...)
}
