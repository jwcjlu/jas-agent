package graphrag

import (
	"context"
	"fmt"
	"jas-agent/agent/rag/embedding"
	"jas-agent/agent/rag/loader"
	"jas-agent/agent/rag/vectordb"
	"strings"
)

// HybridSearchResult 混合搜索结果
type HybridSearchResult struct {
	// 向量搜索结果
	VectorResults []vectordb.SearchResult `json:"vector_results"`
	// 图搜索结果（从向量结果相关的实体出发）
	GraphResults []loader.LocalNodeResult `json:"graph_results"`
	// 路径搜索结果（实体之间的路径）
	PathResults []loader.PathResult `json:"path_results"`
	// 社区搜索结果
	CommunityResults []loader.GlobalCommunityResult `json:"community_results"`
	// 合并后的上下文文本
	CombinedContext string `json:"combined_context"`
}

// HybridSearchOptions 混合搜索选项
type HybridSearchOptions struct {
	// 向量搜索相关
	VectorTopK int // 向量搜索返回的topK结果，默认10
	// 图搜索相关
	GraphNodeTopK      int // 从向量结果相关的实体中选取的topK节点，默认5
	GraphNeighborTopK  int // 每个节点的邻居数量，默认3
	GraphPathMaxDepth  int // 路径搜索的最大深度，默认3
	GraphCommunityTopK int // 社区搜索的topK，默认3
	// 是否启用图搜索
	EnableGraphSearch bool // 是否启用图搜索，默认true
	// 知识库ID过滤
	KnowledgeBaseID int // 可选：限制搜索范围到特定知识库
}

// DefaultHybridSearchOptions 返回默认的混合搜索选项
func DefaultHybridSearchOptions() HybridSearchOptions {
	return HybridSearchOptions{
		VectorTopK:         10,
		GraphNodeTopK:      5,
		GraphNeighborTopK:  3,
		GraphPathMaxDepth:  3,
		GraphCommunityTopK: 3,
		EnableGraphSearch:  true,
	}
}

// HybridSearchEngine 混合检索引擎，结合向量搜索和图搜索
type HybridSearchEngine struct {
	engine      *Engine
	vectorStore vectordb.VectorStore
	embedder    embedding.Embedder
}

// NewHybridSearchEngine 创建混合检索引擎
func NewHybridSearchEngine(engine *Engine, vectorStore vectordb.VectorStore, embedder embedding.Embedder) *HybridSearchEngine {
	return &HybridSearchEngine{
		engine:      engine,
		vectorStore: vectorStore,
		embedder:    embedder,
	}
}

// Search 执行混合搜索
func (h *HybridSearchEngine) Search(ctx context.Context, query string, opts HybridSearchOptions) (*HybridSearchResult, error) {
	result := &HybridSearchResult{}

	// 1. 向量搜索：找到相关的文档块
	queryVector, err := h.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	// 构建过滤器
	filter := make(map[string]string)
	if opts.KnowledgeBaseID > 0 {
		filter["knowledge_base_id"] = fmt.Sprintf("%d", opts.KnowledgeBaseID)
	}

	vectorTopK := opts.VectorTopK
	if vectorTopK <= 0 {
		vectorTopK = 10
	}

	vectorResults, err := h.vectorStore.Search(ctx, queryVector, vectorTopK, filter)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	result.VectorResults = vectorResults

	// 2. 如果启用图搜索，从向量结果中提取实体并进行图搜索
	if opts.EnableGraphSearch && h.engine != nil && h.engine.neo4jStore != nil {
		// 2.1 从向量结果中提取相关的实体节点ID
		entityIDs := h.extractEntityIDsFromVectorResults(vectorResults)

		// 2.2 从这些实体节点出发进行图搜索
		if len(entityIDs) > 0 {
			graphNodeTopK := opts.GraphNodeTopK
			if graphNodeTopK <= 0 {
				graphNodeTopK = 5
			}

			// 执行本地搜索（从相关实体出发）
			graphResults := h.searchFromEntities(ctx, entityIDs, query, graphNodeTopK, opts.GraphNeighborTopK)
			result.GraphResults = graphResults

			// 2.3 路径搜索：在相关实体之间找路径
			if len(graphResults) > 1 {
				pathMaxDepth := opts.GraphPathMaxDepth
				if pathMaxDepth <= 0 {
					pathMaxDepth = 3
				}
				pathResults := h.searchPathsBetweenEntities(ctx, graphResults, pathMaxDepth)
				result.PathResults = pathResults
			}

			// 2.4 社区搜索
			communityTopK := opts.GraphCommunityTopK
			if communityTopK <= 0 {
				communityTopK = 3
			}
			communityResults := h.engine.GlobalSearch(query, communityTopK)
			result.CommunityResults = communityResults
		}
	}

	// 3. 合并上下文
	result.CombinedContext = h.buildCombinedContext(result)

	return result, nil
}

// extractEntityIDsFromVectorResults 从向量结果中提取实体节点ID
func (h *HybridSearchEngine) extractEntityIDsFromVectorResults(vectorResults []vectordb.SearchResult) []string {
	entityIDSet := make(map[string]bool)
	entityIDs := make([]string, 0)

	for _, result := range vectorResults {
		// 方法1: 从文档元数据中提取实体ID
		if result.Metadata != nil {
			if entityID, ok := result.Metadata["entity_id"]; ok && entityID != "" {
				if !entityIDSet[entityID] {
					entityIDSet[entityID] = true
					entityIDs = append(entityIDs, entityID)
				}
			}
		}

		// 方法2: 从文档元数据中提取实体名称
		doc := result.GetDocument()
		if doc != nil && doc.Metadata != nil {
			if entityName, ok := doc.Metadata["entity_name"]; ok && entityName != "" {
				entityID := normalizeEntity(entityName)
				if entityID != "" && !entityIDSet[entityID] {
					entityIDSet[entityID] = true
					entityIDs = append(entityIDs, entityID)
				}
			}
		}

		// 方法3: 从文档内容中提取实体（如果图数据库中有匹配的实体）
		// 这里我们通过搜索图数据库来找到与文档内容相关的实体
		if doc != nil && doc.Text != "" {
			// 从文档文本中提取可能的实体名称
			// 使用简单的实体提取逻辑（实际可以使用更复杂的NLP方法）
			potentialEntities := extractPotentialEntities(doc.Text)
			for _, entityName := range potentialEntities {
				entityID := normalizeEntity(entityName)
				if entityID == "" {
					continue
				}
				// 检查图数据库中是否存在该实体
				ctx := context.Background()
				if h.engine != nil && h.engine.neo4jStore != nil {
					node, err := h.engine.neo4jStore.GetNode(ctx, entityID)
					if err == nil && node != nil {
						if !entityIDSet[entityID] {
							entityIDSet[entityID] = true
							entityIDs = append(entityIDs, entityID)
						}
					}
				}
			}
		}
	}

	return entityIDs
}

// extractPotentialEntities 从文本中提取可能的实体名称（简单实现）
func extractPotentialEntities(text string) []string {
	// 这里使用简单的启发式方法提取实体
	// 实际应用中可以使用更复杂的NLP方法
	entities := make([]string, 0)

	// 提取可能的中文实体（2-10个字符的连续中文字符串）
	runes := []rune(text)
	var current strings.Builder
	for i, r := range runes {
		if isHan(r) {
			current.WriteRune(r)
		} else {
			if current.Len() >= 2 && current.Len() <= 10 {
				entity := current.String()
				entities = append(entities, entity)
			}
			current.Reset()
		}
		// 处理最后一个实体
		if i == len(runes)-1 && current.Len() >= 2 && current.Len() <= 10 {
			entity := current.String()
			entities = append(entities, entity)
		}
	}

	// 去重
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, e := range entities {
		if !seen[e] {
			seen[e] = true
			result = append(result, e)
		}
	}

	return result
}

// isHan 检查是否是中文字符
func isHan(r rune) bool {
	return r >= 0x4E00 && r <= 0x9FFF
}

// searchFromEntities 从实体节点出发进行图搜索
func (h *HybridSearchEngine) searchFromEntities(ctx context.Context, entityIDs []string, query string, topKNodes, topKNeighbors int) []loader.LocalNodeResult {
	allResults := make([]loader.LocalNodeResult, 0)
	seenNodeIDs := make(map[string]bool)

	queryTokens := tokenize(query)

	for _, entityID := range entityIDs {
		// 获取节点
		node, err := h.engine.neo4jStore.GetNode(ctx, entityID)
		if err != nil {
			continue
		}

		if seenNodeIDs[node.ID] {
			continue
		}
		seenNodeIDs[node.ID] = true

		// 计算相似度分数
		nodeText := node.Summary + " " + strings.Join(node.Snippets, " ")
		score := semanticSimilarity(queryTokens, tokenize(nodeText))

		// 获取邻居
		neighbors := make([]loader.Neighbor, 0)
		edges, err := h.engine.neo4jStore.GetNeighbors(ctx, node.ID)
		if err == nil {
			sortEdgesByWeight(edges)
			for i, edge := range edges {
				if i >= topKNeighbors {
					break
				}

				targetID := edge.Target
				if targetID == node.ID {
					targetID = edge.Source
				}

				targetNode, err := h.engine.neo4jStore.GetNode(ctx, targetID)
				if err != nil {
					continue
				}

				neighborScore := semanticSimilarity(
					queryTokens,
					tokenize(targetNode.Summary+" "+edge.Evidence),
				)

				neighbors = append(neighbors, loader.Neighbor{
					NodeID:   targetNode.ID,
					Name:     targetNode.Name,
					Relation: edge.Relation,
					Evidence: edge.Evidence,
					Weight:   edge.Weight,
					Summary:  targetNode.Summary,
					Score:    neighborScore,
				})
			}
		}

		allResults = append(allResults, loader.LocalNodeResult{
			NodeID:     node.ID,
			Name:       node.Name,
			Summary:    node.Summary,
			Snippets:   node.Snippets,
			Score:      score,
			Neighbors:  neighbors,
			Metadata:   node.Metadata,
			Occurrence: node.Occurrence,
		})
	}

	// 按分数排序
	sortLocalResults(allResults)

	// 返回topK
	if len(allResults) > topKNodes {
		allResults = allResults[:topKNodes]
	}

	return allResults
}

// searchPathsBetweenEntities 在实体之间搜索路径
func (h *HybridSearchEngine) searchPathsBetweenEntities(ctx context.Context, graphResults []loader.LocalNodeResult, maxDepth int) []loader.PathResult {
	if len(graphResults) < 2 {
		return nil
	}

	var pathResults []loader.PathResult
	seenPaths := make(map[string]bool)

	// 在top结果之间找路径
	for i := 0; i < len(graphResults) && i < 5; i++ {
		for j := i + 1; j < len(graphResults) && j < 5; j++ {
			startID := graphResults[i].NodeID
			targetID := graphResults[j].NodeID

			pathKey := fmt.Sprintf("%s-%s", startID, targetID)
			if seenPaths[pathKey] {
				continue
			}
			seenPaths[pathKey] = true

			path := h.engine.findPath(ctx, startID, targetID, maxDepth)
			if path != nil {
				pathResults = append(pathResults, *path)
			}
		}
	}

	sortPathResults(pathResults)
	if len(pathResults) > 3 {
		pathResults = pathResults[:3]
	}

	return pathResults
}

// buildCombinedContext 构建合并后的上下文文本
func (h *HybridSearchEngine) buildCombinedContext(result *HybridSearchResult) string {
	var parts []string

	// 1. 添加向量搜索结果
	if len(result.VectorResults) > 0 {
		parts = append(parts, "=== 相关文档片段 ===")
		for i, vr := range result.VectorResults {
			if i >= 5 { // 只取前5个
				break
			}
			text := vr.Text
			if len(text) > 300 {
				text = text[:300] + "..."
			}
			parts = append(parts, fmt.Sprintf("[文档片段 %d] %s", i+1, text))
		}
	}

	// 2. 添加图搜索结果
	if len(result.GraphResults) > 0 {
		parts = append(parts, "\n=== 相关实体和关系 ===")
		for i, gr := range result.GraphResults {
			if i >= 3 { // 只取前3个
				break
			}
			parts = append(parts, fmt.Sprintf("[实体] %s: %s", gr.Name, gr.Summary))
			if len(gr.Neighbors) > 0 {
				for j, neighbor := range gr.Neighbors {
					if j >= 2 { // 每个实体只显示2个邻居
						break
					}
					parts = append(parts, fmt.Sprintf("  - %s %s %s", gr.Name, neighbor.Relation, neighbor.Name))
				}
			}
		}
	}

	// 3. 添加路径结果
	if len(result.PathResults) > 0 {
		parts = append(parts, "\n=== 实体关系路径 ===")
		for i, pr := range result.PathResults {
			if i >= 2 { // 只取前2个路径
				break
			}
			var pathNames []string
			for _, node := range pr.Nodes {
				pathNames = append(pathNames, node.Name)
			}
			parts = append(parts, fmt.Sprintf("[路径] %s", strings.Join(pathNames, " -> ")))
		}
	}

	// 4. 添加社区结果
	if len(result.CommunityResults) > 0 {
		parts = append(parts, "\n=== 相关主题社区 ===")
		for i, cr := range result.CommunityResults {
			if i >= 2 { // 只取前2个社区
				break
			}
			parts = append(parts, fmt.Sprintf("[主题] %s", cr.Summary))
		}
	}

	return strings.Join(parts, "\n\n")
}
