package framework

import (
	"context"
	"fmt"
	"sync"

	"jas-agent/agent/core"
	"jas-agent/agent/llm"
)

// Collaborator 多智能体协作框架
// 负责协调多个专用智能体的工作，模拟专家团队协作模式
type Collaborator struct {
	agents      map[AgentRole]Agent
	coordinator *Coordinator
	context     *CollaborationContext
	mu          sync.RWMutex
}

// GetContext 获取协作上下文
func (c *Collaborator) GetContext() *CollaborationContext {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.context
}

// AgentRole 智能体角色
type AgentRole string

const (
	RolePlanner  AgentRole = "planner"  // 任务规划智能体
	RoleMetrics  AgentRole = "metrics"  // 指标分析智能体
	RoleLogs     AgentRole = "logs"     // 日志分析智能体
	RoleTopology AgentRole = "topology" // 拓扑感知智能体
	RoleDecision AgentRole = "decision" // 分析决策智能体
	RoleOutput   AgentRole = "output"   // 最终输出智能体
)

// Agent 智能体接口
type Agent interface {
	Role() AgentRole
	Name() string
	Execute(ctx context.Context, task *Task) (*TaskResult, error)
	CanHandle(task *Task) bool
}

// Task 任务定义
type Task struct {
	ID         string
	Type       TaskType
	Query      string                 // 自然语言查询
	TimeRange  TimeRange              // 时间范围
	Services   []string               // 相关服务
	Alerts     []Alert                // 告警信息
	Context    map[string]interface{} // 上下文数据
	ParentTask *Task                  // 父任务（如果有）
	SubTasks   []*Task                // 子任务
}

// TaskType 任务类型
type TaskType string

const (
	TaskTypeRootCauseAnalysis TaskType = "root_cause_analysis" // 根因分析
	TaskTypeAlertCorrelation  TaskType = "alert_correlation"   // 告警关联
	TaskTypeAnomalyDetection  TaskType = "anomaly_detection"   // 异常检测
	TaskTypeMetricsAnalysis   TaskType = "metrics_analysis"    // 指标分析
	TaskTypeLogsAnalysis      TaskType = "logs_analysis"       // 日志分析
	TaskTypeTopologyAnalysis  TaskType = "topology_analysis"   // 拓扑分析
	TaskTypeDecision          TaskType = "decision"            // 决策分析
	TaskTypeOutput            TaskType = "output"              // 输出生成
)

// TimeRange 时间范围
type TimeRange struct {
	StartTime int64 // Unix 时间戳
	EndTime   int64 // Unix 时间戳
}

// Alert 告警信息
type Alert struct {
	ID        string
	Service   string
	Severity  string // CRITICAL, HIGH, MEDIUM, LOW
	Message   string
	Timestamp int64
	Source    string // prometheus, zabbix, etc.
	Labels    map[string]string
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID      string
	AgentRole   AgentRole
	Success     bool
	Evidence    []Evidence             // 证据链
	Findings    []Finding              // 发现
	Confidence  float64                // 置信度 [0,1]
	Metadata    map[string]interface{} // 元数据
	NextActions []string               // 下一步行动建议
}

// Evidence 证据
type Evidence struct {
	Type        string // metrics, logs, traces, topology
	Service     string
	Timestamp   int64
	Description string
	Data        interface{}
	Score       float64 // 证据强度
}

// Finding 发现
type Finding struct {
	Type        string // anomaly, correlation, root_cause
	Service     string
	Description string
	Severity    string
	Score       float64
}

// CollaborationContext 协作上下文
type CollaborationContext struct {
	chat       llm.Chat
	memory     core.Memory
	traceID    string
	tenantID   string
	sharedData map[string]interface{}
	mu         sync.RWMutex
}

// NewCollaborator 创建多智能体协作框架
func NewCollaborator(ctx context.Context, chat llm.Chat, memory core.Memory, traceID, tenantID string) *Collaborator {
	collabCtx := &CollaborationContext{
		chat:       chat,
		memory:     memory,
		traceID:    traceID,
		tenantID:   tenantID,
		sharedData: make(map[string]interface{}),
	}

	coordinator := NewCoordinator(collabCtx)

	collaborator := &Collaborator{
		agents:      make(map[AgentRole]Agent),
		coordinator: coordinator,
		context:     collabCtx,
	}

	return collaborator
}

// RegisterAgent 注册智能体
func (c *Collaborator) RegisterAgent(agent Agent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.agents[agent.Role()] = agent
}

// GetAgent 获取指定角色的智能体
func (c *Collaborator) GetAgent(role AgentRole) (Agent, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	agent, ok := c.agents[role]
	if !ok {
		return nil, fmt.Errorf("agent with role %s not found", role)
	}
	return agent, nil
}

// Collaborate 执行协作分析
// 这是多智能体协作的主入口
func (c *Collaborator) Collaborate(ctx context.Context, query string, timeRange TimeRange, services []string, alerts []Alert) (*AnalysisReport, error) {
	// 1. 创建根任务
	rootTask := &Task{
		ID:        GenerateTaskID(),
		Type:      TaskTypeRootCauseAnalysis,
		Query:     query,
		TimeRange: timeRange,
		Services:  services,
		Alerts:    alerts,
		Context:   make(map[string]interface{}),
	}

	// 2. 使用任务规划智能体生成分析计划
	planner, err := c.GetAgent(RolePlanner)
	if err != nil {
		return nil, fmt.Errorf("get planner agent: %w", err)
	}

	planResult, err := planner.Execute(ctx, rootTask)
	if err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	// 3. 从规划结果中提取子任务
	subTasks, ok := planResult.Metadata["sub_tasks"].([]*Task)
	if !ok {
		// 如果规划智能体没有返回子任务，使用默认流程
		subTasks = c.generateDefaultSubTasks(rootTask)
	}

	// 4. 并行执行子任务（由各个专用智能体处理）
	results := c.executeTasksInParallel(ctx, subTasks)

	// 5. 使用分析决策智能体综合所有结果
	decisionAgent, err := c.GetAgent(RoleDecision)
	if err != nil {
		return nil, fmt.Errorf("get decision agent: %w", err)
	}

	decisionTask := &Task{
		ID:        GenerateTaskID(),
		Type:      TaskTypeDecision,
		Query:     query,
		TimeRange: timeRange,
		Services:  services,
		Context: map[string]interface{}{
			"task_results": results,
		},
		ParentTask: rootTask,
	}

	decisionResult, err := decisionAgent.Execute(ctx, decisionTask)
	if err != nil {
		return nil, fmt.Errorf("decision failed: %w", err)
	}

	// 6. 使用最终输出智能体生成报告
	outputAgent, err := c.GetAgent(RoleOutput)
	if err != nil {
		return nil, fmt.Errorf("get output agent: %w", err)
	}

	outputTask := &Task{
		ID:        GenerateTaskID(),
		Type:      TaskTypeOutput,
		Query:     query,
		TimeRange: timeRange,
		Context: map[string]interface{}{
			"decision_result": decisionResult,
			"task_results":    results,
			"original_query":  query,
		},
		ParentTask: rootTask,
	}

	outputResult, err := outputAgent.Execute(ctx, outputTask)
	if err != nil {
		return nil, fmt.Errorf("output generation failed: %w", err)
	}

	// 7. 构建最终报告
	report := c.buildReport(rootTask, results, decisionResult, outputResult)

	// 8. 记录到上下文（用于知识沉淀）
	c.recordAnalysis(report)

	return report, nil
}

// executeTasksInParallel 并行执行任务
func (c *Collaborator) executeTasksInParallel(ctx context.Context, tasks []*Task) map[string]*TaskResult {
	results := make(map[string]*TaskResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, task := range tasks {
		wg.Add(1)
		go func(t *Task) {
			defer wg.Done()

			// 根据任务类型选择合适的智能体
			agent := c.findAgentForTask(t)
			if agent == nil {
				return
			}

			result, err := agent.Execute(ctx, t)
			if err != nil {
				result = &TaskResult{
					TaskID:    t.ID,
					AgentRole: agent.Role(),
					Success:   false,
					Metadata: map[string]interface{}{
						"error": err.Error(),
					},
				}
			}

			mu.Lock()
			results[t.ID] = result
			mu.Unlock()
		}(task)
	}

	wg.Wait()
	return results
}

// findAgentForTask 根据任务类型找到合适的智能体
func (c *Collaborator) findAgentForTask(task *Task) Agent {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, agent := range c.agents {
		if agent.CanHandle(task) {
			return agent
		}
	}

	// 根据任务类型映射到智能体
	switch task.Type {
	case TaskTypeMetricsAnalysis:
		return c.agents[RoleMetrics]
	case TaskTypeLogsAnalysis:
		return c.agents[RoleLogs]
	case TaskTypeTopologyAnalysis:
		return c.agents[RoleTopology]
	default:
		return nil
	}
}

// generateDefaultSubTasks 生成默认子任务流程
func (c *Collaborator) generateDefaultSubTasks(rootTask *Task) []*Task {
	return []*Task{
		{
			ID:         GenerateTaskID(),
			Type:       TaskTypeMetricsAnalysis,
			Query:      rootTask.Query,
			TimeRange:  rootTask.TimeRange,
			Services:   rootTask.Services,
			Alerts:     rootTask.Alerts,
			ParentTask: rootTask,
		},
		{
			ID:         GenerateTaskID(),
			Type:       TaskTypeLogsAnalysis,
			Query:      rootTask.Query,
			TimeRange:  rootTask.TimeRange,
			Services:   rootTask.Services,
			Alerts:     rootTask.Alerts,
			ParentTask: rootTask,
		},
		{
			ID:         GenerateTaskID(),
			Type:       TaskTypeTopologyAnalysis,
			Query:      rootTask.Query,
			TimeRange:  rootTask.TimeRange,
			Services:   rootTask.Services,
			Alerts:     rootTask.Alerts,
			ParentTask: rootTask,
		},
	}
}

// AnalysisReport 分析报告
type AnalysisReport struct {
	ID               string
	Query            string
	TimeRange        TimeRange
	Summary          string                 // 摘要
	RootCause        string                 // 根因
	AffectedServices []string               // 受影响服务
	EvidenceChain    []Evidence             // 证据链
	Findings         []Finding              // 所有发现
	Recommendations  []string               // 修复建议
	Confidence       float64                // 整体置信度
	Timeline         []TimelineEvent        // 时间线
	Metadata         map[string]interface{} // 元数据
}

// TimelineEvent 时间线事件
type TimelineEvent struct {
	Timestamp   int64
	Service     string
	EventType   string
	Description string
	Severity    string
}

// buildReport 构建最终报告
func (c *Collaborator) buildReport(
	rootTask *Task,
	results map[string]*TaskResult,
	decisionResult *TaskResult,
	outputResult *TaskResult,
) *AnalysisReport {
	report := &AnalysisReport{
		ID:               rootTask.ID,
		Query:            rootTask.Query,
		TimeRange:        rootTask.TimeRange,
		AffectedServices: rootTask.Services,
		EvidenceChain:    make([]Evidence, 0),
		Findings:         make([]Finding, 0),
		Recommendations:  make([]string, 0),
		Metadata:         make(map[string]interface{}),
	}

	// 收集所有证据
	for _, result := range results {
		if result.Success {
			report.EvidenceChain = append(report.EvidenceChain, result.Evidence...)
			report.Findings = append(report.Findings, result.Findings...)
		}
	}

	// 从决策结果提取根因和建议
	if decisionResult.Success {
		if rootCause, ok := decisionResult.Metadata["root_cause"].(string); ok {
			report.RootCause = rootCause
		}
		if confidence, ok := decisionResult.Metadata["confidence"].(float64); ok {
			report.Confidence = confidence
		}
	}

	// 从输出结果提取摘要和建议
	if outputResult.Success {
		if summary, ok := outputResult.Metadata["summary"].(string); ok {
			report.Summary = summary
		}
		if recommendations, ok := outputResult.Metadata["recommendations"].([]string); ok {
			report.Recommendations = recommendations
		}
		if timeline, ok := outputResult.Metadata["timeline"].([]TimelineEvent); ok {
			report.Timeline = timeline
		}
	}

	return report
}

// recordAnalysis 记录分析结果（用于知识沉淀）
func (c *Collaborator) recordAnalysis(report *AnalysisReport) {
	c.context.mu.Lock()
	defer c.context.mu.Unlock()

	c.context.sharedData[fmt.Sprintf("report_%s", report.ID)] = report
}

// Coordinator 协调器（用于智能体间通信和调度）
type Coordinator struct {
	context *CollaborationContext
}

// NewCoordinator 创建协调器
func NewCoordinator(ctx *CollaborationContext) *Coordinator {
	return &Coordinator{context: ctx}
}
