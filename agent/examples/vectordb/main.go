package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"jas-agent/agent/rag/embedding"
	"jas-agent/agent/rag/loader"
	"jas-agent/agent/rag/vectordb"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <api_key> <document_path> [document_path...]")
		os.Exit(1)
	}

	apiKey := os.Args[1]
	docPaths := os.Args[2:]

	ctx := context.Background()

	// 1. 加载文档
	fmt.Println("Loading documents...")
	docs, err := loader.LoadDocuments(
		ctx,
		docPaths,
		loader.WithChunkSize(512),
		loader.WithChunkOverlap(80),
	)
	if err != nil {
		log.Fatalf("Failed to load documents: %v", err)
	}

	fmt.Printf("Loaded %d document chunks\n", len(docs))

	// 2. 创建 embedding 生成器
	fmt.Println("Creating embedder...")
	embedder := embedding.NewOpenAIEmbedder(embedding.DefaultConfig(apiKey))
	fmt.Printf("Embedding dimensions: %d\n", embedder.Dimensions())

	// 3. 创建向量数据库
	fmt.Println("Creating vector store...")
	store := vectordb.NewInMemoryStore(embedder.Dimensions())

	// 4. 创建摄入配置
	config := vectordb.DefaultIngestConfig(embedder, store)

	// 5. 将文档存储到向量数据库
	fmt.Println("Ingesting documents into vector database...")
	result, err := vectordb.IngestDocuments(ctx, docs, config)
	if err != nil {
		log.Fatalf("Failed to ingest documents: %v", err)
	}

	fmt.Printf("\nIngestion completed:\n")
	fmt.Printf("  Total documents: %d\n", result.TotalDocs)
	fmt.Printf("  Success: %d\n", result.Success)
	fmt.Printf("  Failed: %d\n", result.Failed)
	fmt.Printf("  Vectors stored: %d\n", result.Vectors)

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	// 6. 获取统计信息
	stats, err := store.Stats(ctx)
	if err != nil {
		log.Fatalf("Failed to get stats: %v", err)
	}
	fmt.Printf("\nVector store stats:\n")
	fmt.Printf("  Count: %d\n", stats.Count)
	fmt.Printf("  Dimensions: %d\n", stats.Dimensions)

	// 7. 搜索示例
	fmt.Println("\nSearching for 'document'...")
	searchResults, err := vectordb.SearchDocuments(ctx, "document", 5, config, nil)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}

	fmt.Printf("Found %d results:\n\n", len(searchResults))
	for i, res := range searchResults {
		text := res.Vector.Document.Text
		if len(text) > 100 {
			text = text[:100] + "..."
		}
		fmt.Printf("%d. ID: %s, Score: %.4f\n", i+1, res.Vector.ID, res.Score)
		fmt.Printf("   Text: %s\n", text)
		if res.Vector.Document.Metadata != nil {
			if source, ok := res.Vector.Document.Metadata["source_path"]; ok {
				fmt.Printf("   Source: %s\n", source)
			}
		}
		fmt.Println()
	}
}
