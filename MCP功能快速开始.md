# 🔌 MCP 管理功能 - 快速开始

## 🎉 新功能已实现！

现在您可以在 Web 界面上动态管理 MCP 服务，无需修改代码！

---

## 🚀 5分钟快速体验

### 步骤1: 启动服务

```bash
# 终端1: 启动后端
cd cmd/server
go run main.go -apiKey YOUR_API_KEY -baseUrl YOUR_BASE_URL

# 终端2: 启动前端
cd web
npm run dev
```

### 步骤2: 打开 MCP 管理界面

1. 访问 http://localhost:3000
2. 找到配置面板
3. 点击 **"🔌 MCP 服务"** 按钮

### 步骤3: 添加 MCP 服务

**在弹出的界面中:**
1. 填写服务名称: `my-mcp`
2. 填写服务端点: `http://localhost:8080/mcp`
3. 点击"添加服务"

**成功后看到:**
```
✅ 成功添加MCP服务 'my-mcp'

my-mcp          ✅ 活跃      [🗑️ 移除]
端点: http://localhost:8080/mcp
工具数量: 5
创建时间: 2025-11-03 10:23:45
```

### 步骤4: 选择使用的服务

1. 关闭 MCP 管理界面
2. 在配置面板看到 **"启用的 MCP 服务"**
3. 勾选要使用的服务：
   ```
   ☑ my-mcp (5 工具)
   ```

### 步骤5: 使用 MCP 工具

1. 输入问题（使用MCP工具的场景）
2. 点击发送
3. Agent 会自动调用勾选服务的工具

**流式响应效果:**
```
[💭 思考]
Thought: 我需要调用 MCP 工具
Action: my-mcp@some_tool[参数]

[👁️ 观察]
Observation: 工具执行结果

============================================================
📊 最终答案：
============================================================
基于工具结果的回答
```

---

## 📸 界面预览

### MCP 管理界面

```
┌──────────────────────────────────────────┐
│ 🔌 MCP 服务管理                    [×]   │
├──────────────────────────────────────────┤
│                                          │
│ 【添加 MCP 服务】                        │
│                                          │
│ 服务名称: [weather-mcp          ]       │
│ 服务端点: [http://localhost:9000/mcp]   │
│           [添加服务]                     │
│                                          │
│ 【已添加的 MCP 服务 (2)】                │
│                                          │
│ ┌────────────────────────────────────┐  │
│ │ weather-mcp   ✅ 活跃   [🗑️ 移除]  │  │
│ │ 端点: http://localhost:9000/mcp    │  │
│ │ 工具数量: 5                        │  │
│ │ 创建时间: 2025-11-03 10:23:45     │  │
│ │ 最后刷新: 2025-11-03 10:25:12     │  │
│ └────────────────────────────────────┘  │
│                                          │
│ ┌────────────────────────────────────┐  │
│ │ db-mcp        ✅ 活跃   [🗑️ 移除]  │  │
│ │ 端点: http://localhost:9001/mcp    │  │
│ │ 工具数量: 3                        │  │
│ └────────────────────────────────────┘  │
└──────────────────────────────────────────┘
```

### 配置面板 - MCP 选择

```
┌──────────────────────────────────────────┐
│ Agent 类型: [ReAct Agent ▼]             │
│ 模型: [GPT-3.5 Turbo ▼]                 │
├──────────────────────────────────────────┤
│ 启用的 MCP 服务:                         │
│ ┌────────────────────────────────────┐  │
│ │ ☑ weather-mcp (5 工具)             │  │
│ │ ☑ db-mcp (3 工具)                  │  │
│ │ □ file-mcp (8 工具)                │  │
│ └────────────────────────────────────┘  │
│                                          │
│ [清空对话] [查看工具] [🔌 MCP 服务]      │
└──────────────────────────────────────────┘
```

---

## 🎯 功能特点

### ✨ 动态管理
- 无需重启服务即可添加/删除 MCP 服务
- 实时工具发现和刷新（每5秒）
- 即时生效，立即可用

### 🎯 灵活选择
- 可以添加多个 MCP 服务
- 按需勾选要使用的服务
- 避免不必要的工具调用

### 📊 状态监控
- 查看服务连接状态
- 查看工具数量
- 查看最后刷新时间

### 🛡️ 安全控制
- 完全掌控使用的服务
- 随时可以移除服务
- 服务隔离，互不影响

---

## 📝 使用示例

### 示例1: 添加天气服务

**场景:** 需要查询天气信息

**操作:**
```
1. 点击"🔌 MCP 服务"
2. 填写:
   名称: weather-service
   端点: http://localhost:9000/mcp
3. 添加服务
4. 勾选 weather-service
5. 输入: "上海明天天气怎么样？"
```

**Agent 执行:**
```
[💭 思考]
Action: weather-service@get_forecast[上海, 明天]

[👁️ 观察]
Observation: 明天上海多云，15-22度

📊 最终答案：
上海明天天气多云，温度15-22度。
```

### 示例2: 多服务组合使用

**场景:** 需要同时使用多个外部服务

**操作:**
```
1. 添加 weather-mcp
2. 添加 database-mcp
3. 添加 api-mcp
4. 勾选需要的服务
5. 发送复杂查询
```

**Agent 可以:**
- 查询天气（weather-mcp@get_weather）
- 查询数据库（database-mcp@query）
- 调用API（api-mcp@call）
- 综合处理结果

### 示例3: 按需启用服务

**场景:** 根据任务类型选择服务

**天气查询任务:**
```
☑ weather-mcp
□ database-mcp
□ file-mcp
```

**数据分析任务:**
```
□ weather-mcp
☑ database-mcp
□ file-mcp
```

**文件处理任务:**
```
□ weather-mcp
□ database-mcp
☑ file-mcp
```

---

## 🔧 API 使用

### JavaScript/Fetch

```javascript
// 添加 MCP 服务
fetch('http://localhost:8080/api/mcp/services', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    name: 'my-mcp',
    endpoint: 'http://localhost:9000/mcp'
  })
});

// 列出 MCP 服务
fetch('http://localhost:8080/api/mcp/services')
  .then(res => res.json())
  .then(data => console.log(data.services));

// 删除 MCP 服务
fetch('http://localhost:8080/api/mcp/services/my-mcp', {
  method: 'DELETE'
});
```

### Python

```python
import requests

# 添加 MCP 服务
response = requests.post('http://localhost:8080/api/mcp/services', json={
    'name': 'my-mcp',
    'endpoint': 'http://localhost:9000/mcp'
})
print(response.json())

# 列出 MCP 服务
response = requests.get('http://localhost:8080/api/mcp/services')
print(response.json())

# 删除 MCP 服务
response = requests.delete('http://localhost:8080/api/mcp/services/my-mcp')
print(response.json())
```

### curl

```bash
# 添加
curl -X POST http://localhost:8080/api/mcp/services \
  -H "Content-Type: application/json" \
  -d '{"name": "my-mcp", "endpoint": "http://localhost:9000/mcp"}'

# 列出
curl http://localhost:8080/api/mcp/services

# 删除
curl -X DELETE http://localhost:8080/api/mcp/services/my-mcp
```

---

## ⚠️ 注意事项

### 服务命名
- 使用小写字母和连字符
- 避免特殊字符 `@`, `/`, `\` 等
- 推荐格式: `service-name-mcp`

### 端点URL
- 必须包含协议: `http://` 或 `https://`
- 确保端口正确
- 确保路径完整

### 安全性
- 只添加信任的 MCP 服务
- 验证服务来源
- 注意工具的权限

### 性能
- 不要添加过多服务
- 按需启用服务
- 定期清理不用的服务

---

## 🐛 故障排查

### 问题1: 添加服务失败

**检查:**
- MCP 服务是否正在运行
- 端点 URL 是否正确
- 网络连接是否正常

**查看日志:**
```
后端终端查看错误信息
```

### 问题2: 工具数量为0

**原因:**
- 服务刚添加，工具还未发现
- 服务没有提供工具
- 工具发现失败

**解决:**
- 等待 5-10 秒让工具发现完成
- 检查 MCP 服务的工具列表
- 查看后端日志

### 问题3: 工具调用失败

**检查:**
- 是否勾选了对应的 MCP 服务
- MCP 服务是否正常工作
- 工具参数是否正确

---

## 🎊 总结

MCP 管理功能让 JAS Agent 变得更加强大和灵活：

✅ **动态扩展**: 随时添加新工具  
✅ **灵活配置**: 按需选择服务  
✅ **简单管理**: 可视化界面操作  
✅ **实时更新**: 自动工具发现  
✅ **安全可控**: 完全掌控服务  

---

**立即体验:** 访问 http://localhost:3000，点击"🔌 MCP 服务"开始管理！

**详细文档:** [MCP 服务管理指南](docs/MCP_MANAGEMENT_GUIDE.md)

