package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"jas-agent/agent/rag/embedding"
	"jas-agent/agent/rag/loader"
	"jas-agent/agent/rag/quality"
	"jas-agent/agent/rag/vectordb"
	"jas-agent/internal/biz"
	"jas-agent/internal/conf"
)

// KnowledgeService 知识库服务
type KnowledgeService struct {
	knowledgeUsecase *biz.KnowledgeUsecase
	llmConfig        *conf.LLM
	uploadDir        string
}

// NewKnowledgeService 创建知识库服务
func NewKnowledgeService(knowledgeUsecase *biz.KnowledgeUsecase, llmConfig *conf.LLM) *KnowledgeService {
	return &KnowledgeService{
		knowledgeUsecase: knowledgeUsecase,
		llmConfig:        llmConfig,
		uploadDir:        "./uploads", // 默认上传目录
	}
}

// UploadDocument 上传并处理文档
func (s *KnowledgeService) UploadDocument(
	ctx context.Context,
	knowledgeBaseID int,
	fileName string,
	fileType string,
	fileReader io.Reader,
) (*biz.Document, error) {
	// 1. 创建文档记录
	doc := &biz.Document{
		KnowledgeBaseID: knowledgeBaseID,
		Name:            fileName,
		FileType:        fileType,
		Status:          "pending",
		Metadata:        "{}",
	}

	if err := s.knowledgeUsecase.CreateDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("create document record: %w", err)
	}

	// 2. 保存文件
	uploadPath := filepath.Join(s.uploadDir, fmt.Sprintf("%d_%s", doc.ID, fileName))
	if err := os.MkdirAll(filepath.Dir(uploadPath), 0755); err != nil {
		s.knowledgeUsecase.UpdateDocumentStatus(ctx, doc.ID, "failed", err.Error())
		return nil, fmt.Errorf("create upload directory: %w", err)
	}

	file, err := os.Create(uploadPath)
	if err != nil {
		s.knowledgeUsecase.UpdateDocumentStatus(ctx, doc.ID, "failed", err.Error())
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	fileSize, err := io.Copy(file, fileReader)
	if err != nil {
		s.knowledgeUsecase.UpdateDocumentStatus(ctx, doc.ID, "failed", err.Error())
		return nil, fmt.Errorf("save file: %w", err)
	}

	// 更新文件路径和大小
	doc.FilePath = uploadPath
	doc.FileSize = fileSize
	doc.Status = "processing"
	if err := s.knowledgeUsecase.UpdateDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("update document: %w", err)
	}

	// 3. 异步处理文档（解析和向量化）
	go s.processDocument(ctx, doc)

	return doc, nil
}

// processDocument 处理文档：解析、向量化、存储
func (s *KnowledgeService) processDocument(ctx context.Context, doc *biz.Document) {
	defer func() {
		if r := recover(); r != nil {
			s.knowledgeUsecase.UpdateDocumentStatus(ctx, doc.ID, "failed", fmt.Sprintf("panic: %v", r))
		}
	}()

	// 获取知识库配置
	kb, err := s.knowledgeUsecase.GetKnowledgeBase(ctx, doc.KnowledgeBaseID)
	if err != nil {
		s.knowledgeUsecase.UpdateDocumentStatus(ctx, doc.ID, "failed", fmt.Sprintf("get knowledge base: %v", err))
		return
	}

	// 1. 加载文档
	loaderOpts := []loader.Option{
		loader.WithChunkSize(kb.ChunkSize),
		loader.WithChunkOverlap(kb.ChunkOverlap),
	}

	docs, err := loader.LoadDocuments(ctx, []string{doc.FilePath}, loaderOpts...)
	if err != nil {
		s.knowledgeUsecase.UpdateDocumentStatus(ctx, doc.ID, "failed", fmt.Sprintf("load document: %v", err))
		return
	}

	// 2. 应用质量过滤
	filterConfig := quality.DefaultFilterConfig()
	filter := quality.NewDocumentFilter(filterConfig)
	validDocs, filterResult := filter.FilterDocuments(docs)

	// 3. 创建向量存储
	var store vectordb.VectorStore
	switch kb.VectorStoreType {
	case "milvus":
		// TODO: 实现 Milvus 存储
		fallthrough
	default:
		// 使用内存存储（需要知道维度）
		dimensions := 1536 // 默认维度
		if kb.EmbeddingModel == "text-embedding-3-large" {
			dimensions = 3072
		}
		store = vectordb.NewInMemoryStore(dimensions)
	}

	// 4. 创建嵌入生成器
	if s.llmConfig == nil || s.llmConfig.ApiKey == "" {
		s.knowledgeUsecase.UpdateDocumentStatus(ctx, doc.ID, "failed", "LLM API key not configured")
		return
	}

	embedder := embedding.NewOpenAIEmbedder(embedding.Config{
		ApiKey:  s.llmConfig.ApiKey,
		BaseURL: s.llmConfig.BaseUrl,
		Model:   kb.EmbeddingModel,
	})

	// 5. 配置摄入管道
	ingestConfig := vectordb.DefaultIngestConfig(embedder, store)
	ingestConfig.QualityFilter = filter
	ingestConfig.BatchSize = 50

	// 6. 摄入文档到向量数据库
	result, err := vectordb.IngestDocuments(ctx, validDocs, ingestConfig)
	if err != nil {
		s.knowledgeUsecase.UpdateDocumentStatus(ctx, doc.ID, "failed", fmt.Sprintf("ingest documents: %v", err))
		return
	}

	// 7. 更新文档状态
	doc.ChunkCount = result.Vectors
	doc.Status = "completed"
	now := time.Now()
	doc.ProcessedAt = &now

	// 保存元数据
	metadata := fmt.Sprintf(`{"filtered": %d, "valid": %d, "vectors": %d}`,
		filterResult.Filtered, filterResult.Valid, result.Vectors)
	doc.Metadata = metadata

	if err := s.knowledgeUsecase.UpdateDocument(ctx, doc); err != nil {
		// 即使更新失败，文档已经处理完成
		fmt.Printf("failed to update document status: %v\n", err)
	}
}
