# JAS Agent

一个基于 ReAct (Reasoning and Acting) 框架的 Go 语言 AI 代理系统，支持工具调用、逐步推理、MCP 协议集成和 SQL 查询生成。

## 特性

- 🤖 **ReAct 框架**: 实现思考-行动-观察的循环推理
- 📝 **SummaryAgent**: 自动总结执行过程，提供清晰答案
- 🗄️ **SQL Agent**: 专业的 SQL 查询生成和执行代理
- 🛠️ **工具系统**: 可扩展的工具管理器和执行器
- 🔌 **MCP 支持**: 集成 [Model Context Protocol](https://github.com/metoro-io/mcp-golang) 工具发现
- 💬 **LLM 集成**: 支持 OpenAI Function Calling 和文本补全
- 🧠 **内存管理**: 对话历史和上下文管理
- 🔧 **模块化设计**: 清晰的架构，易于扩展

## 架构

```
jas-agent/
├── agent/              # 代理核心
│   ├── agent.go        # Agent 接口和执行器
│   ├── agent_context.go # 上下文管理
│   ├── base_react.go   # BaseReact 基础类
│   ├── react_agent.go  # ReAct 代理实现
│   ├── sql_agent.go    # SQL 代理实现
│   └── summary_agent.go # 总结代理实现
├── core/               # 核心类型和接口
│   ├── message.go      # 消息类型
│   ├── memory.go       # 内存接口
│   ├── tool.go         # 工具接口
│   └── prompt.go       # 提示词模板
├── llm/                # LLM 集成
│   ├── chat.go         # 聊天客户端接口
│   └── types.go        # 请求响应类型
├── memory/             # 内存实现
│   └── memory.go       # 内存存储
├── tools/              # 工具实现
│   ├── tool.go         # 工具管理器
│   ├── calculator.go   # 计算器工具
│   ├── sql_tools.go    # SQL 工具集
│   └── mcp.go          # MCP 工具支持
└── examples/           # 示例代码
    ├── react/          # ReAct 示例
    │   ├── main.go     # 主程序
    │   └── tools/      # 示例工具
    │       └── tool.go # 狗狗体重查询工具
    └── sql/            # SQL Agent 示例
        ├── main.go     # SQL Agent 主程序
        └── README.md   # SQL 示例文档
```

## 快速开始

### 1. 安装依赖

```bash
go get github.com/sashabaranov/go-openai
go get github.com/metoro-io/mcp-golang
go get go.starlark.net/starlark
go get github.com/go-sql-driver/mysql  # SQL Agent 需要
```

或者

```bash
go mod tidy
```

### 2. 运行示例

**ReAct Agent 示例:**

```bash
cd examples/react
go run . -apiKey YOUR_API_KEY -baseUrl YOUR_BASE_URL
```

**SQL Agent 示例:**

```bash
cd examples/sql
go run . -apiKey YOUR_API_KEY -baseUrl YOUR_BASE_URL -dsn "root:password@tcp(localhost:3306)/testdb"
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

### Agent 类型

- **ReactAgent**: 通用推理代理，支持多种工具调用
- **SQLAgent**: SQL 查询专家，专注于数据库查询任务
- **SummaryAgent**: 总结代理，提供执行过程总结

### BaseReact 基础类

`BaseReact` 是 ReactAgent 和 SQLAgent 的共享基础实现，封装了核心的 ReAct 循环逻辑：

- **Thought()**: 调用 LLM 进行思考，解析工具调用
- **Action()**: 执行工具调用，添加观察结果
- **Step()**: 协调思考和行动

**特性**:
- 支持 Function Calling（MCP 工具）
- 支持文本解析（普通工具）
- 统一的错误处理
- 自动状态管理

### ReAct 循环

1. **思考 (Thought)**: 分析当前情况，决定下一步行动
2. **行动 (Action)**: 执行工具调用或完成任务
3. **观察 (Observation)**: 获取行动结果，为下一步思考提供信息
4. **总结 (Summary)**: SummaryAgent 自动总结执行过程，提供清晰答案

### 工具系统

#### 工具接口

```go
type Tool interface {
    Name() string
    Description() string
    Handler(ctx context.Context, input string) (string, error)
    Input() any          // JSON Schema（用于 Function Calling）
    Type() ToolType      // Normal 或 Mcp
}
```

#### 工具类型

- **Normal**: 普通工具（通过系统提示词告知 LLM，文本解析调用）
- **Mcp**: MCP 工具（通过 OpenAI Function Calling 调用）

#### 定义工具

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

#### 内置工具

**普通工具:**
- **Calculator**: 数学表达式计算（使用 Starlark 求值器）
- **AverageDogWeight**: 狗狗品种平均体重查询

**SQL 工具集:**
- **list_tables**: 列出数据库中的所有表
- **tables_schema**: 获取指定表的结构信息（列名、数据类型、约束等）
- **execute_sql**: 执行 SQL 查询并返回结果（仅支持 SELECT）

### MCP 工具支持

使用 [mcp-golang](https://github.com/metoro-io/mcp-golang) 库集成 MCP 工具：

#### 创建 MCP 工具管理器

```go
import "jas-agent/tools"

// 创建 MCP 工具管理器（HTTP Transport）
mcpManager, err := tools.NewMCPToolManager("my-mcp", "http://localhost:8080/mcp")
if err != nil {
    log.Fatal(err)
}

// 启动工具发现（后台自动刷新，每5秒）
mcpManager.Start()
```

#### MCP 工具特性

1. **自动发现**: 定期刷新工具列表
2. **双缓冲机制**: 使用原子操作实现无锁切换
3. **工具前缀**: 自动添加前缀避免命名冲突（格式：`name@toolName`）
4. **Function Calling**: MCP 工具通过 OpenAI Function Calling 调用
5. **HTTP Transport**: 使用 HTTP 协议与 MCP 服务器通信

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
// ReAct Agent（默认10步）
executor := agent.NewAgentExecutor(context)

// SQL Agent（默认15步，适应复杂查询）
executor := agent.NewSQLAgentExecutor(context, "MySQL: dbname")
```

## SQL Agent 详解

### 核心职责

SQL Agent 专注于生成准确、高效的 SQL 查询，具备以下能力：

1. **Schema 理解**: 自动探索数据库结构
2. **SQL 生成**: 基于自然语言生成标准 SQL
3. **查询执行**: 安全执行查询并返回结果
4. **结果分析**: 智能解析和总结查询结果

### 工作流程

```
用户问题 → 了解表结构 → 编写SQL → 执行查询 → 分析结果 → 提供答案
```

### 可用工具

| 工具 | 功能 | 输入 | 输出 |
|------|------|------|------|
| list_tables | 列出所有表 | 无 | 表名列表 |
| tables_schema | 获取表结构 | 表名（逗号分隔） | 列信息、类型、约束 |
| execute_sql | 执行SQL查询 | SQL语句（SELECT） | 查询结果（JSON） |

### 安全特性

- ✅ **只读模式**: 仅允许 SELECT 查询
- ✅ **SQL 验证**: 检查查询类型，拒绝 INSERT/UPDATE/DELETE
- ✅ **错误处理**: 完善的错误提示和异常处理
- ✅ **结果限制**: 建议使用 LIMIT 控制返回数据量

### 使用示例

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "jas-agent/agent"
    "jas-agent/tools"
)

func main() {
    // 1. 连接数据库
    db, _ := sql.Open("mysql", "user:pass@tcp(localhost:3306)/dbname")
    defer db.Close()
    
    // 2. 注册 SQL 工具
    sqlConn := &tools.SQLConnection{DB: db}
    tools.RegisterSQLTools(sqlConn)
    
    // 3. 创建上下文
    context := agent.NewContext(
        agent.WithModel(openai.GPT3Dot5Turbo),
        agent.WithChat(chat),
    )
    
    // 4. 创建 SQL Agent 执行器
    executor := agent.NewSQLAgentExecutor(context, "MySQL: dbname")
    
    // 5. 执行查询
    result := executor.Run("查询：查询销售额最高的前10个产品")
    fmt.Println(result)
}
```

### 查询示例

**简单查询:**
```
问题: 用户表有多少条记录？
执行: list_tables[] → tables_schema[users] → execute_sql[SELECT COUNT(*) FROM users]
结果: 用户表共有 5 条记录
```

**关联查询:**
```
问题: 查询每个用户的订单数量
执行: tables_schema[users,orders] → execute_sql[
    SELECT u.username, COUNT(o.id) as order_count 
    FROM users u 
    LEFT JOIN orders o ON u.id = o.user_id 
    GROUP BY u.id
]
```

**聚合查询:**
```
问题: 统计每月的订单总金额
执行: execute_sql[
    SELECT DATE_FORMAT(order_date, '%Y-%m') as month, SUM(amount) 
    FROM orders 
    GROUP BY month 
    ORDER BY month DESC
]
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

## 示例场景

### 数学计算

```go
executor := agent.NewAgentExecutor(context)
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
// 创建 MCP 工具管理器
mcpManager, _ := tools.NewMCPToolManager("weather-mcp", "http://weather-api:8080/mcp")
mcpManager.Start()

result := executor.Run("北京的天气怎么样？")
// LLM 会通过 Function Calling 调用 MCP 工具
```

### SQL 查询

```go
import "database/sql"
import _ "github.com/go-sql-driver/mysql"

// 连接数据库
db, _ := sql.Open("mysql", "root:password@tcp(localhost:3306)/testdb")
defer db.Close()

// 注册 SQL 工具
sqlConn := &tools.SQLConnection{DB: db}
tools.RegisterSQLTools(sqlConn)

// 创建 SQL Agent 执行器
executor := agent.NewSQLAgentExecutor(context, "MySQL Database: testdb")

// 查询示例
result := executor.Run("查询：查询每个用户的订单总金额")

// 执行流程：
// 1. Thought: 需要了解数据库表结构
// 2. Action: list_tables[] 
// 3. Observation: Tables: users, orders
// 4. Thought: 需要查看 users 和 orders 表的结构
// 5. Action: tables_schema[users,orders]
// 6. Observation: [表结构详情]
// 7. Thought: 编写 SQL 查询
// 8. Action: execute_sql[SELECT u.username, SUM(o.amount) as total FROM users u LEFT JOIN orders o ON u.id=o.user_id GROUP BY u.id]
// 9. Observation: [查询结果]
// 10. Summary: 根据查询结果，每个用户的订单总金额为...
```

## 工具调用机制

### 混合调用模式

JAS Agent 支持两种工具调用方式：

#### 1. 文本解析（Normal 工具）

- 通过系统提示词列出工具名称和描述
- LLM 输出格式：`Action: toolName[input]`
- 正则表达式解析并执行

**示例:**
```
Thought: 我需要计算15和27的和
Action: calculator[15 + 27]
```

#### 2. Function Calling（Mcp 工具）

- 通过 OpenAI Tools/Function Calling 机制
- 自动生成工具的 JSON Schema
- LLM 直接调用，返回结构化参数

**示例:**
```json
{
  "tool_calls": [{
    "function": {
      "name": "weather-mcp@get_weather",
      "arguments": "{\"city\": \"Beijing\"}"
    }
  }]
}
```

### 工具执行流程

```go
// BaseReact.Action()
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

### MCP 工具命名

MCP 工具使用前缀机制避免命名冲突：

```
格式: {manager_name}@{tool_name}
示例: weather-mcp@get_weather
```

## 扩展开发

### 添加新的 Agent 类型

使用 BaseReact 快速创建新 Agent：

```go
type MyAgent struct {
    *BaseReact
    systemPrompt string
}

func (a *MyAgent) Type() AgentType {
    return "MyAgent"
}

func NewMyAgent(context *Context, executor *AgentExecutor) Agent {
    // 构建系统提示词
    systemPrompt := "你是一个..."
    context.memory.AddMessage(core.Message{
        Role:    core.MessageRoleSystem,
        Content: systemPrompt,
    })
    
    return &MyAgent{
        BaseReact:    NewBaseReact(context, executor),
        systemPrompt: systemPrompt,
    }
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

## 依赖

- `github.com/sashabaranov/go-openai`: OpenAI API 客户端
- `github.com/metoro-io/mcp-golang`: Model Context Protocol 支持
- `go.starlark.net/starlark`: 数学表达式计算
- `github.com/go-sql-driver/mysql`: MySQL 数据库驱动（SQL Agent）

## 故障排查

### 找不到工具

**问题**: LLM 输出 `Action: search[...]` 但系统没有该工具

**原因**:
1. MCP 工具未成功发现/注册
2. 工具名称不匹配
3. 系统提示词中工具列表为空

**解决方案**:
```go
// 1. 确保 MCP 工具管理器已启动
mcpManager, _ := tools.NewMCPToolManager("my-mcp", "http://localhost:8080/mcp")
mcpManager.Start()

// 2. 检查工具列表
tools := context.GetToolManager().AvailableTools()
for _, tool := range tools {
    fmt.Printf("Tool: %s - %s (Type: %v)\n", 
        tool.Name(), tool.Description(), tool.Type())
}

// 3. 确保工具名称一致
// MCP 工具会自动添加前缀: "my-mcp@toolName"
```

### MCP 连接失败

**问题**: `failed to initialize MCP client`

**解决方案**:
- 确认 MCP 服务器已启动并可访问
- 检查 HTTP 端点是否正确
- 查看 MCP 服务器日志

### SQL 执行失败

**问题**: `only SELECT queries are allowed`

**解决方案**:
- SQL Agent 仅支持 SELECT 查询
- 不支持 INSERT/UPDATE/DELETE 操作
- 检查 SQL 语句是否以 SELECT 开头

## 性能优化

### 内存管理

- 使用 `memory.NewMemory()` 创建轻量级内存实例
- 定期调用 `memory.Clear()` 清理历史消息

### 工具优化

- 限制工具数量，避免系统提示过长
- 使用 `FilterFunc` 过滤不相关工具
- MCP 工具使用双缓冲，无需手动刷新

### 执行优化

- 合理设置 `maxSteps` 避免无限循环
- 使用状态管理及时终止执行
- 对于 SQL 查询，建议增加步骤限制

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

### 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 更新日志

### v1.3.0
- 添加 SQL Agent 专业数据库查询代理
- 实现 SQL 工具集（list_tables, tables_schema, execute_sql）
- 支持 MySQL 数据库（可扩展其他数据库）
- 提供完整的 SQL 查询工作流程
- 添加安全限制（仅 SELECT 查询）
- 重构为 BaseReact 基础类，提高代码复用性

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

## 相关资源

- [ReAct 论文](https://arxiv.org/abs/2210.03629)
- [Model Context Protocol](https://github.com/metoro-io/mcp-golang)
- [OpenAI Function Calling](https://platform.openai.com/docs/guides/function-calling)
