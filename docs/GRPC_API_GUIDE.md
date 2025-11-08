# JAS Agent gRPC API 使用指南

本文档介绍如何使用 JAS Agent 的 gRPC 和 HTTP API 进行对话交互。

## 目录

- [快速开始](#快速开始)
- [gRPC API](#grpc-api)
- [HTTP API](#http-api)
- [前端使用](#前端使用)
- [示例代码](#示例代码)

---

## 快速开始

### 1. 安装依赖

```bash
# 安装 protoc (如果还没安装)
# macOS
brew install protobuf

# Linux
apt-get install protobuf-compiler

# Windows
# 从 https://github.com/protocolbuffers/protobuf/releases 下载

# 安装 Go 依赖
go mod tidy
```

### 2. 生成 Proto 代码

```bash
# Linux/macOS
chmod +x scripts/generate_proto.sh
./scripts/generate_proto.sh

# Windows
scripts\generate_proto.bat
```

### 3. 启动服务器

```bash
cd cmd/server
go run main.go -apiKey YOUR_API_KEY -baseUrl YOUR_BASE_URL
```

服务器将启动在：
- HTTP API: `http://localhost:8080/api`
- 前端界面: `http://localhost:8080`

### 4. 访问前端

打开浏览器访问 `http://localhost:8080`，即可使用 Web 界面进行对话。

---

## gRPC API

### Proto 定义

完整的 Proto 定义见 `api/proto/agent_service.proto`

### 服务接口

#### 1. Chat - 单次对话

```protobuf
rpc Chat(ChatRequest) returns (ChatResponse);
```

**请求参数：**
```json
{
  "query": "计算 15 + 27",
  "agent_type": "REACT",  // REACT, CHAIN, PLAN, SQL
  "model": "gpt-3.5-turbo",
  "system_prompt": "",    // 可选
  "max_steps": 10,
  "config": {},           // 额外配置
  "session_id": ""        // 会话ID
}
```

**响应：**
```json
{
  "response": "根据计算，15 + 27 = 42",
  "agent_type": "ReAct",
  "metadata": {
    "total_steps": 2,
    "tools_called": 1,
    "tool_names": ["calculator"],
    "execution_time_ms": 1523,
    "state": "Finish"
  },
  "success": true
}
```

#### 2. StreamChat - 流式对话

```protobuf
rpc StreamChat(ChatRequest) returns (stream ChatStreamResponse);
```

流式响应类型：
- `THINKING`: 思考过程
- `ACTION`: 执行动作
- `OBSERVATION`: 观察结果
- `FINAL`: 最终答案
- `ERROR`: 错误信息
- `METADATA`: 元数据

#### 3. ListAgentTypes - 获取Agent类型

```protobuf
rpc ListAgentTypes(Empty) returns (AgentTypesResponse);
```

#### 4. ListTools - 获取可用工具

```protobuf
rpc ListTools(Empty) returns (ToolsResponse);
```

---

## HTTP API

HTTP API 是 gRPC 的 RESTful 网关，使用 JSON 格式。

### 端点

#### POST /api/chat

单次对话接口

**请求：**
```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "query": "计算 15 + 27",
    "agent_type": "react",
    "model": "gpt-3.5-turbo",
    "max_steps": 10
  }'
```

**响应：**
```json
{
  "response": "根据计算，15 + 27 = 42",
  "agent_type": "ReAct",
  "metadata": {
    "total_steps": 2,
    "tools_called": 1,
    "tool_names": ["calculator"],
    "execution_time_ms": 1523,
    "state": "Finish"
  },
  "success": true
}
```

#### WebSocket /api/chat/stream

流式对话接口（WebSocket）

**JavaScript 示例：**
```javascript
const ws = new WebSocket('ws://localhost:8080/api/chat/stream');

ws.onopen = () => {
  ws.send(JSON.stringify({
    query: "计算 15 + 27",
    agent_type: "react",
    model: "gpt-3.5-turbo"
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(`[${data.type}] ${data.content}`);
  
  if (data.type === 'final') {
    console.log('Final:', data.content);
    console.log('Metadata:', data.metadata);
    ws.close();
  }
};
```

#### GET /api/agents

获取可用的 Agent 类型

**响应：**
```json
{
  "agents": [
    {
      "type": "react",
      "name": "ReAct Agent",
      "description": "通用推理代理，支持思考-行动-观察循环",
      "available": true
    },
    {
      "type": "chain",
      "name": "Chain Agent",
      "description": "链式代理，按预定义流程执行多个Agent",
      "available": true
    },
    {
      "type": "plan",
      "name": "Plan Agent",
      "description": "计划代理，先规划后执行复杂任务",
      "available": true
    }
  ]
}
```

#### GET /api/tools

获取可用的工具列表

**响应：**
```json
{
  "tools": [
    {
      "name": "calculator",
      "description": "数学表达式计算器",
      "type": "Normal"
    },
    {
      "name": "averageDogWeight",
      "description": "查询狗狗品种的平均体重",
      "type": "Normal"
    }
  ]
}
```

---

## 前端使用

### 功能特性

1. **Agent 选择**
   - 支持选择不同类型的 Agent（ReAct、Chain、Plan）
   - 每种 Agent 有不同的能力和使用场景

2. **模型配置**
   - 支持选择不同的 LLM 模型
   - 可配置最大执行步数
   - 可自定义系统提示词

3. **对话模式**
   - **普通模式**：一次性返回完整结果
   - **流式模式**：实时显示执行过程

4. **工具查看**
   - 查看所有可用工具
   - 了解工具的功能和用途

### 使用步骤

1. **选择 Agent 类型**
   - ReAct：适合通用问题
   - Chain：适合流程化任务
   - Plan：适合复杂多步骤任务

2. **配置参数**
   - 选择模型（建议使用 GPT-3.5 Turbo 或 GPT-4）
   - 设置最大步数（默认 10 步）
   - （可选）设置系统提示词

3. **输入问题**
   - 在输入框输入您的问题
   - 点击发送或按 Enter 键

4. **查看结果**
   - 流式模式：实时看到执行过程
   - 普通模式：等待最终结果

### 示例问题

**数学计算：**
```
计算 (15 + 27) * 3 的结果
```

**信息查询：**
```
我有一只边境牧羊犬和一只苏格兰梗，它们的总体重是多少？
```

**复杂任务：**
```
我有3只狗，分别是border collie、scottish terrier和toy poodle。
请查询它们的平均体重，然后计算总重量。
```

---

## 示例代码

### Go 客户端示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    pb "jas-agent/api/agent/service/v1"
    "google.golang.org/grpc"
)

func main() {
    // 连接到服务器
    conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("连接失败: %v", err)
    }
    defer conn.Close()
    
    client := pb.NewAgentServiceClient(conn)
    
    // 发送请求
    req := &pb.ChatRequest{
        Query:     "计算 15 + 27",
        AgentType: pb.AgentType_REACT,
        Model:     "gpt-3.5-turbo",
        MaxSteps:  10,
    }
    
    resp, err := client.Chat(context.Background(), req)
    if err != nil {
        log.Fatalf("请求失败: %v", err)
    }
    
    fmt.Printf("响应: %s\n", resp.Response)
    fmt.Printf("步骤: %d\n", resp.Metadata.TotalSteps)
}
```

### Python 客户端示例

```python
import grpc
import agent_service_pb2
import agent_service_pb2_grpc

# 连接到服务器
channel = grpc.insecure_channel('localhost:9090')
stub = agent_service_pb2_grpc.AgentServiceStub(channel)

# 发送请求
request = agent_service_pb2.ChatRequest(
    query="计算 15 + 27",
    agent_type=agent_service_pb2.REACT,
    model="gpt-3.5-turbo",
    max_steps=10
)

response = stub.Chat(request)
print(f"响应: {response.response}")
print(f"步骤: {response.metadata.total_steps}")
```

### JavaScript/Node.js 客户端示例

```javascript
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');

// 加载 proto
const packageDefinition = protoLoader.loadSync('agent_service.proto');
const proto = grpc.loadPackageDefinition(packageDefinition).agent;

// 创建客户端
const client = new proto.AgentService(
    'localhost:9090',
    grpc.credentials.createInsecure()
);

// 发送请求
client.Chat({
    query: "计算 15 + 27",
    agent_type: 0, // REACT
    model: "gpt-3.5-turbo",
    max_steps: 10
}, (error, response) => {
    if (error) {
        console.error('错误:', error);
        return;
    }
    console.log('响应:', response.response);
    console.log('步骤:', response.metadata.total_steps);
});
```

---

## 配置选项

### Agent 类型

| 类型 | 值 | 描述 |
|------|-----|------|
| ReAct | `react` / `REACT` | 通用推理代理 |
| Chain | `chain` / `CHAIN` | 链式执行代理 |
| Plan | `plan` / `PLAN` | 计划执行代理 |
| SQL | `sql` / `SQL` | SQL查询代理 |

### 配置参数

```json
{
  "config": {
    "enable_replan": "true",  // Plan Agent: 启用重新规划
    "db_info": "MySQL: db"    // SQL Agent: 数据库信息
  }
}
```

---

## 故障排查

### 常见问题

**Q: 服务器启动失败？**
A: 确保已安装所有依赖并提供了正确的 API Key 和 Base URL。

**Q: WebSocket 连接失败？**
A: 检查防火墙设置，确保 8080 端口可访问。

**Q: 响应时间过长？**
A: 可以调整 `max_steps` 参数，或使用流式模式查看执行进度。

**Q: Agent 返回错误？**
A: 检查系统提示词是否正确，查看返回的错误信息。

---

## 安全建议

1. **API Key 保护**：不要在前端暴露 API Key
2. **CORS 配置**：生产环境应限制允许的来源
3. **访问控制**：建议添加认证机制
4. **请求限制**：添加速率限制防止滥用

---

## 性能优化

1. **会话管理**：使用 `session_id` 保持上下文
2. **连接池**：gRPC 客户端使用连接池
3. **流式响应**：大任务使用流式模式
4. **缓存**：缓存常见查询结果

---

更多信息请参考主 README 和其他文档。


