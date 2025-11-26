package vectordb

import (
	"context"
	"fmt"
	"jas-agent/agent/rag/embedding"
	"jas-agent/agent/rag/loader"
	"jas-agent/agent/rag/quality"
)

// IngestConfig 摄入配置
type IngestConfig struct {
	Embedder      embedding.Embedder
	VectorStore   VectorStore
	BatchSize     int
	QualityFilter *quality.DocumentFilter // 文档质量过滤器（可选）
}

// DefaultIngestConfig 返回默认摄入配置
func DefaultIngestConfig(embedder embedding.Embedder, store VectorStore) *IngestConfig {
	return &IngestConfig{
		Embedder:    embedder,
		VectorStore: store,
		BatchSize:   100, // 每次批量处理 100 个文档
	}
}

// WithBatchSize 设置批量大小
func (c *IngestConfig) WithBatchSize(size int) *IngestConfig {
	if size > 0 {
		c.BatchSize = size
	}
	return c
}

// IngestResult 摄入结果
type IngestResult struct {
	TotalDocs   int                   `json:"total_docs"`
	Success     int                   `json:"success"`
	Failed      int                   `json:"failed"`
	Vectors     int                   `json:"vectors"`
	Errors      []string              `json:"errors,omitempty"`
	FilterStats *quality.FilterResult `json:"filter_stats,omitempty"` // 质量过滤统计
}

// IngestDocuments 将文档加载并存储到向量数据库
func IngestDocuments(ctx context.Context, docs []loader.Document, config *IngestConfig) (*IngestResult, error) {
	if config == nil {
		return nil, fmt.Errorf("ingest config is required")
	}
	if config.Embedder == nil {
		return nil, fmt.Errorf("embedder is required")
	}
	if config.VectorStore == nil {
		return nil, fmt.Errorf("vector store is required")
	}

	result := &IngestResult{
		TotalDocs: len(docs),
		Errors:    make([]string, 0),
	}

	// 应用质量过滤
	filteredDocs := docs
	if config.QualityFilter != nil {
		var filterResult *quality.FilterResult
		filteredDocs, filterResult = config.QualityFilter.FilterDocuments(docs)
		result.FilterStats = filterResult
		result.Failed += filterResult.Filtered
	}

	// 批量处理文档
	for i := 0; i < len(filteredDocs); i += config.BatchSize {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		end := i + config.BatchSize
		if end > len(filteredDocs) {
			end = len(filteredDocs)
		}

		batch := filteredDocs[i:end]
		if err := ingestBatch(ctx, batch, config, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("batch %d-%d: %v", i, end, err))
			result.Failed += len(batch)
			continue
		}

		result.Success += len(batch)
		result.Vectors += len(batch)
	}

	return result, nil
}

func ingestBatch(ctx context.Context, docs []loader.Document, config *IngestConfig, result *IngestResult) error {
	// 提取文本
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Text
	}

	// 批量生成 embedding
	embeddings, err := config.Embedder.EmbedBatch(ctx, texts)
	if err != nil {
		return fmt.Errorf("generate embeddings: %w", err)
	}

	if len(embeddings) != len(docs) {
		return fmt.Errorf("embedding count mismatch: expected %d, got %d", len(docs), len(embeddings))
	}

	// 构建向量对象并验证向量质量
	vectors := make([]Vector, 0, len(docs))
	for i, doc := range docs {
		// 验证向量质量
		if err := quality.ValidateVector(embeddings[i]); err != nil {
			// 跳过无效向量
			result.Errors = append(result.Errors, fmt.Sprintf("doc %s: invalid vector: %v", doc.ID, err))
			continue
		}

		// 确保 Document 有 Metadata
		if doc.Metadata == nil {
			doc.Metadata = make(map[string]string)
		}

		// 添加质量评分到元数据
		score := quality.ScoreDocument(doc)
		doc.Metadata["quality_score"] = fmt.Sprintf("%.3f", score)

		// 确保元数据被正确复制
		docMeta := make(map[string]string)
		if doc.Metadata != nil {
			for k, v := range doc.Metadata {
				docMeta[k] = v
			}
		}

		vectors = append(vectors, Vector{
			ID:       doc.ID,
			Document: &doc,
			Vector:   embeddings[i],
			Metadata: docMeta, // 使用复制的元数据，确保修改不会影响原文档
		})
	}

	// 存储到向量数据库
	if err := config.VectorStore.Insert(ctx, vectors); err != nil {
		return fmt.Errorf("insert vectors: %w", err)
	}

	return nil
}

// SearchConfig 搜索配置
type SearchConfig struct {
	IngestConfig    *IngestConfig
	MinScore        float64 // 最小相似度阈值
	MaxResults      int     // 最大返回结果数
	DiversityRerank bool    // 是否进行多样性重排序
	DiversityTopK   int     // 多样性重排序的 topK
}

// DefaultSearchConfig 返回默认搜索配置
func DefaultSearchConfig(ingestConfig *IngestConfig) *SearchConfig {
	return &SearchConfig{
		IngestConfig:    ingestConfig,
		MinScore:        0.0, // 默认不过滤
		MaxResults:      10,
		DiversityRerank: false, // 默认不进行多样性重排序
		DiversityTopK:   5,
	}
}

// SearchDocuments 在向量数据库中搜索相似文档
func SearchDocuments(ctx context.Context, query string, topK int, config *IngestConfig, filter map[string]string) ([]SearchResult, error) {
	searchConfig := DefaultSearchConfig(config)
	return SearchDocumentsWithConfig(ctx, query, topK, searchConfig, filter)
}

// SearchDocumentsWithConfig 使用配置搜索文档
func SearchDocumentsWithConfig(ctx context.Context, query string, topK int, config *SearchConfig, filter map[string]string) ([]SearchResult, error) {
	if config == nil || config.IngestConfig == nil {
		return nil, fmt.Errorf("invalid search config")
	}

	ingestConfig := config.IngestConfig
	if ingestConfig.Embedder == nil || ingestConfig.VectorStore == nil {
		return nil, fmt.Errorf("invalid ingest config")
	}

	// 生成查询向量
	queryVector, err := ingestConfig.Embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}

	// 验证查询向量质量
	if err := quality.ValidateVector(queryVector); err != nil {
		return nil, fmt.Errorf("invalid query vector: %w", err)
	}

	// 在向量数据库中搜索
	searchTopK := topK
	if config.MaxResults > 0 && searchTopK > config.MaxResults {
		searchTopK = config.MaxResults * 2 // 搜索更多结果，用于后续过滤
	}

	results, err := ingestConfig.VectorStore.Search(ctx, queryVector, searchTopK, filter)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	// 应用质量过滤
	filterConfig := quality.DefaultSearchFilterConfig()
	filterConfig.MinScore = config.MinScore
	filterConfig.MaxResults = config.MaxResults
	if filterConfig.MaxResults == 0 {
		filterConfig.MaxResults = topK
	}

	// 转换为 SearchResult 接口类型进行过滤
	searchResults := make([]quality.SearchResult, len(results))
	for i := range results {
		searchResults[i] = &searchResultAdapter{&results[i]}
	}

	// 应用过滤
	searchResults = quality.FilterSearchResultsByScore(searchResults, filterConfig)

	// 多样性重排序
	if config.DiversityRerank && len(searchResults) > 1 {
		rerankTopK := config.DiversityTopK
		if rerankTopK == 0 {
			rerankTopK = topK
		}
		searchResults = quality.RerankByDiversity(searchResults, rerankTopK)
	}

	// 转换回 SearchResult 类型
	filteredResults := make([]SearchResult, len(searchResults))
	for i, sr := range searchResults {
		if adapter, ok := sr.(*searchResultAdapter); ok {
			filteredResults[i] = *adapter.result
		}
	}
	results = filteredResults

	return results, nil
}

// searchResultAdapter 适配 SearchResult 到 quality.SearchResult 接口
type searchResultAdapter struct {
	result *SearchResult
}

func (a *searchResultAdapter) GetScore() float64 {
	return a.result.Score
}

func (a *searchResultAdapter) GetVector() []float32 {
	if a.result.Vector != nil {
		return a.result.Vector.Vector
	}
	return nil
}

func (a *searchResultAdapter) GetDocumentID() string {
	if a.result.Vector != nil && a.result.Vector.Document != nil {
		return a.result.Vector.Document.ID
	}
	return ""
}
