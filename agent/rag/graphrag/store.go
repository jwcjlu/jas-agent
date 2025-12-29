package graphrag

import (
	"context"
	"errors"
	"fmt"
	"jas-agent/agent/rag/loader"
	"sync"
	"time"
)

// Store 抽象存储接口，提供统一的图数据读写能力
type Store interface {
	UpsertNode(ctx context.Context, node *loader.GraphNode) error
	UpsertEdge(ctx context.Context, edge *loader.GraphEdge) error
	GetNode(ctx context.Context, nodeID string) (*loader.GraphNode, error)
	ListNodes(ctx context.Context) ([]*loader.GraphNode, error)
	ListEdges(ctx context.Context) ([]*loader.GraphEdge, error)
	GetNeighbors(ctx context.Context, nodeID string) ([]*loader.GraphEdge, error)
	Clear(ctx context.Context) error
	Close(ctx context.Context) error
}

type graphStore struct {
	mu    sync.RWMutex
	nodes map[string]*loader.GraphNode
	edges map[string]map[string][]*loader.GraphEdge
}

func newGraphStore() *graphStore {
	return &graphStore{
		nodes: make(map[string]*loader.GraphNode),
		edges: make(map[string]map[string][]*loader.GraphEdge),
	}
}

func (s *graphStore) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes = make(map[string]*loader.GraphNode)
	s.edges = make(map[string]map[string][]*loader.GraphEdge)
}

func (s *graphStore) upsertNode(id string, update func(*loader.GraphNode, bool)) *loader.GraphNode {
	s.mu.Lock()
	defer s.mu.Unlock()
	node, ok := s.nodes[id]
	if !ok {
		node = &loader.GraphNode{
			ID:         id,
			Name:       id,
			Metadata:   map[string]string{},
			SourceDocs: map[string]int{},
			CreatedAt:  time.Now(),
		}
		s.nodes[id] = node
	}
	update(node, !ok)
	node.UpdatedAt = time.Now()
	return node
}

func (s *graphStore) addEdge(edge *loader.GraphEdge) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.edges[edge.Source]; !ok {
		s.edges[edge.Source] = make(map[string][]*loader.GraphEdge)
	}
	s.edges[edge.Source][edge.Target] = append(s.edges[edge.Source][edge.Target], edge)
	edge.UpdatedAt = time.Now()
}

func (s *graphStore) getNode(id string) (*loader.GraphNode, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	node, ok := s.nodes[id]
	if !ok {
		return nil, false
	}
	copied := *node
	copied.Metadata = cloneMap(node.Metadata)
	copied.SourceDocs = cloneIntMap(node.SourceDocs)
	copied.Snippets = append([]string(nil), node.Snippets...)
	copied.CommunityIDs = append([]string(nil), node.CommunityIDs...)
	return &copied, true
}

func (s *graphStore) listNodes() []*loader.GraphNode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nodes := make([]*loader.GraphNode, 0, len(s.nodes))
	for _, node := range s.nodes {
		copied := *node
		copied.Metadata = cloneMap(node.Metadata)
		copied.SourceDocs = cloneIntMap(node.SourceDocs)
		copied.Snippets = append([]string(nil), node.Snippets...)
		copied.CommunityIDs = append([]string(nil), node.CommunityIDs...)
		nodes = append(nodes, &copied)
	}
	return nodes
}

func (s *graphStore) listEdges() []*loader.GraphEdge {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var edges []*loader.GraphEdge
	for _, targets := range s.edges {
		for _, e := range targets {
			for _, item := range e {
				copied := *item
				edges = append(edges, &copied)
			}
		}
	}
	return edges
}

func (s *graphStore) neighbors(nodeID string) []*loader.GraphEdge {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var edges []*loader.GraphEdge
	if targets, ok := s.edges[nodeID]; ok {
		for _, list := range targets {
			for _, edge := range list {
				copied := *edge
				edges = append(edges, &copied)
			}
		}
	}
	// 反向
	for source, targets := range s.edges {
		if list, ok := targets[nodeID]; ok {
			for _, edge := range list {
				copied := *edge
				// 反向边权重稍微降低
				copied.Weight *= 0.9
				copied.Source = source
				copied.Target = nodeID
				edges = append(edges, &copied)
			}
		}
	}
	return edges
}

func cloneMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneIntMap(src map[string]int) map[string]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

var (
	_           Store = (*graphStore)(nil)
	ErrNotFound       = errors.New("graphrag store: not found")
)

// UpsertNode 满足 Store 接口，直接写入或覆盖节点
func (s *graphStore) UpsertNode(ctx context.Context, node *loader.GraphNode) error {
	if node == nil || node.ID == "" {
		return fmt.Errorf("invalid node")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := cloneGraphNode(node)
	if existing, ok := s.nodes[node.ID]; ok {
		*existing = *cloned
	} else {
		s.nodes[node.ID] = cloned
	}
	return nil
}

// UpsertEdge 满足 Store 接口，直接追加边
func (s *graphStore) UpsertEdge(ctx context.Context, edge *loader.GraphEdge) error {
	if edge == nil || edge.Source == "" || edge.Target == "" {
		return fmt.Errorf("invalid edge")
	}
	s.addEdge(cloneGraphEdge(edge))
	return nil
}

// GetNode 满足 Store 接口
func (s *graphStore) GetNode(ctx context.Context, nodeID string) (*loader.GraphNode, error) {
	node, ok := s.getNode(nodeID)
	if !ok {
		return nil, ErrNotFound
	}
	return node, nil
}

// ListNodes 满足 Store 接口
func (s *graphStore) ListNodes(ctx context.Context) ([]*loader.GraphNode, error) {
	return s.listNodes(), nil
}

// ListEdges 满足 Store 接口
func (s *graphStore) ListEdges(ctx context.Context) ([]*loader.GraphEdge, error) {
	return s.listEdges(), nil
}

// GetNeighbors 满足 Store 接口
func (s *graphStore) GetNeighbors(ctx context.Context, nodeID string) ([]*loader.GraphEdge, error) {
	return s.neighbors(nodeID), nil
}

// Clear 满足 Store 接口
func (s *graphStore) Clear(ctx context.Context) error {
	s.reset()
	return nil
}

// Close 满足 Store 接口
func (s *graphStore) Close(ctx context.Context) error {
	return nil
}

func cloneGraphNode(node *loader.GraphNode) *loader.GraphNode {
	if node == nil {
		return nil
	}
	cloned := *node
	cloned.Metadata = cloneMap(node.Metadata)
	cloned.SourceDocs = cloneIntMap(node.SourceDocs)
	cloned.Snippets = append([]string(nil), node.Snippets...)
	cloned.CommunityIDs = append([]string(nil), node.CommunityIDs...)
	return &cloned
}

func cloneGraphEdge(edge *loader.GraphEdge) *loader.GraphEdge {
	if edge == nil {
		return nil
	}
	cloned := *edge
	return &cloned
}
