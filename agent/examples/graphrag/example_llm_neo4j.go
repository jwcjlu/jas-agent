package main

import (
	"context"
	"jas-agent/agent/llm"
	"jas-agent/agent/rag/graphrag"
)

// ExampleLLMNeo4jGraphRAG 示例：使用 LLM 提取实体和关系并存储到 Neo4j
func main() {
	ctx := context.Background()

	// 1. 创建 LLM Chat 客户端
	llmConfig := &llm.Config{
		ApiKey:  "sk-DKPXsITvLkWVdtN_gTRPdTmv9ILx3VJ1FVy9AI8ur4R9P_3oM4INlil_Or8",
		BaseURL: "http://10.86.3.248:3000/v1",
	}
	chat := llm.NewChat(llmConfig)

	// 2. 创建 Neo4j 存储
	neo4jConfig := graphrag.Neo4jConfig{
		URI:      "bolt://10.86.1.21:7687",
		Username: "neo4j",
		Password: "jw123456",
		Database: "neo4j",
	}
	neo4jStore := graphrag.NewNeo4jStore(ctx, neo4jConfig)
	defer neo4jStore.Close(ctx)

	// 3. 创建 LLM 提取器
	llmExtractor := graphrag.NewLLMExtractor(chat, "gpt-3.5-turbo")

	// 4. 创建 GraphRAG Engine
	engine := graphrag.NewEngine(graphrag.Options{}, neo4jStore, llmExtractor)

	// 5. 准备文档
	/*docs := []loader.Document{
		{
			ID:   "doc-1",
			Text: "GraphRAG 是一个基于知识图谱的检索增强生成系统。它使用 Neo4j 图数据库存储实体和关系。",
			Metadata: map[string]string{
				"source": "example",
			},
		},
		{
			ID:   "doc-2",
			Text: "OpenAI 开发了 GPT 模型，这些模型可以用于自然语言处理任务。",
			Metadata: map[string]string{
				"source": "example",
			},
		},
	}

	// 6. 摄入文档（会自动使用 LLM 提取实体和关系并存储到 Neo4j）
	stats, err := engine.IngestDocuments(ctx, docs)
	if err != nil {
		fmt.Printf("ingest documents: %w", err)
	}*/

	/*fmt.Printf("Ingested %d documents: %d nodes, %d edges\n",
		stats.Documents, stats.Nodes, stats.Edges)

	// 7. 从 Neo4j 查询节点
	nodes, err := neo4jStore.ListNodes(ctx)
	if err != nil {
		fmt.Printf("list nodes: %w", err)
	}
	fmt.Printf("Total nodes in Neo4j: %d\n", len(nodes))

	// 8. 从 Neo4j 查询边
	edges, err := neo4jStore.ListEdges(ctx)
	if err != nil {
		fmt.Printf("list edges: %w", err)
	}
	fmt.Printf("Total edges in Neo4j: %d\n", len(edges))*/

	global := engine.GlobalSearch("串流失败有哪些原因", 3)
	if len(global) == 0 {
		panic("expected global search results")
	}

	local := engine.LocalSearch("串流失败有哪些原因", 3, 2)
	if len(local) == 0 {
		panic("expected local nodes")
	}

	paths := engine.PathSearch("串流失败有哪些原因 and Neo4j", 4)
	if len(paths) == 0 {
		panic("expected at least one path")
	}

}
