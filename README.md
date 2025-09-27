# JAS Agent

一个基于 ReAct (Reasoning and Acting) 框架的 Go 语言 AI 代理系统，支持工具调用和逐步推理。

## 特性

- 🤖 **ReAct 框架**: 实现思考-行动-观察的循环推理
- 🛠️ **工具系统**: 可扩展的工具管理器和执行器
- 💬 **LLM 集成**: 支持 OpenAI 兼容的 API
- 🧠 **内存管理**: 对话历史和上下文管理
- 🔧 **模块化设计**: 清晰的架构，易于扩展

## 架构

```
jas-agent/
├── agent/           # 代理核心
│   ├── agent.go     # Agent 接口和执行器
│   ├── agent_context.go  # 上下文管理
│   └── react_agent.go    # ReAct 代理实现
├── core/            # 核心类型和接口
│   ├── message.go   # 消息类型
│   ├── memory.go    # 内存接口
│   ├── tool.go      # 工具接口
│   └── prompt.go    # 提示词模板
├── llm/             # LLM 集成
│   ├── chat.go      # 聊天客户端
│   └── types.go     # 请求响应类型
├── memory/          # 内存实现
│   └── memory.go    # 内存存储
├── tools/           # 工具实现
│   ├── tool.go      # 工具管理器
│   ├── calculator.go # 计算器工具
│   └── mcp.go       # MCP 工具
└── examples/        # 示例代码
    └── react/       # ReAct 示例
```

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 运行示例

```bash
cd examples/react
go run .
```

### 3. 基本使用

```go
package main

import (
    "fmt"
    "jas-agent/agent"
    "jas-agent/llm"
    "github.com/sashabaranov/go-openai"
)

func main() {
    // 创建 LLM 客户端
    chat := llm.NewChat(&llm.Config{
        ApiKey:  "your-api-key",
        BaseURL: "your-api-base-url",
    })
    
    // 创建代理上下文
    context := agent.NewContext(
        agent.WithModel(openai.GPT3Dot5Turbo),
        agent.WithChat(chat),
    )
    
    // 创建执行器
    executor := agent.NewAgentExecutor(context)
    
    // 运行查询
    result := executor.Run("计算 15 + 27 的结果")
    fmt.Println(result)
}
```

## 核心概念

### Agent 接口

```go
type Agent interface {
    Type() AgentType
    Step() string
}
```

### ReAct 循环

1. **思考 (Thought)**: 分析当前情况，决定下一步行动
2. **行动 (Action)**: 执行工具调用或完成任务
3. **观察 (Observation)**: 获取行动结果，为下一步思考提供信息

### 工具系统

#### 定义工具

```go
type MyTool struct{}

func (t *MyTool) Name() string {
    return "myTool"
}

func (t *MyTool) Description() string {
    return "我的自定义工具"
}

func (t *MyTool) Handler(ctx context.Context, input string) (string, error) {
    // 工具逻辑
    return "结果", nil
}

// 注册工具
func init() {
    tools.GetToolManager().RegisterTool(&MyTool{})
}
```

#### 内置工具

- **Calculator**: 数学表达式计算
- **AverageDogWeight**: 狗狗品种平均体重查询

## 配置选项

### 上下文选项

```go
// 设置模型
agent.WithModel(openai.GPT4)

// 设置聊天客户端
agent.WithChat(chat)

// 设置内存
agent.WithMemory(memory)

// 设置工具管理器
agent.WithToolManager(toolManager)
```

### 执行器配置

```go
executor := &AgentExecutor{
    maxSteps: 10,        // 最大执行步数
    currentStep: 0,      // 当前步数
    state: IdleState,    // 执行状态
}
```

## 状态管理

- **IdleState**: 空闲状态
- **RunningState**: 运行中
- **FinishState**: 完成
- **ErrorState**: 错误

## 消息类型

- **System**: 系统消息
- **User**: 用户消息
- **Assistant**: 助手消息
- **Function**: 函数调用
- **Tool**: 工具响应

## 扩展开发

### 添加新的 Agent 类型

```go
type MyAgent struct {
    context *Context
}

func (a *MyAgent) Type() AgentType {
    return "MyAgent"
}

func (a *MyAgent) Step() string {
    // 实现步骤逻辑
    return "结果"
}
```

### 添加新的内存实现

```go
type MyMemory struct {
    // 实现 core.Memory 接口
}

func (m *MyMemory) AddMessage(message core.Message) {
    // 实现添加消息
}
```

## 示例场景

### 数学计算

```go
result := executor.Run("计算 (15 + 27) * 3 的结果")
// 输出: Final answer: 126
```

### 多步骤推理

```go
result := executor.Run("我有3只狗，一只边境牧羊犬、一只苏格兰梗和一只玩具贵宾犬。它们的总重量是多少？")
// 输出: Final answer: 64
```

## 依赖

- `github.com/sashabaranov/go-openai`: OpenAI API 客户端
- `go.starlark.net`: 数学表达式计算

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 更新日志

### v1.0.0
- 初始版本发布
- 实现 ReAct 框架
- 支持工具调用和逐步推理
- 提供完整的示例和文档
