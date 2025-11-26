package vectordb

import (
	"context"
	"errors"
	"fmt"
	"jas-agent/agent/rag/loader"
	"math"
	"sort"
	"sync"
)

// VectorStore 向量数据库接口
type VectorStore interface {
	// Insert 插入文档向量
	Insert(ctx context.Context, vectors []Vector) error
	// Search 向量相似度搜索
	Search(ctx context.Context, queryVector []float32, topK int, filter map[string]string) ([]SearchResult, error)
	// Delete 根据 ID 删除向量
	Delete(ctx context.Context, ids []string) error
	// GetByID 根据 ID 获取向量
	GetByID(ctx context.Context, id string) (*Vector, error)
	// BatchGet 批量获取向量
	BatchGet(ctx context.Context, ids []string) ([]*Vector, error)
	// Stats 获取统计信息
	Stats(ctx context.Context) (Stats, error)
}

// Vector 表示一个向量和其关联的文档
type Vector struct {
	ID       string            `json:"id"`
	Document *loader.Document  `json:"document"`
	Vector   []float32         `json:"vector"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Vector   *Vector `json:"vector"`
	Score    float64 `json:"score"`
	Distance float64 `json:"distance"`
	// 便捷访问字段（从 Vector 和 Document 中提取）
	ID       string            `json:"id"`       // 文档ID
	Text     string            `json:"text"`     // 文档文本
	Metadata map[string]string `json:"metadata"` // 元数据（合并 Vector.Metadata 和 Document.Metadata）
}

// GetMetadata 获取元数据值（优先从 Document.Metadata，其次从 Vector.Metadata）
func (r *SearchResult) GetMetadata(key string) string {
	if r.Metadata != nil {
		if value, ok := r.Metadata[key]; ok {
			return value
		}
	}
	return ""
}

// GetDocument 获取文档对象
func (r *SearchResult) GetDocument() *loader.Document {
	if r.Vector != nil && r.Vector.Document != nil {
		return r.Vector.Document
	}
	return nil
}

// Stats 向量数据库统计信息
type Stats struct {
	Count      int `json:"count"`
	Dimensions int `json:"dimensions"`
}

// Filter 用于过滤搜索结果的选项
type Filter struct {
	Metadata map[string]string
}

// inMemoryStore 内存向量存储实现（用于测试和小规模数据）
type inMemoryStore struct {
	mu         sync.RWMutex
	vectors    map[string]*Vector
	dimensions int
}

// NewInMemoryStore 创建内存向量存储
func NewInMemoryStore(dimensions int) VectorStore {
	return &inMemoryStore{
		vectors:    make(map[string]*Vector),
		dimensions: dimensions,
	}
}

func (s *inMemoryStore) Insert(ctx context.Context, vectors []Vector) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range vectors {
		v := &vectors[i]
		// 验证向量维度
		if len(v.Vector) != s.dimensions {
			return fmt.Errorf("vector dimension mismatch: expected %d, got %d", s.dimensions, len(v.Vector))
		}
		s.vectors[v.ID] = v
	}
	return nil
}

func (s *inMemoryStore) Search(ctx context.Context, queryVector []float32, topK int, filter map[string]string) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(queryVector) != s.dimensions {
		return nil, fmt.Errorf("query vector dimension mismatch: expected %d, got %d", s.dimensions, len(queryVector))
	}

	type scoredVector struct {
		vector *Vector
		score  float64
	}

	var results []scoredVector

	for _, v := range s.vectors {
		// 应用过滤器
		if filter != nil && !s.matchesFilter(v, filter) {
			continue
		}

		score := cosineSimilarity(queryVector, v.Vector)
		results = append(results, scoredVector{
			vector: v,
			score:  score,
		})
	}

	// 按相似度排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// 返回 topK 结果
	if topK > len(results) {
		topK = len(results)
	}

	searchResults := make([]SearchResult, topK)
	for i := 0; i < topK; i++ {
		distance := 1 - results[i].score // cosine distance
		v := results[i].vector

		// 提取基本信息
		id := v.ID
		var text string
		if v.Document != nil {
			text = v.Document.Text
		}

		// 合并元数据（优先使用 Document.Metadata，其次使用 Vector.Metadata）
		metadata := make(map[string]string)

		// 先添加 Vector.Metadata
		if v.Metadata != nil {
			for k, v := range v.Metadata {
				metadata[k] = v
			}
		}

		// 再添加 Document.Metadata（会覆盖重复的键）
		if v.Document != nil && v.Document.Metadata != nil {
			for k, v := range v.Document.Metadata {
				metadata[k] = v
			}
		}

		searchResults[i] = SearchResult{
			Vector:   v,
			Score:    results[i].score,
			Distance: distance,
			ID:       id,
			Text:     text,
			Metadata: metadata,
		}
	}

	return searchResults, nil
}

func (s *inMemoryStore) Delete(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		delete(s.vectors, id)
	}
	return nil
}

func (s *inMemoryStore) GetByID(ctx context.Context, id string) (*Vector, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.vectors[id]
	if !ok {
		return nil, errors.New("vector not found")
	}

	// 返回副本
	copied := *v
	copied.Metadata = cloneMap(v.Metadata)
	return &copied, nil
}

func (s *inMemoryStore) BatchGet(ctx context.Context, ids []string) ([]*Vector, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Vector, 0, len(ids))
	for _, id := range ids {
		if v, ok := s.vectors[id]; ok {
			copied := *v
			copied.Metadata = cloneMap(v.Metadata)
			result = append(result, &copied)
		}
	}
	return result, nil
}

func (s *inMemoryStore) Stats(ctx context.Context) (Stats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return Stats{
		Count:      len(s.vectors),
		Dimensions: s.dimensions,
	}, nil
}

func (s *inMemoryStore) matchesFilter(v *Vector, filter map[string]string) bool {
	if v.Document == nil || v.Document.Metadata == nil {
		return len(filter) == 0
	}

	for key, value := range filter {
		if v.Document.Metadata[key] != value {
			return false
		}
	}
	return true
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var normA, normB float64

	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// sqrt 包装 math.Sqrt 以保持一致性
func sqrt(x float64) float64 {
	return math.Sqrt(x)
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
