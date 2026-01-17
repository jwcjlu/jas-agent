package datasource

import (
	"context"
	"fmt"
	agents2 "jas-agent/agent/agent/aiops/agents"
	"jas-agent/agent/agent/aiops/framework"
	"time"
)

// ExampleUsage 演示如何使用数据源适配器
func ExampleUsage() {
	ctx := context.Background()

	// 1. 创建 Prometheus 指标数据源
	prometheusDS := NewPrometheusDataSource(
		"http://prometheus:9090",
		30*time.Second,
	)

	// 2. 创建 Elasticsearch 日志数据源
	esDS := NewElasticsearchLogDataSource(
		"http://elasticsearch:9200",
		"elastic",
		"password",
		30*time.Second,
	)

	// 3. 创建 Jaeger Trace 数据源
	jaegerDS := NewJaegerTraceDataSource(
		"http://jaeger:16686",
		30*time.Second,
	)

	// 5. 定义时间范围
	timeRange := framework.TimeRange{
		StartTime: time.Now().Add(-1 * time.Hour).Unix(),
		EndTime:   time.Now().Unix(),
	}

	// 6. 查询指标数据
	metrics, err := prometheusDS.FetchMetrics(ctx, []string{"order-service"}, timeRange)
	if err != nil {
		fmt.Printf("Failed to fetch metrics: %v\n", err)
	} else {
		fmt.Printf("Fetched %d metric data points\n", len(metrics))
		for _, m := range metrics[:minInt(5, len(metrics))] {
			fmt.Printf("  %s[%s] at %d: %.2f\n", m.Service, m.Metric, m.Timestamp, m.Value)
		}
	}

	// 7. 查询日志数据
	logs, err := esDS.FetchLogs(ctx, []string{"order-service"}, timeRange)
	if err != nil {
		fmt.Printf("Failed to fetch logs: %v\n", err)
	} else {
		fmt.Printf("Fetched %d log entries\n", len(logs))
		for _, log := range logs[:minInt(5, len(logs))] {
			fmt.Printf("  [%s] %s: %s\n", log.Level, log.Service, log.Message)
		}
	}

	// 8. 查询 Jaeger Trace 数据
	jaegerTraces, err := jaegerDS.FetchTraces(ctx, []string{"order-service"}, timeRange)
	if err != nil {
		fmt.Printf("Failed to fetch Jaeger traces: %v\n", err)
	} else {
		fmt.Printf("Fetched %d traces from Jaeger\n", len(jaegerTraces))
		for _, trace := range jaegerTraces[:minInt(3, len(jaegerTraces))] {
			fmt.Printf("  Trace ID: %s, Services: %v, Has Error: %v\n", trace.TraceID, trace.Services, trace.HasError)
		}
	}

}

// MetricsDataSourceWrapper Prometheus 数据源包装器
type MetricsDataSourceWrapper struct {
	ds *PrometheusDataSource
}

// FetchMetrics 实现 agents.DataSource 接口
func (m *MetricsDataSourceWrapper) FetchMetrics(
	ctx context.Context,
	services []string,
	timeRange framework.TimeRange,
) ([]agents2.MetricsData, error) {
	return m.ds.FetchMetrics(ctx, services, timeRange)
}

// LogsDataSourceWrapper Elasticsearch 数据源包装器
type LogsDataSourceWrapper struct {
	ds *ElasticsearchLogDataSource
}

// FetchLogs 实现 agents.LogDataSource 接口
func (l *LogsDataSourceWrapper) FetchLogs(
	ctx context.Context,
	services []string,
	timeRange framework.TimeRange,
) ([]agents2.LogEntry, error) {
	return l.ds.FetchLogs(ctx, services, timeRange)
}

// ExampleIntegration 演示如何集成到 AIOps 系统
func ExampleIntegration() {
	//ctx := context.Background()

	// 1. 创建数据源
	prometheusDS := NewPrometheusDataSource("http://prometheus:9090", 30*time.Second)
	esDS := NewElasticsearchLogDataSource("http://elasticsearch:9200", "", "", 30*time.Second)

	// 2. 创建包装器
	metricsDS := &MetricsDataSourceWrapper{ds: prometheusDS}
	logsDS := &LogsDataSourceWrapper{ds: esDS}

	// 3. 创建协作上下文（假设已存在）
	// collaboratorCtx := framework.NewCollaborationContext(...)

	// 4. 创建智能体（需要实际的上下文）
	// metricsAgent := agents.NewMetricsAgent(collaboratorCtx, metricsDS)
	// logsAgent := agents.NewLogsAgent(collaboratorCtx, logsDS)

	fmt.Printf("Created metrics data source: %T\n", metricsDS)
	fmt.Printf("Created logs data source: %T\n", logsDS)
}

// minInt 返回两个整数中的较小值
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
