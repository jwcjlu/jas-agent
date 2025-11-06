package agent

// NewESAgentExecutor 创建Elasticsearch Agent执行器
func NewESAgentExecutor(context *Context, clusterInfo string) *AgentExecutor {
	executor := NewAgentExecutor(context)
	esAgent := NewESAgent(context, executor, clusterInfo)
	executor.agent = esAgent
	return executor
}
