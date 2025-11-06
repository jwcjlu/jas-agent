# Chain 和 Plan 框架使用指南

本文档介绍 JAS Agent 中的两种高级代理框架：**链式框架 (Chain Framework)** 和 **计划框架 (Plan Framework)**。

## 目录

- [链式框架 (Chain Framework)](#链式框架-chain-framework)
  - [核心概念](#核心概念)
  - [使用场景](#使用场景)
  - [快速开始](#快速开始)
  - [高级功能](#高级功能)
- [计划框架 (Plan Framework)](#计划框架-plan-framework)
  - [核心概念](#核心概念-1)
  - [使用场景](#使用场景-1)
  - [快速开始](#快速开始-1)
  - [高级功能](#高级功能-1)
- [对比与选择](#对比与选择)

---

## 链式框架 (Chain Framework)

### 核心概念

链式框架允许你将多个 Agent 按照预定义的流程串联起来，前一个 Agent 的输出会作为下一个 Agent 的输入。这类似于流水线或工作流的概念。

**关键特性：**
- ✅ 预定义的执行流程
- ✅ 节点间的数据传递
- ✅ 支持条件分支
- ✅ 输出转换功能
- ✅ 灵活的流程编排

**核心组件：**

```go
// ChainNode - 链式节点
type ChainNode struct {
    Name        string                   // 节点名称
    Agent       Agent                    // 执行的Agent
    Transform   func(string) string      // 输出转换函数
    Condition   func(string) bool        // 执行条件
    MaxSteps    int                      // 最大步数
    NextNodes   []*ChainNode            // 下一个节点（支持分支）
    Description string                   // 节点描述
}
```

### 使用场景

链式框架适合以下场景：

1. **多阶段数据处理**：数据收集 → 清洗 → 分析 → 报告
2. **工作流自动化**：审批流程、订单处理等
3. **条件路由**：根据中间结果选择不同的处理路径
4. **管道式任务**：需要按固定顺序执行的任务序列

### 快速开始

#### 示例 1: 简单线性链

```go
package main

import (
    "jas-agent/agent"
    "jas-agent/llm"
    "github.com/sashabaranov/go-openai"
)

func main() {
    // 创建上下文
    chat := llm.NewChat(&llm.Config{
        ApiKey:  "your-api-key",
        BaseURL: "your-base-url",
    })
    
    context := agent.NewContext(
        agent.WithModel(openai.GPT3Dot5Turbo),
        agent.WithChat(chat),
    )

    // 构建链式Agent
    builder := agent.NewChainBuilder(context)
    
    // 添加节点：查询狗狗信息 -> 计算总和
    builder.
        AddNode("query_weights", agent.ReactAgentType, 5).
        AddNode("calculate_total", agent.ReactAgentType, 3).
        Link("query_weights", "calculate_total")

    chainAgent := builder.Build()
    executor := agent.NewChainAgentExecutor(context, chainAgent)

    result := executor.Run("我有一只边境牧羊犬和一只苏格兰梗，它们的总体重是多少？")
    fmt.Printf("最终结果: %s\n", result)
}
```

#### 示例 2: 带转换函数的链

```go
// 构建链式Agent
builder := agent.NewChainBuilder(context)

builder.
    AddNode("query_info", agent.ReactAgentType, 5).
    AddNode("summarize", agent.ReactAgentType, 3).
    Link("query_info", "summarize")

// 为第一个节点设置转换函数：提取关键信息
builder.SetTransform("query_info", func(output string) string {
    // 自定义转换逻辑
    if strings.Contains(output, "磅") || strings.Contains(output, "lbs") {
        return "已获取体重信息: " + output
    }
    return output
})

chainAgent := builder.Build()
executor := agent.NewChainAgentExecutor(context, chainAgent)

result := executor.Run("玩具贵宾犬的平均体重是多少？")
```

#### 示例 3: 条件分支链

```go
// 构建带条件分支的链
builder := agent.NewChainBuilder(context)

builder.
    AddNode("check_value", agent.ReactAgentType, 3).
    AddNode("large_process", agent.ReactAgentType, 3).
    AddNode("small_process", agent.ReactAgentType, 3).
    Link("check_value", "large_process").
    Link("check_value", "small_process")

// 设置条件：如果结果表示大型犬，走large_process
builder.SetCondition("large_process", func(input string) bool {
    return strings.Contains(input, "大") || 
           strings.Contains(input, "50") ||
           strings.Contains(input, "60")
})

// 设置条件：如果结果表示小型犬，走small_process
builder.SetCondition("small_process", func(input string) bool {
    return strings.Contains(input, "小") || 
           strings.Contains(input, "10") ||
           strings.Contains(input, "7")
})

chainAgent := builder.Build()
executor := agent.NewChainAgentExecutor(context, chainAgent)

result := executor.Run("玩具贵宾犬的平均体重是多少？")
```

### 高级功能

#### 1. 智能路由 Agent

使用 AI 自动选择处理路径：

```go
// 定义不同的处理路由
routes := map[string]Agent{
    "sql":   sqlAgent,
    "react": reactAgent,
    "plan":  planAgent,
}

// 路由描述
descriptions := map[string]string{
    "sql":   "处理数据库查询任务",
    "react": "处理通用推理任务",
    "plan":  "处理需要规划的复杂任务",
}

// 创建AI路由Agent
routeAgent := agent.NewAIRouteAgent(context, executor, routes, descriptions)

result := executor.Run("查询数据库中的用户数量")
// AI会自动选择SQL路由
```

#### 2. 自定义路由函数

```go
// 基于规则的路由
routeFunc := func(input string) string {
    if strings.Contains(input, "数据库") || strings.Contains(input, "SQL") {
        return "sql"
    } else if strings.Contains(input, "计划") || strings.Contains(input, "步骤") {
        return "plan"
    }
    return "react"
}

routeAgent := agent.NewRouteAgent(context, executor, routeFunc, routes)
```

#### 3. 多步骤数据处理链

```go
builder := agent.NewChainBuilder(context)

// 数据处理流程：收集 -> 清洗 -> 分析 -> 报告
builder.
    AddNode("collect", agent.ReactAgentType, 5).
    AddNode("clean", agent.ReactAgentType, 3).
    AddNode("analyze", agent.ReactAgentType, 5).
    AddNode("report", agent.ReactAgentType, 3).
    Link("collect", "clean").
    Link("clean", "analyze").
    Link("analyze", "report")

// 为每个节点设置转换函数
builder.SetTransform("collect", func(output string) string {
    return fmt.Sprintf("[已收集] %s", output)
})

builder.SetTransform("clean", func(output string) string {
    return fmt.Sprintf("[已清洗] %s", output)
})

builder.SetTransform("analyze", func(output string) string {
    return fmt.Sprintf("[已分析] %s", output)
})

chainAgent := builder.Build()
executor := agent.NewChainAgentExecutor(context, chainAgent)

result := executor.Run("收集并分析三种狗狗的体重数据")
```

---

## 计划框架 (Plan Framework)

### 核心概念

计划框架采用 "先规划，后执行" 的策略。它会首先分析任务，生成完整的执行计划，然后按照计划逐步执行。

**关键特性：**
- ✅ 先规划再执行
- ✅ 支持步骤依赖
- ✅ 可视化执行计划
- ✅ 自动错误处理
- ✅ 支持重新规划

**核心组件：**

```go
// PlanStep - 计划步骤
type PlanStep struct {
    ID          int      `json:"id"`
    Description string   `json:"description"`
    Tool        string   `json:"tool"`
    Input       string   `json:"input"`
    Status      string   `json:"status"`       // pending, executing, completed, failed
    Result      string   `json:"result"`
    Dependencies []int   `json:"dependencies"` // 依赖的步骤ID
}

// Plan - 执行计划
type Plan struct {
    Goal     string      `json:"goal"`
    Steps    []*PlanStep `json:"steps"`
    Created  time.Time   `json:"created"`
    Updated  time.Time   `json:"updated"`
    Status   string      `json:"status"` // planning, executing, completed, failed
}
```

### 使用场景

计划框架适合以下场景：

1. **复杂多步骤任务**：需要多个工具协同完成的任务
2. **有依赖关系的任务**：某些步骤必须在其他步骤完成后才能执行
3. **需要全局优化的任务**：提前规划可以避免重复操作
4. **可能需要调整的任务**：执行过程中可能需要修改计划

### 快速开始

#### 示例 1: 基本计划执行

```go
package main

import (
    "jas-agent/agent"
    "jas-agent/llm"
    "github.com/sashabaranov/go-openai"
)

func main() {
    // 创建上下文
    chat := llm.NewChat(&llm.Config{
        ApiKey:  "your-api-key",
        BaseURL: "your-base-url",
    })
    
    context := agent.NewContext(
        agent.WithModel(openai.GPT3Dot5Turbo),
        agent.WithChat(chat),
    )

    // 创建Plan Agent执行器（不启用重新规划）
    executor := agent.NewPlanAgentExecutor(context, false)

    result := executor.Run("计算15 + 27的结果，然后乘以3")
    fmt.Printf("最终结果:\n%s\n", result)
}
```

**执行过程：**

```
📋 Generating execution plan...

📝 Generated Plan:
Goal: 计算(15 + 27) * 3
Steps:
  1. 计算15 + 27
  2. 将结果乘以3

⚙️  Executing step 1: 计算15 + 27
✅ Step 1 completed: 42

⚙️  Executing step 2: 将结果乘以3
✅ Step 2 completed: 126

📊 Generating summary...
最终结果: 126
```

#### 示例 2: 带依赖的复杂计划

```go
// 创建Plan Agent执行器
executor := agent.NewPlanAgentExecutor(context, false)

result := executor.Run("我有3只狗，分别是border collie、scottish terrier和toy poodle。请查询它们的平均体重，然后计算总重量")

// 生成的计划可能如下：
// Step 1: 查询border collie的平均体重 (dependencies: [])
// Step 2: 查询scottish terrier的平均体重 (dependencies: [])
// Step 3: 查询toy poodle的平均体重 (dependencies: [])
// Step 4: 计算总重量 (dependencies: [1, 2, 3])
```

#### 示例 3: 启用重新规划

```go
// 创建Plan Agent执行器（启用重新规划）
executor := agent.NewPlanAgentExecutor(context, true)

result := executor.Run("查询拉布拉多和金毛的体重差异，并计算平均值")

// 如果某个步骤失败，系统会自动重新规划
```

### 高级功能

#### 1. 步骤依赖管理

计划框架支持复杂的依赖关系：

```json
{
  "goal": "数据分析任务",
  "steps": [
    {
      "id": 1,
      "description": "列出所有表",
      "tool": "list_tables",
      "input": "",
      "dependencies": []
    },
    {
      "id": 2,
      "description": "查看users表结构",
      "tool": "tables_schema",
      "input": "users",
      "dependencies": [1]
    },
    {
      "id": 3,
      "description": "查看orders表结构",
      "tool": "tables_schema",
      "input": "orders",
      "dependencies": [1]
    },
    {
      "id": 4,
      "description": "执行关联查询",
      "tool": "execute_sql",
      "input": "SELECT u.name, COUNT(o.id) FROM users u LEFT JOIN orders o ON u.id=o.user_id GROUP BY u.id",
      "dependencies": [2, 3]
    }
  ]
}
```

#### 2. 依赖引用

在步骤的输入中可以引用之前步骤的结果：

```json
{
  "id": 4,
  "description": "计算总重量",
  "tool": "calculator",
  "input": "${step.1} + ${step.2} + ${step.3}",
  "dependencies": [1, 2, 3]
}
```

系统会自动将 `${step.1}` 替换为步骤1的实际执行结果。

#### 3. 自动重新规划

启用重新规划后，如果某个步骤失败，系统会：

1. 分析失败原因
2. 收集当前执行状态
3. 生成新的执行计划
4. 继续执行新计划

```go
// 启用重新规划
executor := agent.NewPlanAgentExecutor(context, true)

// 系统会在遇到问题时自动调整计划
result := executor.Run("复杂的多步骤任务")
```

#### 4. 计划可视化

计划框架会在执行过程中输出详细的进度信息：

```
📋 Generating execution plan...

📝 Generated Plan:
Goal: 查询三只狗的总体重
Steps:
  1. 查询边境牧羊犬体重
  2. 查询苏格兰梗体重
  3. 查询玩具贵宾犬体重
  4. 计算总重量 (depends on: [1, 2, 3])

⚙️  Executing step 1: 查询边境牧羊犬体重
✅ Step 1 completed: 37 lbs

⚙️  Executing step 2: 查询苏格兰梗体重
✅ Step 2 completed: 20 lbs

⚙️  Executing step 3: 查询玩具贵宾犬体重
✅ Step 3 completed: 7 lbs

⚙️  Executing step 4: 计算总重量
✅ Step 4 completed: 64

📊 Generating summary...
三只狗的总体重约为64磅。
```

---

## 对比与选择

### 功能对比

| 特性 | Chain Framework | Plan Framework |
|------|----------------|----------------|
| 执行方式 | 流式执行 | 先规划后执行 |
| 灵活性 | 预定义流程 | 动态生成计划 |
| 复杂度 | 中等 | 较高 |
| 适合任务 | 固定流程 | 复杂多步骤 |
| 错误处理 | 条件分支 | 重新规划 |
| 依赖管理 | 线性依赖 | 复杂依赖图 |
| 可视化 | 节点流程 | 执行计划 |

### 选择建议

**使用 Chain Framework 当：**
- ✅ 任务流程相对固定
- ✅ 需要精确控制每个步骤
- ✅ 有明确的数据转换需求
- ✅ 需要条件分支功能
- ✅ 想要更好的性能（预定义流程）

**使用 Plan Framework 当：**
- ✅ 任务较复杂，步骤不确定
- ✅ 需要 AI 自动分解任务
- ✅ 步骤间有复杂依赖关系
- ✅ 可能需要动态调整计划
- ✅ 希望看到完整的执行计划

### 混合使用

两种框架可以配合使用：

```go
// 在Chain的某个节点中使用Plan Agent
builder := agent.NewChainBuilder(context)

// 创建Plan Agent
planAgent := agent.NewPlanAgent(context, executor, true)

// 在Chain中添加节点
builder.
    AddNode("preprocess", agent.ReactAgentType, 3).
    AddNode("complex_task", agent.PlanAgentType, 20).  // 使用Plan处理复杂任务
    AddNode("postprocess", agent.ReactAgentType, 3).
    Link("preprocess", "complex_task").
    Link("complex_task", "postprocess")

chainAgent := builder.Build()
```

---

## 完整示例

### Chain Framework 完整示例

查看 `examples/chain/main.go` 获取完整的链式框架示例。

运行示例：
```bash
cd examples/chain
go run . -apiKey YOUR_API_KEY -baseUrl YOUR_BASE_URL
```

### Plan Framework 完整示例

查看 `examples/plan/main.go` 获取完整的计划框架示例。

运行示例：
```bash
cd examples/plan
go run . -apiKey YOUR_API_KEY -baseUrl YOUR_BASE_URL
```

---

## 最佳实践

### Chain Framework 最佳实践

1. **合理设置节点步数**：避免单个节点执行时间过长
2. **使用转换函数**：清理和格式化节点输出
3. **设计清晰的条件**：确保分支逻辑正确
4. **避免循环依赖**：保持链式结构的单向性
5. **记录节点职责**：使用Description字段说明节点功能

### Plan Framework 最佳实践

1. **提供清晰的任务描述**：帮助AI生成更准确的计划
2. **合理使用依赖**：明确步骤间的先后关系
3. **启用重新规划**：对于可能失败的任务
4. **限制步骤数量**：避免计划过于复杂
5. **验证工具可用性**：确保计划中的工具都已注册

---

## 故障排查

### Chain Framework 常见问题

**Q: 链式执行卡住不动？**
A: 检查条件函数是否正确，确保至少有一个分支满足条件。

**Q: 节点间数据传递失败？**
A: 检查Transform函数，确保返回值格式正确。

**Q: 执行顺序不对？**
A: 检查Link调用顺序，确保节点正确连接。

### Plan Framework 常见问题

**Q: 计划生成失败？**
A: 检查可用工具列表，确保有足够的工具完成任务。

**Q: 步骤执行失败？**
A: 启用重新规划功能，或检查工具输入格式。

**Q: 依赖解析错误？**
A: 检查依赖引用格式（`${step.X}`），确保引用的步骤已完成。

---

## 总结

Chain 和 Plan 框架为 JAS Agent 提供了强大的任务编排能力：

- **Chain Framework** 适合流程化、确定性的任务
- **Plan Framework** 适合复杂、需要规划的任务

根据你的具体需求选择合适的框架，或者将两者结合使用以获得最佳效果。

如有问题，欢迎提交 Issue 或查看示例代码！


