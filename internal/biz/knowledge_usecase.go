package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"jas-agent/internal/conf"
	"os"
	"path/filepath"
	"time"

	"jas-agent/agent/rag/embedding"
	"jas-agent/agent/rag/loader"
	"jas-agent/agent/rag/quality"
	"jas-agent/agent/rag/vectordb"

	"github.com/go-kratos/kratos/v2/log"
)

// KnowledgeUsecase 知识库业务逻辑
type KnowledgeUsecase struct {
	kbRepo       KnowledgeBaseRepo
	docRepo      DocumentRepo
	logger       *log.Helper
	openaiAPIKey string // OpenAI API Key
	uploadDir    string // 文件上传目录
	embedder     embedding.Embedder
}

// NewKnowledgeUsecase 创建知识库业务逻辑
func NewKnowledgeUsecase(kbRepo KnowledgeBaseRepo, docRepo DocumentRepo, logger log.Logger, c *conf.Bootstrap, embedder embedding.Embedder) *KnowledgeUsecase {

	uploadDir := "./uploads"

	return &KnowledgeUsecase{
		kbRepo:       kbRepo,
		docRepo:      docRepo,
		logger:       log.NewHelper(log.With(logger, "module", "biz/knowledge")),
		openaiAPIKey: c.Llm.ApiKey,
		uploadDir:    uploadDir,
		embedder:     embedder,
	}
}

// CreateKnowledgeBase 创建知识库
func (s *KnowledgeUsecase) CreateKnowledgeBase(ctx context.Context, kb *KnowledgeBase) error {
	if kb.Name == "" {
		return fmt.Errorf("name is required")
	}

	// 设置默认值
	if kb.EmbeddingModel == "" {
		kb.EmbeddingModel = "text-embedding-3-small"
	}
	if kb.ChunkSize == 0 {
		kb.ChunkSize = 800
	}
	if kb.ChunkOverlap == 0 {
		kb.ChunkOverlap = 120
	}
	if kb.VectorStoreType == "" {
		kb.VectorStoreType = "memory"
	}
	if kb.VectorStoreConfig == "" {
		kb.VectorStoreConfig = "{}"
	}
	if kb.Tags == nil {
		kb.Tags = []string{}
	}

	return s.kbRepo.CreateKnowledgeBase(ctx, kb)
}

// UpdateKnowledgeBase 更新知识库
func (s *KnowledgeUsecase) UpdateKnowledgeBase(ctx context.Context, kb *KnowledgeBase) error {
	if kb.ID <= 0 {
		return fmt.Errorf("knowledge base id is required")
	}
	return s.kbRepo.UpdateKnowledgeBase(ctx, kb)
}

// DeleteKnowledgeBase 删除知识库
func (s *KnowledgeUsecase) DeleteKnowledgeBase(ctx context.Context, id int) error {
	// 检查是否有文档关联
	docs, err := s.docRepo.ListDocuments(ctx, id)
	if err != nil {
		return fmt.Errorf("list documents: %w", err)
	}
	if len(docs) > 0 {
		return fmt.Errorf("cannot delete knowledge base with documents, please delete documents first")
	}
	return s.kbRepo.DeleteKnowledgeBase(ctx, id)
}

// GetKnowledgeBase 获取知识库
func (s *KnowledgeUsecase) GetKnowledgeBase(ctx context.Context, id int) (*KnowledgeBase, error) {
	return s.kbRepo.GetKnowledgeBase(ctx, id)
}

// ListKnowledgeBases 列出知识库（支持搜索）
func (s *KnowledgeUsecase) ListKnowledgeBases(ctx context.Context, searchQuery string, tags []string) ([]*KnowledgeBase, error) {
	return s.kbRepo.ListKnowledgeBases(ctx, searchQuery, tags)
}

// CreateDocument 创建文档记录
func (s *KnowledgeUsecase) CreateDocument(ctx context.Context, doc *Document) error {
	if doc.KnowledgeBaseID <= 0 {
		return fmt.Errorf("knowledge_base_id is required")
	}
	if doc.Name == "" {
		return fmt.Errorf("name is required")
	}
	if doc.Status == "" {
		doc.Status = "pending"
	}
	return s.docRepo.CreateDocument(ctx, doc)
}

// UpdateDocument 更新文档
func (s *KnowledgeUsecase) UpdateDocument(ctx context.Context, doc *Document) error {
	if doc.ID <= 0 {
		return fmt.Errorf("document id is required")
	}
	return s.docRepo.UpdateDocument(ctx, doc)
}

// DeleteDocument 删除文档
func (s *KnowledgeUsecase) DeleteDocument(ctx context.Context, id int) error {
	return s.docRepo.DeleteDocument(ctx, id)
}

// GetDocument 获取文档
func (s *KnowledgeUsecase) GetDocument(ctx context.Context, id int) (*Document, error) {
	return s.docRepo.GetDocument(ctx, id)
}

// ListDocuments 列出文档
func (s *KnowledgeUsecase) ListDocuments(ctx context.Context, knowledgeBaseID int) ([]*Document, error) {
	return s.docRepo.ListDocuments(ctx, knowledgeBaseID)
}

// UpdateDocumentStatus 更新文档状态
func (s *KnowledgeUsecase) UpdateDocumentStatus(ctx context.Context, id int, status string, errorMsg string) error {
	return s.docRepo.UpdateDocumentStatus(ctx, id, status, errorMsg)
}

// UploadAndProcessDocument 上传并处理文档（解析、分块、向量化、存储）
func (s *KnowledgeUsecase) UploadAndProcessDocument(ctx context.Context, knowledgeBaseID int, fileName string, fileContent io.Reader, fileSize int64) (*Document, error) {
	// 1. 获取知识库配置
	kb, err := s.kbRepo.GetKnowledgeBase(ctx, knowledgeBaseID)
	if err != nil {
		return nil, fmt.Errorf("get knowledge base: %w", err)
	}

	// 2. 创建文档记录（状态为 pending）
	doc := &Document{
		KnowledgeBaseID: knowledgeBaseID,
		Name:            fileName,
		FileSize:        fileSize,
		FileType:        s.detectFileType(fileName),
		Status:          "pending",
		ChunkCount:      0,
	}

	if err := s.docRepo.CreateDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("create document record: %w", err)
	}

	// 3. 更新状态为 processing
	if err := s.docRepo.UpdateDocumentStatus(ctx, doc.ID, "processing", ""); err != nil {
		return nil, fmt.Errorf("update document status: %w", err)
	}

	// 4. 保存文件到本地
	filePath := filepath.Join(s.uploadDir, fmt.Sprintf("kb_%d_doc_%d_%s", knowledgeBaseID, doc.ID, fileName))
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		s.docRepo.UpdateDocumentStatus(ctx, doc.ID, "failed", fmt.Sprintf("create upload directory: %v", err))
		return nil, fmt.Errorf("create upload directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		s.docRepo.UpdateDocumentStatus(ctx, doc.ID, "failed", fmt.Sprintf("create file: %v", err))
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, fileContent); err != nil {
		s.docRepo.UpdateDocumentStatus(ctx, doc.ID, "failed", fmt.Sprintf("save file: %v", err))
		return nil, fmt.Errorf("save file: %w", err)
	}

	// 5. 更新文档记录的文件路径
	doc.FilePath = filePath
	if err := s.docRepo.UpdateDocument(ctx, doc); err != nil {
		s.logger.Warnf("update document file path failed: %v", err)
	}

	// 6. 异步处理文档
	go func() {
		processCtx := context.Background()
		if err = s.processDocument(processCtx, doc, kb); err != nil {
			s.logger.Errorf("process document %d failed: %v", doc.ID, err)
			s.docRepo.UpdateDocumentStatus(processCtx, doc.ID, "failed", err.Error())
		}
	}()

	return doc, nil
}

// processDocument 处理文档：解析、分块、向量化、存储
func (s *KnowledgeUsecase) processDocument(ctx context.Context, doc *Document, kb *KnowledgeBase) error {
	// 1. 加载文档
	chunkingConfig := loader.DefaultChunkingConfig()
	chunkingConfig.WithSemanticChunking(s.embedder)
	docs, err := loader.LoadDocuments(ctx, []string{doc.FilePath},
		loader.WithChunkSize(kb.ChunkSize),
		loader.WithChunkOverlap(kb.ChunkOverlap),
		loader.WithChunkingConfig(chunkingConfig),
	)
	if err != nil {
		return fmt.Errorf("load documents: %w", err)
	}

	if len(docs) == 0 {
		return fmt.Errorf("no documents loaded from file")
	}

	// 2. 添加知识库相关的元数据
	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = make(map[string]string)
		}
		docs[i].Metadata["knowledge_base_id"] = fmt.Sprintf("%d", kb.ID)
		docs[i].Metadata["knowledge_base_name"] = kb.Name
		docs[i].Metadata["document_id"] = fmt.Sprintf("%d", doc.ID)
		docs[i].Metadata["document_name"] = doc.Name
		docs[i].Metadata["file_type"] = doc.FileType
	}

	// 4. 创建 vector store
	var store vectordb.VectorStore
	if kb.VectorStoreType == "milvus" {
		// 解析 Milvus 配置
		var milvusConfig map[string]interface{}
		if kb.VectorStoreConfig != "" && kb.VectorStoreConfig != "{}" {
			if err := json.Unmarshal([]byte(kb.VectorStoreConfig), &milvusConfig); err != nil {
				return fmt.Errorf("parse milvus config: %w", err)
			}
		}
		collectionName := fmt.Sprintf("kb_%d", kb.ID)
		if name, ok := milvusConfig["collection"].(string); ok && name != "" {
			collectionName = name
		}
		milvusCfg := vectordb.DefaultMilvusConfig(collectionName, s.embedder.Dimensions())
		if host, ok := milvusConfig["host"].(string); ok {
			milvusCfg.Host = host
		}
		if port, ok := milvusConfig["port"].(float64); ok {
			milvusCfg.Port = int(port)
		}
		var err error
		store, err = vectordb.NewMilvusStore(ctx, milvusCfg)
		if err != nil {
			return fmt.Errorf("create milvus store: %w", err)
		}
	} else {
		// 默认使用内存存储
		store = vectordb.NewInMemoryStore(s.embedder.Dimensions())
	}

	// 5. 创建质量过滤器
	qualityFilter := quality.NewDocumentFilter(quality.DefaultFilterConfig())

	// 6. 创建 ingest config
	ingestConfig := vectordb.DefaultIngestConfig(s.embedder, store)
	ingestConfig.QualityFilter = qualityFilter

	// 7. 执行文档入库
	result, err := vectordb.IngestDocuments(ctx, docs, ingestConfig)
	if err != nil {
		return fmt.Errorf("ingest documents: %w", err)
	}

	// 8. 更新文档状态和元数据
	now := time.Now()
	metadata := map[string]interface{}{
		"chunk_count":  result.Vectors,
		"total_docs":   result.TotalDocs,
		"success_docs": result.Success,
		"failed_docs":  result.Failed,
		"processed_at": now.Format(time.RFC3339),
	}
	if result.FilterStats != nil {
		metadata["filter_stats"] = result.FilterStats
	}
	metadataJSON, _ := json.Marshal(metadata)

	doc.Status = "completed"
	doc.ChunkCount = result.Vectors
	doc.ProcessedAt = &now
	doc.Metadata = string(metadataJSON)

	if err := s.docRepo.UpdateDocument(ctx, doc); err != nil {
		return fmt.Errorf("update document: %w", err)
	}

	s.logger.Infof("Document %d processed successfully: %d chunks ingested", doc.ID, result.Vectors)
	return nil
}

// detectFileType 根据文件名检测文件类型
func (s *KnowledgeUsecase) detectFileType(fileName string) string {
	ext := filepath.Ext(fileName)
	switch ext {
	case ".pdf":
		return "pdf"
	case ".txt", ".text":
		return "txt"
	case ".html", ".htm":
		return "html"
	case ".md", ".markdown":
		return "markdown"
	case ".xlsx", ".xls":
		return "excel"
	case ".csv":
		return "csv"
	case ".docx", ".doc":
		return "docx"
	case ".json":
		return "json"
	default:
		return "unknown"
	}
}
