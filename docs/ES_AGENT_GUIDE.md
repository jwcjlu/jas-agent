# Elasticsearch Agent 使用指南

## 概述

Elasticsearch Agent 是一个智能的 Elasticsearch 查询助手，可以帮助您轻松地搜索、分析和理解 Elasticsearch 中的数据。它基于 ReAct 框架，能够自动理解用户需求、探索索引结构、构建查询并执行分析。

## 核心特性

### 🔍 智能搜索
- 自动理解自然语言查询意图
- 构建复杂的 ES Query DSL
- 支持全文搜索、精确匹配、范围查询等

### 📊 数据分析
- 执行聚合查询（Aggregations）
- 统计分析（sum, avg, max, min）
- 时间序列分析（date_histogram）
- Terms 聚合和桶聚合

### 🗂️ 索引管理
- 列出所有索引信息
- 查看索引映射（mapping）
- 了解字段类型和结构

### 📄 文档操作
- 搜索文档
- 获取指定文档
- 批量查询

## 快速开始

### 1. 创建 ES 连接

```go
import "jas-agent/tools"

// 创建连接
esConn := tools.NewESConnection(
    "http://localhost:9200",  // ES地址
    "elastic",                 // 用户名（可选）
    "password",                // 密码（可选）
)

// 注册工具
tools.RegisterESTools(esConn)
```

### 2. 创建 ES Agent

```go
import (
    "jas-agent/agent"
    "jas-agent/llm"
    "jas-agent/memory"
    "jas-agent/tools"
)

// 创建LLM
chat := llm.NewChat(&llm.Config{
    ApiKey:  "your-api-key",
    BaseURL: "your-base-url",
})

// 创建内存
mem := memory.NewSimpleMemory()

// 创建工具管理器
toolManager := tools.GetToolManager()

// 创建Agent上下文
context := agent.NewContext(chat, mem, toolManager)

// 创建执行器
executor := agent.NewAgentExecutor(context, 20)

// 创建ES Agent
clusterInfo := "Elasticsearch cluster at http://localhost:9200"
esAgent := agent.NewESAgent(context, executor, clusterInfo)

// 设置Agent
executor.SetAgent(esAgent)
```

### 3. 执行查询

```go
// 执行查询
result, err := executor.Run("搜索最近的10条错误日志")
if err != nil {
    log.Fatal(err)
}

fmt.Println(result)
```

## 可用工具

### 1. list_indices
列出所有索引及其基本信息。

**输入**: 无

**输出**: 索引列表，包含健康状态、文档数量、存储大小

**示例**:
```
Found 3 indices:

- logs-2024-01
  Health: green, Docs: 15234, Size: 2.3mb
- logs-2024-02
  Health: green, Docs: 18567, Size: 3.1mb
- products
  Health: yellow, Docs: 1500, Size: 450kb
```

### 2. get_index_mapping
获取索引的映射结构。

**输入**: 索引名称（字符串）

**输出**: 索引的字段映射（JSON格式）

**示例**:
```json
Mapping for index 'logs':
{
  "logs": {
    "mappings": {
      "properties": {
        "timestamp": { "type": "date" },
        "level": { "type": "keyword" },
        "message": { "type": "text" },
        "user_id": { "type": "keyword" }
      }
    }
  }
}
```

### 3. search_documents
搜索文档。

**输入**: JSON格式
```json
{
  "index": "logs",
  "query": {
    "match": {
      "message": "error"
    }
  },
  "size": 10
}
```

**输出**: 匹配的文档列表

**示例**:
```
Found 156 documents (showing 10):

Document 1 (ID: abc123, Score: 5.23):
  {
    "timestamp": "2024-01-15T10:30:00Z",
    "level": "ERROR",
    "message": "Database connection failed"
  }
```

### 4. get_document
根据ID获取文档。

**输入**: JSON格式
```json
{
  "index": "logs",
  "id": "abc123"
}
```

**输出**: 文档内容

### 5. aggregate_data
执行聚合查询。

**输入**: JSON格式
```json
{
  "index": "logs",
  "aggs": {
    "error_count": {
      "terms": {
        "field": "level.keyword",
        "size": 10
      }
    }
  }
}
```

**输出**: 聚合结果

## 查询示例

### 简单搜索

**用户**: "搜索包含'error'的日志"

**Agent 执行流程**:
1. 使用 `list_indices` 查找日志索引
2. 使用 `get_index_mapping` 了解字段结构
3. 构建搜索查询：
   ```json
   {
     "index": "logs",
     "query": {
       "match": {
         "message": "error"
       }
     },
     "size": 10
   }
   ```
4. 使用 `search_documents` 执行搜索
5. 返回结果并解释

### 范围查询

**用户**: "查找今天的错误日志"

**Agent 执行流程**:
1. 了解索引结构
2. 构建带日期范围的查询：
   ```json
   {
     "index": "logs",
     "query": {
       "bool": {
         "must": [
           { "term": { "level.keyword": "ERROR" } }
         ],
         "filter": [
           {
             "range": {
               "timestamp": {
                 "gte": "now/d",
                 "lte": "now"
               }
             }
           }
         ]
       }
     }
   }
   ```
3. 执行搜索并返回结果

### 聚合分析

**用户**: "统计每小时的日志数量"

**Agent 执行流程**:
1. 确定时间字段名称
2. 构建date_histogram聚合：
   ```json
   {
     "index": "logs",
     "aggs": {
       "logs_per_hour": {
         "date_histogram": {
           "field": "timestamp",
           "calendar_interval": "hour"
         }
       }
     }
   }
   ```
3. 使用 `aggregate_data` 执行聚合
4. 解释聚合结果

### 复杂查询

**用户**: "统计每个用户的错误日志数量，并按数量降序排列"

**Agent 执行流程**:
1. 了解字段结构
2. 构建复合查询和聚合：
   ```json
   {
     "index": "logs",
     "query": {
       "term": {
         "level.keyword": "ERROR"
       }
     },
     "aggs": {
       "errors_by_user": {
         "terms": {
           "field": "user_id.keyword",
           "size": 20,
           "order": { "_count": "desc" }
         }
       }
     }
   }
   ```
3. 执行查询
4. 格式化并解释结果

## Query DSL 参考

### Match Query (全文搜索)
```json
{
  "match": {
    "message": "error database"
  }
}
```

### Term Query (精确匹配)
```json
{
  "term": {
    "level.keyword": "ERROR"
  }
}
```

### Range Query (范围查询)
```json
{
  "range": {
    "timestamp": {
      "gte": "2024-01-01",
      "lte": "2024-01-31"
    }
  }
}
```

### Bool Query (复合查询)
```json
{
  "bool": {
    "must": [
      { "match": { "message": "error" } }
    ],
    "filter": [
      { "term": { "level.keyword": "ERROR" } }
    ],
    "must_not": [
      { "term": { "user_id.keyword": "test" } }
    ],
    "should": [
      { "match": { "message": "critical" } }
    ]
  }
}
```

## Aggregations 参考

### Terms Aggregation (分组统计)
```json
{
  "aggs": {
    "top_users": {
      "terms": {
        "field": "user_id.keyword",
        "size": 10
      }
    }
  }
}
```

### Date Histogram (时间序列)
```json
{
  "aggs": {
    "logs_over_time": {
      "date_histogram": {
        "field": "timestamp",
        "calendar_interval": "day"
      }
    }
  }
}
```

### Metric Aggregations (统计)
```json
{
  "aggs": {
    "stats": {
      "stats": {
        "field": "response_time"
      }
    },
    "avg_response": {
      "avg": {
        "field": "response_time"
      }
    }
  }
}
```

### Nested Aggregations (嵌套聚合)
```json
{
  "aggs": {
    "daily_logs": {
      "date_histogram": {
        "field": "timestamp",
        "calendar_interval": "day"
      },
      "aggs": {
        "by_level": {
          "terms": {
            "field": "level.keyword"
          }
        }
      }
    }
  }
}
```

## 使用场景

### 日志分析
- 查找错误日志
- 统计错误趋势
- 分析错误分布
- 追踪特定错误

### 数据探索
- 了解索引结构
- 查看数据样例
- 验证数据质量
- 发现数据模式

### 业务分析
- 用户行为分析
- 销售数据统计
- 性能指标监控
- 转化率分析

### 安全审计
- 查询异常访问
- 统计安全事件
- 追踪用户操作
- 合规性检查

## 最佳实践

### 1. 逐步探索
```
不要直接编写复杂查询，先：
1. 使用 list_indices 了解可用索引
2. 使用 get_index_mapping 查看字段结构
3. 根据实际结构构建查询
```

### 2. 控制返回量
```json
// 使用 size 参数
{
  "index": "logs",
  "query": { ... },
  "size": 10  // 限制返回数量
}
```

### 3. 使用过滤器
```json
// filter 不计算相关性分数，更快
{
  "bool": {
    "filter": [
      { "term": { "status": "active" } }
    ]
  }
}
```

### 4. 字段类型选择
```
- text 字段用 match 查询（全文搜索）
- keyword 字段用 term 查询（精确匹配）
- 日期字段用 range 查询
```

### 5. 聚合优化
```
- 限制 terms 聚合的 size
- 使用 filter 减少聚合数据量
- 避免深度嵌套聚合
```

## 故障排除

### 连接问题
```bash
# 测试连接
curl http://localhost:9200

# 检查认证
curl -u elastic:password http://localhost:9200
```

### 查询错误
- 检查索引名称是否正确
- 验证字段名称和类型
- 确保 JSON 格式正确
- 查看 ES 错误信息

### 性能问题
- 减少返回文档数量
- 使用过滤器代替查询
- 优化聚合参数
- 考虑使用 scroll API

## 进阶功能

### 1. Scroll API
用于处理大量数据的深度分页。

### 2. Highlight
高亮显示匹配的文本片段。

### 3. Suggest
自动完成和拼写建议。

### 4. Pipeline Aggregations
基于其他聚合结果的二次聚合。

## 集成示例

### 命令行使用
参见 `examples/es/main.go`

### Web API 集成
可以将 ES Agent 集成到 gRPC/HTTP API 中，提供 Web 界面。

### 自动化脚本
用于日志分析、报告生成等自动化任务。

## 参考资源

- [Elasticsearch 官方文档](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html)
- [Query DSL](https://www.elastic.co/guide/en/elasticsearch/reference/current/query-dsl.html)
- [Aggregations](https://www.elastic.co/guide/en/elasticsearch/reference/current/search-aggregations.html)

## 下一步

1. ✅ 安装和配置 Elasticsearch
2. ✅ 运行示例代码
3. ✅ 尝试不同类型的查询
4. ✅ 探索聚合分析功能
5. ✅ 集成到您的应用中

祝使用愉快！🎉

