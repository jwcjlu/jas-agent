package agents

import (
	"context"
	"fmt"
	"jas-agent/agent/agent/aiops/framework"
	"math"
	"time"

	"jas-agent/agent/core"
	"jas-agent/agent/llm"
)

// TopologyAgent 拓扑感知智能体
// 理解系统架构和服务依赖关系，能基于CMDB和实时调用链，分析故障的传播路径
type TopologyAgent struct {
	*BaseAgent
	dataSource TopologyDataSource
}

// NewTopologyAgent 创建拓扑感知智能体
func NewTopologyAgent(ctx *framework.CollaborationContext, dataSource TopologyDataSource) *TopologyAgent {
	base := NewBaseAgent(
		framework.RoleTopology,
		"拓扑感知智能体",
		`你是一个专业的拓扑分析专家。你的职责是：
1. 理解系统架构和服务依赖关系
2. 基于CMDB和实时调用链分析故障的传播路径
3. 确定影响范围
4. 识别关键路径和瓶颈服务

分析时应该：
- 关注服务间的调用关系
- 识别上游和下游依赖
- 分析故障传播路径
- 评估影响范围
- 识别关键路径上的瓶颈服务`,
		ctx.Chat(),
		ctx.Memory(),
	)

	agent := &TopologyAgent{
		BaseAgent:  base,
		dataSource: dataSource,
	}

	return agent
}

// Execute 执行拓扑分析
func (a *TopologyAgent) Execute(ctx context.Context, task *framework.Task) (*framework.TaskResult, error) {
	result := &framework.TaskResult{
		TaskID:    task.ID,
		AgentRole: framework.RoleTopology,
		Success:   true,
		Evidence:  make([]framework.Evidence, 0),
		Findings:  make([]framework.Finding, 0),
		Metadata:  make(map[string]interface{}),
	}

	// 1. 获取拓扑数据
	topology, err := a.fetchTopology(ctx, task)
	if err != nil {
		result.Success = false
		result.Metadata["error"] = err.Error()
		return result, nil
	}

	// 2. 分析依赖关系
	dependencies := a.analyzeDependencies(topology, task.Services)

	// 3. 分析故障传播路径
	propagationPaths := a.analyzePropagationPaths(topology, task.Services)

	// 4. 评估影响范围
	impactScope := a.assessImpactScope(topology, task.Services)

	// 5. 识别关键路径
	criticalPaths := a.identifyCriticalPaths(topology, task.Services)

	// 6. 构建证据和发现
	for _, path := range propagationPaths {
		result.Evidence = append(result.Evidence, framework.Evidence{
			Type:        "topology",
			Service:     path.SourceService,
			Timestamp:   task.TimeRange.StartTime,
			Description: fmt.Sprintf("故障传播路径: %s -> %s", path.SourceService, path.TargetService),
			Data:        path,
			Score:       path.ImpactScore,
		})
	}

	for _, path := range criticalPaths {
		result.Findings = append(result.Findings, framework.Finding{
			Type:        "critical_path",
			Service:     path.Service,
			Description: fmt.Sprintf("关键路径上的瓶颈服务: %s", path.Service),
			Severity:    path.Severity,
			Score:       path.Criticality,
		})
	}

	// 7. 使用 LLM 进行高级分析
	analysis := a.analyzeWithLLM(ctx, task, topology, dependencies, propagationPaths, impactScope)

	result.Metadata["topology"] = topology
	result.Metadata["dependencies"] = dependencies
	result.Metadata["propagation_paths"] = propagationPaths
	result.Metadata["impact_scope"] = impactScope
	result.Metadata["critical_paths"] = criticalPaths
	result.Metadata["analysis"] = analysis
	result.Confidence = a.calculateConfidence(topology, propagationPaths, criticalPaths)

	return result, nil
}

// ServiceNode 服务节点
type ServiceNode struct {
	Name         string
	Type         string // service, database, cache, queue, etc.
	Status       string // healthy, degraded, down
	Dependencies []string
	Dependents   []string // 依赖此服务的服务
}

// ServiceTopology 服务拓扑
type ServiceTopology struct {
	Nodes     map[string]*ServiceNode
	Edges     []ServiceEdge
	UpdatedAt int64
}

// ServiceEdge 服务边（调用关系）
type ServiceEdge struct {
	Source    string
	Target    string
	Type      string  // http, grpc, database, cache, etc.
	Weight    float64 // 调用频率或重要性
	Latency   float64 // 平均延迟
	ErrorRate float64 // 错误率
}

// Dependency 依赖关系
type Dependency struct {
	Service    string
	Upstream   []string // 上游服务（被依赖）
	Downstream []string // 下游服务（依赖此服务）
	Depth      int      // 依赖深度
}

// PropagationPath 传播路径
type PropagationPath struct {
	SourceService string
	TargetService string
	Path          []string // 传播路径上的服务序列
	ImpactScore   float64  // 影响评分
	Severity      string
}

// CriticalPath 关键路径
type CriticalPath struct {
	Service     string
	Path        []string
	Criticality float64 // 关键性评分
	Severity    string
	Bottleneck  bool
}

// fetchTopology 获取拓扑数据
func (a *TopologyAgent) fetchTopology(ctx context.Context, task *framework.Task) (*ServiceTopology, error) {
	if a.dataSource == nil {
		return a.generateMockTopology(task.Services), nil
	}
	return a.dataSource.FetchTopology(ctx, task.Services, task.TimeRange)
}

// generateMockTopology 生成模拟拓扑数据（用于测试）
func (a *TopologyAgent) generateMockTopology(services []string) *ServiceTopology {
	nodes := make(map[string]*ServiceNode)
	edges := make([]ServiceEdge, 0)
	// 为每个服务创建节点
	for _, service := range services {
		nodes[service] = &ServiceNode{
			Name:         service,
			Type:         "service",
			Status:       "healthy",
			Dependencies: make([]string, 0),
			Dependents:   make([]string, 0),
		}
	}

	// 创建依赖关系（示例）
	if len(services) > 1 {
		for i := 0; i < len(services)-1; i++ {
			source := services[i]
			target := services[i+1]
			nodes[source].Dependencies = append(nodes[source].Dependencies, target)
			nodes[target].Dependents = append(nodes[target].Dependents, source)
			edges = append(edges, ServiceEdge{
				Source:    source,
				Target:    target,
				Type:      "http",
				Weight:    1.0,
				Latency:   100,
				ErrorRate: 0.01,
			})
		}
	}

	return &ServiceTopology{
		Nodes:     nodes,
		Edges:     edges,
		UpdatedAt: time.Now().Unix(),
	}
}

// analyzeDependencies 分析依赖关系
func (a *TopologyAgent) analyzeDependencies(topology *ServiceTopology, services []string) []Dependency {
	dependencies := make([]Dependency, 0)
	for _, service := range services {
		node, ok := topology.Nodes[service]
		if !ok {
			continue
		}

		// 计算依赖深度
		depth := a.calculateDepth(topology, service, make(map[string]bool))

		dependencies = append(dependencies, Dependency{
			Service:    service,
			Upstream:   node.Dependencies,
			Downstream: node.Dependents,
			Depth:      depth,
		})
	}

	return dependencies
}

// calculateDepth 计算依赖深度
func (a *TopologyAgent) calculateDepth(topology *ServiceTopology, service string, visited map[string]bool) int {
	if visited[service] {
		return 0 // 避免循环
	}
	visited[service] = true

	node, ok := topology.Nodes[service]
	if !ok || len(node.Dependencies) == 0 {
		return 0
	}

	maxDepth := 0
	for _, dep := range node.Dependencies {
		depth := a.calculateDepth(topology, dep, visited) + 1
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return maxDepth
}

// analyzePropagationPaths 分析故障传播路径
func (a *TopologyAgent) analyzePropagationPaths(topology *ServiceTopology, services []string) []PropagationPath {
	paths := make([]PropagationPath, 0)

	for _, service := range services {
		// 分析从此服务向下的传播路径
		downstreamPaths := a.findPaths(topology, service, true)
		for _, path := range downstreamPaths {
			impactScore := a.calculateImpactScore(topology, path)
			severity := "MEDIUM"
			if impactScore > 0.8 {
				severity = "CRITICAL"
			} else if impactScore > 0.6 {
				severity = "HIGH"
			}

			paths = append(paths, PropagationPath{
				SourceService: service,
				TargetService: path[len(path)-1],
				Path:          path,
				ImpactScore:   impactScore,
				Severity:      severity,
			})
		}
	}

	return paths
}

// findPaths 查找路径（DFS）
func (a *TopologyAgent) findPaths(topology *ServiceTopology, start string, downstream bool) [][]string {
	paths := make([][]string, 0)
	visited := make(map[string]bool)

	var dfs func(current string, path []string)
	dfs = func(current string, path []string) {
		if visited[current] {
			return
		}
		visited[current] = true
		path = append(path, current)

		node, ok := topology.Nodes[current]
		if !ok {
			return
		}

		targets := node.Dependencies
		if !downstream {
			targets = node.Dependents
		}

		if len(targets) == 0 {
			// 到达叶子节点，保存路径
			if len(path) > 1 {
				newPath := make([]string, len(path))
				copy(newPath, path)
				paths = append(paths, newPath)
			}
			return
		}

		for _, target := range targets {
			dfs(target, path)
		}
	}

	dfs(start, []string{})
	return paths
}

// calculateImpactScore 计算影响评分
func (a *TopologyAgent) calculateImpactScore(topology *ServiceTopology, path []string) float64 {
	if len(path) < 2 {
		return 0.5
	}

	// 基于路径长度、服务数量、错误率等因素计算
	baseScore := float64(len(path)) / 10.0 // 路径越长，影响越大
	if baseScore > 1.0 {
		baseScore = 1.0
	}

	// 考虑路径上各边的错误率
	totalErrorRate := 0.0
	for i := 0; i < len(path)-1; i++ {
		edge := a.findEdge(topology, path[i], path[i+1])
		if edge != nil {
			totalErrorRate += edge.ErrorRate
		}
	}
	errorRateScore := totalErrorRate * 10
	if errorRateScore > 0.5 {
		errorRateScore = 0.5
	}

	return baseScore + errorRateScore
}

// findEdge 查找边
func (a *TopologyAgent) findEdge(topology *ServiceTopology, source, target string) *ServiceEdge {
	for i := range topology.Edges {
		if topology.Edges[i].Source == source && topology.Edges[i].Target == target {
			return &topology.Edges[i]
		}
	}
	return nil
}

// assessImpactScope 评估影响范围
func (a *TopologyAgent) assessImpactScope(topology *ServiceTopology, services []string) []string {
	impactScope := make(map[string]bool)

	// 将源服务加入影响范围
	for _, service := range services {
		impactScope[service] = true
	}

	// 查找所有下游服务
	for _, service := range services {
		node, ok := topology.Nodes[service]
		if !ok {
			continue
		}

		// 递归查找所有依赖的服务
		a.findAllDependents(topology, node, impactScope)
	}

	result := make([]string, 0, len(impactScope))
	for service := range impactScope {
		result = append(result, service)
	}

	return result
}

// findAllDependents 查找所有依赖的服务
func (a *TopologyAgent) findAllDependents(topology *ServiceTopology, node *ServiceNode, visited map[string]bool) {
	for _, dep := range node.Dependencies {
		if !visited[dep] {
			visited[dep] = true
			depNode, ok := topology.Nodes[dep]
			if ok {
				a.findAllDependents(topology, depNode, visited)
			}
		}
	}
}

// identifyCriticalPaths 识别关键路径
func (a *TopologyAgent) identifyCriticalPaths(topology *ServiceTopology, services []string) []CriticalPath {
	criticalPaths := make([]CriticalPath, 0)

	for _, service := range services {
		// 找到从根服务到此服务的所有路径
		paths := a.findPaths(topology, service, false) // 向上查找
		for _, path := range paths {
			if len(path) == 0 {
				continue
			}

			criticality := a.calculateCriticality(topology, path)
			severity := "MEDIUM"
			if criticality > 0.8 {
				severity = "CRITICAL"
			} else if criticality > 0.6 {
				severity = "HIGH"
			}

			// 检查是否为瓶颈
			isBottleneck := a.isBottleneck(topology, path)

			criticalPaths = append(criticalPaths, CriticalPath{
				Service:     path[0],
				Path:        path,
				Criticality: criticality,
				Severity:    severity,
				Bottleneck:  isBottleneck,
			})
		}
	}

	return criticalPaths
}

// calculateCriticality 计算关键性
func (a *TopologyAgent) calculateCriticality(topology *ServiceTopology, path []string) float64 {
	if len(path) == 0 {
		return 0
	}

	// 基于路径长度、服务重要性等因素
	baseCriticality := float64(len(path)) / 10.0
	if baseCriticality > 0.7 {
		baseCriticality = 0.7
	}

	// 考虑路径上的延迟和错误率
	totalLatency := 0.0
	totalErrorRate := 0.0
	for i := 0; i < len(path)-1; i++ {
		edge := a.findEdge(topology, path[i], path[i+1])
		if edge != nil {
			totalLatency += edge.Latency
			totalErrorRate += edge.ErrorRate
		}
	}

	latencyScore := math.Min(totalLatency/1000.0, 0.15) // 总延迟超过1秒
	errorRateScore := math.Min(totalErrorRate*5, 0.15)  // 总错误率

	return baseCriticality + latencyScore + errorRateScore
}

// isBottleneck 判断是否为瓶颈
func (a *TopologyAgent) isBottleneck(topology *ServiceTopology, path []string) bool {
	// 如果路径上的某个服务有多个依赖者，可能是瓶颈
	for _, service := range path {
		node, ok := topology.Nodes[service]
		if !ok {
			continue
		}
		if len(node.Dependents) > 3 { // 超过3个服务依赖此服务
			return true
		}
	}
	return false
}

// analyzeWithLLM 使用 LLM 进行高级分析
func (a *TopologyAgent) analyzeWithLLM(
	ctx context.Context,
	task *framework.Task,
	topology *ServiceTopology,
	dependencies []Dependency,
	propagationPaths []PropagationPath,
	impactScope []string,
) string {
	prompt := fmt.Sprintf(`分析以下服务拓扑和依赖关系：

服务数量: %d
依赖关系数: %d
传播路径数: %d
影响范围: %d 个服务

主要依赖关系：
%s

主要传播路径：
%s

请分析：
1. 故障可能的传播路径
2. 受影响的服务范围
3. 关键路径和瓶颈服务
4. 故障根因可能的位置`,
		len(topology.Nodes),
		len(dependencies),
		len(propagationPaths),
		len(impactScope),
		a.formatDependencies(dependencies[:minInt(5, len(dependencies))]),
		a.formatPropagationPaths(propagationPaths[:minInt(5, len(propagationPaths))]))

	resp, err := a.chat.Completions(ctx, llm.NewChatRequest(
		"gpt-3.5-turbo",
		[]core.Message{
			{
				Role:    core.MessageRoleSystem,
				Content: a.description,
			},
			{
				Role:    core.MessageRoleUser,
				Content: prompt,
			},
		},
	))
	if err != nil {
		return "LLM 分析失败: " + err.Error()
	}

	return resp.Content()
}

func (a *TopologyAgent) formatDependencies(dependencies []Dependency) string {
	result := ""
	for _, dep := range dependencies {
		result += fmt.Sprintf("- %s: 上游 %v, 下游 %v, 深度 %d\n", dep.Service, dep.Upstream, dep.Downstream, dep.Depth)
	}
	return result
}

func (a *TopologyAgent) formatPropagationPaths(paths []PropagationPath) string {
	result := ""
	for _, path := range paths {
		result += fmt.Sprintf("- %s -> %s: %v (影响评分: %.2f)\n", path.SourceService, path.TargetService, path.Path, path.ImpactScore)
	}
	return result
}

func (a *TopologyAgent) calculateConfidence(topology *ServiceTopology, propagationPaths []PropagationPath, criticalPaths []CriticalPath) float64 {
	if topology == nil || len(topology.Nodes) == 0 {
		return 0.3
	}

	baseConfidence := 0.5
	pathBonus := math.Min(float64(len(propagationPaths))*0.05, 0.2)
	criticalBonus := math.Min(float64(len(criticalPaths))*0.1, 0.3)

	return math.Min(baseConfidence+pathBonus+criticalBonus, 1.0)
}

// TopologyDataSource 拓扑数据源接口
type TopologyDataSource interface {
	FetchTopology(ctx context.Context, services []string, timeRange framework.TimeRange) (*ServiceTopology, error)
}
