# Agent 管理指南

## 概述

JAS Agent 系统现在支持完整的 Agent 管理功能，允许您创建、配置、管理多个不同用途的 AI Agent。每个 Agent 可以配置不同的框架类型、模型、提示词和绑定的 MCP 服务。

## 核心概念

### Agent 配置项

每个 Agent 包含以下配置：

| 配置项 | 说明 | 必填 |
|--------|------|------|
| 名称 (name) | Agent 的唯一标识名称 | ✅ |
| 框架类型 (framework) | react / plan / chain | ✅ |
| 描述 (description) | Agent 功能描述 | ❌ |
| 系统提示词 (system_prompt) | 自定义的系统提示 | ❌ |
| 模型 (model) | 使用的 LLM 模型 | ✅ |
| 最大步数 (max_steps) | 执行的最大步骤数 | ✅ |
| MCP 服务 (mcp_services) | 绑定的 MCP 服务列表 | ❌ |

### Agent 框架类型

#### ReAct Agent
- **特点**: 推理与行动循环 (Reasoning and Acting)
- **适用场景**: 通用问答、工具调用、复杂推理
- **执行流程**: 思考 → 行动 → 观察 → 思考...

#### Plan Agent  
- **特点**: 先规划后执行
- **适用场景**: 复杂多步骤任务、需要提前规划的场景
- **执行流程**: 生成计划 → 执行步骤1 → 执行步骤2 → ...

#### Chain Agent
- **特点**: 链式调用多个 Agent
- **适用场景**: 工作流编排、多阶段任务
- **执行流程**: Agent1 → Agent2 → Agent3 → ...

## 使用指南

### 1. 数据库初始化

首先需要创建 MySQL 数据库：

```bash
# 连接到 MySQL
mysql -u root -p

# 创建数据库
CREATE DATABASE jas_agent CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE jas_agent;

# 导入表结构
source scripts/schema.sql;
```

### 2. 启动服务器

启动服务器时需要指定数据库 DSN：

```bash
cd cmd/server

# Windows
go run main.go -apiKey YOUR_API_KEY -baseUrl YOUR_BASE_URL -dsn "root:password@tcp(localhost:3306)/jas_agent"

# Linux/Mac
go run main.go -apiKey YOUR_API_KEY -baseUrl YOUR_BASE_URL -dsn "root:password@tcp(localhost:3306)/jas_agent"
```

参数说明：
- `-apiKey`: OpenAI API密钥
- `-baseUrl`: OpenAI API基础URL
- `-dsn`: MySQL数据源名称（可选，不提供则无数据库功能）

### 3. Web 界面管理 Agent

#### 3.1 访问 Agent 管理

1. 访问 `http://localhost:8080` 打开 Web 界面
2. 点击 **"🤖 管理 Agent"** 按钮
3. 进入 Agent 管理界面

#### 3.2 创建 Agent

1. 点击 **"➕ 添加 Agent"**
2. 填写 Agent 配置：
   - **名称**: 例如 "默认助手"
   - **框架类型**: 选择 react / plan / chain
   - **描述**: 简要说明 Agent 用途
   - **系统提示词**: 自定义 Agent 行为（可选）
   - **模型**: 例如 gpt-3.5-turbo
   - **最大步数**: 建议 10-20
   - **MCP 服务**: 选择需要绑定的 MCP 服务
3. 点击 **"保存"**

#### 3.3 编辑 / 删除 Agent

- 点击 Agent 卡片上的 **"✏️ 编辑"** 按钮修改配置
- 点击 **"🗑️ 删除"** 按钮删除 Agent（需确认）

#### 3.4 使用 Agent 进行对话

1. 在主界面的下拉框中选择要使用的 Agent
2. 输入问题，点击发送
3. 系统会使用所选 Agent 的配置执行任务

⚠️ **注意**: 必须选择一个 Agent 才能发送消息！

### 4. API 管理 Agent

#### 4.1 列出所有 Agent

```bash
curl http://localhost:8080/api/agents
```

响应示例：
```json
{
  "agents": [
    {
      "id": 1,
      "name": "默认助手",
      "framework": "react",
      "description": "通用智能助手",
      "system_prompt": "",
      "max_steps": 10,
      "model": "gpt-3.5-turbo",
      "mcp_services": ["infra-mcp"],
      "created_at": "2025-11-03 10:00:00",
      "updated_at": "2025-11-03 10:00:00",
      "is_active": true
    }
  ]
}
```

#### 4.2 创建 Agent

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "数据分析师",
    "framework": "plan",
    "description": "专业的数据分析助手",
    "system_prompt": "你是一个专业的数据分析师",
    "max_steps": 20,
    "model": "gpt-4",
    "mcp_services": ["data-tools"]
  }'
```

#### 4.3 更新 Agent

```bash
curl -X PUT http://localhost:8080/api/agents/1 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "默认助手",
    "framework": "react",
    "description": "更新后的描述",
    "max_steps": 15
  }'
```

#### 4.4 删除 Agent

```bash
curl -X DELETE http://localhost:8080/api/agents/1
```

#### 4.5 使用 Agent 进行对话

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "query": "你好",
    "agent_id": 1,
    "session_id": "test_session"
  }'
```

## 最佳实践

### 1. Agent 命名规范

- 使用清晰、描述性的名称
- 建议格式: "{用途}{类型}"
  - 例如: "数据分析助手"、"代码审查专家"、"客服机器人"

### 2. 框架选择建议

| 场景 | 推荐框架 | 原因 |
|------|----------|------|
| 通用对话 | ReAct | 灵活、适应性强 |
| 复杂任务 | Plan | 提前规划，执行更有条理 |
| 工作流 | Chain | 串联多个专业Agent |

### 3. 系统提示词编写

**好的提示词示例**:
```
你是一个专业的数据分析师，擅长：
1. 数据清洗和预处理
2. 统计分析和可视化
3. 洞察提取和报告撰写

请始终保持专业、准确、客观的态度。
```

**避免**:
- 过于模糊的描述
- 相互矛盾的指令
- 过长的提示词（>500字）

### 4. MCP 服务绑定

- 只绑定 Agent 真正需要的 MCP 服务
- 避免绑定过多服务影响性能
- 定期检查服务可用性

### 5. 性能优化

- **最大步数**: 
  - 简单任务: 5-10步
  - 中等复杂: 10-20步
  - 复杂任务: 20-50步

- **模型选择**:
  - 快速响应: gpt-3.5-turbo
  - 复杂推理: gpt-4
  - 长文本: gpt-4-turbo

## 故障排除

### 问题 1: 无法创建 Agent

**可能原因**:
- 数据库未连接
- Agent 名称重复
- 无效的框架类型

**解决方法**:
1. 检查服务器启动日志
2. 确认数据库连接正常
3. 使用唯一的 Agent 名称
4. 确保框架类型为 react/plan/chain 之一

### 问题 2: Agent 不显示在下拉列表

**可能原因**:
- 数据库查询失败
- Agent 被标记为 is_active=false
- 前端未正确加载

**解决方法**:
1. 刷新页面
2. 检查浏览器控制台错误
3. 确认 Agent 的 is_active 字段为 true

### 问题 3: 使用 Agent 时报错

**可能原因**:
- 未选择 Agent
- Agent 配置的模型不可用
- 绑定的 MCP 服务离线

**解决方法**:
1. 确保选择了 Agent
2. 检查模型配置是否正确
3. 验证 MCP 服务状态

## 示例场景

### 场景 1: 创建代码审查助手

```json
{
  "name": "代码审查专家",
  "framework": "react",
  "description": "专业的代码审查和优化建议助手",
  "system_prompt": "你是一个资深的代码审查专家，请从以下方面审查代码：1. 代码质量 2. 性能优化 3. 安全问题 4. 最佳实践",
  "max_steps": 15,
  "model": "gpt-4",
  "mcp_services": ["github-tools"]
}
```

### 场景 2: 创建数据分析助手

```json
{
  "name": "数据分析师",
  "framework": "plan",
  "description": "擅长数据清洗、分析和可视化",
  "system_prompt": "你是一个专业的数据分析师，擅长使用Python进行数据分析。",
  "max_steps": 25,
  "model": "gpt-4-turbo",
  "mcp_services": ["data-tools", "visualization-tools"]
}
```

### 场景 3: 创建客服机器人

```json
{
  "name": "智能客服",
  "framework": "react",
  "description": "友好、专业的客户服务助手",
  "system_prompt": "你是一个友好、耐心的客服，请用礼貌、专业的态度回答用户问题。",
  "max_steps": 10,
  "model": "gpt-3.5-turbo",
  "mcp_services": ["knowledge-base", "ticketing-system"]
}
```

## 进阶功能

### 1. Agent 继承和扩展

可以基于现有 Agent 创建新的变体：

1. 复制现有 Agent 配置
2. 修改名称和关键配置
3. 保存为新 Agent

### 2. Agent 性能监控

通过查看对话历史可以评估 Agent 性能：

- 平均步骤数
- 成功率
- 响应时间

### 3. Agent 版本管理

建议：
- 为关键 Agent 创建备份
- 记录重要配置变更
- 定期审查和优化

## 相关文档

- [MCP 服务管理指南](./MCP_MANAGEMENT_GUIDE.md)
- [系统架构说明](../README.md)
- [API 参考文档](./API_REFERENCE.md)

## 总结

Agent 管理功能让 JAS Agent 系统更加灵活和强大。通过合理配置和使用不同的 Agent，您可以构建适应各种场景的智能助手系统。

如有问题或建议，欢迎提 Issue！

