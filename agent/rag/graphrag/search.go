package graphrag

import (
	"context"
	"jas-agent/agent/rag/loader"
	"sort"
)

type pathState struct {
	NodeID   string
	Path     []string
	Evidence []string
	Score    float64
}

func sortByScore(results []loader.GlobalCommunityResult) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].CommunityID < results[j].CommunityID
		}
		return results[i].Score > results[j].Score
	})
}

func sortEdgesByWeight(edges []*loader.GraphEdge) {
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Weight > edges[j].Weight
	})
}

func sortLocalResults(results []loader.LocalNodeResult) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Occurrence > results[j].Occurrence
		}
		return results[i].Score > results[j].Score
	})
}

func sortPathResults(results []loader.PathResult) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return len(results[i].Nodes) < len(results[j].Nodes)
		}
		return results[i].Score > results[j].Score
	})
}

func (e *Engine) findPath(ctx context.Context, start, target string, maxDepth int) *loader.PathResult {
	if start == target {
		return nil
	}
	queue := []pathState{{
		NodeID: start,
		Path:   []string{start},
	}}
	visited := map[string]int{start: 1}

	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]
		if len(state.Path) > maxDepth+1 {
			continue
		}
		neighbors, err := e.store.GetNeighbors(ctx, state.NodeID)
		if err != nil {
			continue
		}
		for _, edge := range neighbors {
			next := edge.Target
			if next == state.NodeID {
				next = edge.Source
			}
			if next == "" {
				continue
			}
			pathLen := len(state.Path) + 1
			if prev, ok := visited[next]; ok && prev <= pathLen {
				continue
			}
			newState := pathState{
				NodeID:   next,
				Path:     append(append([]string(nil), state.Path...), next),
				Evidence: append(append([]string(nil), state.Evidence...), edge.Evidence),
				Score:    state.Score + edge.Weight,
			}
			if next == target {
				return e.buildPathResult(ctx, newState)
			}
			visited[next] = pathLen
			queue = append(queue, newState)
		}
	}
	return nil
}

func (e *Engine) buildPathResult(ctx context.Context, state pathState) *loader.PathResult {
	pathNodes := make([]loader.PathNode, 0, len(state.Path))
	for _, nodeID := range state.Path {
		node, err := e.store.GetNode(ctx, nodeID)
		if err != nil {
			pathNodes = append(pathNodes, loader.PathNode{NodeID: nodeID})
			continue
		}
		pathNodes = append(pathNodes, loader.PathNode{
			NodeID:  node.ID,
			Name:    node.Name,
			Summary: node.Summary,
		})
	}
	return &loader.PathResult{
		Nodes:    pathNodes,
		Edges:    append([]string(nil), state.Evidence...),
		Evidence: append([]string(nil), state.Evidence...),
		Score:    state.Score,
	}
}
