package graphrag

import (
	"context"
	"errors"
	"fmt"
	"jas-agent/agent/rag/loader"
	"sort"
	"strings"
	"sync"
	"time"
)

// Options 控制 GraphRAG 引擎行为
type Options struct {
	GlobalTopK         int
	LocalNodeTopK      int
	LocalNeighborTopK  int
	PathMaxDepth       int
	MaxSnippetsPerNode int
	MaxSummaryLength   int
}

func (o *Options) normalize() {
	if o.GlobalTopK <= 0 {
		o.GlobalTopK = 4
	}
	if o.LocalNodeTopK <= 0 {
		o.LocalNodeTopK = 5
	}
	if o.LocalNeighborTopK <= 0 {
		o.LocalNeighborTopK = 4
	}
	if o.PathMaxDepth <= 0 {
		o.PathMaxDepth = 4
	}
	if o.MaxSnippetsPerNode <= 0 {
		o.MaxSnippetsPerNode = 6
	}
	if o.MaxSummaryLength <= 0 {
		o.MaxSummaryLength = 800
	}
}

// Engine 是 GraphRAG 的核心实现
type Engine struct {
	store         Store
	options       Options
	neo4jStore    *Neo4jStore   // Neo4j 存储（可选）
	llmExtractor  *LLMExtractor // LLM 提取器（可选）
	communities   map[string]*loader.Community
	communitiesMu sync.RWMutex
}

var (
	defaultEngine *Engine
	engineOnce    sync.Once
)

// DefaultEngine 返回全局默认引擎
func DefaultEngine() *Engine {
	engineOnce.Do(func() {
		defaultEngine = NewEngine(Options{}, nil, nil)
	})
	return defaultEngine
}

// NewEngine 创建新的 GraphRAG 引擎
func NewEngine(opts Options, neo4jStore *Neo4jStore, llmExtractor *LLMExtractor) *Engine {
	opts.normalize()
	engine := &Engine{
		options:      opts,
		llmExtractor: llmExtractor,
		communities:  make(map[string]*loader.Community),
	}
	if neo4jStore != nil {
		engine.store = neo4jStore
		engine.neo4jStore = neo4jStore
	} else {
		engine.store = newGraphStore()
	}
	return engine
}

// UseStore 切换底层 Store 实现
func (e *Engine) UseStore(store Store) {
	if store == nil {
		store = newGraphStore()
	}
	e.store = store
	if neo, ok := store.(*Neo4jStore); ok {
		e.neo4jStore = neo
	} else {
		e.neo4jStore = nil
	}
	e.clearCommunities()
}

// Reset 清空所有数据
func (e *Engine) Reset() {
	if err := e.store.Clear(context.Background()); err != nil {
		fmt.Printf("clear store failed: %v\n", err)
	}
	e.clearCommunities()
}

// Stats 返回当前图的节点和边数量
func (e *Engine) Stats() (nodes, edges int) {
	ctx := context.Background()
	if nodeList, err := e.store.ListNodes(ctx); err == nil {
		nodes = len(nodeList)
	} else {
		fmt.Printf("list nodes failed: %v\n", err)
	}
	if edgeList, err := e.store.ListEdges(ctx); err == nil {
		edges = len(edgeList)
	} else {
		fmt.Printf("list edges failed: %v\n", err)
	}
	return nodes, edges
}

// IngestDocuments 摄入文档
func (e *Engine) IngestDocuments(ctx context.Context, docs []loader.Document) (*loader.IngestStats, error) {
	if len(docs) == 0 {
		return nil, errors.New("documents is empty")
	}
	stats := &loader.IngestStats{}
	for _, doc := range docs {
		if strings.TrimSpace(doc.Text) == "" {
			continue
		}
		if doc.ID == "" {
			doc.ID = fmt.Sprintf("doc-%d", time.Now().UnixNano())
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var (
			addedNodes int
			addedEdges int
			err        error
		)

		// 如果配置了 LLM 提取器和 Neo4j 存储，使用 LLM 提取
		if e.llmExtractor != nil && e.neo4jStore != nil {
			addedNodes, addedEdges, err = e.llmExtractor.IngestDocumentWithLLM(ctx, &doc, e.neo4jStore)
			if err != nil {
				// 如果 LLM 提取失败，回退到传统方法
				fmt.Printf("LLM extraction failed, falling back to traditional method: %v\n", err)
				addedNodes, addedEdges, err = e.ingestDocument(ctx, &doc)
			}
		} else {
			// 使用传统方法提取
			addedNodes, addedEdges, err = e.ingestDocument(ctx, &doc)
		}
		if err != nil {
			return nil, err
		}

		stats.Documents++
		stats.Nodes += addedNodes
		stats.Edges += addedEdges
	}
	if err := e.rebuildCommunities(ctx); err != nil {
		return stats, err
	}
	return stats, nil
}

// GlobalSearch 返回社区级别的上下文
func (e *Engine) GlobalSearch(query string, topK int) []loader.GlobalCommunityResult {
	if topK <= 0 {
		topK = e.options.GlobalTopK
	}
	communities, err := e.ensureCommunities(context.Background())
	if err != nil {
		fmt.Printf("ensure communities failed: %v\n", err)
		return nil
	}
	queryTokens := tokenize(query)
	results := make([]loader.GlobalCommunityResult, 0, topK)

	for _, community := range communities {
		score := semanticSimilarity(queryTokens, tokenize(community.Summary))
		if score <= 0 {
			continue
		}
		results = append(results, loader.GlobalCommunityResult{
			CommunityID: community.ID,
			Summary:     community.Summary,
			NodeIDs:     community.NodeIDs,
			Keywords:    community.Keywords,
			Score:       score,
		})
	}

	sortByScore(results)
	if len(results) > topK {
		results = results[:topK]
	}
	return results
}

// LocalSearch 返回节点级别的上下文
func (e *Engine) LocalSearch(query string, topKNodes, topKNeighbors int) []loader.LocalNodeResult {
	if topKNodes <= 0 {
		topKNodes = e.options.LocalNodeTopK
	}
	if topKNeighbors <= 0 {
		topKNeighbors = e.options.LocalNeighborTopK
	}
	ctx := context.Background()
	nodes, err := e.store.ListNodes(ctx)
	if err != nil {
		fmt.Printf("list nodes failed: %v\n", err)
		return nil
	}
	queryTokens := tokenize(query)
	candidates := make([]loader.LocalNodeResult, 0, topKNodes)

	for _, node := range nodes {
		score := semanticSimilarity(queryTokens, tokenize(node.Summary+strings.Join(node.Snippets, " ")))
		if score <= 0 {
			continue
		}
		local := loader.LocalNodeResult{
			NodeID:     node.ID,
			Name:       node.Name,
			Summary:    node.Summary,
			Snippets:   node.Snippets,
			Score:      score,
			Metadata:   node.Metadata,
			Occurrence: node.Occurrence,
		}

		edges, err := e.store.GetNeighbors(ctx, node.ID)
		if err != nil {
			fmt.Printf("get neighbors failed: %v\n", err)
			continue
		}
		sortEdgesByWeight(edges)
		for _, edge := range edges {
			if len(local.Neighbors) >= topKNeighbors {
				break
			}
			targetID := edge.Target
			if targetID == node.ID {
				targetID = edge.Source
			}
			targetNode, err := e.store.GetNode(ctx, targetID)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					fmt.Printf("get node %s failed: %v\n", targetID, err)
				}
				continue
			}
			local.Neighbors = append(local.Neighbors, loader.Neighbor{
				NodeID:   targetNode.ID,
				Name:     targetNode.Name,
				Relation: edge.Relation,
				Evidence: edge.Evidence,
				Weight:   edge.Weight,
				Summary:  targetNode.Summary,
				Score: semanticSimilarity(
					queryTokens,
					tokenize(targetNode.Summary+" "+edge.Evidence),
				),
			})
		}
		candidates = append(candidates, local)
	}

	sortLocalResults(candidates)
	if len(candidates) > topKNodes {
		candidates = candidates[:topKNodes]
	}
	return candidates
}

// PathSearch 搜索关键路径
func (e *Engine) PathSearch(query string, maxDepth int) []loader.PathResult {
	if maxDepth <= 0 {
		maxDepth = e.options.PathMaxDepth
	}
	nodes := e.LocalSearch(query, e.options.LocalNodeTopK, 2)
	if len(nodes) <= 1 {
		return nil
	}

	var results []loader.PathResult
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			path := e.findPath(context.Background(), nodes[i].NodeID, nodes[j].NodeID, maxDepth)
			if path == nil {
				continue
			}
			results = append(results, *path)
		}
	}

	sortPathResults(results)
	if len(results) > 3 {
		results = results[:3]
	}
	return results
}

func (e *Engine) setCommunities(data map[string]*loader.Community) {
	e.communitiesMu.Lock()
	defer e.communitiesMu.Unlock()
	e.communities = data
}

func (e *Engine) getCommunities() []*loader.Community {
	e.communitiesMu.RLock()
	defer e.communitiesMu.RUnlock()
	result := make([]*loader.Community, 0, len(e.communities))
	for _, community := range e.communities {
		copied := *community
		copied.NodeIDs = append([]string(nil), community.NodeIDs...)
		copied.Keywords = append([]string(nil), community.Keywords...)
		result = append(result, &copied)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result
}

func (e *Engine) clearCommunities() {
	e.communitiesMu.Lock()
	defer e.communitiesMu.Unlock()
	e.communities = make(map[string]*loader.Community)
}

func (e *Engine) ensureCommunities(ctx context.Context) ([]*loader.Community, error) {
	communities := e.getCommunities()
	if len(communities) > 0 {
		return communities, nil
	}
	if err := e.rebuildCommunities(ctx); err != nil {
		return nil, err
	}
	return e.getCommunities(), nil
}
