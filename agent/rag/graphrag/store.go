package graphrag

import (
	"fmt"
	"jas-agent/agent/rag/loader"
	"sort"
	"sync"
	"time"
)

type graphStore struct {
	mu          sync.RWMutex
	nodes       map[string]*loader.GraphNode
	edges       map[string]map[string][]*loader.GraphEdge
	communities map[string]*loader.Community
	documents   map[string]*loader.Document
}

func newGraphStore() *graphStore {
	return &graphStore{
		nodes:       make(map[string]*loader.GraphNode),
		edges:       make(map[string]map[string][]*loader.GraphEdge),
		communities: make(map[string]*loader.Community),
		documents:   make(map[string]*loader.Document),
	}
}

func (s *graphStore) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes = make(map[string]*loader.GraphNode)
	s.edges = make(map[string]map[string][]*loader.GraphEdge)
	s.communities = make(map[string]*loader.Community)
	s.documents = make(map[string]*loader.Document)
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

func (s *graphStore) addDocument(doc *loader.Document) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents[doc.ID] = doc
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

func (s *graphStore) updateCommunities(communities map[string]*loader.Community) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.communities = communities

	// 更新节点的社区信息
	for _, node := range s.nodes {
		node.CommunityIDs = nil
	}
	for id, community := range communities {
		for _, nodeID := range community.NodeIDs {
			if node, ok := s.nodes[nodeID]; ok {
				node.CommunityIDs = append(node.CommunityIDs, id)
			}
		}
	}
}

func (s *graphStore) listCommunities() []*loader.Community {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*loader.Community, 0, len(s.communities))
	for _, c := range s.communities {
		copied := *c
		copied.NodeIDs = append([]string(nil), c.NodeIDs...)
		copied.Keywords = append([]string(nil), c.Keywords...)
		result = append(result, &copied)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result
}

func (s *graphStore) stats() (nodes int, edges int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nodes = len(s.nodes)
	for _, targets := range s.edges {
		for _, list := range targets {
			edges += len(list)
		}
	}
	return nodes, edges
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

func edgeKey(source, target string) string {
	return fmt.Sprintf("%s->%s", source, target)
}
