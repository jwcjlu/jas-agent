package core

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

type Tool interface {
	Name() string
	Description() string
	Handler(ctx context.Context, input string) (string, error)
	Input() any
	Type() ToolType
}

type ToolType int

const (
	Normal ToolType = 1
	Mcp    ToolType = 2
)

type FilterFunc func(tool Tool) bool

type DataHandler func(ctx context.Context, data string) (string, error)

type DataHandlerFilter func(DataHandler) DataHandler

func DataHandlerChain(handlers ...DataHandlerFilter) DataHandlerFilter {
	return func(next DataHandler) DataHandler {
		for i := len(handlers) - 1; i >= 0; i-- {
			next = handlers[i](next)
		}
		return next
	}
}

// LoggingDataHandlerFilter 一个简单的日志过滤器，用于在调用前后打印输入和输出。
// 自动注入TraceID和SpanID到日志中
func LoggingDataHandlerFilter(logger log.Logger) DataHandlerFilter {
	return func(next DataHandler) DataHandler {
		return func(ctx context.Context, data string) (string, error) {
			LogInfo(ctx, logger, "DataHandler start", "input", data)
			out, err := next(ctx, data)
			if err != nil {
				LogError(ctx, logger, err, "DataHandler error")
			} else {
				LogInfo(ctx, logger, "DataHandler done", "output", out)
			}
			return out, err
		}
	}
}
