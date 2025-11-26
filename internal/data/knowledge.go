package data

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"jas-agent/internal/biz"

	"gorm.io/gorm"
)

type knowledgeBaseRepo struct {
	data *Data
}

func NewKnowledgeBaseRepo(data *Data) biz.KnowledgeBaseRepo {
	return &knowledgeBaseRepo{data: data}
}

func (r *knowledgeBaseRepo) db() (*gorm.DB, error) {
	if r.data == nil || r.data.DB() == nil {
		return nil, errDBNotConfigured
	}
	return r.data.DB(), nil
}

func (r *knowledgeBaseRepo) CreateKnowledgeBase(ctx context.Context, kb *biz.KnowledgeBase) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	model := knowledgeBaseModelFromBiz(kb)
	if err := db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("create knowledge base: %w", err)
	}
	kb.ID = model.ID
	return nil
}

func (r *knowledgeBaseRepo) UpdateKnowledgeBase(ctx context.Context, kb *biz.KnowledgeBase) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	model := knowledgeBaseModelFromBiz(kb)
	return db.WithContext(ctx).Model(&KnowledgeBaseModel{ID: model.ID}).Updates(map[string]interface{}{
		"name":                model.Name,
		"description":         model.Description,
		"tags":                model.Tags,
		"embedding_model":     model.EmbeddingModel,
		"chunk_size":          model.ChunkSize,
		"chunk_overlap":       model.ChunkOverlap,
		"vector_store_type":   model.VectorStoreType,
		"vector_store_config": model.VectorStoreConfig,
		"is_active":           model.IsActive,
	}).Error
}

func (r *knowledgeBaseRepo) DeleteKnowledgeBase(ctx context.Context, id int) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	return db.WithContext(ctx).Where("id = ?", id).Delete(&KnowledgeBaseModel{}).Error
}

func (r *knowledgeBaseRepo) GetKnowledgeBase(ctx context.Context, id int) (*biz.KnowledgeBase, error) {
	db, err := r.db()
	if err != nil {
		return nil, err
	}

	var model KnowledgeBaseModel
	if err := db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("knowledge base not found: %d", id)
		}
		return nil, fmt.Errorf("query knowledge base: %w", err)
	}

	kb := model.ToBiz()

	// 统计文档数量
	var docCount int64
	db.WithContext(ctx).Model(&DocumentModel{}).Where("knowledge_base_id = ?", id).Count(&docCount)
	kb.DocumentCount = int(docCount)

	return kb, nil
}

func (r *knowledgeBaseRepo) ListKnowledgeBases(ctx context.Context, searchQuery string, tags []string) ([]*biz.KnowledgeBase, error) {
	db, err := r.db()
	if err != nil {
		return nil, err
	}

	var models []KnowledgeBaseModel
	query := db.WithContext(ctx)

	// 按名字搜索
	if searchQuery != "" {
		query = query.Where("name LIKE ?", "%"+searchQuery+"%")
	}

	// 按标签搜索
	if len(tags) > 0 {
		for _, tag := range tags {
			query = query.Where("JSON_CONTAINS(tags, ?)", fmt.Sprintf(`"%s"`, tag))
		}
	}

	if err := query.Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list knowledge bases: %w", err)
	}

	kbs := make([]*biz.KnowledgeBase, 0, len(models))
	for _, model := range models {
		kb := model.ToBiz()
		// 统计文档数量
		var docCount int64
		db.WithContext(ctx).Model(&DocumentModel{}).Where("knowledge_base_id = ?", model.ID).Count(&docCount)
		kb.DocumentCount = int(docCount)
		kbs = append(kbs, kb)
	}
	return kbs, nil
}

type KnowledgeBaseModel struct {
	ID                int       `gorm:"column:id;primaryKey"`
	Name              string    `gorm:"column:name"`
	Description       string    `gorm:"column:description"`
	Tags              JSONArray `gorm:"column:tags;type:json"`
	EmbeddingModel    string    `gorm:"column:embedding_model"`
	ChunkSize         int       `gorm:"column:chunk_size"`
	ChunkOverlap      int       `gorm:"column:chunk_overlap"`
	VectorStoreType   string    `gorm:"column:vector_store_type"`
	VectorStoreConfig string    `gorm:"column:vector_store_config"`
	IsActive          bool      `gorm:"column:is_active"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

func (KnowledgeBaseModel) TableName() string {
	return "knowledge_bases"
}

func (m KnowledgeBaseModel) ToBiz() *biz.KnowledgeBase {
	return &biz.KnowledgeBase{
		ID:                m.ID,
		Name:              m.Name,
		Description:       m.Description,
		Tags:              []string(m.Tags),
		EmbeddingModel:    m.EmbeddingModel,
		ChunkSize:         m.ChunkSize,
		ChunkOverlap:      m.ChunkOverlap,
		VectorStoreType:   m.VectorStoreType,
		VectorStoreConfig: m.VectorStoreConfig,
		IsActive:          m.IsActive,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

func knowledgeBaseModelFromBiz(kb *biz.KnowledgeBase) *KnowledgeBaseModel {
	return &KnowledgeBaseModel{
		ID:                kb.ID,
		Name:              kb.Name,
		Description:       kb.Description,
		Tags:              JSONArray(kb.Tags),
		EmbeddingModel:    kb.EmbeddingModel,
		ChunkSize:         kb.ChunkSize,
		ChunkOverlap:      kb.ChunkOverlap,
		VectorStoreType:   kb.VectorStoreType,
		VectorStoreConfig: kb.VectorStoreConfig,
		IsActive:          kb.IsActive,
	}
}

// JSONArray 用于处理 JSON 数组字段
type JSONArray []string

func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = []string{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONArray", value)
	}

	if len(bytes) == 0 || string(bytes) == "[]" {
		*j = []string{}
		return nil
	}

	return json.Unmarshal(bytes, j)
}

func (j JSONArray) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "[]", nil
	}
	return json.Marshal(j)
}

// DocumentRepo 实现

type documentRepo struct {
	data *Data
}

func NewDocumentRepo(data *Data) biz.DocumentRepo {
	return &documentRepo{data: data}
}

func (r *documentRepo) db() (*gorm.DB, error) {
	if r.data == nil || r.data.DB() == nil {
		return nil, errDBNotConfigured
	}
	return r.data.DB(), nil
}

func (r *documentRepo) CreateDocument(ctx context.Context, doc *biz.Document) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	model := documentModelFromBiz(doc)
	if err := db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("create document: %w", err)
	}
	doc.ID = model.ID
	return nil
}

func (r *documentRepo) UpdateDocument(ctx context.Context, doc *biz.Document) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	model := documentModelFromBiz(doc)
	return db.WithContext(ctx).Model(&DocumentModel{ID: model.ID}).Updates(map[string]interface{}{
		"name":          model.Name,
		"file_path":     model.FilePath,
		"file_size":     model.FileSize,
		"file_type":     model.FileType,
		"status":        model.Status,
		"chunk_count":   model.ChunkCount,
		"processed_at":  model.ProcessedAt,
		"error_message": model.ErrorMessage,
		"metadata":      model.Metadata,
	}).Error
}

func (r *documentRepo) DeleteDocument(ctx context.Context, id int) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	return db.WithContext(ctx).Where("id = ?", id).Delete(&DocumentModel{}).Error
}

func (r *documentRepo) GetDocument(ctx context.Context, id int) (*biz.Document, error) {
	db, err := r.db()
	if err != nil {
		return nil, err
	}

	var model DocumentModel
	if err := db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("document not found: %d", id)
		}
		return nil, fmt.Errorf("query document: %w", err)
	}

	return model.ToBiz(), nil
}

func (r *documentRepo) ListDocuments(ctx context.Context, knowledgeBaseID int) ([]*biz.Document, error) {
	db, err := r.db()
	if err != nil {
		return nil, err
	}

	var models []DocumentModel
	if err := db.WithContext(ctx).Where("knowledge_base_id = ?", knowledgeBaseID).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}

	docs := make([]*biz.Document, 0, len(models))
	for _, model := range models {
		docs = append(docs, model.ToBiz())
	}
	return docs, nil
}

func (r *documentRepo) UpdateDocumentStatus(ctx context.Context, id int, status string, errorMsg string) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"status": status,
	}
	if status == "completed" {
		now := time.Now()
		updates["processed_at"] = &now
	}
	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}

	return db.WithContext(ctx).Model(&DocumentModel{ID: id}).Updates(updates).Error
}

type DocumentModel struct {
	ID              int        `gorm:"column:id;primaryKey"`
	KnowledgeBaseID int        `gorm:"column:knowledge_base_id"`
	Name            string     `gorm:"column:name"`
	FilePath        string     `gorm:"column:file_path"`
	FileSize        int64      `gorm:"column:file_size"`
	FileType        string     `gorm:"column:file_type"`
	Status          string     `gorm:"column:status"`
	ChunkCount      int        `gorm:"column:chunk_count"`
	ProcessedAt     *time.Time `gorm:"column:processed_at"`
	ErrorMessage    string     `gorm:"column:error_message"`
	Metadata        string     `gorm:"column:metadata"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (DocumentModel) TableName() string {
	return "documents"
}

func (m DocumentModel) ToBiz() *biz.Document {
	return &biz.Document{
		ID:              m.ID,
		KnowledgeBaseID: m.KnowledgeBaseID,
		Name:            m.Name,
		FilePath:        m.FilePath,
		FileSize:        m.FileSize,
		FileType:        m.FileType,
		Status:          m.Status,
		ChunkCount:      m.ChunkCount,
		ProcessedAt:     m.ProcessedAt,
		ErrorMessage:    m.ErrorMessage,
		Metadata:        m.Metadata,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func documentModelFromBiz(doc *biz.Document) *DocumentModel {
	return &DocumentModel{
		ID:              doc.ID,
		KnowledgeBaseID: doc.KnowledgeBaseID,
		Name:            doc.Name,
		FilePath:        doc.FilePath,
		FileSize:        doc.FileSize,
		FileType:        doc.FileType,
		Status:          doc.Status,
		ChunkCount:      doc.ChunkCount,
		ProcessedAt:     doc.ProcessedAt,
		ErrorMessage:    doc.ErrorMessage,
		Metadata: func(metadata string) string {
			if len(metadata) > 0 {
				return metadata
			}
			return "{}"
		}(doc.Metadata),
	}
}
