package biz

import (
	"context"
	"encoding/json"
	"time"
)

// KnowledgeBase 知识库领域模型
type KnowledgeBase struct {
	ID                int
	Name              string
	Description       string
	Tags              []string // 标签列表
	EmbeddingModel    string
	ChunkSize         int
	ChunkOverlap      int
	VectorStoreType   string
	VectorStoreConfig string
	IsActive          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DocumentCount     int // 文档数量（统计）
}

// Document 文档领域模型
type Document struct {
	ID              int
	KnowledgeBaseID int
	Name            string
	FilePath        string
	FileSize        int64
	FileType        string
	Status          string // pending, processing, completed, failed
	ChunkCount      int
	ProcessedAt     *time.Time
	ErrorMessage    string
	Metadata        string // JSON格式
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// KnowledgeBaseRepo 知识库数据访问接口
type KnowledgeBaseRepo interface {
	CreateKnowledgeBase(ctx context.Context, kb *KnowledgeBase) error
	UpdateKnowledgeBase(ctx context.Context, kb *KnowledgeBase) error
	DeleteKnowledgeBase(ctx context.Context, id int) error
	GetKnowledgeBase(ctx context.Context, id int) (*KnowledgeBase, error)
	ListKnowledgeBases(ctx context.Context, searchQuery string, tags []string) ([]*KnowledgeBase, error)
}

// DocumentRepo 文档数据访问接口
type DocumentRepo interface {
	CreateDocument(ctx context.Context, doc *Document) error
	UpdateDocument(ctx context.Context, doc *Document) error
	DeleteDocument(ctx context.Context, id int) error
	GetDocument(ctx context.Context, id int) (*Document, error)
	ListDocuments(ctx context.Context, knowledgeBaseID int) ([]*Document, error)
	UpdateDocumentStatus(ctx context.Context, id int, status string, errorMsg string) error
}

// TagsToJSON 将标签列表转换为JSON字符串
func TagsToJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(tags)
	return string(data)
}

// JSONToTags 将JSON字符串转换为标签列表
func JSONToTags(jsonStr string) []string {
	if jsonStr == "" || jsonStr == "[]" {
		return []string{}
	}
	var tags []string
	_ = json.Unmarshal([]byte(jsonStr), &tags)
	return tags
}
