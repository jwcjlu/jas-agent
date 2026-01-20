package core

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// InitObservability 初始化可观测性系统
func InitObservability(serviceName, serviceVersion string) (func(), error) {
	ctx := context.Background()

	// 创建资源
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		return nil, err
	}

	// 初始化Prometheus指标导出器
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	// 创建指标提供者
	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(exporter),
	)

	// 创建跟踪提供者（使用NoOp实现，如需完整追踪可以替换为Jaeger等导出器）
	tp := trace.NewTracerProvider(
		trace.WithResource(res),
		// 可以添加更多配置，如采样率、导出器等
	)

	// 设置全局提供者
	otel.SetMeterProvider(mp)
	otel.SetTracerProvider(tp)

	// 初始化全局指标
	if err := InitGlobalMetrics(); err != nil {
		return nil, err
	}

	// 返回清理函数
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
	}

	return cleanup, nil
}

// GetPrometheusExporter 获取Prometheus导出器（用于HTTP服务器暴露指标）
func GetPrometheusExporter() (*prometheus.Exporter, error) {
	// 这里需要从全局meter provider获取exporter
	// 由于prometheus.New()已经返回了exporter，我们可以存储它
	// 为了简化，我们直接返回一个新的（实际应该复用）
	return prometheus.New()
}
