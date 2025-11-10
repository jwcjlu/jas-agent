# MCP 服务管理指南

## 概述

JAS Agent 支持动态管理 MCP (Model Context Protocol) 服务，您可以在 Web 界面上：
- 添加自定义 MCP 服务
- 查看 MCP 服务状态和工具列表
- 选择在对话中使用哪些 MCP 服务
- 移除不需要的 MCP 服务

## 🌐 Web 界面使用

### 1. 打开 MCP 服务管理

1. 访问 http://localhost:3000
2. 点击配置面板中的 **"🔌 MCP 服务"** 按钮
3. 打开 MCP 服务管理界面

### 2. 添加 MCP 服务

**步骤:**
1. 在"添加 MCP 服务"表单中填写：
   - **服务名称**: 自定义名称，如 `weather-mcp`
   - **服务端点**: MCP服务的URL，如 `http://localhost:8080/mcp`
2. 点击"添加服务"按钮
3. 等待服务连接和工具发现

**示例:**
```
服务名称: weather-mcp
服务端点: http://localhost:9000/mcp
```

**成功后:**
- ✅ 服务出现在列表中
- ✅ 显示状态为"活跃"
- ✅ 显示发现的工具数量
- ✅ 自动刷新可用工具列表

### 3. 查看 MCP 服务

已添加的服务会显示以下信息：
- **服务名称**: 您设置的名称
- **状态**: 活跃/未激活
- **端点**: 服务URL
- **工具数量**: 该服务提供的工具数
- **创建时间**: 添加时间
- **最后刷新**: 工具列表刷新时间

### 4. 选择使用的 MCP 服务

添加 MCP 服务后，在配置面板会出现"启用的 MCP 服务"选项：

**操作:**
1. 在配置面板中找到"启用的 MCP 服务"
2. 勾选想要使用的服务
3. 发送消息时，只有勾选的服务的工具可用

**示例:**
```
□ weather-mcp (5 工具)
☑ database-mcp (3 工具)
□ file-mcp (8 工具)
```

勾选后，只有 `database-mcp` 的 3 个工具会被 Agent 使用。

### 5. 移除 MCP 服务

**操作:**
1. 在 MCP 服务列表中找到要移除的服务
2. 点击"🗑️ 移除"按钮
3. 确认删除
4. 服务及其工具将被移除

## 🔌 API 使用

### HTTP API 端点

#### 获取 MCP 服务列表

```bash
GET /api/mcp/services
```

**响应:**
```json
{
  "services": [
    {
      "name": "weather-mcp",
      "endpoint": "http://localhost:9000/mcp",
      "active": true,
      "tool_count": 5,
      "created_at": "2025-11-03 10:23:45",
      "last_refresh": "2025-11-03 10:25:12"
    }
  ]
}
```

#### 添加 MCP 服务

```bash
POST /api/mcp/services
Content-Type: application/json

{
  "name": "weather-mcp",
  "endpoint": "http://localhost:9000/mcp"
}
```

**响应:**
```json
{
  "success": true,
  "message": "成功添加MCP服务 'weather-mcp'",
  "service": {
    "name": "weather-mcp",
    "endpoint": "http://localhost:9000/mcp",
    "active": true,
    "tool_count": 5,
    "created_at": "2025-11-03 10:23:45",
    "last_refresh": "2025-11-03 10:23:45"
  }
}
```

#### 移除 MCP 服务

```bash
DELETE /api/mcp/services/{name}
```

**示例:**
```bash
curl -X DELETE http://localhost:8080/api/mcp/services/weather-mcp
```

**响应:**
```json
{
  "success": true,
  "message": "成功移除MCP服务 'weather-mcp'"
}
```

### 在对话中使用 MCP 服务

在发送对话请求时，包含 `enabled_mcp_services` 参数：

```javascript
{
  "query": "查询北京的天气",
  "agent_type": "react",
  "model": "gpt-3.5-turbo",
  "enabled_mcp_services": ["weather-mcp"]
}
```

## 📝 使用场景

### 场景1: 天气查询服务

**添加服务:**
```
名称: weather-mcp
端点: http://localhost:9000/mcp
```

**使用:**
1. 勾选 `weather-mcp`
2. 输入: "北京今天的天气怎么样？"
3. Agent 会调用天气服务的工具

### 场景2: 文件操作服务

**添加服务:**
```
名称: file-mcp
端点: http://localhost:9001/mcp
```

**使用:**
1. 勾选 `file-mcp`
2. 输入: "读取 config.json 文件的内容"
3. Agent 会调用文件操作工具

### 场景3: 多服务组合

**添加多个服务:**
```
weather-mcp -> 天气查询
database-mcp -> 数据库操作
file-mcp -> 文件操作
```

**使用:**
1. 根据需要勾选服务
2. Agent 可以使用所有勾选服务的工具
3. 实现复杂的跨服务任务

## 🛠️ MCP 工具命名

MCP 工具会自动添加服务前缀，避免命名冲突：

**格式:**
```
{服务名}@{工具名}
```

**示例:**
```
weather-mcp@get_weather
weather-mcp@get_forecast
database-mcp@query
database-mcp@insert
```

在 Agent 执行时，您会看到完整的工具名称。

## 📊 工具列表查看

点击"查看工具"按钮，可以看到：
- **Normal 工具**: 内置工具（calculator、averageDogWeight等）
- **MCP 工具**: 来自MCP服务的工具（带服务前缀）

**示例显示:**
```
calculator (Normal)
  数学表达式计算器

averageDogWeight (Normal)
  查询狗狗品种的平均体重

weather-mcp@get_weather (MCP)
  获取指定城市的天气信息
  来自: weather-mcp

weather-mcp@get_forecast (MCP)
  获取天气预报
  来自: weather-mcp
```

## ⚙️ 高级配置

### 工具自动发现

MCP 服务添加后：
- 每 5 秒自动刷新工具列表
- 新工具会自动被发现
- 工具数量实时更新

### 服务状态监控

服务信息包括：
- **Active**: 服务是否活跃
- **Tool Count**: 提供的工具数量
- **Last Refresh**: 最后刷新时间

### 选择性启用

您可以添加多个 MCP 服务，但只在需要时启用特定服务：
- ✅ 节省资源
- ✅ 提高响应速度
- ✅ 避免工具冲突

## 🚨 故障排查

### 问题1: 添加服务失败

**可能原因:**
- MCP 服务端点不可访问
- URL 格式错误
- 服务未启动

**解决方案:**
- 确认 MCP 服务正在运行
- 检查端点 URL 是否正确
- 查看后端终端日志

### 问题2: 工具数量为0

**可能原因:**
- MCP 服务没有提供工具
- 工具发现失败
- 网络连接问题

**解决方案:**
- 检查 MCP 服务是否正常工作
- 查看后端日志
- 等待自动刷新（5秒）

### 问题3: 工具调用失败

**可能原因:**
- 未启用对应的 MCP 服务
- 工具参数错误
- MCP 服务异常

**解决方案:**
- 确认勾选了正确的 MCP 服务
- 检查 Agent 的工具调用参数
- 查看 MCP 服务日志

## 💡 最佳实践

### 1. 服务命名

**推荐:**
- 使用描述性名称：`weather-service`, `db-connector`
- 使用小写和连字符：`my-mcp-service`
- 避免特殊字符：不要使用 `@`, `/` 等

### 2. 端点配置

**确保:**
- URL 包含协议：`http://` 或 `https://`
- 端口号正确：`:8080`, `:9000` 等
- 路径完整：`/mcp`, `/api/mcp` 等

### 3. 服务管理

**建议:**
- 定期检查服务状态
- 移除不再使用的服务
- 根据任务需求选择性启用

### 4. 安全考虑

**注意:**
- 只添加信任的 MCP 服务
- 验证服务端点的安全性
- 注意工具的权限范围

## 📖 示例

### 完整的使用流程

**步骤1: 添加天气服务**
```
1. 点击"🔌 MCP 服务"
2. 填写名称: weather-service
3. 填写端点: http://localhost:9000/mcp
4. 点击"添加服务"
5. 看到服务添加成功提示
```

**步骤2: 启用服务**
```
1. 关闭 MCP 管理界面
2. 在配置面板看到新增的"启用的 MCP 服务"选项
3. 勾选 weather-service
```

**步骤3: 使用服务**
```
1. 输入: "北京今天的天气怎么样？"
2. 点击发送
3. Agent 会自动调用 weather-service@get_weather 工具
4. 返回天气信息
```

**步骤4: 查看执行过程（流式模式）**
```
[💭 思考]
Thought: 我需要查询北京的天气
Action: weather-service@get_weather[北京]

[👁️ 观察]
Observation: {"city": "北京", "temp": 15, "weather": "晴"}

============================================================
📊 最终答案：
============================================================
北京今天天气晴朗，温度15度。
```

---

## 🎉 总结

MCP 服务管理功能让您可以：

✅ **动态扩展**: 随时添加新的外部工具  
✅ **灵活选择**: 按需启用需要的服务  
✅ **实时更新**: 自动发现新工具  
✅ **简单管理**: 可视化界面操作  
✅ **安全可控**: 完全掌控使用的服务  

通过 MCP 服务管理，JAS Agent 可以连接到任何支持 MCP 协议的服务，大大扩展了系统的能力！

---

**立即体验:** 打开 Web 界面，点击"🔌 MCP 服务"开始管理！

