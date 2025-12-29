# RAG 质量控制示例

本示例展示了如何使用完整的 RAG 质量控制流程，包括：

1. **数据清洗与预处理**
   - 去重（完全重复和近乎重复）
   - 清理导航栏、广告等噪音
   - 处理不完整内容

2. **智能分块**
   - 固定大小分块（默认）
   - 语义分块（基于嵌入模型）
   - 递归分块（先按段落，再按句子）
   - 多粒度分块（粗粒度和细粒度）

3. **丰富元数据**
   - 文件信息（路径、大小、修改时间）
   - 内容信息（类型、词数、段落数）
   - 目录信息（可作为topic）
   - 语言检测

## 使用示例

### 基础使用（固定大小分块）

```go
docs, err := loader.LoadDocuments(ctx, []string{"testdata"},
    loader.WithChunkSize(500),
    loader.WithChunkOverlap(50),
)
```

### 语义分块

```go
embedder := embedding.NewOpenAIEmbedder(embedding.DefaultConfig(apiKey))
chunkConfig := loader.DefaultChunkingConfig().
    WithSemanticChunking(embedder)

docs, err := loader.LoadDocuments(ctx, []string{"testdata"},
    loader.WithChunkingConfig(chunkConfig),
)
```

### 多粒度分块

```go
chunkConfig := loader.DefaultChunkingConfig().
    WithMultiGranularity()

docs, err := loader.LoadDocuments(ctx, []string{"testdata"},
    loader.WithChunkingConfig(chunkConfig),
)
```

### 完整质量控制流程

```go
// 1. 数据清洗
cleaner := quality.NewDocumentCleaner(quality.DefaultCleaningConfig())
cleanedDocs := cleaner.CleanDocuments(docs)

// 2. 检测重复
duplicates := cleaner.DetectNearDuplicates(cleanedDocs)

// 3. 检查完整性
for _, doc := range cleanedDocs {
    isComplete, issues := cleaner.CheckCompleteness(doc)
    if !isComplete {
        // 处理不完整的文档
    }
}

// 4. 质量过滤
filter := quality.NewDocumentFilter(quality.DefaultFilterConfig())
validDocs, _ := filter.FilterDocuments(cleanedDocs)

// 5. 文档摄入（自动添加质量评分到元数据）
ingestConfig := vectordb.DefaultIngestConfig(embedder, store)
ingestConfig.QualityFilter = filter
result, _ := vectordb.IngestDocuments(ctx, validDocs, ingestConfig)

// 6. 质量控制的搜索
searchConfig := vectordb.DefaultSearchConfig(ingestConfig)
searchConfig.MinScore = 0.7         // 最小相似度阈值
searchConfig.DiversityRerank = true // 多样性重排序
results, _ := vectordb.SearchDocumentsWithConfig(ctx, query, 5, searchConfig, nil)
```

## 元数据字段

每个文档块包含以下元数据：

- `source_path`: 文件路径
- `source_name`: 文件名
- `source_ext`: 文件扩展名
- `source_dir`: 目录路径
- `last_updated`: 最后更新时间
- `file_size`: 文件大小
- `topic`: 主题（通常来自目录名）
- `author`: 作者（从文件名提取）
- `chunk_index`: 块索引
- `chunk_type`: 块类型（title/paragraph/code/list/table）
- `chunk_size`: 块大小（字符数）
- `word_count`: 词数
- `paragraph_count`: 段落数
- `sentence_count`: 句子数
- `content_type`: 内容类型
- `language`: 语言（zh/en）
- `quality_score`: 质量评分（0-1）

## 运行示例

```bash
export OPENAI_API_KEY=your_api_key
go run agent/examples/rag_quality/main.go
```

