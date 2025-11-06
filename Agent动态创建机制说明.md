# Agent 动态创建机制说明

## 🎯 核心改进

现在对话时，后台会**根据 `agent_id` 从数据库查询 Agent 配置，并动态创建相应的 Agent**，包括自动处理 SQL 和 Elasticsearch 的连接配置。

---

## 📋 完整流程

### 1. 前端发送请求

```javascript
// web/src/App.jsx
const request = {
  query: "查询用户表有多少条记录",
  agent_id: 1,              // ⭐ 必须传递
  session_id: sessionId,
  // 以下可选，用于临时覆盖配置
  agent_type: config.agentType,
  model: config.model,
  max_steps: config.maxSteps,
  system_prompt: config.systemPrompt,
  enabled_mcp_services: config.enabledMCPServices
};
```

### 2. HTTP Gateway 接收

```go
// server/http_gateway.go
type ChatRequestHTTP struct {
    Query              string   `json:"query"`
    AgentID            int32    `json:"agent_id"`              // ⭐ 必须
    SessionID          string   `json:"session_id"`
    AgentType          string   `json:"agent_type,omitempty"`  // 可选覆盖
    Model              string   `json:"model,omitempty"`
    SystemPrompt       string   `json:"system_prompt,omitempty"`
    MaxSteps           int      `json:"max_steps,omitempty"`
    EnabledMCPServices []string `json:"enabled_mcp_services,omitempty"`
}

// 转换为 gRPC 请求
grpcReq := &pb.ChatRequest{
    Query:   req.Query,
    AgentId: req.AgentID,  // ⭐ 传递到 gRPC
    // ...
}
```

### 3. gRPC 处理请求

```go
// server/grpc_server.go
func (s *AgentServer) Chat(ctx, req) {
    // 创建执行器（内部会使用 agent_id）
    executor, err := s.createExecutor(req, agentCtx)
    
    // 执行查询
    result := executor.Run(req.Query)
}
```

### 4. 动态创建 Agent

```go
// server/grpc_server.go
func (s *AgentServer) createExecutor(req, ctx) {
    // ⭐ 第1步: 验证 agent_id
    if req.AgentId == 0 {
        return nil, fmt.Errorf("agent_id is required")
    }

    // ⭐ 第2步: 从数据库查询 Agent 配置
    agentConfig, err := s.db.GetAgent(int(req.AgentId))
    // 返回: {
    //   id: 1,
    //   name: "SQL查询助手",
    //   framework: "sql",
    //   connection_config: "{\"host\":\"localhost\", ...}",
    //   max_steps: 15,
    //   model: "gpt-3.5-turbo"
    // }

    // ⭐ 第3步: 根据框架类型创建对应的 Agent
    switch agentConfig.Framework {
    case "sql":
        // 3.1 解析 SQL 连接配置
        connConfig := parseSQLConnectionConfig(agentConfig.ConnectionConfig)
        
        // 3.2 创建 MySQL 连接
        db := sql.Open("mysql", dsn)
        
        // 3.3 注册 SQL 工具
        tools.RegisterSQLTools(sqlConn)
        
        // 3.4 创建 SQL Agent
        executor = agent.NewSQLAgentExecutor(ctx, dbInfo)
        
    case "elasticsearch":
        // 3.1 解析 ES 连接配置
        esConfig := parseESConnectionConfig(agentConfig.ConnectionConfig)
        
        // 3.2 创建 ES 连接
        esConn := tools.NewESConnection(host, user, pass)
        
        // 3.3 注册 ES 工具
        tools.RegisterESTools(esConn)
        
        // 3.4 创建 ES Agent
        executor = agent.NewESAgentExecutor(ctx, clusterInfo)
        
    case "react", "plan", "chain":
        // 直接创建对应的 Executor
        executor = agent.NewXXXExecutor(ctx)
    }

    // ⭐ 第4步: 设置参数
    executor.SetMaxSteps(agentConfig.MaxSteps)
    
    // ⭐ 第5步: 返回配置好的 Executor
    return executor
}
```

### 5. 连接配置解析

```go
// server/connection_parser.go

// SQL 连接配置
func parseSQLConnectionConfig(configJSON string) (*SQLConnectionConfig, error) {
    // 解析 JSON: {"host":"localhost","port":3306,...}
    // 验证必填字段
    // 返回配置结构
}

// ES 连接配置
func parseESConnectionConfig(configJSON string) (*ESConnectionConfig, error) {
    // 解析 JSON: {"host":"http://localhost:9200",...}
    // 验证必填字段
    // 返回配置结构
}
```

---

## 🔄 数据流向图

```
┌──────────────────┐
│  前端发送请求    │
│  agent_id: 1     │
└────────┬─────────┘
         ↓
┌──────────────────────────────────────┐
│  HTTP Gateway (http_gateway.go)      │
│  接收 agent_id                        │
│  转换为 gRPC 请求                     │
└────────┬─────────────────────────────┘
         ↓
┌──────────────────────────────────────┐
│  gRPC Server (grpc_server.go)        │
│  Chat() 或 StreamChat()               │
└────────┬─────────────────────────────┘
         ↓
┌──────────────────────────────────────┐
│  createExecutor()                     │
│  ├─ 验证 agent_id != 0               │
│  ├─ db.GetAgent(agent_id)            │
│  └─ 加载配置                          │
└────────┬─────────────────────────────┘
         ↓
┌──────────────────────────────────────┐
│  数据库查询 (agent_repo.go)          │
│  SELECT * FROM agents WHERE id = 1   │
│  返回: AgentConfig{                  │
│    Framework: "sql",                 │
│    ConnectionConfig: "{...}",        │
│    MaxSteps: 15,                     │
│    ...                               │
│  }                                   │
└────────┬─────────────────────────────┘
         ↓
┌──────────────────────────────────────┐
│  根据 Framework 创建 Agent            │
│                                       │
│  if framework == "sql":               │
│    ├─ 解析 connection_config         │
│    ├─ 创建 MySQL 连接                │
│    ├─ 注册 SQL 工具                  │
│    └─ NewSQLAgentExecutor()          │
│                                       │
│  if framework == "elasticsearch":    │
│    ├─ 解析 connection_config         │
│    ├─ 创建 ES 连接                   │
│    ├─ 注册 ES 工具                   │
│    └─ NewESAgentExecutor()           │
│                                       │
│  if framework == "react/plan/chain": │
│    └─ 直接创建 Executor              │
└────────┬─────────────────────────────┘
         ↓
┌──────────────────────────────────────┐
│  配置 Executor                        │
│  - SetMaxSteps(agentConfig.MaxSteps) │
│  - 应用系统提示词                     │
│  - 启用 MCP 服务                      │
└────────┬─────────────────────────────┘
         ↓
┌──────────────────────────────────────┐
│  执行查询                             │
│  executor.Run(query)                 │
└────────┬─────────────────────────────┘
         ↓
┌──────────────────────────────────────┐
│  返回结果                             │
└──────────────────────────────────────┘
```

---

## ✅ 已修改的文件

### 1. HTTP Gateway (`server/http_gateway.go`)

**ChatRequestHTTP 结构体**:
```go
type ChatRequestHTTP struct {
    Query              string   `json:"query"`
    AgentID            int32    `json:"agent_id"`              // ⭐ 新增
    SessionID          string   `json:"session_id"`
    AgentType          string   `json:"agent_type,omitempty"`
    Model              string   `json:"model,omitempty"`
    SystemPrompt       string   `json:"system_prompt,omitempty"`
    MaxSteps           int      `json:"max_steps,omitempty"`
    Config             map[string]string `json:"config,omitempty"`
    EnabledMCPServices []string `json:"enabled_mcp_services,omitempty"` // ⭐ 新增
}
```

**请求转换**:
```go
grpcReq := &pb.ChatRequest{
    Query:              req.Query,
    AgentId:            req.AgentID,  // ⭐ 传递 agent_id
    SessionId:          req.SessionID,
    EnabledMcpServices: req.EnabledMCPServices,  // ⭐ 传递 mcp_services
    // ...
}
```

### 2. gRPC Server (`server/grpc_server.go`)

**createExecutor 函数**:
```go
func (s *AgentServer) createExecutor(req *pb.ChatRequest, ctx *agent.Context) {
    // ⭐ 必须提供 agent_id
    if req.AgentId == 0 {
        return nil, fmt.Errorf("agent_id is required")
    }

    // ⭐ 从数据库加载配置
    agentConfig, err := s.db.GetAgent(int(req.AgentId))

    // ⭐ 根据框架类型动态创建
    switch agentConfig.Framework {
    case "sql":
        // 自动创建 MySQL 连接和 SQL Agent
    case "elasticsearch":
        // 自动创建 ES 连接和 ES Agent
    case "react", "plan", "chain":
        // 创建对应的 Agent
    }
}
```

### 3. 连接配置解析器 (`server/connection_parser.go`)

**新增文件**:
```go
// SQLConnectionConfig SQL连接配置
type SQLConnectionConfig struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Database string `json:"database"`
    Username string `json:"username"`
    Password string `json:"password"`
}

// ESConnectionConfig ES连接配置
type ESConnectionConfig struct {
    Host     string `json:"host"`
    Username string `json:"username"`
    Password string `json:"password"`
}

// 解析和验证方法
func parseSQLConnectionConfig(configJSON string) (*SQLConnectionConfig, error)
func parseESConnectionConfig(configJSON string) (*ESConnectionConfig, error)
```

---

## 📊 请求示例

### HTTP API 请求

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "query": "查询用户表的记录数",
    "agent_id": 1,
    "session_id": "test_session"
  }'
```

### WebSocket 请求

```javascript
const client = new ChatStreamClient();
client.connect();
client.send({
  query: "搜索错误日志",
  agent_id: 2,
  session_id: sessionId
});
```

---

## 🎯 参数优先级

### 配置优先级规则

```
请求参数 > Agent配置 > 默认值
```

**示例**:

如果 Agent 配置为:
```json
{
  "id": 1,
  "max_steps": 15,
  "model": "gpt-3.5-turbo"
}
```

请求中可以覆盖:
```json
{
  "agent_id": 1,
  "max_steps": 20,      // ⭐ 覆盖配置的15
  "model": "gpt-4"      // ⭐ 覆盖配置的gpt-3.5
}
```

最终使用:
- `max_steps`: 20（来自请求）
- `model`: gpt-4（来自请求）

---

## 🔒 验证机制

### 1. Agent ID 验证

```go
if req.AgentId == 0 {
    return fmt.Errorf("agent_id is required")
}
```

### 2. 数据库验证

```go
agentConfig, err := s.db.GetAgent(int(req.AgentId))
if err != nil {
    return fmt.Errorf("Agent不存在或加载失败")
}
```

### 3. 连接配置验证

**SQL**:
```go
connConfig, err := parseSQLConnectionConfig(agentConfig.ConnectionConfig)
// 验证: host, port, database, username 必填
```

**Elasticsearch**:
```go
esConfig, err := parseESConnectionConfig(agentConfig.ConnectionConfig)
// 验证: host 必填
```

### 4. 连接测试

**SQL**:
```go
db, err := sql.Open("mysql", dsn)
if err := db.Ping(); err != nil {
    return fmt.Errorf("MySQL连接失败")
}
```

**Elasticsearch**:
```go
esConn := tools.NewESConnection(host, user, pass)
// 工具执行时会测试连接
```

---

## 💡 使用场景

### 场景 1: SQL 查询

```
1. 在 Web 界面创建 SQL Agent (ID=1)
   - 框架: sql
   - MySQL配置: localhost:3306/sales_db

2. 对话请求
   POST /api/chat
   {
     "agent_id": 1,
     "query": "查询本月销售额"
   }

3. 后台自动执行
   ✅ 加载 Agent 配置
   ✅ 解析 MySQL 连接配置
   ✅ 创建数据库连接
   ✅ 注册 SQL 工具
   ✅ 创建 SQL Agent
   ✅ 执行查询
```

### 场景 2: Elasticsearch 搜索

```
1. 在 Web 界面创建 ES Agent (ID=2)
   - 框架: elasticsearch
   - ES配置: http://localhost:9200

2. 对话请求
   POST /api/chat
   {
     "agent_id": 2,
     "query": "搜索今天的错误日志"
   }

3. 后台自动执行
   ✅ 加载 Agent 配置
   ✅ 解析 ES 连接配置
   ✅ 创建 ES HTTP 客户端
   ✅ 注册 ES 工具
   ✅ 创建 ES Agent
   ✅ 执行搜索
```

### 场景 3: 多 Agent 切换

```
用户创建了多个 Agent:
- ID=1: "销售数据助手" (sql)
- ID=2: "日志分析助手" (elasticsearch)
- ID=3: "通用助手" (react)

对话时可以灵活切换:
- 查询销售数据 → agent_id=1
- 分析日志 → agent_id=2
- 通用问题 → agent_id=3
```

---

## 🛡️ 错误处理

### 1. Agent 不存在

```
请求: { "agent_id": 999, "query": "..." }
响应: { 
  "success": false, 
  "error": "failed to load agent config: Agent不存在: id=999"
}
```

### 2. 连接配置缺失

```
Agent框架=sql，但connection_config为空
响应: {
  "success": false,
  "error": "invalid SQL connection config: SQL connection config is required"
}
```

### 3. 数据库连接失败

```
MySQL连接失败
响应: {
  "success": false,
  "error": "failed to connect to MySQL: ..."
}
```

### 4. 数据库未配置

```
服务器启动时未提供 -dsn 参数
响应: {
  "success": false,
  "error": "database is not configured"
}
```

---

## 📝 关键代码片段

### 完整的 createExecutor 函数

```go
func (s *AgentServer) createExecutor(req *pb.ChatRequest, ctx *agent.Context) (*agent.AgentExecutor, error) {
    // 1. 验证和加载配置
    if req.AgentId == 0 {
        return nil, fmt.Errorf("agent_id is required")
    }
    agentConfig, err := s.db.GetAgent(int(req.AgentId))
    
    // 2. 参数处理（支持覆盖）
    maxSteps := req.MaxSteps || agentConfig.MaxSteps || 10
    model := req.Model || agentConfig.Model
    systemPrompt := req.SystemPrompt || agentConfig.SystemPrompt
    
    // 3. 创建 Agent
    var executor *agent.AgentExecutor
    switch agentConfig.Framework {
    case "sql":
        // SQL Agent 创建逻辑
    case "elasticsearch":
        // ES Agent 创建逻辑
    case "react", "plan", "chain":
        // 其他 Agent 创建逻辑
    }
    
    // 4. 配置和返回
    executor.SetMaxSteps(maxSteps)
    return executor, nil
}
```

---

## ✨ 功能亮点

### 1. 完全配置驱动
```
✅ Agent 的所有行为由数据库配置决定
✅ 无需在代码中硬编码连接信息
✅ 可以随时修改配置
```

### 2. 自动连接管理
```
✅ SQL Agent → 自动创建 MySQL 连接
✅ ES Agent → 自动创建 ES HTTP 客户端
✅ 自动注册相应的工具集
```

### 3. 灵活的参数覆盖
```
✅ 请求中可以临时覆盖配置
✅ 适合测试和调试
✅ 不影响数据库中的配置
```

### 4. 统一的错误处理
```
✅ Agent 不存在
✅ 连接配置无效
✅ 数据库连接失败
✅ 友好的错误消息
```

---

## 🚀 启动和使用

### 1. 启动服务器（必须带数据库）

```bash
go run cmd/server/main.go \
  -apiKey YOUR_KEY \
  -baseUrl YOUR_URL \
  -dsn "root:pass@tcp(localhost:3306)/jas_agent"
```

⚠️ **注意**: 必须提供 `-dsn` 参数，否则无法加载 Agent 配置！

### 2. 创建 Agent

访问 http://localhost:8080
- 创建 SQL Agent（配置 MySQL 连接）
- 创建 ES Agent（配置 ES 连接）

### 3. 开始对话

选择 Agent → 输入查询 → 后台自动：
- 加载配置
- 创建连接
- 注册工具
- 执行查询

---

## 📋 测试清单

- [x] agent_id 必填验证 ✅
- [x] 从数据库加载配置 ✅
- [x] SQL Agent 动态创建 ✅
- [x] ES Agent 动态创建 ✅
- [x] MySQL 连接自动创建 ✅
- [x] ES 连接自动创建 ✅
- [x] 工具自动注册 ✅
- [x] 参数覆盖支持 ✅
- [x] 错误处理完善 ✅
- [x] 编译验证通过 ✅

---

## 🎊 总结

现在系统完全基于 **Agent ID** 驱动：

1. ✅ 前端传递 `agent_id`
2. ✅ 后端从数据库加载配置
3. ✅ 自动创建相应的 Agent
4. ✅ 自动处理连接配置（SQL/ES）
5. ✅ 自动注册工具集
6. ✅ 执行查询并返回结果

**无需在代码中硬编码任何连接信息，完全由数据库配置驱动！** 🚀

