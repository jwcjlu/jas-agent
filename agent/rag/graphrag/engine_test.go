package graphrag

import (
	"context"
	"jas-agent/agent/rag/loader"
	"testing"
)

func TestEngineIngestAndSearch(t *testing.T) {
	engine := NewEngine(Options{}, nil, nil)
	docs := []loader.Document{
		{
			ID:   "doc-1",
			Text: "Neo4j 图数据库用于存储关系型图结构。GraphRAG 使用 Neo4j 以及 OpenAI 来完成检索增强生成。",
			Metadata: map[string]string{
				"source": "whitepaper",
			},
		},
		{
			ID:   "doc-2",
			Text: "LlamaIndex 提供 GraphRAG 能力，GraphRAG 通过 Global Search 找到社区并结合 Local Search 进行推理。",
		},
	}
	stats, err := engine.IngestDocuments(context.Background(), docs)
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if stats.Documents != 2 {
		t.Fatalf("unexpected docs: %v", stats.Documents)
	}
	nodes, edges := engine.Stats()
	if nodes == 0 || edges == 0 {
		t.Fatalf("graph not built, nodes=%d edges=%d", nodes, edges)
	}

	global := engine.GlobalSearch("GraphRAG 如何利用 Neo4j", 3)
	if len(global) == 0 {
		t.Fatalf("expected global search results")
	}

	local := engine.LocalSearch("GraphRAG", 3, 2)
	if len(local) == 0 {
		t.Fatalf("expected local nodes")
	}

	paths := engine.PathSearch("GraphRAG and Neo4j", 4)
	if len(paths) == 0 {
		t.Fatalf("expected at least one path")
	}
}
