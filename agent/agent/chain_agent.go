package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"jas-agent/agent/core"
	"jas-agent/agent/llm"

	"github.com/go-kratos/kratos/v2/log"
)

var chainLogger = log.NewHelper(log.With(log.NewStdLogger(os.Stdout), "module", "agent/chain_agent"))

// ChainNode 链式节点
type ChainNode struct {
	Name        string              // 节点名称
	Agent       Agent               // 执行的Agent
	Transform   func(string) string // 输出转换函数
	Condition   func(string) bool   // 执行条件
	MaxSteps    int                 // 最大步数
	NextNodes   []*ChainNode        // 下一个节点（支持分支）
	Description string              // 节点描述
}

// ChainAgent 链式Agent
type ChainAgent struct {
	context      *Context
	executor     *AgentExecutor
	rootNode     *ChainNode
	currentNode  *ChainNode
	chainResult  map[string]string // 存储每个节点的结果
	systemPrompt string
}

func (a *ChainAgent) Type() AgentType {
	return ChainAgentType
}

func (a *ChainAgent) Step() string {
	if a.currentNode == nil {
		return "Chain execution completed"
	}

	// 执行当前节点
	nodeName := a.currentNode.Name
	chainLogger.Infof("🔗 Executing chain node: %s", nodeName)

	// 获取上一个节点的输出作为输入
	var input string
	if len(a.chainResult) > 0 {
		// 找到最后一个执行的结果
		for _, result := range a.chainResult {
			input = result
		}
	}

	// 检查执行条件
	if a.currentNode.Condition != nil && !a.currentNode.Condition(input) {
		chainLogger.Infof("⏭️  Skipping node %s (condition not met)", nodeName)
		// 跳过当前节点，移到下一个
		if len(a.currentNode.NextNodes) > 0 {
			a.currentNode = a.currentNode.NextNodes[0]
			return a.Step()
		}
		a.currentNode = nil
		return "Chain execution completed (skipped)"
	}

	// 创建节点专用的执行器
	nodeExecutor := &AgentExecutor{
		context:      a.context,
		maxSteps:     a.currentNode.MaxSteps,
		currentStep:  0,
		state:        IdleState,
		agent:        a.currentNode.Agent,
		summaryAgent: NewSummaryAgent(a.context, a.executor),
	}

	// 如果有输入，添加到内存
	if input != "" {
		a.context.memory.AddMessage(core.Message{
			Role:    core.MessageRoleUser,
			Content: fmt.Sprintf("基于上一步的结果继续处理: %s", input),
		})
	}

	// 执行节点
	result := nodeExecutor.Run("")

	// 应用转换函数
	if a.currentNode.Transform != nil {
		result = a.currentNode.Transform(result)
	}

	// 保存结果
	a.chainResult[nodeName] = result
	chainLogger.Infof("✅ Node %s completed with result: %s", nodeName, truncateString(result, 100))

	// 选择下一个节点
	if len(a.currentNode.NextNodes) == 0 {
		a.currentNode = nil
		a.executor.UpdateState(FinishState)
		return result
	}

	// 如果有多个下一个节点，根据结果选择（简单实现：选择第一个条件满足的）
	for _, nextNode := range a.currentNode.NextNodes {
		if nextNode.Condition == nil || nextNode.Condition(result) {
			a.currentNode = nextNode
			return "Moving to next node: " + nextNode.Name
		}
	}

	// 如果没有满足条件的下一个节点，使用第一个
	a.currentNode = a.currentNode.NextNodes[0]
	return "Moving to next node: " + a.currentNode.Name
}

// ChainBuilder 链式构建器
type ChainBuilder struct {
	context *Context
	nodes   map[string]*ChainNode
	root    *ChainNode
}

// NewChainBuilder 创建链式构建器
func NewChainBuilder(context *Context) *ChainBuilder {
	return &ChainBuilder{
		context: context,
		nodes:   make(map[string]*ChainNode),
	}
}

// AddNode 添加节点
func (b *ChainBuilder) AddNode(name string, agentType AgentType, maxSteps int) *ChainBuilder {
	var agent Agent
	executor := &AgentExecutor{context: b.context}

	switch agentType {
	case ReactAgentType:
		agent = NewReactAgent(b.context, executor)
	case SQLAgentType:
		agent = NewSQLAgent(b.context, executor, "default")
	default:
		agent = NewReactAgent(b.context, executor)
	}

	if maxSteps == 0 {
		maxSteps = 10
	}

	node := &ChainNode{
		Name:      name,
		Agent:     agent,
		MaxSteps:  maxSteps,
		NextNodes: []*ChainNode{},
	}

	b.nodes[name] = node

	// 如果是第一个节点，设置为根节点
	if b.root == nil {
		b.root = node
	}

	return b
}

// SetTransform 设置节点的转换函数
func (b *ChainBuilder) SetTransform(nodeName string, transform func(string) string) *ChainBuilder {
	if node, ok := b.nodes[nodeName]; ok {
		node.Transform = transform
	}
	return b
}

// SetCondition 设置节点的执行条件
func (b *ChainBuilder) SetCondition(nodeName string, condition func(string) bool) *ChainBuilder {
	if node, ok := b.nodes[nodeName]; ok {
		node.Condition = condition
	}
	return b
}

// Link 连接两个节点
func (b *ChainBuilder) Link(fromNode, toNode string) *ChainBuilder {
	if from, ok := b.nodes[fromNode]; ok {
		if to, ok := b.nodes[toNode]; ok {
			from.NextNodes = append(from.NextNodes, to)
		}
	}
	return b
}

// Build 构建链式Agent
func (b *ChainBuilder) Build() Agent {
	systemPrompt := `你是一个链式执行代理，将按照预定义的流程逐步处理任务。`

	return &ChainAgent{
		context:      b.context,
		rootNode:     b.root,
		currentNode:  b.root,
		chainResult:  make(map[string]string),
		systemPrompt: systemPrompt,
	}
}

// NewChainAgentExecutor 创建链式Agent执行器
func NewChainAgentExecutor(context *Context, chainAgent Agent) *AgentExecutor {
	executor := &AgentExecutor{
		context:     context,
		maxSteps:    100, // 链式执行可能需要更多步骤
		currentStep: 0,
		state:       IdleState,
		agent:       chainAgent,
	}

	// 将executor注入到chainAgent
	if ca, ok := chainAgent.(*ChainAgent); ok {
		ca.executor = executor
	}

	// 设置summaryAgent
	executor.summaryAgent = NewSummaryAgent(context, executor)

	return executor
}

// 辅助函数：截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// RouteAgent 路由Agent - 根据输入选择不同的处理链路
type RouteAgent struct {
	context      *Context
	executor     *AgentExecutor
	routes       map[string]Agent
	routeFunc    func(string) string // 路由函数
	systemPrompt string
}

func (a *RouteAgent) Type() AgentType {
	return "RouteAgent"
}

func (a *RouteAgent) Step() string {
	// 获取用户输入
	messages := a.context.memory.GetMessages()
	var userInput string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == core.MessageRoleUser {
			userInput = messages[i].Content
			break
		}
	}

	// 使用路由函数确定路由
	routeKey := a.routeFunc(userInput)
	chainLogger.Infof("🔀 Routing to: %s", routeKey)

	// 获取对应的Agent
	targetAgent, ok := a.routes[routeKey]
	if !ok {
		return fmt.Sprintf("No route found for key: %s", routeKey)
	}

	// 执行目标Agent
	return targetAgent.Step()
}

// NewRouteAgent 创建路由Agent
func NewRouteAgent(context *Context, executor *AgentExecutor, routeFunc func(string) string, routes map[string]Agent) Agent {
	systemPrompt := core.GetReactSystemPrompt(core.ReactSystemPrompt{
		Date:  time.Now().Format("2006-01-02 15:04:05"),
		Tools: []core.ToolData{},
	})

	context.memory.AddMessage(core.Message{
		Role:    core.MessageRoleSystem,
		Content: systemPrompt,
	})

	return &RouteAgent{
		context:      context,
		executor:     executor,
		routes:       routes,
		routeFunc:    routeFunc,
		systemPrompt: systemPrompt,
	}
}

// AIRouteAgent 使用AI进行智能路由的Agent
type AIRouteAgent struct {
	context           *Context
	executor          *AgentExecutor
	routes            map[string]Agent
	routeDescriptions map[string]string
	systemPrompt      string
}

func (a *AIRouteAgent) Type() AgentType {
	return "AIRouteAgent"
}

func (a *AIRouteAgent) Step() string {
	// 获取用户输入
	messages := a.context.memory.GetMessages()
	var userInput string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == core.MessageRoleUser {
			userInput = messages[i].Content
			break
		}
	}

	// 构建路由选择提示
	var routeOptions strings.Builder
	routeOptions.WriteString("请选择最合适的处理路由:\n")
	for key, desc := range a.routeDescriptions {
		routeOptions.WriteString(fmt.Sprintf("- %s: %s\n", key, desc))
	}
	routeOptions.WriteString(fmt.Sprintf("\n用户输入: %s\n", userInput))
	routeOptions.WriteString("\n请只返回路由名称（如 'sql', 'react' 等）")

	// 使用LLM选择路由
	routeMessages := []core.Message{
		{
			Role:    core.MessageRoleSystem,
			Content: "你是一个智能路由助手，根据用户输入选择最合适的处理方式。",
		},
		{
			Role:    core.MessageRoleUser,
			Content: routeOptions.String(),
		},
	}

	resp, err := a.context.chat.Completions(context.TODO(), llm.NewChatRequest(a.context.model, routeMessages))
	if err != nil {
		return fmt.Sprintf("Route selection failed: %s", err.Error())
	}

	routeKey := strings.TrimSpace(strings.ToLower(resp.Content()))
	chainLogger.Infof("🤖 AI selected route: %s", routeKey)

	// 获取对应的Agent
	targetAgent, ok := a.routes[routeKey]
	if !ok {
		return fmt.Sprintf("Invalid route selected: %s", routeKey)
	}

	// 执行目标Agent
	return targetAgent.Step()
}

// NewAIRouteAgent 创建AI路由Agent
func NewAIRouteAgent(context *Context, exec *AgentExecutor, routes map[string]Agent, descriptions map[string]string) Agent {
	return &AIRouteAgent{
		context:           context,
		executor:          exec,
		routes:            routes,
		routeDescriptions: descriptions,
		systemPrompt:      "AI智能路由代理",
	}
}
