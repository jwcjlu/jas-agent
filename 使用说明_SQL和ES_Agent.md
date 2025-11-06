# SQL 和 Elasticsearch Agent 使用说明

## 🎯 功能概述

现在您可以在 Web 界面创建 **SQL Agent** 和 **Elasticsearch Agent**，这两种 Agent 需要配置数据源连接信息。

---

## 📝 创建 SQL Agent

### 步骤 1: 打开 Agent 管理

访问 `http://localhost:8080`，点击 **"🤖 管理 Agent"** 按钮。

### 步骤 2: 添加 SQL Agent

点击 **"➕ 添加 Agent"**，填写以下信息：

#### 基本信息
```
Agent 名称: SQL查询助手
框架类型: 🗄️ SQL - MySQL数据库查询（需配置数据库）
描述: 专业的MySQL数据库查询和分析助手
系统提示词: (可选留空)
模型: GPT-3.5 Turbo
最大步数: 15
```

#### MySQL 连接配置 ⭐

选择 SQL 框架后，会自动显示 MySQL 连接配置区域：

```
📊 MySQL 连接配置
┌────────────────────────────────────┐
│ 主机 *         端口 *              │
│ localhost      3306                │
│                                    │
│ 数据库名称 *                       │
│ testdb                             │
│                                    │
│ 用户名 *       密码                │
│ root           ********            │
└────────────────────────────────────┘
```

**字段说明**:
- **主机**: MySQL 服务器地址（localhost 或 IP）
- **端口**: 默认 3306
- **数据库名称**: 要连接的数据库
- **用户名**: 数据库用户（建议使用只读用户）
- **密码**: 数据库密码（可选）

#### 保存

点击 **"保存"** 按钮，Agent 创建完成！

---

## 🔍 创建 Elasticsearch Agent

### 步骤 1: 打开 Agent 管理

同上。

### 步骤 2: 添加 ES Agent

点击 **"➕ 添加 Agent"**，填写以下信息：

#### 基本信息
```
Agent 名称: 日志分析助手
框架类型: 🔍 Elasticsearch - 日志搜索分析（需配置ES）
描述: Elasticsearch 日志搜索和数据分析专家
系统提示词: (可选留空)
模型: GPT-3.5 Turbo
最大步数: 15
```

#### Elasticsearch 连接配置 ⭐

选择 Elasticsearch 框架后，会自动显示 ES 连接配置区域：

```
🔍 Elasticsearch 连接配置
┌────────────────────────────────────┐
│ ES 服务地址 *                      │
│ http://localhost:9200              │
│                                    │
│ 用户名 (可选)  密码 (可选)        │
│ elastic         ********           │
└────────────────────────────────────┘
```

**字段说明**:
- **ES 服务地址**: 完整的 HTTP 地址（必须包含 http:// 或 https://）
- **用户名**: ES 用户名（如不需要认证可留空）
- **密码**: ES 密码（如不需要认证可留空）

#### 保存

点击 **"保存"** 按钮，Agent 创建完成！

---

## 💬 使用 Agent 进行对话

### SQL Agent 使用示例

#### 1. 选择 SQL Agent

在主界面下拉框中选择 **"SQL查询助手"**。

#### 2. 输入查询

```
示例 1: "查询用户表的所有记录"
示例 2: "统计每个月的订单总额"
示例 3: "查找最近注册的10个用户"
示例 4: "分析销售数据，找出销售额最高的产品"
```

#### 3. 查看执行过程

Agent 会自动执行以下步骤：

```
💭 思考: 需要先了解数据库结构
⚙️ 执行: list_tables[]
👁️ 观察: 找到 users, orders, products 表

💭 思考: 查看表结构
⚙️ 执行: tables_schema[users]
👁️ 观察: id, name, email, created_at...

💭 思考: 构建 SQL 查询
⚙️ 执行: execute_sql[SELECT * FROM users LIMIT 10]
👁️ 观察: 返回查询结果

📊 最终答案: 用户表共有 X 条记录，以下是前10条...
```

### Elasticsearch Agent 使用示例

#### 1. 选择 ES Agent

在主界面下拉框中选择 **"日志分析助手"**。

#### 2. 输入查询

```
示例 1: "搜索包含error的日志"
示例 2: "统计每小时的日志数量"
示例 3: "查找今天的错误日志"
示例 4: "分析用户访问行为"
```

#### 3. 查看执行过程

Agent 会自动执行以下步骤：

```
💭 思考: 需要先了解可用的索引
⚙️ 执行: list_indices[]
👁️ 观察: 找到 logs-2024-11 索引

💭 思考: 查看索引结构
⚙️ 执行: get_index_mapping[logs-2024-11]
👁️ 观察: timestamp, level, message 等字段

💭 思考: 构建搜索查询
⚙️ 执行: search_documents[{"index":"logs-2024-11","query":{"match":{"message":"error"}}}]
👁️ 观察: 找到 X 条匹配的日志

📊 最终答案: 找到 X 条包含error的日志，详情如下...
```

---

## ⚙️ 配置格式参考

### MySQL 连接配置 JSON

存储在数据库中的格式：

```json
{
  "host": "localhost",
  "port": 3306,
  "database": "testdb",
  "username": "root",
  "password": "mypassword"
}
```

### Elasticsearch 连接配置 JSON

存储在数据库中的格式：

```json
{
  "host": "http://localhost:9200",
  "username": "elastic",
  "password": "changeme"
}
```

---

## 🔒 安全建议

### 生产环境配置

#### SQL Agent
```
✅ 使用只读数据库用户
✅ 限制可访问的表
✅ 使用专用数据库连接
❌ 不要使用 root 用户
❌ 不要授予写权限
```

创建只读用户：
```sql
CREATE USER 'readonly'@'%' IDENTIFIED BY 'secure_password';
GRANT SELECT ON mydb.* TO 'readonly'@'%';
FLUSH PRIVILEGES;
```

#### Elasticsearch Agent
```
✅ 使用只读角色
✅ 限制索引访问权限
✅ 使用 HTTPS 连接
❌ 不要使用超级管理员账户
❌ 不要授予写权限
```

创建只读角色（Kibana）：
```json
{
  "cluster": ["monitor"],
  "indices": [
    {
      "names": ["logs-*"],
      "privileges": ["read", "view_index_metadata"]
    }
  ]
}
```

---

## 🐛 故障排除

### SQL Agent

**问题**: "连接失败"

**解决**:
```bash
# 1. 测试 MySQL 连接
mysql -h localhost -u root -p testdb

# 2. 检查防火墙
# 3. 验证用户权限
SHOW GRANTS FOR 'root'@'localhost';
```

**问题**: "表不存在"

**解决**:
```sql
-- 列出所有表
SHOW TABLES;

-- 检查数据库
SHOW DATABASES;
```

### Elasticsearch Agent

**问题**: "连接失败"

**解决**:
```bash
# 1. 测试 ES 连接
curl http://localhost:9200

# 2. 测试认证
curl -u elastic:password http://localhost:9200

# 3. 检查网络
ping localhost
```

**问题**: "索引不存在"

**解决**:
```bash
# 列出所有索引
curl http://localhost:9200/_cat/indices?v

# 创建测试索引
curl -X PUT http://localhost:9200/test-index
```

---

## 📚 相关文档

- **配置详细指南**: [docs/SQL_ES_AGENT_CONFIG.md](docs/SQL_ES_AGENT_CONFIG.md)
- **ES Agent 使用**: [docs/ES_AGENT_GUIDE.md](docs/ES_AGENT_GUIDE.md)
- **Agent 管理**: [docs/AGENT_MANAGEMENT_GUIDE.md](docs/AGENT_MANAGEMENT_GUIDE.md)
- **SQL 示例**: [examples/sql/README.md](examples/sql/README.md)
- **ES 示例**: [examples/es/README.md](examples/es/README.md)

---

## 🎊 开始使用

1. ✅ 准备好 MySQL 或 Elasticsearch
2. ✅ 在 Web 界面创建对应的 Agent
3. ✅ 填写连接配置信息
4. ✅ 保存并选择 Agent
5. ✅ 开始智能数据查询和分析！

有任何问题，请查看详细文档或提交 Issue。

祝使用愉快！🚀

