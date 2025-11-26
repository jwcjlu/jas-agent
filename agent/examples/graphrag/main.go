package main

import (
	"context"
	"fmt"

	"jas-agent/agent/rag/graphrag"
	"jas-agent/agent/rag/loader"
)

func main() {
	engine := graphrag.DefaultEngine()
	engine.Reset()
	ctx := context.Background()
	engine.IngestDocuments(ctx, []loader.Document{
		{
			ID:   "doc-1",
			Text: "GraphRAG 将文档拆分为实体和关系，并通过 Global Search 聚合社区信息。",
		},
		{
			ID:   "doc-2",
			Text: "Local Search 提取关键节点，Path Search 则帮助我们解释实体如何联动。",
		},
	})

	result := engine.LocalSearch("GraphRAG 如何工作", 2, 2)
	fmt.Printf("Local Search => %+v\n", result)
}
