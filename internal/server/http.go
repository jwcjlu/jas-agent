package server

import (
	"jas-agent/internal/server/middleware"
	"jas-agent/internal/service"
	"net/http"
	"reflect"

	"github.com/go-kratos/aegis/ratelimit/bbr"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	v1 "jas-agent/api/agent/service/v1"
	"jas-agent/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	httptransport "github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer 创建 Kratos HTTP 服务。
func NewHTTPServer(c *conf.Server, agentSvc *service.AgentService, logger log.Logger) *httptransport.Server {
	addr := ":0"
	if c != nil && c.HTTP != nil && c.HTTP.Addr != "" {
		addr = c.HTTP.Addr
	}

	opts := []httptransport.ServerOption{
		httptransport.Address(addr),
		httptransport.ResponseEncoder(responseEncoder),
		httptransport.ErrorEncoder(errorEncoder),
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
		middleware.ServerMetrics(),
		middleware.Validator(),
		middleware.TraceparentMiddleware(),
		middleware.MetaData(),
		ratelimit.Server(ratelimit.WithLimiter(bbr.NewLimiter())),
	))

	srv := httptransport.NewServer(opts...)
	v1.RegisterAgentServiceHTTPServer(srv, agentSvc)
	srv.Handle("/api/chat/stream", http.HandlerFunc(agentSvc.WebSocket))
	return srv
}

type BizResp interface {
	GetRet() *v1.BaseResponse
}

func responseEncoder(w http.ResponseWriter, r *http.Request, v interface{}) error {
	if v == nil {
		return nil
	}
	if rd, ok := v.(httptransport.Redirector); ok {
		url, code := rd.Redirect()
		http.Redirect(w, r, url, code)
		return nil
	}
	bizResp, ok := v.(BizResp)
	if ok && bizResp != nil && bizResp.GetRet() == nil {
		val := reflect.ValueOf(v).Elem()
		ret := val.FieldByName("Ret")
		if ret.CanSet() {
			ret.Set(reflect.ValueOf(&v1.BaseResponse{Code: 0, Message: "", Reason: ""}))
		}
	}
	codec, _ := httptransport.CodecForRequest(r, "Accept")
	data, err := codec.Marshal(v)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	return err
}

func errorEncoder(w http.ResponseWriter, r *http.Request, err error) {
	se := errors.FromError(err)
	codec, _ := httptransport.CodecForRequest(r, "Accept")
	bizCode := v1.ErrorReason_value[se.Reason]
	if bizCode == 0 {
		bizCode = se.GetCode()
	}
	rsp := v1.Response{
		Ret: &v1.BaseResponse{
			Code:    bizCode,
			Reason:  se.Reason,
			Message: se.Message,
		},
	}
	body, err := codec.Marshal(&rsp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(se.Code))
	_, _ = w.Write(body)
}
