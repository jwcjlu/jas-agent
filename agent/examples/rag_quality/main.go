package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"jas-agent/agent/rag/embedding"
	"jas-agent/agent/rag/loader"
	"jas-agent/agent/rag/quality"
	"jas-agent/agent/rag/vectordb"
)

func main() {
	ctx := context.Background()

	// 1. 创建文档加载器
	docs, err := loader.LoadDocuments(ctx, []string{"testdata"},
		loader.WithChunkSize(500),
		loader.WithChunkOverlap(50),
	)
	if err != nil {
		log.Fatalf("Failed to load documents: %v", err)
	}

	fmt.Printf("Loaded %d documents\n", len(docs))

	// 2. 创建质量过滤器
	filterConfig := quality.DefaultFilterConfig()
	filterConfig.MinTextLength = 50
	filterConfig.MinWordCount = 5
	filterConfig.MinAlphaRatio = 0.5
	filterConfig.RemoveDuplicates = true

	filter := quality.NewDocumentFilter(filterConfig)
	validDocs, filterResult := filter.FilterDocuments(docs)

	fmt.Printf("\n=== 质量过滤统计 ===\n")
	fmt.Printf("总文档数: %d\n", filterResult.Total)
	fmt.Printf("过滤文档数: %d\n", filterResult.Filtered)
	fmt.Printf("有效文档数: %d\n", filterResult.Valid)
	fmt.Printf("重复文档数: %d\n", filterResult.Duplicates)

	if len(filterResult.FilteredDocs) > 0 {
		fmt.Printf("\n被过滤的文档:\n")
		for _, reason := range filterResult.FilteredDocs[:min(10, len(filterResult.FilteredDocs))] {
			fmt.Printf("  - %s\n", reason)
		}
		if len(filterResult.FilteredDocs) > 10 {
			fmt.Printf("  ... 还有 %d 个文档被过滤\n", len(filterResult.FilteredDocs)-10)
		}
	}

	// 3. 显示文档质量评分
	fmt.Printf("\n=== 文档质量评分（前10个） ===\n")
	for i, doc := range validDocs[:min(10, len(validDocs))] {
		score := quality.ScoreDocument(doc)
		fmt.Printf("%d. %s (评分: %.3f)\n", i+1, doc.ID, score)
	}

	// 4. 创建嵌入生成器（需要设置 API Key）
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	embedder := embedding.NewOpenAIEmbedder(embedding.DefaultConfig(apiKey))

	// 5. 创建向量数据库（需要知道 embedding 的维度）
	store := vectordb.NewInMemoryStore(embedder.Dimensions())

	// 6. 配置摄入管道（启用质量控制）
	ingestConfig := vectordb.DefaultIngestConfig(embedder, store)
	ingestConfig.QualityFilter = filter // 使用质量过滤器
	ingestConfig.BatchSize = 50

	// 7. 将文档摄入向量数据库
	fmt.Printf("\n=== 开始文档摄入 ===\n")
	result, err := vectordb.IngestDocuments(ctx, validDocs, ingestConfig)
	if err != nil {
		log.Fatalf("Failed to ingest documents: %v", err)
	}

	fmt.Printf("\n=== 摄入结果 ===\n")
	fmt.Printf("总文档数: %d\n", result.TotalDocs)
	fmt.Printf("成功: %d\n", result.Success)
	fmt.Printf("失败: %d\n", result.Failed)
	fmt.Printf("向量数: %d\n", result.Vectors)

	if result.FilterStats != nil {
		fmt.Printf("\n质量过滤统计:\n")
		fmt.Printf("  过滤: %d\n", result.FilterStats.Filtered)
		fmt.Printf("  有效: %d\n", result.FilterStats.Valid)
	}

	// 8. 执行搜索（启用质量控制）
	searchConfig := vectordb.DefaultSearchConfig(ingestConfig)
	searchConfig.MinScore = 0.7         // 最小相似度阈值
	searchConfig.MaxResults = 5         // 最多返回 5 个结果
	searchConfig.DiversityRerank = true // 启用多样性重排序
	searchConfig.DiversityTopK = 3

	query := "什么是人工智能？"
	fmt.Printf("\n=== 搜索: %s ===\n", query)
	results, err := vectordb.SearchDocumentsWithConfig(ctx, query, 5, searchConfig, nil)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}

	fmt.Printf("找到 %d 个相关结果:\n", len(results))
	for i, r := range results {
		fmt.Printf("\n%d. 相似度: %.3f (距离: %.3f)\n", i+1, r.Score, r.Distance)
		fmt.Printf("   文档ID: %s\n", r.ID)

		// 显示元数据（使用便捷字段）
		if r.Metadata != nil {
			// 显示常用元数据
			if source, ok := r.Metadata["source_path"]; ok {
				fmt.Printf("   来源: %s\n", source)
			}
			if topic, ok := r.Metadata["topic"]; ok {
				fmt.Printf("   主题: %s\n", topic)
			}
			if chunkType, ok := r.Metadata["chunk_type"]; ok {
				fmt.Printf("   块类型: %s\n", chunkType)
			}
			if qualityScore, ok := r.Metadata["quality_score"]; ok {
				fmt.Printf("   质量评分: %s\n", qualityScore)
			}
			if lastUpdated, ok := r.Metadata["last_updated"]; ok {
				fmt.Printf("   更新时间: %s\n", lastUpdated)
			}
			if author, ok := r.Metadata["author"]; ok {
				fmt.Printf("   作者: %s\n", author)
			}
			if language, ok := r.Metadata["language"]; ok {
				fmt.Printf("   语言: %s\n", language)
			}

			// 显示所有元数据（调试用）
			fmt.Printf("   所有元数据: %+v\n", r.Metadata)
		}

		// 显示内容预览
		preview := r.Text
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		fmt.Printf("   内容预览: %s\n", preview)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
