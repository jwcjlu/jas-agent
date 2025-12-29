package main

import (
	"context"
	"fmt"
	"jas-agent/agent/llm"
	"jas-agent/agent/rag/embedding"
	"jas-agent/agent/rag/graphrag"
	"jas-agent/agent/rag/loader"
	"jas-agent/agent/rag/vectordb"
)

// ExampleHybridSearch 示例：使用混合检索（向量搜索 + 图搜索）
func ExampleHybridSearch() {
	ctx := context.Background()

	// 1. 创建 LLM 客户端
	chat := llm.NewChat(&llm.Config{
		ApiKey:  "YOUR_API_KEY",
		BaseURL: "https://api.openai.com/v1",
	})

	// 2. 创建 Embedder
	embedder := embedding.NewOpenAIEmbedder(embedding.Config{
		ApiKey:  "YOUR_API_KEY",
		BaseURL: "https://api.openai.com/v1",
		Model:   "text-embedding-3-small",
	})

	// 3. 创建 Neo4j 存储
	neo4jConfig := graphrag.Neo4jConfig{
		URI:      "neo4j://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	}
	neo4jStore := graphrag.NewNeo4jStore(ctx, neo4jConfig)
	defer neo4jStore.Close(ctx)

	// 4. 创建 LLM 提取器
	llmExtractor := graphrag.NewLLMExtractor(chat, "gpt-3.5-turbo")

	// 5. 创建 GraphRAG 引擎
	engine := graphrag.NewEngine(graphrag.Options{}, neo4jStore, llmExtractor)

	// 6. 创建向量存储（这里使用内存存储作为示例）
	vectorStore := vectordb.NewInMemoryStore(embedder.Dimensions())

	// 7. 创建混合检索引擎
	hybridEngine := graphrag.NewHybridSearchEngine(engine, vectorStore, embedder)

	// 8. 准备示例文档
	docs := []loader.Document{
		{
			ID:   "doc1",
			Text: "GraphRAG 是一个基于知识图谱的检索增强生成系统。它使用 Neo4j 图数据库存储实体和关系。",
			Metadata: map[string]string{
				"knowledge_base_id": "1",
			},
		},
		{
			ID:   "doc2",
			Text: "向量数据库用于存储文档的向量表示，支持相似度搜索。Milvus 是一个流行的向量数据库。",
			Metadata: map[string]string{
				"knowledge_base_id": "1",
			},
		},
		{
			ID:   "doc3",
			Text: "混合检索结合了向量搜索和图搜索的优势，能够提供更准确的检索结果。",
			Metadata: map[string]string{
				"knowledge_base_id": "1",
			},
		},
	}

	// 9. 将文档存储到向量数据库和图数据库
	// 9.1 存储到向量数据库
	ingestConfig := vectordb.DefaultIngestConfig(embedder, vectorStore)
	_, err := vectordb.IngestDocuments(ctx, docs, ingestConfig)
	if err != nil {
		fmt.Printf("Failed to ingest documents to vector store: %v\n", err)
		return
	}

	// 9.2 存储到图数据库
	_, err = engine.IngestDocuments(ctx, docs)
	if err != nil {
		fmt.Printf("Failed to ingest documents to graph store: %v\n", err)
		return
	}

	// 10. 执行混合搜索
	query := "GraphRAG 如何结合向量数据库和图数据库？"
	opts := graphrag.DefaultHybridSearchOptions()
	opts.VectorTopK = 5
	opts.GraphNodeTopK = 3
	opts.GraphNeighborTopK = 2
	opts.KnowledgeBaseID = 1

	result, err := hybridEngine.Search(ctx, query, opts)
	if err != nil {
		fmt.Printf("Hybrid search failed: %v\n", err)
		return
	}

	// 11. 输出结果
	fmt.Println("=== 混合搜索结果 ===")
	fmt.Printf("\n向量搜索结果数量: %d\n", len(result.VectorResults))
	for i, vr := range result.VectorResults {
		fmt.Printf("\n[向量结果 %d]\n", i+1)
		fmt.Printf("  相似度: %.4f\n", vr.Score)
		fmt.Printf("  文本: %s\n", truncate(vr.Text, 100))
	}

	fmt.Printf("\n图搜索结果数量: %d\n", len(result.GraphResults))
	for i, gr := range result.GraphResults {
		fmt.Printf("\n[图结果 %d]\n", i+1)
		fmt.Printf("  实体: %s\n", gr.Name)
		fmt.Printf("  摘要: %s\n", truncate(gr.Summary, 100))
		fmt.Printf("  分数: %.4f\n", gr.Score)
		fmt.Printf("  邻居数量: %d\n", len(gr.Neighbors))
		for j, neighbor := range gr.Neighbors {
			if j >= 2 {
				break
			}
			fmt.Printf("    - %s %s %s (权重: %.2f)\n", gr.Name, neighbor.Relation, neighbor.Name, neighbor.Weight)
		}
	}

	fmt.Printf("\n路径搜索结果数量: %d\n", len(result.PathResults))
	for i, pr := range result.PathResults {
		fmt.Printf("\n[路径结果 %d]\n", i+1)
		var pathNames []string
		for _, node := range pr.Nodes {
			pathNames = append(pathNames, node.Name)
		}
		fmt.Printf("  路径: %s\n", fmt.Sprintf("%s -> %s", pathNames[0], pathNames[len(pathNames)-1]))
		fmt.Printf("  分数: %.4f\n", pr.Score)
	}

	fmt.Printf("\n社区搜索结果数量: %d\n", len(result.CommunityResults))
	for i, cr := range result.CommunityResults {
		fmt.Printf("\n[社区结果 %d]\n", i+1)
		fmt.Printf("  摘要: %s\n", truncate(cr.Summary, 100))
		fmt.Printf("  分数: %.4f\n", cr.Score)
	}

	fmt.Println("\n=== 合并后的上下文 ===")
	fmt.Println(result.CombinedContext)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// 注意：这个示例需要单独运行，不要与main.go中的main函数冲突
// 可以重命名为 ExampleHybridSearchMain 或单独放在一个文件中
func ExampleHybridSearchMain() {
	ExampleHybridSearch()
}
