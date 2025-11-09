# JAS Agent 服务器快速开始

本示例演示如何启动 JAS Agent 的 gRPC/HTTP 服务器和 Web 前端。

## 功能特性

- ✅ gRPC API 服务
- ✅ HTTP RESTful API
- ✅ WebSocket 流式响应
- ✅ Web 前端界面
- ✅ 会话管理
- ✅ 多种 Agent 类型支持

## 快速启动

### 1. 准备工作

确保已安装 Go 1.16+ 和 Node.js（可选，仅用于前端开发）

### 2. 安装依赖

```bash
# 在项目根目录
cd jas-agent
go mod tidy
```

### 3. 启动服务器

```bash
cd cmd/server
go run main.go \
  -apiKey YOUR_OPENAI_API_KEY \
  -baseUrl YOUR_API_BASE_URL \
  -http :8080
```

**参数说明：**
- `-apiKey`: OpenAI API Key（必需）
- `-baseUrl`: OpenAI API Base URL（必需）
- `-http`: HTTP服务器地址（默认 :8080）
- `-model`: 默认模型（默认 gpt-3.5-turbo）

### 4. 访问服务

启动成功后，您将看到：

```
🚀 启动 JAS Agent 服务器...
✅ gRPC服务已创建
🌐 HTTP服务器启动在 :8080
📡 API端点: http://localhost:8080/api
🌍 前端界面: http://localhost:8080
```

**访问方式：**
- Web界面: `http://localhost:8080`
- API端点: `http://localhost:8080/api`

## 使用示例

### Web 界面使用

1. 打开浏览器访问 `http://localhost:8080`
2. 选择 Agent 类型（ReAct、Chain、Plan）
3. 选择模型和配置参数
4. 输入问题并发送
5. 查看结果

### API 调用示例

#### curl 调用

```bash
# 单次对话
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "query": "计算 15 + 27 * 3",
    "agent_type": "react",
    "model": "gpt-3.5-turbo",
    "max_steps": 10
  }'
```

#### Python 调用

```python
import requests
import json

url = "http://localhost:8080/api/chat"
data = {
    "query": "我有一只边境牧羊犬，它的平均体重是多少？",
    "agent_type": "react",
    "model": "gpt-3.5-turbo",
    "max_steps": 10
}

response = requests.post(url, json=data)
result = response.json()

print(f"响应: {result['response']}")
if result.get('metadata'):
    print(f"执行步骤: {result['metadata']['total_steps']}")
    print(f"使用工具: {result['metadata']['tool_names']}")
```

#### JavaScript 调用

```javascript
// 普通HTTP请求
fetch('http://localhost:8080/api/chat', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    query: "计算 (15 + 27) * 3",
    agent_type: "react",
    model: "gpt-3.5-turbo",
    max_steps: 10
  })
})
.then(res => res.json())
.then(data => {
  console.log('响应:', data.response);
  console.log('元数据:', data.metadata);
});

// WebSocket流式请求
const ws = new WebSocket('ws://localhost:8080/api/chat/stream');

ws.onopen = () => {
  ws.send(JSON.stringify({
    query: "我有3只狗，计算它们的总体重",
    agent_type: "plan",
    model: "gpt-3.5-turbo",
    max_steps: 15
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(`[${data.type}] ${data.content}`);
  
  if (data.type === 'final') {
    console.log('最终结果:', data.content);
    ws.close();
  }
};
```

## Agent 类型说明

### ReAct Agent

**适用场景：** 通用推理任务

**示例：**
```json
{
  "query": "计算 15 + 27 * 3",
  "agent_type": "react",
  "max_steps": 10
}
```

### Chain Agent

**适用场景：** 流程化任务

**示例：**
```json
{
  "query": "查询狗狗体重并计算总和",
  "agent_type": "chain",
  "max_steps": 10
}
```

### Plan Agent

**适用场景：** 复杂多步骤任务

**示例：**
```json
{
  "query": "我有3只狗，查询它们的体重并计算平均值",
  "agent_type": "plan",
  "max_steps": 20,
  "config": {
    "enable_replan": "true"
  }
}
```

## 配置选项

### 环境变量

```bash
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="your-base-url"
export HTTP_PORT="8080"
export DEFAULT_MODEL="gpt-3.5-turbo"
```

### 命令行参数

```bash
go run main.go \
  -apiKey $OPENAI_API_KEY \
  -baseUrl $OPENAI_BASE_URL \
  -http :$HTTP_PORT \
  -model $DEFAULT_MODEL
```

## 开发模式

### 前端开发

前端文件位于 `web/` 目录：

```
web/
├── index.html    # HTML主文件
├── style.css     # 样式文件
└── app.js        # JavaScript应用
```

修改后刷新浏览器即可看到变化（无需重启服务器）。

### 后端开发

```bash
# 监听文件变化自动重启（需要安装 air）
air

# 或手动重启
go run cmd/server/main.go -apiKey KEY -baseUrl URL
```

## 部署建议

### Docker 部署

```dockerfile
FROM golang:1.21-alpine

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o server cmd/server/main.go

EXPOSE 8080

CMD ["./server", "-apiKey", "$API_KEY", "-baseUrl", "$BASE_URL", "-http", ":8080"]
```

### 生产环境

1. **使用环境变量**：不要在代码中硬编码 API Key
2. **启用 HTTPS**：使用 TLS 证书
3. **添加认证**：实现用户认证机制
4. **日志记录**：添加详细的日志
5. **监控告警**：配置性能监控

## 故障排查

### 常见问题

**Q: 服务器启动失败**
```bash
# 检查端口是否被占用
lsof -i :8080

# 检查参数是否正确
go run main.go -apiKey YOUR_KEY -baseUrl YOUR_URL
```

**Q: API 调用失败**
```bash
# 检查 CORS 设置
# 检查 API Key 是否正确
# 查看服务器日志
```

**Q: WebSocket 连接失败**
```bash
# 检查浏览器控制台错误
# 检查防火墙设置
# 确认 WebSocket 端口可访问
```

## 更多信息

- [gRPC API 使用指南](../../../docs/GRPC_API_GUIDE.md)
- [Chain 和 Plan 框架指南](../../../docs/CHAIN_AND_PLAN_FRAMEWORK.md)
- [主 README](../../../README.md)

## 许可证

MIT License


