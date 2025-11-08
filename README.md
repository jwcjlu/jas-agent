# JAS Agent

JAS Agent 是一个面向多代理场景的智能助手平台，支持 ReAct、Chain、Plan、SQL、Elasticsearch 等执行框架，可连接 MCP（Model Context Protocol）服务，并提供可视化管理界面与 gRPC/HTTP API。

## 功能亮点

- **多代理框架**：内置 ReAct、Chain、Plan、SQL、Elasticsearch 等策略，可扩展自定义逻辑。
- **实时对话**：支持普通请求和 WebSocket 流式输出，展示思考、动作、观察等阶段。
- **Agent 管理**：界面和 API 双通道创建/编辑/删除代理，配置模型、系统提示、最大步数、MCP 服务等。
- **MCP 服务管理**：统一注册/启用/停用外部工具，显示工具数量与状态。
- **前端可视化**：Vite + React + TypeScript 打造的现代 UI，包括对话区、配置面板、状态栏及管理弹窗。
- **统一 API**：同时提供 gRPC 与 HTTP/JSON 接口，便于不同客户端集成。

## 仓库结构

```
.
├── api/                    # Protobuf 定义与生成的 gRPC/HTTP 代码
├── cmd/server/             # 后端入口，Kratos 应用（wire 依赖注入）
├── configs/                # 配置文件（YAML）
├── internal/               # 后端核心逻辑
│   ├── biz/                # 领域用例（Agent、MCP、聊天执行等）
│   ├── data/               # 数据访问层（GORM）
│   ├── server/             # HTTP/GRPC Server 适配
│   └── service/            # gRPC/HTTP Handler
├── third_party/            # Proto 依赖
└── web/                    # 前端工程 (Vite + React + TypeScript)
```

## 环境要求

- Go 1.21+
- Node.js 18+
- MySQL（如启用 SQL Agent）
- Elasticsearch（如启用 Elasticsearch Agent）

## 配置说明

主要配置位于 `configs/config.yaml`：

```yaml
server:
  http:
    addr: "0.0.0.0:8080"
  grpc:
    addr: "0.0.0.0:9000"

llm:
  api_key: "YOUR_API_KEY"
  base_url: "https://api.openai.com/v1"
  model: "gpt-3.5-turbo"

data:
  database:
    driver: "mysql"
    source: "user:password@tcp(127.0.0.1:3306)/jas_agent?parseTime=true"
    max_idle_conns: 5
    max_open_conns: 25
    conn_max_lifetime: 300
```

- 若 `data.database` 留空则以内存模式运行（Agent/MCP 管理受限）。
- `llm` 中配置默认模型、Key 和 Base URL，可替换为自建服务。

## 后端运行

```bash
# 可选：需要时重新生成 wire / proto
wire ./cmd/server
protoc -I . -I third_party --go_out=. --go-grpc_out=. --go-http_out=. api/proto/agent_service.proto

# 启动服务
go run cmd/server/main.go -conf configs/config.yaml
```

默认暴露：
- HTTP: `http://localhost:8080`
- gRPC: `localhost:9000`

## 前端运行

```bash
cd web
npm install
npm run dev       # 开发模式
npm run build     # 生产构建 (输出 web/dist)
npm run preview   # 构建后预览
```

开发模式下，Vite 会将 `/api` 请求自动代理到 `http://localhost:8080`，WebSocket (`ws://.../api/chat/stream`) 亦同步代理。

## 核心接口

### 聊天

- 普通：`POST /api/chat`
- 流式：`ws://{host}/api/chat/stream`
  - 请求字段：
    - `agent_id`（必填）
    - `agent_type`（整型枚举：0=ReAct, 1=Chain, 2=Plan, 3=SQL, 4=Elasticsearch）
    - `query`、`model`、`max_steps`、`system_prompt`、`enabled_mcp_services` 等
  - 返回消息类型：
    - `thinking`、`action`、`observation`、`metadata`、`final`、`error`

### Agent 管理

- `GET /api/agents`
- `POST /api/agents`
- `PUT /api/agents/{id}`
- `DELETE /api/agents/{id}`
- `GET /api/agents/{id}`

数据结构详见 `api/proto/agent_service.proto` 中 `AgentConfigRequest`/`AgentConfigResponse`。

### MCP 服务

- `GET /api/mcp/services`
- `POST /api/mcp/services`
- `DELETE /api/mcp/services/{name}`

## 前端特性

- **聊天视图**：展示用户、助手、执行步骤（思考/动作/观察）。
- **配置面板**：选择 Agent、模型、最大步数、系统提示、启用 MCP 服务。
- **状态栏**：实时显示执行状态、调用次数、耗时等。
- **Agent/MCP 管理**：模态框形式管理代理与外部工具，内置 SQL/ES 连接设置模板。

## 开发提示

- 依赖注入通过 `wireApp` 组装 Kratos 组件，更新后执行 `wire ./cmd/server`。
- 数据层使用 GORM（`internal/data`），配合 `internal/biz` 实现业务逻辑。
- HTTP 注解映射 `/api/...`，与前端请求保持一致；若改动 Protobuf 后记得重新生成代码。
- gRPC 和 HTTP 均复用同一领域逻辑，避免多处维护。

## 常见问题

1. **“invalid request body”**：多因 `agent_id` 或 `agent_type` 缺失/类型不符，`agent_type` 必须为整型枚举。
2. **WebSocket 提示“已关闭”**：通常是请求参数错误后返回 `type:error`。可在浏览器 Network > WS 观察消息并排查。
3. **SQL/ES Agent**：需在 Agent 表单中提供 JSON 连接信息（界面已有提示）。
4. **CORS/部署**：生产环境通常在反向代理层处理跨域或直接同域部署，内置 HTTP Server 默认允许所有请求。

## Roadmap

- Agent/MCP 列表支持搜索、分页、过滤。
- 聊天执行过程持久化并支持回放。
- 多模型、多租户隔离策略。
- 更完善的权限和审计机制。

---

如需更多示例、截图或部署脚本，可在此基础上扩展。前端详情请参考 `web/README.md`。

