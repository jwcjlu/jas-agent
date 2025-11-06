# 🎉 SQL 和 Elasticsearch Agent 功能演示

## ✨ 新功能亮点

参照 SQL Agent 的实现，我们成功开发了 **Elasticsearch Agent**，并为两种 Agent 添加了**完整的连接配置功能**。

---

## 📺 界面演示

### 1. 创建 SQL Agent

#### 表单界面

```
┌────────────────────────────────────────────────┐
│  🤖 Agent 管理                       [×]      │
├────────────────────────────────────────────────┤
│                                                │
│  Agent 名称 *                                  │
│  ┌──────────────────────────────────────────┐ │
│  │ SQL查询助手                              │ │
│  └──────────────────────────────────────────┘ │
│                                                │
│  框架类型 *                                    │
│  ┌──────────────────────────────────────────┐ │
│  │ 🗄️ SQL - MySQL数据库查询（需配置数据库）│ │ ← 选择这个
│  └──────────────────────────────────────────┘ │
│                                                │
│  ┌────────────────────────────────────────┐   │
│  │  📊 MySQL 连接配置                     │   │ ← 自动显示
│  ├────────────────────────────────────────┤   │
│  │  主机 *          端口 *                │   │
│  │  ┌───────────┐   ┌────┐              │   │
│  │  │ localhost │   │3306│              │   │
│  │  └───────────┘   └────┘              │   │
│  │                                       │   │
│  │  数据库名称 *                         │   │
│  │  ┌─────────────────────────────────┐ │   │
│  │  │ testdb                          │ │   │
│  │  └─────────────────────────────────┘ │   │
│  │                                       │   │
│  │  用户名 *        密码                 │   │
│  │  ┌──────────┐   ┌────────────────┐  │   │
│  │  │ root     │   │ ************** │  │   │
│  │  └──────────┘   └────────────────┘  │   │
│  └────────────────────────────────────────┘   │
│                                                │
│  描述 (可选)                                   │
│  ┌──────────────────────────────────────────┐ │
│  │ 专业的MySQL数据库查询和分析助手          │ │
│  └──────────────────────────────────────────┘ │
│                                                │
│  模型            最大步数                      │
│  ┌───────────┐   ┌───┐                       │
│  │GPT-3.5▼   │   │15 │                       │
│  └───────────┘   └───┘                       │
│                                                │
│                          [取消]   [保存]      │
└────────────────────────────────────────────────┘
```

### 2. 创建 Elasticsearch Agent

#### 表单界面

```
┌────────────────────────────────────────────────┐
│  🤖 Agent 管理                       [×]      │
├────────────────────────────────────────────────┤
│                                                │
│  Agent 名称 *                                  │
│  ┌──────────────────────────────────────────┐ │
│  │ 日志分析助手                             │ │
│  └──────────────────────────────────────────┘ │
│                                                │
│  框架类型 *                                    │
│  ┌──────────────────────────────────────────┐ │
│  │ 🔍 Elasticsearch - 日志搜索分析（需配置ES）│ │ ← 选择这个
│  └──────────────────────────────────────────┘ │
│                                                │
│  ┌────────────────────────────────────────┐   │
│  │  🔍 Elasticsearch 连接配置             │   │ ← 自动显示
│  ├────────────────────────────────────────┤   │
│  │  ES 服务地址 *                         │   │
│  │  ┌─────────────────────────────────┐  │   │
│  │  │ http://localhost:9200           │  │   │
│  │  └─────────────────────────────────┘  │   │
│  │                                       │   │
│  │  用户名 (可选)   密码 (可选)          │   │
│  │  ┌──────────┐   ┌────────────────┐  │   │
│  │  │ elastic  │   │ ************** │  │   │
│  │  └──────────┘   └────────────────┘  │   │
│  └────────────────────────────────────────┘   │
│                                                │
│  描述 (可选)                                   │
│  ┌──────────────────────────────────────────┐ │
│  │ Elasticsearch日志搜索和数据分析专家      │ │
│  └──────────────────────────────────────────┘ │
│                                                │
│  模型            最大步数                      │
│  ┌───────────┐   ┌───┐                       │
│  │GPT-3.5▼   │   │15 │                       │
│  └───────────┘   └───┘                       │
│                                                │
│                          [取消]   [保存]      │
└────────────────────────────────────────────────┘
```

---

## 🎬 使用演示

### SQL Agent 对话示例

```
┌─────────────────────────────────────────────┐
│ 选择 Agent: [SQL查询助手 (SQL) ▼]         │
│                              [🤖 管理]      │
└─────────────────────────────────────────────┘

用户: 查询销售额最高的前10个客户

AI 执行过程:

[💭 思考]
我需要先了解数据库中有哪些表
─────────────────────────────────────────────

[⚙️ 执行]
Action: list_tables[]
─────────────────────────────────────────────

[👁️ 观察]
Tables: customers, orders, products
─────────────────────────────────────────────

[💭 思考]
需要查看表结构以了解字段信息
─────────────────────────────────────────────

[⚙️ 执行]
Action: tables_schema[customers, orders]
─────────────────────────────────────────────

[👁️ 观察]
Table: customers
Columns:
  - id (INT) [PRIMARY KEY]
  - name (VARCHAR)
  - email (VARCHAR)

Table: orders
Columns:
  - id (INT) [PRIMARY KEY]
  - customer_id (INT)
  - amount (DECIMAL)
  - order_date (DATE)
─────────────────────────────────────────────

[💭 思考]
现在我可以构建SQL查询来获取销售额最高的客户
─────────────────────────────────────────────

[⚙️ 执行]
Action: execute_sql[
  SELECT c.name, SUM(o.amount) as total_sales
  FROM customers c
  JOIN orders o ON c.id = o.customer_id
  GROUP BY c.id, c.name
  ORDER BY total_sales DESC
  LIMIT 10
]
─────────────────────────────────────────────

[👁️ 观察]
Query returned 10 rows:
[
  {"name": "张三", "total_sales": 150000},
  {"name": "李四", "total_sales": 120000},
  ...
]
─────────────────────────────────────────────

[📊 最终答案]
销售额最高的前10个客户如下：

1. 张三 - ¥150,000
2. 李四 - ¥120,000
3. 王五 - ¥98,000
...

这些客户贡献了总销售额的 65%。
```

### Elasticsearch Agent 对话示例

```
┌─────────────────────────────────────────────┐
│ 选择 Agent: [日志分析助手 (ELASTICSEARCH) ▼]│
│                              [🤖 管理]      │
└─────────────────────────────────────────────┘

用户: 统计今天每小时的错误日志数量

AI 执行过程:

[💭 思考]
我需要先查看有哪些索引
─────────────────────────────────────────────

[⚙️ 执行]
Action: list_indices[]
─────────────────────────────────────────────

[👁️ 观察]
Found 2 indices:
- logs-2024-11
  Health: green, Docs: 15234, Size: 2.3mb
- logs-2024-10
  Health: green, Docs: 18567, Size: 3.1mb
─────────────────────────────────────────────

[💭 思考]
需要查看索引的字段结构
─────────────────────────────────────────────

[⚙️ 执行]
Action: get_index_mapping[logs-2024-11]
─────────────────────────────────────────────

[👁️ 观察]
Mapping for index 'logs-2024-11':
{
  "properties": {
    "timestamp": {"type": "date"},
    "level": {"type": "keyword"},
    "message": {"type": "text"}
  }
}
─────────────────────────────────────────────

[💭 思考]
我需要构建一个聚合查询，统计今天每小时的错误数
─────────────────────────────────────────────

[⚙️ 执行]
Action: aggregate_data[{
  "index": "logs-2024-11",
  "query": {
    "bool": {
      "must": [
        {"term": {"level.keyword": "ERROR"}},
        {"range": {"timestamp": {"gte": "now/d"}}}
      ]
    }
  },
  "aggs": {
    "hourly_errors": {
      "date_histogram": {
        "field": "timestamp",
        "calendar_interval": "hour"
      }
    }
  }
}]
─────────────────────────────────────────────

[👁️ 观察]
Aggregation results:
{
  "hourly_errors": {
    "buckets": [
      {"key": "2024-11-04T00:00:00", "doc_count": 12},
      {"key": "2024-11-04T01:00:00", "doc_count": 8},
      {"key": "2024-11-04T02:00:00", "doc_count": 15},
      ...
    ]
  }
}
─────────────────────────────────────────────

[📊 最终答案]
今天每小时的错误日志统计：

00:00-01:00  ████████████  12条
01:00-02:00  ████████       8条
02:00-03:00  ███████████████ 15条
...

高峰时段在 02:00-03:00，共15条错误。
建议重点关注这个时段的系统状态。
```

---

## 🔧 技术实现

### 动态配置表单

**React 代码片段**:

```jsx
{formData.framework === 'sql' && (
  <div className="connection-config-section">
    <h4>📊 MySQL 连接配置</h4>
    <input name="host" required />
    <input name="port" type="number" required />
    <input name="database" required />
    <input name="username" required />
    <input name="password" type="password" />
  </div>
)}

{formData.framework === 'elasticsearch' && (
  <div className="connection-config-section">
    <h4>🔍 Elasticsearch 连接配置</h4>
    <input name="host" required />
    <input name="username" />
    <input name="password" type="password" />
  </div>
)}
```

### 数据存储

**数据库存储**:
```sql
UPDATE agents 
SET connection_config = '{"host":"localhost","port":3306,"database":"testdb","username":"root","password":"pass"}'
WHERE id = 1;
```

**前端提交**:
```javascript
const submitData = {
  name: "SQL查询助手",
  framework: "sql",
  connection_config: JSON.stringify({
    host: "localhost",
    port: 3306,
    database: "testdb",
    username: "root",
    password: "mypass"
  })
};
```

---

## 📊 功能对比

### SQL Agent vs Elasticsearch Agent

| 特性 | SQL Agent | Elasticsearch Agent |
|------|-----------|---------------------|
| **框架标识** | `sql` | `elasticsearch` |
| **连接协议** | MySQL 协议 | HTTP/HTTPS |
| **查询语言** | SQL | Query DSL (JSON) |
| **主要用途** | 结构化数据查询 | 全文搜索、日志分析 |
| **工具数量** | 3个 | 5个 |
| **聚合分析** | SQL聚合函数 | Aggregations API |
| **必需配置** | host, port, db, user | host |
| **可选配置** | password | username, password |

### 配置字段对比

**SQL Agent 连接配置**:
```json
{
  "host": "localhost",      // 必填
  "port": 3306,            // 必填
  "database": "testdb",    // 必填
  "username": "root",      // 必填
  "password": "pass"       // 可选
}
```

**ES Agent 连接配置**:
```json
{
  "host": "http://localhost:9200",  // 必填
  "username": "elastic",             // 可选
  "password": "changeme"             // 可选
}
```

---

## 🎯 实际应用场景

### 场景 1: 业务数据分析

**创建 SQL Agent**:
```
名称: 销售数据分析师
框架: sql
MySQL: 连接到销售数据库 (sales_db)
```

**使用示例**:
```
✅ "查询本月销售额"
✅ "分析各区域销售情况"
✅ "找出滞销产品"
✅ "统计客户复购率"
```

### 场景 2: 日志故障排查

**创建 ES Agent**:
```
名称: 故障排查助手
框架: elasticsearch
ES: 连接到生产日志集群
```

**使用示例**:
```
✅ "查找过去1小时的所有500错误"
✅ "分析错误日志的分布趋势"
✅ "追踪特定用户的操作记录"
✅ "统计各服务的错误率"
```

### 场景 3: 安全审计

**创建 ES Agent**:
```
名称: 安全审计助手
框架: elasticsearch
ES: 连接到审计日志
```

**使用示例**:
```
✅ "查找异常登录尝试"
✅ "统计每个IP的请求数量"
✅ "检测可疑的访问模式"
✅ "生成安全审计报告"
```

---

## 💡 最佳实践

### 1. Agent 命名

**推荐命名格式**: `{用途} + {数据源}`

```
✅ 好的命名:
  - "销售数据库查询助手"
  - "生产环境日志分析"
  - "用户行为分析助手"
  - "安全审计助手"

❌ 不好的命名:
  - "Agent1"
  - "测试"
  - "数据库"
```

### 2. 连接配置安全

**开发环境**:
```json
{
  "host": "localhost",
  "username": "dev_user",
  "password": "dev_pass"
}
```

**生产环境**:
```json
{
  "host": "prod-server.internal",
  "username": "readonly_user",    // ⭐ 使用只读用户
  "password": "secure_password"
}
```

### 3. 权限控制

**SQL Agent** - 创建只读用户:
```sql
CREATE USER 'readonly'@'%' IDENTIFIED BY 'secure_pass';
GRANT SELECT ON mydb.* TO 'readonly'@'%';
FLUSH PRIVILEGES;
```

**ES Agent** - 使用只读角色:
```json
{
  "cluster": ["monitor"],
  "indices": [{
    "names": ["logs-*"],
    "privileges": ["read", "view_index_metadata"]
  }]
}
```

### 4. Agent 描述

写清晰的描述帮助团队成员理解 Agent 用途：

```
✅ 好的描述:
"专业的MySQL销售数据库查询助手，擅长复杂SQL分析、销售报表生成和趋势预测"

❌ 简单的描述:
"查询数据库"
```

---

## 🔍 调试和测试

### 测试 SQL 连接

```bash
# 在创建 Agent 前，先测试连接
mysql -h localhost -u root -p testdb

# 验证权限
SHOW GRANTS;
```

### 测试 ES 连接

```bash
# 测试连接
curl http://localhost:9200

# 测试认证
curl -u elastic:password http://localhost:9200

# 列出索引
curl http://localhost:9200/_cat/indices?v
```

### 验证 Agent 配置

1. 创建 Agent 后，在列表中查看
2. 点击编辑，确认连接配置正确
3. 选择 Agent，发送简单测试查询
4. 观察 Agent 是否能正确连接数据源

---

## 📈 扩展功能建议

### 未来可以添加

1. **连接测试按钮**
   - 在保存前测试连接是否成功
   - 显示连接状态和错误信息

2. **更多数据源**
   - PostgreSQL
   - MongoDB
   - Redis
   - ClickHouse

3. **连接池管理**
   - 复用数据库连接
   - 连接池配置
   - 超时控制

4. **密码加密**
   - 加密存储密码
   - 使用密钥管理服务
   - 支持环境变量

5. **性能监控**
   - 查询耗时统计
   - 连接状态监控
   - 慢查询告警

---

## 📚 完整文件列表

### 新增的文件

```
jas-agent/
├── agent/
│   ├── es_agent.go              ✅ ES Agent
│   └── es_executor.go           ✅ ES Executor
├── tools/
│   └── es_tools.go              ✅ ES 工具集
├── core/
│   ├── prompt.go                ✅ 更新
│   └── prompt_es.go             ✅ ES 提示词
├── storage/
│   ├── db.go                    ✅ 更新（connection_config字段）
│   └── agent_repo.go            ✅ 更新（CRUD支持连接配置）
├── server/
│   ├── agent_service.go         ✅ 更新（支持sql/es框架）
│   └── agent_handlers.go        ✅ 更新（connection_config）
├── api/proto/
│   └── agent_service.proto      ✅ 更新（connection_config字段）
├── web/src/components/
│   ├── AgentManageModal.jsx     ✅ 更新（动态配置表单）
│   └── AgentManageModal.css     ✅ 更新（配置区域样式）
├── examples/es/
│   ├── main.go                  ✅ ES 示例
│   └── README.md                ✅ 文档
├── docs/
│   ├── ES_AGENT_GUIDE.md        ✅ ES 详细指南
│   └── SQL_ES_AGENT_CONFIG.md   ✅ 配置指南
├── scripts/
│   └── schema.sql               ✅ 更新（connection_config字段）
└── 使用说明_SQL和ES_Agent.md    ✅ 使用说明
```

---

## ✅ 验证清单

- [x] 数据库表结构更新 ✅
- [x] Proto 定义扩展 ✅
- [x] 后端 CRUD 支持连接配置 ✅
- [x] 前端动态表单 ✅
- [x] SQL Agent 实现 ✅
- [x] Elasticsearch Agent 实现 ✅
- [x] ES 工具集（5个工具）✅
- [x] ES 系统提示词 ✅
- [x] 示例代码 ✅
- [x] 完整文档 ✅
- [x] 前端构建成功 ✅
- [x] 后端编译成功 ✅

---

## 🚀 立即开始使用

### 第一步: 启动服务

```bash
go run cmd/server/main.go \
  -apiKey YOUR_KEY \
  -baseUrl YOUR_URL \
  -dsn "root:pass@tcp(localhost:3306)/jas_agent"
```

### 第二步: 准备数据源

**MySQL**:
```bash
mysql -u root -p
CREATE DATABASE testdb;
```

**Elasticsearch**:
```bash
# 确保 ES 运行
curl http://localhost:9200
```

### 第三步: 创建 Agent

访问 `http://localhost:8080`
- 点击 **"🤖 管理 Agent"**
- 添加 SQL 或 ES Agent
- 配置连接信息
- 保存

### 第四步: 开始对话

- 选择创建的 Agent
- 输入查询
- 享受智能数据分析！

---

**所有功能已完成，立即体验！** 🎊

