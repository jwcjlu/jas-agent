# SQL 和 Elasticsearch Agent 配置指南

## 概述

JAS Agent 现在支持 SQL 和 Elasticsearch 两种特殊框架的 Agent，这些 Agent 需要配置数据源连接信息才能正常工作。

## SQL Agent 配置

### 连接配置字段

| 字段 | 说明 | 必填 | 示例 |
|------|------|------|------|
| host | MySQL 服务器地址 | ✅ | localhost |
| port | MySQL 端口 | ✅ | 3306 |
| database | 数据库名称 | ✅ | mydb |
| username | 用户名 | ✅ | root |
| password | 密码 | ❌ | mypassword |

### Web 界面配置

1. 点击 **"🤖 管理 Agent"** → **"➕ 添加 Agent"**
2. 选择框架类型: **"🗄️ SQL - MySQL数据库查询"**
3. 填写 MySQL 连接配置：
   ```
   主机: localhost
   端口: 3306
   数据库名称: testdb
   用户名: root
   密码: ********
   ```
4. 保存

### API 配置示例

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "SQL查询助手",
    "framework": "sql",
    "description": "MySQL数据库查询专家",
    "max_steps": 15,
    "model": "gpt-3.5-turbo",
    "connection_config": "{\"host\":\"localhost\",\"port\":3306,\"database\":\"testdb\",\"username\":\"root\",\"password\":\"mypass\"}"
  }'
```

### 连接配置 JSON 格式

```json
{
  "host": "localhost",
  "port": 3306,
  "database": "testdb",
  "username": "root",
  "password": "mypass"
}
```

### 可用工具

SQL Agent 创建后会自动获得以下工具：

1. **list_tables** - 列出所有表
2. **tables_schema** - 获取表结构
3. **execute_sql** - 执行 SQL 查询（仅 SELECT）

### 使用示例

```
用户: "查询用户表有多少条记录"

Agent:
  1. 使用 list_tables 查看所有表
  2. 使用 tables_schema 了解 users 表结构
  3. 构建 SQL: SELECT COUNT(*) FROM users
  4. 使用 execute_sql 执行查询
  5. 返回结果
```

---

## Elasticsearch Agent 配置

### 连接配置字段

| 字段 | 说明 | 必填 | 示例 |
|------|------|------|------|
| host | ES 服务地址（含协议） | ✅ | http://localhost:9200 |
| username | 用户名（如需认证） | ❌ | elastic |
| password | 密码（如需认证） | ❌ | changeme |

### Web 界面配置

1. 点击 **"🤖 管理 Agent"** → **"➕ 添加 Agent"**
2. 选择框架类型: **"🔍 Elasticsearch - 日志搜索分析"**
3. 填写 Elasticsearch 连接配置：
   ```
   ES 服务地址: http://localhost:9200
   用户名: elastic (可选)
   密码: ******** (可选)
   ```
4. 保存

### API 配置示例

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "日志分析助手",
    "framework": "elasticsearch",
    "description": "Elasticsearch日志搜索和分析专家",
    "max_steps": 15,
    "model": "gpt-3.5-turbo",
    "connection_config": "{\"host\":\"http://localhost:9200\",\"username\":\"elastic\",\"password\":\"changeme\"}"
  }'
```

### 连接配置 JSON 格式

```json
{
  "host": "http://localhost:9200",
  "username": "elastic",
  "password": "changeme"
}
```

### 可用工具

Elasticsearch Agent 创建后会自动获得以下工具：

1. **list_indices** - 列出所有索引
2. **get_index_mapping** - 获取索引映射
3. **search_documents** - 搜索文档
4. **get_document** - 获取指定文档
5. **aggregate_data** - 聚合查询

### 使用示例

```
用户: "搜索最近的错误日志"

Agent:
  1. 使用 list_indices 查找日志索引
  2. 使用 get_index_mapping 了解字段结构
  3. 构建查询 DSL: {"match": {"level": "ERROR"}}
  4. 使用 search_documents 执行搜索
  5. 返回格式化的结果
```

---

## 完整配置流程

### 步骤 1: 准备数据源

**MySQL**:
```bash
# 确保 MySQL 运行
mysql -u root -p

# 创建测试数据库
CREATE DATABASE testdb;
USE testdb;
CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100));
```

**Elasticsearch**:
```bash
# 确保 ES 运行
curl http://localhost:9200

# 创建测试索引
curl -X PUT http://localhost:9200/logs \
  -H 'Content-Type: application/json' \
  -d '{"mappings": {"properties": {"timestamp": {"type": "date"}, "level": {"type": "keyword"}, "message": {"type": "text"}}}}'
```

### 步骤 2: 在 Web 界面创建 Agent

#### SQL Agent
```
名称: SQL查询助手
框架: sql
描述: 专业的MySQL数据库查询助手
系统提示词: (可选)
模型: gpt-3.5-turbo
最大步数: 15

MySQL 连接配置:
  主机: localhost
  端口: 3306
  数据库名称: testdb
  用户名: root
  密码: yourpass
```

#### Elasticsearch Agent
```
名称: 日志分析助手
框架: elasticsearch
描述: Elasticsearch日志搜索和分析专家
系统提示词: (可选)
模型: gpt-3.5-turbo
最大步数: 15

Elasticsearch 连接配置:
  ES 服务地址: http://localhost:9200
  用户名: elastic (可选)
  密码: changeme (可选)
```

### 步骤 3: 选择并使用 Agent

1. 在主界面下拉框选择创建的 SQL 或 ES Agent
2. 输入查询，例如：
   - SQL Agent: "查询用户表的所有记录"
   - ES Agent: "搜索包含error的日志"
3. 查看 Agent 自动执行的完整流程

## 安全注意事项

### 密码存储

⚠️ **重要**: 连接配置（包括密码）以明文形式存储在数据库中。

**生产环境建议**:
1. 使用环境变量或密钥管理系统
2. 加密敏感字段
3. 限制数据库访问权限
4. 使用只读数据库用户
5. 定期轮换密码

### 权限控制

**SQL Agent**:
- ✅ 仅支持 SELECT 查询
- ❌ 不支持 INSERT、UPDATE、DELETE
- ✅ 建议使用只读用户

**Elasticsearch Agent**:
- ✅ 仅支持读操作（搜索、聚合）
- ❌ 不支持索引修改
- ❌ 不支持文档写入
- ✅ 建议使用只读角色

## 故障排除

### SQL Agent

**问题**: 连接失败

**解决方法**:
1. 检查 MySQL 是否运行
2. 验证主机、端口、数据库名称
3. 确认用户名密码正确
4. 检查防火墙设置

**问题**: 查询权限不足

**解决方法**:
```sql
GRANT SELECT ON testdb.* TO 'readonly'@'localhost';
FLUSH PRIVILEGES;
```

### Elasticsearch Agent

**问题**: 连接失败

**解决方法**:
1. 检查 ES 是否运行: `curl http://localhost:9200`
2. 验证服务地址格式（需包含 http://）
3. 确认认证信息正确
4. 检查网络连接

**问题**: 索引不存在

**解决方法**:
```bash
# 列出所有索引
curl http://localhost:9200/_cat/indices

# 创建测试索引
curl -X PUT http://localhost:9200/test-index
```

## 配置示例

### 开发环境

**SQL Agent**:
```json
{
  "host": "localhost",
  "port": 3306,
  "database": "dev_db",
  "username": "dev_user",
  "password": "dev_pass"
}
```

**ES Agent**:
```json
{
  "host": "http://localhost:9200",
  "username": "",
  "password": ""
}
```

### 生产环境

**SQL Agent** (使用只读用户):
```json
{
  "host": "prod-mysql.example.com",
  "port": 3306,
  "database": "production",
  "username": "readonly_user",
  "password": "secure_password"
}
```

**ES Agent** (使用认证):
```json
{
  "host": "https://es-cluster.example.com:9200",
  "username": "readonly",
  "password": "secure_password"
}
```

## 最佳实践

### 1. 命名规范
```
✅ 好: "销售数据库查询助手"
✅ 好: "生产环境日志分析"
❌ 差: "Agent1"
❌ 差: "测试"
```

### 2. 连接配置
```
✅ 开发环境使用本地数据库
✅ 生产环境使用只读用户
✅ 定期测试连接
❌ 避免使用 root 用户
❌ 避免暴露敏感信息
```

### 3. Agent 描述
```
✅ 好: "专业的MySQL数据库查询助手，擅长复杂SQL分析和报表生成"
❌ 差: "查询数据库"
```

## 相关文档

- [SQL Agent 示例](../agent/examples/sql/README.md)
- [ES Agent 示例](../agent/examples/es/README.md)
- [ES Agent 详细指南](./ES_AGENT_GUIDE.md)
- [Agent 管理指南](./AGENT_MANAGEMENT_GUIDE.md)

## 总结

SQL 和 Elasticsearch Agent 为 JAS Agent 系统增加了强大的数据查询和分析能力：

✅ **SQL Agent** - 智能SQL查询生成和执行
✅ **ES Agent** - 复杂的日志搜索和数据分析
✅ **连接配置** - 灵活的数据源配置
✅ **安全控制** - 只读操作，权限限制
✅ **易于使用** - Web 界面可视化配置

立即创建您的第一个数据查询 Agent！🚀

