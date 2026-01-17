package biz

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"jas-agent/agent/agent"
	"jas-agent/agent/agent/aiops/agents"
	aiopsDatasource "jas-agent/agent/agent/aiops/datasource"
	aiopsFramework "jas-agent/agent/agent/aiops/framework"
	"jas-agent/agent/llm"
	"jas-agent/agent/tools"
)

type AgentFactory struct {
	factory map[agent.AgentType]IAgent
	chat    llm.Chat // LLM chat 客户端，用于 AIOPS Agent
}

func NewAgentFactory(chat llm.Chat) *AgentFactory {
	af := &AgentFactory{
		factory: make(map[agent.AgentType]IAgent),
		chat:    chat,
	}
	af.RegisterAgent(&reactAgent{})
	af.RegisterAgent(&planAgent{})
	af.RegisterAgent(&chainAgent{})
	af.RegisterAgent(&sqlAgent{})
	af.RegisterAgent(&esAgent{})
	af.RegisterAgent(newAiOpsAgent(chat))
	return af
}

func (factory *AgentFactory) RegisterAgent(agent IAgent) {
	factory.factory[agent.AgentType()] = agent
}

func (factory *AgentFactory) GetAgent(agentType agent.AgentType) (IAgent, error) {
	agent, ok := factory.factory[agentType]
	if !ok {
		return nil, fmt.Errorf("not found agent type")
	}
	return agent, nil
}

// getChat 获取 chat 客户端（用于 AIOPS Agent）
func (factory *AgentFactory) getChat() llm.Chat {
	return factory.chat
}

func (factory *AgentFactory) CreateAgentExecutor(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context) (*agent.AgentExecutor, error) {
	iAgent := factory.findIAgent(agentConfig.Framework)
	if iAgent == nil {
		return agent.NewAgentExecutor(agentCtx), nil
	}
	return iAgent.CreateAgentExecutor(ctx, agentConfig, agentCtx)
}

func (factory *AgentFactory) findIAgent(framework string) IAgent {
	iAgent, ok := factory.factory[agent.AgentType(framework)]
	if ok {
		return iAgent
	}
	for _, v := range factory.factory {
		if v.Alias() == framework {
			return v
		}
	}
	return nil
}

type planAgent struct {
}

func (s *planAgent) Validate() bool {
	return true
}
func (s *planAgent) CreateAgentExecutor(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context) (*agent.AgentExecutor, error) {

	return agent.NewPlanAgentExecutor(agentCtx, true), nil
}
func (s *planAgent) AgentType() agent.AgentType {
	return agent.PlanAgentType
}
func (s *planAgent) Description() string {
	return "计划代理，先规划后执行复杂任务"
}
func (s *planAgent) Alias() string {
	return "plan"
}

type reactAgent struct {
}

func (s *reactAgent) Validate() bool {
	return true
}
func (s *reactAgent) CreateAgentExecutor(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context) (*agent.AgentExecutor, error) {
	return agent.NewAgentExecutor(agentCtx), nil
}
func (s *reactAgent) AgentType() agent.AgentType {
	return agent.ReactAgentType
}
func (s *reactAgent) Description() string {
	return "通用推理代理，支持思考-行动-观察循环"
}
func (s *reactAgent) Alias() string {
	return "react"
}

type chainAgent struct {
}

func (s *chainAgent) Validate() bool {
	return true
}
func (s *chainAgent) CreateAgentExecutor(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context) (*agent.AgentExecutor, error) {
	builder := agent.NewChainBuilder(agentCtx)
	builder.AddNode("main", agent.ReactAgentType, 100)
	ca := builder.Build()
	return agent.NewChainAgentExecutor(agentCtx, ca), nil
}
func (s *chainAgent) AgentType() agent.AgentType {
	return agent.ChainAgentType
}
func (s *chainAgent) Description() string {
	return "链式代理，按预定义流程执行多个Agent"
}
func (s *chainAgent) Alias() string {
	return "chain"
}

type sqlAgent struct {
}

func (s *sqlAgent) Validate() bool {
	return true
}

type sqlConnectionConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

func (s *sqlAgent) parseSQLConnectionConfig(raw string) (*sqlConnectionConfig, error) {
	if raw == "" {
		return nil, fmt.Errorf("SQL 连接配置为空")
	}

	cfg := &sqlConnectionConfig{
		Port: 3306,
	}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("解析 SQL 连接配置失败: %w", err)
	}

	if cfg.Host == "" || cfg.Username == "" || cfg.Database == "" {
		return nil, fmt.Errorf("SQL 连接配置缺少必要字段")
	}
	return cfg, nil
}
func (s *sqlAgent) CreateAgentExecutor(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context) (*agent.AgentExecutor, error) {
	// 解析 SQL 连接配置
	connConfig, err := s.parseSQLConnectionConfig(agentConfig.ConnectionConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid SQL connection config: %w", err)
	}

	// 创建 SQL 连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		connConfig.Username, connConfig.Password,
		connConfig.Host, connConfig.Port, connConfig.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	// 注册 SQL 工具
	sqlConn := &tools.SQLConnection{DB: db}
	tools.RegisterSQLTools(sqlConn, agentCtx.GetToolManager())

	// 创建 SQL Agent
	dbInfo := fmt.Sprintf("MySQL: %s@%s:%d/%s", connConfig.Username, connConfig.Host, connConfig.Port, connConfig.Database)
	return agent.NewSQLAgentExecutor(agentCtx, dbInfo), nil

}
func (s *sqlAgent) AgentType() agent.AgentType {
	return agent.SQLAgentType
}
func (s *sqlAgent) Description() string {
	return "SQL查询专家，生成和执行数据库查询"
}
func (s *sqlAgent) Alias() string {
	return "sql"
}

type esAgent struct {
}

func (s *esAgent) Validate() bool {
	return true
}
func (s *esAgent) CreateAgentExecutor(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context) (*agent.AgentExecutor, error) {
	// 解析 ES 连接配置
	esConfig, err := s.parseESConnectionConfig(agentConfig.ConnectionConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid ES connection config: %w", err)
	}

	// 创建 ES 连接
	esConn := tools.NewESConnection(esConfig.Host, esConfig.Username, esConfig.Password)

	// 注册 ES 工具
	//tools.RegisterESTools(esConn, agentCtx.GetToolManager())
	agentCtx.GetToolManager().RegisterTool(tools.NewGetIndexMapping(esConn))
	agentCtx.GetToolManager().RegisterTool(tools.NewSearchDocuments(esConn), tools.WithLogClustering())
	agentCtx.GetToolManager().RegisterTool(tools.NewGetDocument(esConn))
	agentCtx.GetToolManager().RegisterTool(tools.NewAggregateData(esConn))
	agentCtx.GetToolManager().RegisterTool(tools.NewSearchIndices(esConn)) // 新增：索引模糊搜索
	// 创建 ES Agent
	clusterInfo := fmt.Sprintf("Elasticsearch: %s", esConfig.Host)
	executor := agent.NewESAgentExecutor(agentCtx, clusterInfo)
	return executor, nil

}

type esConnectionConfig struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *esAgent) parseESConnectionConfig(raw string) (*esConnectionConfig, error) {
	if raw == "" {
		return nil, fmt.Errorf("elasticsearch 连接配置为空")
	}

	cfg := &esConnectionConfig{}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("解析 Elasticsearch 连接配置失败: %w", err)
	}

	if cfg.Host == "" {
		return nil, fmt.Errorf("elasticsearch 连接配置缺少 host")
	}
	return cfg, nil
}

func (s *esAgent) AgentType() agent.AgentType {
	return agent.ESAgentType
}
func (s *esAgent) Description() string {
	return "计划代理，先规划后执行复杂任务"
}

func (s *esAgent) Alias() string {
	return "elasticsearch"
}

type aiopsAgent struct {
	chat llm.Chat
}

func newAiOpsAgent(chat llm.Chat) IAgent {
	return &aiopsAgent{chat: chat}
}

func (s *aiopsAgent) Validate() bool {
	return true
}

// aiopsConnectionConfig AIOps 连接配置
type aiopsConnectionConfig struct {
	// Prometheus 配置
	Prometheus struct {
		BaseURL string `json:"base_url"`
		Timeout int    `json:"timeout"` // 秒
	} `json:"prometheus"`
	// Elasticsearch 配置
	Elasticsearch struct {
		BaseURL  string `json:"base_url"`
		Username string `json:"username"`
		Password string `json:"password"`
		Timeout  int    `json:"timeout"` // 秒
	} `json:"elasticsearch"`
	// Jaeger 配置
	Jaeger struct {
		BaseURL string `json:"base_url"`
		Timeout int    `json:"timeout"` // 秒
	} `json:"jaeger"`
	// 服务列表
	Services []ServiceConfig `json:"services"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Name             string `json:"name"`               // 服务名
	LogIndex         string `json:"log_index"`          // 日志索引
	TraceServiceName string `json:"trace_service_name"` // Trace 服务名
}

func (s *aiopsAgent) parseAIOPSConnectionConfig(raw string) (*aiopsConnectionConfig, error) {
	if raw == "" {
		return nil, fmt.Errorf("aiops 连接配置为空")
	}

	cfg := &aiopsConnectionConfig{
		Services: make([]ServiceConfig, 0),
	}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("解析 AIOps 连接配置失败: %w", err)
	}

	// 兼容旧格式：如果 services 是字符串数组，转换为 ServiceConfig 数组
	// 先尝试解析为临时结构，检查 services 字段类型
	var tempConfig map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &tempConfig); err == nil {
		if servicesRaw, ok := tempConfig["services"]; ok {
			if servicesArray, ok := servicesRaw.([]interface{}); ok && len(servicesArray) > 0 {
				// 检查第一个元素是否为字符串（旧格式）
				if _, ok := servicesArray[0].(string); ok {
					// 旧格式：字符串数组，转换为 ServiceConfig 数组
					cfg.Services = make([]ServiceConfig, 0, len(servicesArray))
					for _, item := range servicesArray {
						if serviceName, ok := item.(string); ok {
							cfg.Services = append(cfg.Services, ServiceConfig{
								Name:             serviceName,
								LogIndex:         "",
								TraceServiceName: serviceName, // 默认使用服务名
							})
						}
					}
				}
			}
		}
	}

	// 验证至少配置了一个数据源
	if cfg.Prometheus.BaseURL == "" && cfg.Elasticsearch.BaseURL == "" && cfg.Jaeger.BaseURL == "" {
		return nil, fmt.Errorf("AIOps 连接配置至少需要配置一个数据源（Prometheus、Elasticsearch 或 Jaeger）")
	}

	return cfg, nil
}

// CreateAgentExecutor 实现 IAgent 接口
func (s *aiopsAgent) CreateAgentExecutor(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context) (*agent.AgentExecutor, error) {
	return s.createAgentExecutorWithChat(ctx, agentConfig, agentCtx, s.chat)

}

// createAgentExecutorWithChat 创建 AIOPS Agent Executor（带 chat 参数）
func (s *aiopsAgent) createAgentExecutorWithChat(ctx context.Context,
	agentConfig *Agent,
	agentCtx *agent.Context,
	chat llm.Chat) (*agent.AgentExecutor, error) {
	// 解析 AIOps 连接配置
	connConfig, err := s.parseAIOPSConnectionConfig(agentConfig.ConnectionConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid AIOps connection config: %w", err)
	}

	// 检查 chat 是否配置
	if chat == nil {
		return nil, fmt.Errorf("LLM chat client is not configured for AIOPS Agent")
	}

	traceID := fmt.Sprintf("aiops_%d", time.Now().Unix())
	tenantID := "default"

	// 创建 Collaborator（它会在内部创建 CollaborationContext）
	collaborator := aiopsFramework.NewCollaborator(
		ctx,
		chat,
		agentCtx.GetMemory(),
		traceID,
		tenantID,
	)

	// 获取 CollaborationContext
	collabCtx := collaborator.GetContext()

	// 根据配置创建数据源和 Agent
	var metricsDataSource agents.DataSource
	var logsDataSource agents.LogDataSource
	var topologyDataSource agents.TopologyDataSource

	// 初始化 Prometheus 数据源（Metrics）
	if connConfig.Prometheus.BaseURL != "" {
		timeout := time.Duration(connConfig.Prometheus.Timeout) * time.Second
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		promDS := aiopsDatasource.NewPrometheusDataSource(connConfig.Prometheus.BaseURL, timeout)
		// 创建适配器
		metricsDataSource = &prometheusDataSourceAdapter{ds: promDS}
	}

	// 初始化 Elasticsearch 数据源（Logs）
	if connConfig.Elasticsearch.BaseURL != "" {
		timeout := time.Duration(connConfig.Elasticsearch.Timeout) * time.Second
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		esDS := aiopsDatasource.NewElasticsearchLogDataSource(
			connConfig.Elasticsearch.BaseURL,
			connConfig.Elasticsearch.Username,
			connConfig.Elasticsearch.Password,
			timeout,
		)
		// 创建适配器，传入服务配置用于日志索引映射
		logsDataSource = &elasticsearchLogDataSourceAdapter{
			ds:             esDS,
			serviceConfigs: connConfig.Services,
		}
	}

	// 初始化 Jaeger 数据源（Topology/Traces）
	if connConfig.Jaeger.BaseURL != "" {
		timeout := time.Duration(connConfig.Jaeger.Timeout) * time.Second
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		jaegerDS := aiopsDatasource.NewJaegerTraceDataSource(connConfig.Jaeger.BaseURL, timeout)
		// 创建适配器，将 Jaeger Trace 数据源适配为 Topology 数据源
		topologyDataSource = &jaegerTopologyDataSourceAdapter{
			ds:             jaegerDS,
			serviceConfigs: connConfig.Services,
		}
	}

	// 注册所有 Agent
	// 1. Planner Agent
	collaborator.RegisterAgent(agents.NewPlannerAgent(collabCtx))

	// 2. Metrics Agent（如果有 Prometheus 配置）
	if metricsDataSource != nil {
		collaborator.RegisterAgent(agents.NewMetricsAgent(collabCtx, metricsDataSource))
	}

	// 3. Logs Agent（如果有 Elasticsearch 配置）
	if logsDataSource != nil {
		collaborator.RegisterAgent(agents.NewLogsAgent(collabCtx, logsDataSource))
	}

	// 4. Topology Agent（如果有 Jaeger 配置）
	if topologyDataSource != nil {
		collaborator.RegisterAgent(agents.NewTopologyAgent(collabCtx, topologyDataSource))
	}

	// 5. Decision Agent
	collaborator.RegisterAgent(agents.NewDecisionAgent(collabCtx))

	// 6. Output Agent
	collaborator.RegisterAgent(agents.NewOutputAgent(collabCtx))

	// 创建 AIOPS Agent Executor
	systemPrompt := agentConfig.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "你是一个专业的 AIOPS 智能运维分析专家，能够分析系统故障的根因。"
	}

	// 从配置中提取服务名列表
	services := make([]string, 0, len(connConfig.Services))
	for _, serviceConfig := range connConfig.Services {
		if serviceConfig.Name != "" {
			services = append(services, serviceConfig.Name)
		}
	}

	return agent.NewAIOPSAgentExecutor(agentCtx, collaborator, systemPrompt, services), nil
}

func (s *aiopsAgent) AgentType() agent.AgentType {
	return agent.AIOPSAgentType
}

func (s *aiopsAgent) Description() string {
	return "AIOps 智能运维代理，支持 Metrics、Log、Trace 多数据源分析"
}

func (s *aiopsAgent) Alias() string {
	return "aiops"
}

// prometheusDataSourceAdapter Prometheus 数据源适配器
// 将 datasource.PrometheusDataSource 适配为 agents.DataSource
type prometheusDataSourceAdapter struct {
	ds *aiopsDatasource.PrometheusDataSource
}

func (a *prometheusDataSourceAdapter) FetchMetrics(ctx context.Context, services []string, timeRange aiopsFramework.TimeRange) ([]agents.MetricsData, error) {
	return a.ds.FetchMetrics(ctx, services, timeRange)
}

// elasticsearchLogDataSourceAdapter Elasticsearch 日志数据源适配器
// 将 datasource.ElasticsearchLogDataSource 适配为 agents.LogDataSource
// 支持根据服务配置使用指定的日志索引
type elasticsearchLogDataSourceAdapter struct {
	ds             *aiopsDatasource.ElasticsearchLogDataSource
	serviceConfigs []ServiceConfig
}

func (a *elasticsearchLogDataSourceAdapter) FetchLogs(ctx context.Context, services []string, timeRange aiopsFramework.TimeRange) ([]agents.LogEntry, error) {
	// 构建服务名到日志索引的映射
	serviceToLogIndex := make(map[string]string)
	for _, cfg := range a.serviceConfigs {
		if cfg.LogIndex != "" {
			serviceToLogIndex[cfg.Name] = cfg.LogIndex
		}
	}

	allLogs := make([]agents.LogEntry, 0)
	for _, service := range services {
		var logs []agents.LogEntry
		var err error

		// 如果配置了日志索引，使用指定的索引查询
		if logIndex, ok := serviceToLogIndex[service]; ok && logIndex != "" {
			logs, err = a.ds.QueryLogsWithIndex(ctx, logIndex, service, timeRange)
		} else {
			// 否则使用默认逻辑（尝试多个索引模式）
			logs, err = a.ds.FetchLogs(ctx, []string{service}, timeRange)
		}

		if err != nil {
			// 记录错误但继续查询其他服务
			continue
		}
		allLogs = append(allLogs, logs...)
	}

	return allLogs, nil
}

// jaegerTopologyDataSourceAdapter Jaeger Trace 数据源适配器
// 将 datasource.JaegerTraceDataSource 适配为 agents.TopologyDataSource
// 从 Trace 数据中提取拓扑信息
type jaegerTopologyDataSourceAdapter struct {
	ds             *aiopsDatasource.JaegerTraceDataSource
	serviceConfigs []ServiceConfig
}

func (a *jaegerTopologyDataSourceAdapter) FetchTopology(ctx context.Context, services []string, timeRange aiopsFramework.TimeRange) (*agents.ServiceTopology, error) {
	// 构建服务名到 Trace 服务名的映射
	serviceToTraceService := make(map[string]string)
	for _, cfg := range a.serviceConfigs {
		traceServiceName := cfg.TraceServiceName
		if traceServiceName == "" {
			traceServiceName = cfg.Name
		}
		serviceToTraceService[cfg.Name] = traceServiceName
	}

	// 获取 Trace 数据
	traceServices := make([]string, 0, len(services))
	for _, service := range services {
		if traceServiceName, ok := serviceToTraceService[service]; ok {
			traceServices = append(traceServices, traceServiceName)
		} else {
			traceServices = append(traceServices, service)
		}
	}

	traces, err := a.ds.FetchTraces(ctx, traceServices, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch traces: %w", err)
	}

	// 从 Trace 数据构建拓扑
	return a.buildTopologyFromTraces(traces, services), nil
}

// buildTopologyFromTraces 从 Trace 数据构建服务拓扑
func (a *jaegerTopologyDataSourceAdapter) buildTopologyFromTraces(traces []aiopsDatasource.TraceData, services []string) *agents.ServiceTopology {
	// ServiceTopology.Nodes 是 map[string]*ServiceNode
	nodes := make(map[string]*agents.ServiceNode)
	edges := make([]agents.ServiceEdge, 0)

	// 收集所有服务和调用关系
	serviceSet := make(map[string]bool)
	edgesMap := make(map[string]map[string]bool) // from -> to

	for _, trace := range traces {
		for _, service := range trace.Services {
			serviceSet[service] = true
		}

		// 从调用链提取边
		for i := 0; i < len(trace.CallChain)-1; i++ {
			from := trace.CallChain[i].Service
			to := trace.CallChain[i+1].Service
			if from != "" && to != "" && from != to {
				if edgesMap[from] == nil {
					edgesMap[from] = make(map[string]bool)
				}
				edgesMap[from][to] = true
			}
		}
	}

	// 构建节点
	for service := range serviceSet {
		nodes[service] = &agents.ServiceNode{
			Name: service,
			Type: "service",
		}
	}

	// 构建边
	for from, toMap := range edgesMap {
		for to := range toMap {
			edges = append(edges, agents.ServiceEdge{
				Source: from,
				Target: to,
				Type:   "calls",
			})
		}
	}

	return &agents.ServiceTopology{
		Nodes:     nodes,
		Edges:     edges,
		UpdatedAt: time.Now().Unix(),
	}
}
