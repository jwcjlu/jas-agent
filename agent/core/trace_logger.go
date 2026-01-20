package core

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel/trace"
)

// TraceLogger 带追踪信息的日志包装器
type TraceLogger struct {
	logger log.Logger
}

// NewTraceLogger 创建带追踪信息的日志包装器
func NewTraceLogger(logger log.Logger) *TraceLogger {
	return &TraceLogger{logger: logger}
}

// WithContext 从context中提取TraceID和SpanID，添加到日志字段中
func (tl *TraceLogger) WithContext(ctx context.Context) log.Logger {
	if ctx == nil {
		return tl.logger
	}

	span := trace.SpanFromContext(ctx)
	if span == nil {
		return tl.logger
	}

	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return tl.logger
	}

	// 提取TraceID和SpanID
	traceID := spanCtx.TraceID().String()
	spanID := spanCtx.SpanID().String()

	// 将TraceID和SpanID添加到日志上下文中
	return log.With(tl.logger,
		"trace_id", traceID,
		"span_id", spanID,
	)
}

// TraceHelper 带追踪信息的日志Helper包装器
type TraceHelper struct {
	logger     log.Logger
	baseHelper *log.Helper
}

// NewTraceHelper 创建带追踪信息的日志Helper
func NewTraceHelper(logger log.Logger) *TraceHelper {
	return &TraceHelper{
		logger:     logger,
		baseHelper: log.NewHelper(logger),
	}
}

// WithContext 返回带追踪信息的Helper
func (th *TraceHelper) WithContext(ctx context.Context) *log.Helper {
	traceLogger := NewTraceLogger(th.logger)
	loggerWithTrace := traceLogger.WithContext(ctx)
	return log.NewHelper(loggerWithTrace)
}

// Helper 返回基础Helper（不带context，用于无context的场景）
func (th *TraceHelper) Helper() *log.Helper {
	return th.baseHelper
}

// Info 不带context的Info日志
func (th *TraceHelper) Info(args ...interface{}) {
	th.baseHelper.Info(args...)
}

// Infof 不带context的Infof日志
func (th *TraceHelper) Infof(format string, args ...interface{}) {
	th.baseHelper.Infof(format, args...)
}

// Debug 不带context的Debug日志
func (th *TraceHelper) Debug(args ...interface{}) {
	th.baseHelper.Debug(args...)
}

// Debugf 不带context的Debugf日志
func (th *TraceHelper) Debugf(format string, args ...interface{}) {
	th.baseHelper.Debugf(format, args...)
}

// Warn 不带context的Warn日志
func (th *TraceHelper) Warn(args ...interface{}) {
	th.baseHelper.Warn(args...)
}

// Warnf 不带context的Warnf日志
func (th *TraceHelper) Warnf(format string, args ...interface{}) {
	th.baseHelper.Warnf(format, args...)
}

// Error 不带context的Error日志
func (th *TraceHelper) Error(args ...interface{}) {
	th.baseHelper.Error(args...)
}

// Errorf 不带context的Errorf日志
func (th *TraceHelper) Errorf(format string, args ...interface{}) {
	th.baseHelper.Errorf(format, args...)
}

// Log 从context中提取追踪信息并记录日志
func Log(ctx context.Context, logger log.Logger, level log.Level, keyvals ...interface{}) error {
	traceLogger := NewTraceLogger(logger)
	loggerWithTrace := traceLogger.WithContext(ctx)
	return loggerWithTrace.Log(level, keyvals...)
}

// LogInfo 记录Info级别日志（带追踪信息）
func LogInfo(ctx context.Context, logger log.Logger, msg string, keyvals ...interface{}) {
	traceLogger := NewTraceLogger(logger)
	loggerWithTrace := traceLogger.WithContext(ctx)
	helper := log.NewHelper(loggerWithTrace)
	args := append([]interface{}{"msg", msg}, keyvals...)
	helper.Info(args...)
}

// LogError 记录Error级别日志（带追踪信息）
func LogError(ctx context.Context, logger log.Logger, err error, msg string, keyvals ...interface{}) {
	traceLogger := NewTraceLogger(logger)
	loggerWithTrace := traceLogger.WithContext(ctx)
	helper := log.NewHelper(loggerWithTrace)
	args := append([]interface{}{"msg", msg, "error", err}, keyvals...)
	helper.Error(args...)
}

// LogWarn 记录Warn级别日志（带追踪信息）
func LogWarn(ctx context.Context, logger log.Logger, msg string, keyvals ...interface{}) {
	traceLogger := NewTraceLogger(logger)
	loggerWithTrace := traceLogger.WithContext(ctx)
	helper := log.NewHelper(loggerWithTrace)
	args := append([]interface{}{"msg", msg}, keyvals...)
	helper.Warn(args...)
}

// LogDebug 记录Debug级别日志（带追踪信息）
func LogDebug(ctx context.Context, logger log.Logger, msg string, keyvals ...interface{}) {
	traceLogger := NewTraceLogger(logger)
	loggerWithTrace := traceLogger.WithContext(ctx)
	helper := log.NewHelper(loggerWithTrace)
	args := append([]interface{}{"msg", msg}, keyvals...)
	helper.Debug(args...)
}
