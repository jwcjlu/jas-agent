# Elasticsearch Agent 示例

这个示例展示了如何使用 ES Agent 来搜索和分析 Elasticsearch 数据。

## 功能特性

### 支持的操作

1. **索引管理**
   - 列出所有索引
   - 获取索引映射（mapping）

2. **文档操作**
   - 搜索文档（支持复杂查询DSL）
   - 获取指定文档

3. **数据分析**
   - 执行聚合查询
   - 统计分析

## 前置要求

1. 运行中的 Elasticsearch 实例
2. OpenAI API Key

## 环境配置

```bash
# 必需
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="your-base-url"

# 可选（默认 http://localhost:9200）
export ES_HOST="http://localhost:9200"
export ES_USER="elastic"           # 如果需要认证
export ES_PASS="your-password"     # 如果需要认证
```

## 运行示例

```bash
cd examples/es
go run main.go
```

## 示例查询

### 1. 列出所有索引
```
Query: 列出所有索引及其文档数量
Agent: 使用 list_indices 工具获取所有索引信息
```

### 2. 查看索引结构
```
Query: 查看 logs 索引的结构
Agent: 使用 get_index_mapping 获取字段映射
```

### 3. 搜索文档
```
Query: 搜索最近的10条错误日志
Agent: 
  1. 获取索引mapping
  2. 构建查询DSL
  3. 使用 search_documents 执行搜索
```

### 4. 聚合分析
```
Query: 统计每小时的日志数量
Agent:
  1. 了解索引结构
  2. 构建 date_histogram 聚合
  3. 使用 aggregate_data 执行聚合
```

## 查询 DSL 示例

### 简单搜索
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

### 范围查询
```json
{
  "index": "logs",
  "query": {
    "range": {
      "timestamp": {
        "gte": "2024-01-01",
        "lte": "2024-01-31"
      }
    }
  }
}
```

### 复合查询
```json
{
  "index": "logs",
  "query": {
    "bool": {
      "must": [
        { "match": { "level": "ERROR" } }
      ],
      "filter": [
        { "range": { "timestamp": { "gte": "now-1h" } } }
      ]
    }
  }
}
```

### 聚合查询
```json
{
  "index": "logs",
  "aggs": {
    "hourly_logs": {
      "date_histogram": {
        "field": "timestamp",
        "calendar_interval": "hour"
      }
    }
  }
}
```

### Terms 聚合
```json
{
  "index": "logs",
  "aggs": {
    "top_errors": {
      "terms": {
        "field": "error_type.keyword",
        "size": 10
      }
    }
  }
}
```

## 可用工具

| 工具名称 | 功能 | 输入 |
|---------|------|------|
| `list_indices` | 列出所有索引 | 无 |
| `get_index_mapping` | 获取索引映射 | 索引名称 |
| `search_documents` | 搜索文档 | JSON (index, query, size) |
| `get_document` | 获取文档 | JSON (index, id) |
| `aggregate_data` | 聚合分析 | JSON (index, aggs) |

## 使用场景

### 1. 日志分析
```
- 查找错误日志
- 统计错误趋势
- 分析错误分布
```

### 2. 数据探索
```
- 了解索引结构
- 查看数据样例
- 验证数据质量
```

### 3. 业务分析
```
- 用户行为分析
- 销售数据统计
- 性能指标监控
```

### 4. 安全审计
```
- 查询异常访问
- 统计安全事件
- 追踪用户操作
```

## 最佳实践

1. **逐步探索**: 先了解索引结构，再构建查询
2. **限制结果**: 使用 size 参数控制返回数量
3. **精确查询**: 根据字段类型选择合适的查询类型
4. **合理聚合**: 使用 aggregations 进行数据分析
5. **性能优化**: 避免深度分页，使用 scroll API 处理大量数据

## 故障排除

### 连接失败
```bash
# 检查 ES 是否运行
curl http://localhost:9200

# 检查认证信息
curl -u elastic:password http://localhost:9200
```

### 索引不存在
```
确保索引名称正确，使用 list_indices 查看所有索引
```

### 查询语法错误
```
参考 ES 官方文档，使用正确的 Query DSL 语法
```

## 参考资源

- [Elasticsearch 官方文档](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html)
- [Query DSL 参考](https://www.elastic.co/guide/en/elasticsearch/reference/current/query-dsl.html)
- [Aggregations 参考](https://www.elastic.co/guide/en/elasticsearch/reference/current/search-aggregations.html)

## 扩展

你可以添加更多工具：

1. **索引管理**: 创建、删除、更新索引
2. **文档操作**: 创建、更新、删除文档
3. **批量操作**: bulk API
4. **复杂聚合**: pipeline aggregations
5. **高级搜索**: suggest、highlight 等

## 注意事项

⚠️ 当前实现仅支持只读操作（搜索、聚合），不支持写入操作。

如需支持写入，请在生产环境中实施适当的权限控制和审计机制。

