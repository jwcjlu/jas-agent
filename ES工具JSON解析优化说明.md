# Elasticsearch 工具 JSON 解析优化说明

## 🐛 问题分析

### 遇到的错误

```
Action: search_documents[{"index": "backend-vm_manager-2025.11.06", "query": {"bool": {"must": [{"term": {"L": "ERROR"}}, {"match": {"M": "4103_sujai1datfsi"}}]}}}

错误: invalid input format: unexpected end of JSON input
```

### 问题原因

**JSON 不完整** - 缺少最后的右括号 `]`

完整的应该是：
```
Action: search_documents[{"index": "backend-vm_manager-2025.11.06", "query": {"bool": {"must": [{"term": {"L": "ERROR"}}, {"match": {"M": "4103_sujai1datfsi"}}]}}}]
                                                                                                                                                                ^
                                                                                                                                                          缺少这个
```

### 可能的原因

1. **LLM 生成不完整** - 模型在生成时被截断
2. **Token 限制** - 响应长度限制导致截断
3. **解析器问题** - 我们的括号匹配算法提取不完整

---

## ✅ 优化方案

### 1. 增强错误提示

**优化前**:
```
错误: invalid input format: unexpected end of JSON input
```
❌ 不够清晰，用户不知道哪里错了

**优化后**:
```
错误: JSON解析失败: unexpected end of JSON input

输入内容:
{"index": "backend-vm_manager-2025.11.06", "query": {"bool": {"must": [{"term": {"L": "ERROR"}}, {"match": {"M": "4103_sujai1datfsi"}}]}}}

请确保JSON格式正确，所有括号都已闭合
```
✅ 显示完整输入，方便调试

### 2. aggregate_data 支持 query 参数

**优化前**:
```json
{
  "index": "logs",
  "aggs": {...}
}
```
❌ 无法过滤，聚合所有数据

**优化后**:
```json
{
  "index": "logs",
  "query": {"term": {"L": "ERROR"}},  ⭐ 支持查询过滤
  "aggs": {...}
}
```
✅ 可以先过滤再聚合，更灵活

### 3. 字段名称简化

**L 字段查询**:
```
优化前: {"term": {"L.keyword": "ERROR"}}
优化后: {"term": {"L": "ERROR"}}          ⭐ 去掉 .keyword
```

---

## 🔧 修复的代码

### SearchDocuments 工具

```go
func (t *SearchDocuments) Handler(ctx context.Context, input string) (string, error) {
    var searchReq struct {
        Index string                 `json:"index"`
        Query map[string]interface{} `json:"query"`
        Size  int                    `json:"size"`
    }
    
    if err := json.Unmarshal([]byte(input), &searchReq); err != nil {
        // ⭐ 更友好的错误信息
        return "", fmt.Errorf("JSON解析失败: %w\n\n输入内容:\n%s\n\n请确保JSON格式正确，所有括号都已闭合", err, input)
    }
    
    // ...
}
```

### AggregateData 工具

```go
func (t *AggregateData) Handler(ctx context.Context, input string) (string, error) {
    var aggReq struct {
        Index string                 `json:"index"`
        Query map[string]interface{} `json:"query"` // ⭐ 新增：支持查询过滤
        Aggs  map[string]interface{} `json:"aggs"`
    }
    
    if err := json.Unmarshal([]byte(input), &aggReq); err != nil {
        // ⭐ 更友好的错误信息
        return "", fmt.Errorf("JSON解析失败: %w\n\n输入内容:\n%s\n\n请确保JSON格式正确，所有括号都已闭合", err, input)
    }
    
    // 构建聚合请求
    aggBody := map[string]interface{}{
        "size": 0,
    }
    
    // ⭐ 如果有查询条件，添加到请求中
    if aggReq.Query != nil && len(aggReq.Query) > 0 {
        aggBody["query"] = aggReq.Query
    }
    
    aggBody["aggs"] = aggReq.Aggs
    
    // ...
}
```

---

## 📋 正确的用法

### 正确的 Action 格式

**完整的 Action**（注意最后的 `]`）:
```
Action: search_documents[{"index": "backend-vm_manager-2025.11.06", "query": {"bool": {"must": [{"term": {"L": "ERROR"}}, {"match": {"M": "4103_sujai1datfsi"}}]}}}]
                                                                                                                                                                ^
                                                                                                                                                          必须有这个
```

### 复合查询示例

**查询特定用户的错误日志**:
```json
{
  "index": "backend-vm_manager-2025.11.06",
  "query": {
    "bool": {
      "must": [
        {"term": {"L": "ERROR"}},
        {"term": {"M": "4103_sujai1datfsi"}}    // 使用 term 而不是 match
      ]
    }
  },
  "size": 20
}
```

**注意**:
- ✅ 用 `term` 查询精确匹配（keyword 字段）
- ✅ 用 `match` 查询全文搜索（text 字段）
- ✅ M 字段如果是 ID，应该用 `term`

### 带过滤的聚合查询

**统计某用户的错误趋势**:
```json
{
  "index": "backend-vm_manager-*",
  "query": {                                    // ⭐ 先过滤
    "bool": {
      "must": [
        {"term": {"M": "4103_sujai1datfsi"}},
        {"term": {"L": "ERROR"}}
      ]
    }
  },
  "aggs": {                                     // ⭐ 再聚合
    "hourly": {
      "date_histogram": {
        "field": "T",
        "calendar_interval": "hour"
      }
    }
  }
}
```

---

## 🎯 常见错误和修复

### 错误 1: 括号不匹配

**错误示例**:
```
Action: search_documents[{"query": {"bool": {"must": [...]}}
                         ^                                   ← 缺少 }]
```

**修复**:
```
Action: search_documents[{"query": {"bool": {"must": [...]}}}]
                         ^                                   ^^
```

### 错误 2: 逗号错误

**错误示例**:
```json
{
  "index": "logs",
  "query": {...},    // 多余的逗号
}
```

**修复**:
```json
{
  "index": "logs",
  "query": {...}     // 去掉最后的逗号
}
```

### 错误 3: 引号未闭合

**错误示例**:
```json
{
  "index": "logs,
  "query": {...}
}
```

**修复**:
```json
{
  "index": "logs",   // 引号要闭合
  "query": {...}
}
```

---

## 💡 调试技巧

### 1. 检查括号匹配

```
复制 Action 的 JSON 部分
使用 JSON 格式化工具验证:
  - https://jsonformatter.org/
  - VS Code: Format Document
```

### 2. 逐步简化查询

```
复杂查询出错时，先简化:

第一步: 测试基本查询
Action: search_documents[{"index": "logs", "query": {"match_all": {}}}]

第二步: 添加简单过滤
Action: search_documents[{"index": "logs", "query": {"term": {"L": "ERROR"}}}]

第三步: 添加复合条件
Action: search_documents[{"index": "logs", "query": {"bool": {"must": [...]}}}]
```

### 3. 查看错误信息

**新的错误信息会显示**:
```
JSON解析失败: unexpected end of JSON input

输入内容:
{"index": "backend-vm_manager-2025.11.06", ...}    ← 可以看到实际输入

请确保JSON格式正确，所有括号都已闭合              ← 明确的建议
```

---

## 📝 提示词补充

### 添加 JSON 格式注意事项

```
JSON 格式要求（重要！）:
  ⭐ 确保所有括号都已闭合
  - 每个 { 必须有对应的 }
  - 每个 [ 必须有对应的 ]
  - 最外层的 [...] 是 Action 的参数括号，不要遗漏
  
  常见错误:
  ❌ Action: tool[{"a": {...}]     // 缺少一个 }
  ✅ Action: tool[{"a": {...}}]    // 正确
  
  ❌ Action: tool[{"a": [...]}     // 缺少最后的 ]
  ✅ Action: tool[{"a": [...]}]    // 正确
```

---

## ✅ 优化效果

### 错误提示对比

**优化前**:
```
Tool execution error: invalid input format: unexpected end of JSON input
```
❌ 信息不足，难以定位问题

**优化后**:
```
Tool execution error: JSON解析失败: unexpected end of JSON input

输入内容:
{"index": "backend-vm_manager-2025.11.06", "query": {"bool": {"must": [{"term": {"L": "ERROR"}}, {"match": {"M": "4103_sujai1datfsi"}}]}}}

请确保JSON格式正确，所有括号都已闭合
```
✅ 显示完整输入，明确指出问题

---

## 🎯 正确的查询示例

### 查询特定用户的错误

```
Action: search_documents[{
  "index": "backend-vm_manager-2025.11.06",
  "query": {
    "bool": {
      "must": [
        {"term": {"L": "ERROR"}},
        {"term": {"M": "4103_sujai1datfsi"}}
      ]
    }
  },
  "size": 20
}]
```
✅ 所有括号都闭合

### 聚合查询（带过滤）

```
Action: aggregate_data[{
  "index": "backend-vm_manager-*",
  "query": {
    "term": {"L": "ERROR"}
  },
  "aggs": {
    "hourly": {
      "date_histogram": {
        "field": "T",
        "calendar_interval": "hour"
      }
    }
  }
}]
```
✅ 支持 query 参数过滤后再聚合

---

## ✅ 编译验证

```bash
✅ go build -o jas-agent-server.exe cmd/server/main.go
   编译成功！
```

---

## 🎊 总结

**已优化**:
1. ✅ 更友好的 JSON 解析错误提示
2. ✅ 显示完整的输入内容
3. ✅ aggregate_data 支持 query 参数
4. ✅ L 字段简化（去掉 .keyword）
5. ✅ 明确的格式要求说明

**现在工具错误时会提供更详细的信息，帮助快速定位和修复问题！** 🎉

