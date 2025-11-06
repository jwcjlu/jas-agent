# Elasticsearch 两步查询策略说明

## 🎯 优化需求

当发现多个索引具有相同前缀（仅日期不同）时，采用**两步查询策略**：
1. **第一步**：优先查询**最新的索引**
2. **第二步**：如果查不到数据，再使用**通配符**查询所有索引

---

## ✨ 实现方案

### 1. search_indices 工具优化

**返回格式更新**:
```
找到 3 个包含关键词 'backend' 的索引：

- backend-vm_manager-2025.11.04
  Health: green, Docs: 15234, Size: 2.3mb
- backend-vm_manager-2025.11.03
  Health: green, Docs: 14567, Size: 2.1mb
- backend-vm_manager-2025.11.02
  Health: green, Docs: 13890, Size: 2.0mb

💡 查询策略建议：
   1️⃣ 优先查询最新索引：'backend-vm_manager-2025.11.04'    ⭐
   2️⃣ 如果查不到数据，再使用通配符 'backend-vm_manager-*' 查询所有相关索引
```

**实现逻辑**:
```go
// 1. 检测相同前缀索引
wildcardSuggestion := detectWildcardPattern(indexNames)

// 2. 找出最新的索引
latestIndex := findLatestIndex(indexNames)  // 按字符串排序，日期越新越大

// 3. 提供两步建议
if wildcardSuggestion != "" {
    建议 1: 先查 latestIndex
    建议 2: 查不到再用 wildcardSuggestion
}
```

### 2. 系统提示词更新

**查询策略**:
```
⭐ 当发现多个索引具有相同前缀，仅日期不同时，采用两步查询策略

第一步：优先查询最新索引
  - search_indices 会返回最新的索引建议
  - 先用最新索引查询（通常最新数据概率更高）
  - 例如: backend-vm_manager-2025.11.04（最新）

第二步：如果查不到数据，使用通配符查询所有
  - 如果第一步返回结果为空
  - 使用通配符模式查询所有相关索引
  - 例如: backend-vm_manager-* （所有日期）
```

### 3. Few-shot 示例更新

**示例 6: 优先查询最新索引**
```
Observation: search_indices返回建议
   1️⃣ 优先查询最新索引：'backend-vm_manager-2025.11.04'
   2️⃣ 如果查不到，再用 'backend-vm_manager-*'

Thought: 按建议先查询最新的索引
Action: search_documents[{
  "index": "backend-vm_manager-2025.11.04",    ⭐ 第一步：最新索引
  "query": {"term": {"L.keyword": "ERROR"}}
}]
```

**示例 7: 扩大查询范围**
```
Observation: 在 backend-vm_manager-2025.11.04 中未找到匹配的文档

Thought: 最新索引中没有数据，现在使用通配符查询所有历史索引
Action: search_documents[{
  "index": "backend-vm_manager-*",                     ⭐ 第二步：通配符
  "query": {"term": {"L.keyword": "ERROR"}}
}]
```

---

## 📊 完整执行流程

### 场景: 查询 backend 项目的错误日志

```
用户输入: "查询backend项目的错误日志"
  ↓
┌────────────────────────────────────────┐
│ Step 1: 查找相关索引                   │
├────────────────────────────────────────┤
│ Thought: 用户提到backend，先查找索引   │
│ Action: search_indices[backend]        │
└──────────────┬─────────────────────────┘
               ↓
┌────────────────────────────────────────┐
│ Observation: 工具返回                  │
├────────────────────────────────────────┤
│ 找到 3 个索引:                         │
│ - backend-vm_manager-2025.11.04       │
│ - backend-vm_manager-2025.11.03       │
│ - backend-vm_manager-2025.11.02       │
│                                        │
│ 💡 建议:                               │
│ 1️⃣ 先查: backend-vm_manager-2025.11.04│ ⭐ 最新
│ 2️⃣ 没数据再查: backend-vm_manager-*   │
└──────────────┬─────────────────────────┘
               ↓
┌────────────────────────────────────────┐
│ Step 2: 查询最新索引（第一步）         │
├────────────────────────────────────────┤
│ Thought: 按建议先查最新索引            │
│ Action: search_documents[{             │
│   "index": "backend-vm_manager-2025.11.04" │ ⭐
│   "query": {"term": {"L.keyword": "ERROR"}}│
│ }]                                     │
└──────────────┬─────────────────────────┘
               ↓
        有结果？
         ↓    ↓
      是      否
       ↓       ↓
   返回结果  ┌────────────────────────────────────┐
             │ Step 3: 使用通配符（第二步）       │
             ├────────────────────────────────────┤
             │ Thought: 最新索引没数据，用通配符  │
             │ Action: search_documents[{         │
             │   "index": "backend-vm_manager-*"  │ ⭐
             │   "query": {"term": {"L.keyword": "ERROR"}}│
             │ }]                                 │
             └──────────────┬─────────────────────┘
                            ↓
                        返回结果
```

---

## 🎯 实际使用效果

### 示例 1: 最新索引有数据

```
用户: "查询backend的错误日志"

💭 Thought: 先查找backend相关的索引
⚙️ Action: search_indices[backend]
👁️ Observation:
  找到 3 个索引：
  - backend-vm_manager-2025.11.04
  - backend-vm_manager-2025.11.03
  - backend-vm_manager-2025.11.02
  
  💡 建议:
     1️⃣ 优先查询: backend-vm_manager-2025.11.04
     2️⃣ 查不到再用: backend-vm_manager-*

💭 Thought: 按建议先查询最新索引
⚙️ Action: search_documents[{
  "index": "backend-vm_manager-2025.11.04",          ⭐ 第一步：最新索引
  "query": {"term": {"L.keyword": "ERROR"}},
  "size": 20
}]
👁️ Observation: 找到 15 条错误日志                  ✅ 有数据，完成！

📊 最终答案: 在最新的 backend 日志中找到 15 条错误...
```

### 示例 2: 最新索引无数据，需要扩大范围

```
用户: "查询backend项目中用户4103的错误"

💭 Thought: 先查找backend索引
⚙️ Action: search_indices[backend]
👁️ Observation:
  💡 建议:
     1️⃣ 优先查询: backend-vm_manager-2025.11.04
     2️⃣ 查不到再用: backend-vm_manager-*

💭 Thought: 先查最新索引
⚙️ Action: search_documents[{
  "index": "backend-vm_manager-2025.11.04",          ⭐ 第一步：最新
  "query": {
    "bool": {
      "must": [
        {"term": {"M.keyword": "4103"}},
        {"term": {"L.keyword": "ERROR"}}
      ]
    }
  }
}]
👁️ Observation: 未找到匹配的文档                    ❌ 无数据

💭 Thought: 最新索引中没有该用户的错误，使用通配符查询所有历史索引
⚙️ Action: search_documents[{
  "index": "backend-vm_manager-*",                   ⭐ 第二步：通配符
  "query": {
    "bool": {
      "must": [
        {"term": {"M.keyword": "4103"}},
        {"term": {"L.keyword": "ERROR"}}
      ]
    }
  }
}]
👁️ Observation: 在历史索引中找到 3 条错误日志      ✅ 找到了！

📊 最终答案: 在 backend 项目的历史日志中找到用户4103的 3 条错误...
```

### 示例 3: 趋势分析（直接用通配符）

```
用户: "统计backend项目本月的错误趋势"

💭 Thought: 需要分析趋势，应该查询所有数据
⚙️ Action: search_indices[backend]
👁️ Observation:
  💡 建议:
     1️⃣ 优先查询: backend-vm_manager-2025.11.04
     2️⃣ 查不到再用: backend-vm_manager-*

💭 Thought: 趋势分析需要完整数据，直接使用通配符查询所有索引
⚙️ Action: aggregate_data[{
  "index": "backend-vm_manager-*",                   ⭐ 直接用通配符（因为要全量）
  "query": {"term": {"L.keyword": "ERROR"}},
  "aggs": {
    "daily_trend": {
      "date_histogram": {
        "field": "T",
        "calendar_interval": "day"
      }
    }
  }
}]
👁️ Observation: 本月每天的错误统计

📊 最终答案: 本月错误趋势：
  11-01: 45条
  11-02: 38条
  11-03: 52条
  11-04: 15条（今天，数据还在增长）
```

---

## 🔧 最新索引识别算法

### findLatestIndex 函数

```go
func findLatestIndex(indexNames []string) string {
    latest := indexNames[0]
    
    for _, name := range indexNames {
        // 字符串比较（日期格式天然有序）
        if name > latest {
            latest = name
        }
    }
    
    return latest
}
```

**为什么字符串比较有效？**

```
日期格式通常是有序的:
  2025.11.04 > 2025.11.03  ✅
  2024-11-30 > 2024-11-01  ✅
  20241104 > 20241103      ✅

字符串比较:
  "2025.11.04" > "2025.11.03"  ✅ 正确
  "backend-2025.11.04" > "backend-2025.11.03"  ✅ 正确
```

---

## 📊 策略对比

### 优化前（直接用通配符）

```
Observation: 找到多个同前缀索引
Thought: 使用通配符
Action: search_documents[{"index": "prefix-*", ...}]

问题:
  ❌ 查询所有索引，性能开销大
  ❌ 返回海量数据，需要排序
  ❌ 不够精准
```

### 优化后（两步查询）

```
Observation: 找到多个同前缀索引
  建议:
    1️⃣ 先查最新: prefix-2025.11.04
    2️⃣ 查不到再用: prefix-*

Thought: 先查最新的
Action: search_documents[{"index": "prefix-2025.11.04", ...}]  ⭐ 第一步

如果有结果 → 返回 ✅
如果无结果 → 第二步

Action: search_documents[{"index": "prefix-*", ...}]           ⭐ 第二步

优势:
  ✅ 优先最新数据（性能好）
  ✅ 查不到自动扩大范围
  ✅ 兼顾性能和完整性
```

---

## 🎯 使用场景

### 场景 1: 查询最近的错误（通常在最新索引）

```
用户: "查询backend最近的错误"

策略: 
  第一步: backend-vm_manager-2025.11.04    ⭐ 最新索引，找到15条
  结果: ✅ 直接返回，无需第二步
  
性能: 只查1个索引，速度快
```

### 场景 2: 查询历史问题（可能在旧索引）

```
用户: "查询用户X的历史错误"

策略:
  第一步: backend-vm_manager-2025.11.04    ⭐ 最新索引，未找到
  第二步: backend-vm_manager-*             ⭐ 所有索引，找到3条
  
结果: ✅ 找到历史数据
```

### 场景 3: 趋势分析（需要全量数据）

```
用户: "统计本月的错误趋势"

策略:
  直接使用: backend-vm_manager-*           ⭐ 跳过第一步，直接全量
  原因: 趋势分析需要完整数据
  
结果: ✅ 获得完整趋势
```

---

## 📝 提示词关键部分

### 查询策略说明

```
查询策略（重要！）:
  ⭐ 两步查询策略
  
  第一步：优先查询最新索引
    - 通常最新数据概率更高
    - 性能更好（只查1个索引）
  
  第二步：如果查不到，扩大范围
    - 使用通配符查所有索引
    - 避免遗漏历史数据
```

### Few-shot 示例（现在7个）

| # | 示例 | 重点 |
|---|------|------|
| 1 | backend项目查找 | 优先 search_indices |
| 2 | 按日期查找 | 日期关键词 |
| 3 | 错误恢复 | 索引不存在时查找 |
| 4 | L字段查错误 | L.keyword = ERROR |
| 5 | 错误聚合 | 聚合查询 |
| 6 | **优先最新索引** | **第一步策略** ⭐ |
| 7 | **扩大查询范围** | **第二步策略** ⭐ |

---

## 🔍 算法细节

### 最新索引识别

```go
func findLatestIndex(indexNames []string) string {
    // 示例输入:
    // ["backend-vm_manager-2025.11.02",
    //  "backend-vm_manager-2025.11.04",
    //  "backend-vm_manager-2025.11.03"]
    
    latest := indexNames[0]  // "backend-vm_manager-2025.11.02"
    
    for _, name := range indexNames {
        if name > latest {   // 字符串比较
            latest = name
        }
    }
    
    // 返回: "backend-vm_manager-2025.11.04"  ✅ 最新的
}
```

**为什么有效？**
```
ISO 8601 日期格式天然有序:
  "2025.11.04" > "2025.11.03" ✅
  "2024-11-30" > "2024-11-01" ✅
  
字符串逐字符比较:
  "2025.11.04"
  "2025.11.03"
   ↑↑↑↑↑↑↑↑
   都相同，到这里: '4' > '3' ✅
```

---

## 💡 Agent 决策逻辑

### 何时用第一步（最新索引）？

```
✅ 查询"最近"的数据
✅ 查询"今天"的数据
✅ 查询"当前"状态
✅ 实时监控类查询
```

### 何时用第二步（通配符）？

```
✅ 第一步查询结果为空
✅ 第一步结果不足（如只有1条，需要更多）
✅ 查询"历史"数据
✅ 查询"所有"数据
✅ 趋势分析
✅ 统计汇总
```

### 何时直接用第二步？

```
✅ 用户明确说"所有"、"全部"、"历史"
✅ 趋势分析、统计分析
✅ 时间跨度大（如"本月"、"本周"）

示例:
  "统计本月错误" → 直接用 logs-2024-11-*
  "所有错误日志" → 直接用 logs-*
```

---

## ✅ 编译验证

```bash
✅ go build -o jas-agent-server.exe cmd/server/main.go
   编译成功！
```

---

## 🎊 总结

**新的查询策略**:
1. ✅ **第一步优先** - 查询最新索引（性能优化）
2. ✅ **第二步扩展** - 通配符查询所有（完整性保障）
3. ✅ **智能建议** - search_indices 自动提供两步建议
4. ✅ **灵活应用** - Agent 可根据场景选择策略

**优势**:
- 🚀 **性能优化** - 优先最新索引，通常更快
- 🚀 **完整性** - 查不到自动扩大范围
- 🚀 **智能化** - 工具自动建议，Agent 自动应用
- 🚀 **适应性** - 适合各种查询场景

**现在 ES Agent 会智能地先查最新索引，查不到时自动使用通配符扩大范围！** 🎉

