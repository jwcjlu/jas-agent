# JAS Agent

一个基于 ReAct (Reasoning and Acting) 框架的 Go 语言 AI 代理系统，支持工具调用、逐步推理和 MCP 协议集成。

## 特性

- 🤖 **ReAct 框架**: 实现思考-行动-观察的循环推理
- 📝 **SummaryAgent**: 自动总结执行过程，提供清晰答案
- 🛠️ **工具系统**: 可扩展的工具管理器和执行器
- 🔌 **MCP 支持**: 集成 [Model Context Protocol](https://github.com/metoro-io/mcp-golang) 工具发现
- 💬 **LLM 集成**: 支持 OpenAI 兼容的 API
- 🧠 **内存管理**: 对话历史和上下文管理
- 🔧 **模块化设计**: 清晰的架构，易于扩展

## 架构

```
jas-agent/
├── agent/              # 代理核心
│   ├── agent.go        # Agent 接口和执行器
│   ├── agent_context.go # 上下文管理
│   ├── react_agent.go  # ReAct 代理实现
│   └── summary_agent.go # 总结代理实现
├── core/               # 核心类型和接口
│   ├── message.go      # 消息类型
│   ├── memory.go       # 内存接口
│   ├── tool.go         # 工具接口
│   └── prompt.go       # 提示词模板
├── llm/                # LLM 集成
│   ├── chat.go         # 聊天客户端
│   └── types.go        # 请求响应类型
├── memory/             # 内存实现
│   └── memory.go       # 内存存储
├── tools/              # 工具实现
│   ├── tool.go         # 工具管理器
│   ├── calculator.go   # 计算器工具
│   └── mcp.go          # MCP 工具支持
└── examples/           # 示例代码
    └── react/          # ReAct 示例
        ├── main.go     # 主程序
        └── tools/      # 示例工具
            └── tool.go # 狗狗体重查询工具
```

## 快速开始

### 1. 安装依赖

```bash
go get github.com/sashabaranov/go-openai
go get github.com/metoro-io/mcp-golang
go get go.starlark.net/starlark
```

或者

```bash
go mod tidy
```

### 2. 运行示例

```bash
cd examples/react
go run . -apiKey YOUR_API_KEY -baseUrl YOUR_BASE_URL
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
4. **总结 (Summary)**: SummaryAgent 自动总结执行过程，提供清晰答案

### 工具系统

#### 定义工具

工具需要实现 `core.Tool` 接口：

```go
type Tool interface {
    Name() string
    Description() string
    Handler(ctx context.Context, input string) (string, error)
    Input() any
    Type() ToolType
}
```

示例工具实现：

```go
package tools

import (
    "context"
    "jas-agent/core"
    "jas-agent/tools"
)

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

func (t *MyTool) Input() any {
    return nil // 或返回 JSON Schema
}

func (t *MyTool) Type() core.ToolType {
    return core.Normal
}

// 注册工具
func init() {
    tools.GetToolManager().RegisterTool(&MyTool{})
}
```

#### 工具类型

- **Normal**: 普通工具（通过系统提示词告知 LLM）
- **Mcp**: MCP 工具（通过 Function Calling 方式调用）

#### 内置工具

- **Calculator**: 数学表达式计算（使用 Starlark 求值器）
- **AverageDogWeight**: 狗狗品种平均体重查询

### MCP 工具支持

使用 [mcp-golang](https://github.com/metoro-io/mcp-golang) 库集成 MCP 工具：

#### 创建 MCP 工具管理器

```go
import "jas-agent/tools"

// 创建 MCP 工具管理器
mcpManager, err := tools.NewMCPToolManager("my-mcp", "http://localhost:8080/mcp")
if err != nil {
    log.Fatal(err)
}

// 启动工具发现（后台自动刷新）
mcpManager.Start()
```

#### MCP 工具特性

1. **自动发现**: 定期刷新工具列表（每 5 秒）
2. **双缓冲**: 使用原子操作实现无锁切换
3. **工具前缀**: 自动添加前缀避免命名冲突（格式：`name@toolName`）
4. **Function Calling**: MCP 工具通过 OpenAI Function Calling 调用
5. **HTTP Transport**: 使用 HTTP 协议与 MCP 服务器通信

#### MCP 工具调用流程

```
LLM -> Function Calling -> MCPToolWrapper -> MCP Server
                                ↓
                         解析参数并调用
                                ↓
                           返回结果
```

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

### SummaryAgent 功能

SummaryAgent 会自动分析执行过程并提供总结：

- 默认启用
- 分析整个执行过程
- 提取关键信息和结果
- 提供简洁明了的最终答案

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
// 输出: 基于执行过程分析，15加27等于42，乘以3等于126。因此结果是126。
```

### 多步骤推理

```go
result := executor.Run("我有3只狗，一只边境牧羊犬、一只苏格兰梗和一只玩具贵宾犬。它们的总重量是多少？")
// 执行流程：
// 1. 查询边境牧羊犬平均体重: 37 lbs
// 2. 查询苏格兰梗平均体重: 20 lbs
// 3. 查询玩具贵宾犬平均体重: 7 lbs
// 4. 计算总重量: 37 + 20 + 7 = 64 lbs
// 5. 总结: 三只狗的总重量约为64磅
```

### MCP 工具调用

```go
// 假设 MCP 服务器提供了天气查询工具
mcpManager, _ := tools.NewMCPToolManager("weather-mcp", "http://weather-api:8080/mcp")
mcpManager.Start()

result := executor.Run("北京的天气怎么样？")
// LLM 会通过 Function Calling 调用 MCP 工具
```

## 工具调用机制

### 普通工具（Normal）

- 通过系统提示词列出工具名称和描述
- LLM 输出格式：`Action: toolName[input]`
- 正则解析并执行

### MCP 工具（Mcp）

- 通过 OpenAI Function Calling 机制
- 自动生成工具的 JSON Schema
- LLM 直接调用，无需解析

### 工具执行流程

```go
// ReactAgent.Action()
toolCalls := agent.tools  // 从 LLM 响应获取工具调用

for _, toolCall := range toolCalls {
    // 执行工具
    result, err := agent.context.toolManager.ExecTool(ctx, toolCall)
    
    // 添加观察结果到内存
    agent.context.memory.AddMessage(core.Message{
        Role:    core.MessageRoleUser,
        Content: fmt.Sprintf("Observation: %s", result),
    })
}
```

## 依赖

- `github.com/sashabaranov/go-openai`: OpenAI API 客户端
- `github.com/metoro-io/mcp-golang`: Model Context Protocol 支持
- `go.starlark.net/starlark`: 数学表达式计算

## 故障排查

### 找不到工具

**问题**: LLM 输出 `Action: search[...]` 但系统没有该工具

**原因**:
1. MCP 工具未成功发现/注册
2. 工具名称不匹配
3. 系统提示词中工具列表为空

**解决方案**:
```go
// 1. 确保在创建执行器前发现 MCP 工具
mcpManager, _ := tools.NewMCPToolManager("my-mcp", "http://localhost:8080/mcp")
mcpManager.Start()

// 2. 检查工具列表
tools := context.GetToolManager().AvailableTools()
for _, tool := range tools {
    fmt.Printf("Tool: %s - %s\n", tool.Name(), tool.Description())
}

// 3. 确保工具名称一致
// MCP 工具会自动添加前缀: "my-mcp@toolName"
```

### MCP 连接失败

**问题**: `failed to initialize MCP client`

**解决方案**:
- 确认 MCP 服务器已启动
- 检查 HTTP 端点是否正确
- 查看 MCP 服务器日志

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 更新日志

### v1.2.0
- 集成 [mcp-golang](https://github.com/metoro-io/mcp-golang) 库
- 支持 HTTP Transport MCP 通信
- 实现工具类型区分（Normal/Mcp）
- 添加 MCP 工具自动刷新机制
- 支持 OpenAI Function Calling

### v1.1.0
- 添加 SummaryAgent 总结功能
- 改进 LLM 接口设计
- 优化执行流程

### v1.0.0
- 初始版本发布
- 实现 ReAct 框架
- 支持工具调用和逐步推理
- 提供完整的示例和文档
