package loader

import "time"

// Document 表示需要被索引的原始文档
type Document struct {
	ID       string            `json:"id"`
	Text     string            `json:"text"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Tags     []string          `json:"tags,omitempty"`
}

// GraphNode 表示知识图谱中的实体
type GraphNode struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Summary      string            `json:"summary"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	SourceDocs   map[string]int    `json:"source_docs,omitempty"`
	Snippets     []string          `json:"snippets,omitempty"`
	UpdatedAt    time.Time         `json:"updated_at"`
	CreatedAt    time.Time         `json:"created_at"`
	Occurrence   int               `json:"occurrence"`
	CommunityIDs []string          `json:"community_ids,omitempty"`
}

// GraphEdge 表示实体之间的关系
type GraphEdge struct {
	Source    string    `json:"source"`
	Target    string    `json:"target"`
	Relation  string    `json:"relation"`
	Evidence  string    `json:"evidence"`
	Weight    float64   `json:"weight"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Community 代表社区级别的聚合
type Community struct {
	ID        string    `json:"id"`
	NodeIDs   []string  `json:"node_ids"`
	Summary   string    `json:"summary"`
	Keywords  []string  `json:"keywords"`
	UpdatedAt time.Time `json:"updated_at"`
	Score     float64   `json:"score"`
}

// GlobalCommunityResult 用于 Global Search 返回
type GlobalCommunityResult struct {
	CommunityID string   `json:"community_id"`
	Summary     string   `json:"summary"`
	NodeIDs     []string `json:"node_ids"`
	Keywords    []string `json:"keywords"`
	Score       float64  `json:"score"`
}

// LocalNodeResult 用于 Local Search 返回
type LocalNodeResult struct {
	NodeID     string     `json:"node_id"`
	Name       string     `json:"name"`
	Summary    string     `json:"summary"`
	Snippets   []string   `json:"snippets"`
	Score      float64    `json:"score"`
	Neighbors  []Neighbor `json:"neighbors"`
	Metadata   map[string]string
	Occurrence int `json:"occurrence"`
}

// Neighbor 表示邻居节点信息
type Neighbor struct {
	NodeID   string  `json:"node_id"`
	Name     string  `json:"name"`
	Relation string  `json:"relation"`
	Evidence string  `json:"evidence"`
	Weight   float64 `json:"weight"`
	Summary  string  `json:"summary"`
	Score    float64 `json:"score"`
}

// PathResult 表示 Path Search 的结果
type PathResult struct {
	Nodes    []PathNode `json:"nodes"`
	Edges    []string   `json:"edges"`
	Score    float64    `json:"score"`
	Evidence []string   `json:"evidence"`
}

// PathNode 表示路径中的节点
type PathNode struct {
	NodeID  string `json:"node_id"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

// IngestStats 记录一次摄入的统计
type IngestStats struct {
	Documents int `json:"documents"`
	Nodes     int `json:"nodes"`
	Edges     int `json:"edges"`
}
