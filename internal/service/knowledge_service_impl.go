package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"jas-agent/internal/biz"

	pb "jas-agent/api/agent/service/v1"
)

// KnowledgeServiceImpl 实现知识库服务
type KnowledgeServiceImpl struct {
	pb.UnimplementedKnowledgeServiceServer
	knowledgeUsecase *biz.KnowledgeUsecase
}

// NewKnowledgeServiceImpl 创建知识库服务实现
func NewKnowledgeServiceImpl(knowledgeUsecase *biz.KnowledgeUsecase) *KnowledgeServiceImpl {
	return &KnowledgeServiceImpl{
		knowledgeUsecase: knowledgeUsecase,
	}
}

// CreateKnowledgeBase 创建知识库
func (s *KnowledgeServiceImpl) CreateKnowledgeBase(ctx context.Context, req *pb.KnowledgeBaseRequest) (*pb.KnowledgeBaseResponse, error) {
	result := &pb.KnowledgeBaseResponse{
		Ret: &pb.BaseResponse{
			Code: 0,
		},
	}

	kb := &biz.KnowledgeBase{
		Name:              req.Name,
		Description:       req.Description,
		Tags:              req.Tags,
		EmbeddingModel:    req.EmbeddingModel,
		ChunkSize:         int(req.ChunkSize),
		ChunkOverlap:      int(req.ChunkOverlap),
		VectorStoreType:   req.VectorStoreType,
		VectorStoreConfig: req.VectorStoreConfig,
		IsActive:          req.IsActive,
	}

	if err := s.knowledgeUsecase.CreateKnowledgeBase(ctx, kb); err != nil {
		result.Ret.Code = 1
		result.Ret.Message = err.Error()
		return result, err
	}

	result.KnowledgeBase = knowledgeBaseToProto(kb)
	return result, nil
}

// UpdateKnowledgeBase 更新知识库
func (s *KnowledgeServiceImpl) UpdateKnowledgeBase(ctx context.Context, req *pb.KnowledgeBaseRequest) (*pb.KnowledgeBaseResponse, error) {
	result := &pb.KnowledgeBaseResponse{
		Ret: &pb.BaseResponse{
			Code: 0,
		},
	}

	tags := []string{}
	if req.Tags != nil {
		tags = req.Tags
	}
	kb := &biz.KnowledgeBase{
		ID:                int(req.Id),
		Name:              req.Name,
		Description:       req.Description,
		Tags:              tags,
		EmbeddingModel:    req.EmbeddingModel,
		ChunkSize:         int(req.ChunkSize),
		ChunkOverlap:      int(req.ChunkOverlap),
		VectorStoreType:   req.VectorStoreType,
		VectorStoreConfig: req.VectorStoreConfig,
		IsActive:          req.IsActive,
	}

	if err := s.knowledgeUsecase.UpdateKnowledgeBase(ctx, kb); err != nil {
		result.Ret.Code = 1
		result.Ret.Message = err.Error()
		return result, err
	}

	// 重新获取以获取最新数据
	updated, err := s.knowledgeUsecase.GetKnowledgeBase(ctx, kb.ID)
	if err != nil {
		result.Ret.Code = 1
		result.Ret.Message = err.Error()
		return result, err
	}

	result.KnowledgeBase = knowledgeBaseToProto(updated)
	return result, nil
}

// DeleteKnowledgeBase 删除知识库
func (s *KnowledgeServiceImpl) DeleteKnowledgeBase(ctx context.Context, req *pb.KnowledgeBaseDeleteRequest) (*pb.KnowledgeBaseResponse, error) {
	result := &pb.KnowledgeBaseResponse{
		Ret: &pb.BaseResponse{
			Code: 0,
		},
	}
	if err := s.knowledgeUsecase.DeleteKnowledgeBase(ctx, int(req.Id)); err != nil {
		result.Ret.Code = 1
		result.Ret.Message = err.Error()
		return result, err
	}
	return result, nil
}

// GetKnowledgeBase 获取知识库
func (s *KnowledgeServiceImpl) GetKnowledgeBase(ctx context.Context, req *pb.KnowledgeBaseGetRequest) (*pb.KnowledgeBaseResponse, error) {
	result := &pb.KnowledgeBaseResponse{
		Ret: &pb.BaseResponse{
			Code: 0,
		},
	}
	kb, err := s.knowledgeUsecase.GetKnowledgeBase(ctx, int(req.Id))
	if err != nil {
		result.Ret.Code = 1
		result.Ret.Message = err.Error()
		return result, err
	}
	result.KnowledgeBase = knowledgeBaseToProto(kb)
	return result, nil
}

// ListKnowledgeBases 列出知识库
func (s *KnowledgeServiceImpl) ListKnowledgeBases(ctx context.Context, req *pb.KnowledgeBaseListRequest) (*pb.KnowledgeBaseListResponse, error) {
	result := &pb.KnowledgeBaseListResponse{
		Ret: &pb.BaseResponse{
			Code: 0,
		},
	}

	searchQuery := ""
	if req.Search != "" {
		searchQuery = req.Search
	}

	tags := []string{}
	if req.Tags != nil {
		tags = req.Tags
	}

	kbs, err := s.knowledgeUsecase.ListKnowledgeBases(ctx, searchQuery, tags)
	if err != nil {
		result.Ret.Code = 1
		result.Ret.Message = err.Error()
		return result, err
	}

	result.KnowledgeBases = make([]*pb.KnowledgeBaseInfo, 0, len(kbs))
	for _, kb := range kbs {
		result.KnowledgeBases = append(result.KnowledgeBases, knowledgeBaseToProto(kb))
	}
	return result, nil
}

// ListDocuments 列出文档
func (s *KnowledgeServiceImpl) ListDocuments(ctx context.Context, req *pb.DocumentListRequest) (*pb.DocumentListResponse, error) {
	result := &pb.DocumentListResponse{
		Ret: &pb.BaseResponse{
			Code: 0,
		},
	}
	docs, err := s.knowledgeUsecase.ListDocuments(ctx, int(req.KnowledgeBaseId))
	if err != nil {
		result.Ret.Code = 1
		result.Ret.Message = err.Error()
		return result, err
	}

	result.Documents = make([]*pb.DocumentInfo, 0, len(docs))
	for _, doc := range docs {
		result.Documents = append(result.Documents, documentToProto(doc))
	}
	return result, nil
}

// GetDocument 获取文档
func (s *KnowledgeServiceImpl) GetDocument(ctx context.Context, req *pb.DocumentGetRequest) (*pb.DocumentResponse, error) {
	result := &pb.DocumentResponse{
		Ret: &pb.BaseResponse{
			Code: 0,
		},
	}
	doc, err := s.knowledgeUsecase.GetDocument(ctx, int(req.Id))
	if err != nil {
		result.Ret.Code = 1
		result.Ret.Message = err.Error()
		return result, err
	}
	result.Document = documentToProto(doc)
	return result, nil
}

// DeleteDocument 删除文档
func (s *KnowledgeServiceImpl) DeleteDocument(ctx context.Context, req *pb.DocumentDeleteRequest) (*pb.DocumentResponse, error) {
	result := &pb.DocumentResponse{
		Ret: &pb.BaseResponse{
			Code: 0,
		},
	}
	if err := s.knowledgeUsecase.DeleteDocument(ctx, int(req.Id)); err != nil {
		result.Ret.Code = 1
		result.Ret.Message = err.Error()
		return result, err
	}
	return result, nil
}

// 转换函数
func knowledgeBaseToProto(kb *biz.KnowledgeBase) *pb.KnowledgeBaseInfo {
	return &pb.KnowledgeBaseInfo{
		Id:                int32(kb.ID),
		Name:              kb.Name,
		Description:       kb.Description,
		Tags:              kb.Tags,
		EmbeddingModel:    kb.EmbeddingModel,
		ChunkSize:         int32(kb.ChunkSize),
		ChunkOverlap:      int32(kb.ChunkOverlap),
		VectorStoreType:   kb.VectorStoreType,
		VectorStoreConfig: kb.VectorStoreConfig,
		IsActive:          kb.IsActive,
		DocumentCount:     int32(kb.DocumentCount),
		CreatedAt:         kb.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:         kb.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func documentToProto(doc *biz.Document) *pb.DocumentInfo {
	processedAt := ""
	if doc.ProcessedAt != nil {
		processedAt = doc.ProcessedAt.Format("2006-01-02 15:04:05")
	}
	return &pb.DocumentInfo{
		Id:                 int32(doc.ID),
		KnowledgeBaseId:    int32(doc.KnowledgeBaseID),
		Name:               doc.Name,
		FilePath:           doc.FilePath,
		FileSize:           doc.FileSize,
		FileType:           doc.FileType,
		Status:             doc.Status,
		ChunkCount:         int32(doc.ChunkCount),
		ProcessedAt:        processedAt,
		ErrorMessage:       doc.ErrorMessage,
		Metadata:           doc.Metadata,
		CreatedAt:          doc.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:          doc.UpdatedAt.Format("2006-01-02 15:04:05"),
		EnableGraphExtract: doc.EnableGraphExtraction,
	}
}

// UploadDocument 处理文档上传（HTTP multipart/form-data）
func (s *KnowledgeServiceImpl) UploadDocument(w http.ResponseWriter, r *http.Request) {
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	path := r.URL.Path
	knowledgeBaseIDStr := ""
	if idx := len("/api/knowledge-bases/"); idx < len(path) {
		rest := path[idx:]
		for i, c := range rest {
			if c == '/' {
				knowledgeBaseIDStr = rest[:i]
				break
			}
			if i == len(rest)-1 {
				knowledgeBaseIDStr = rest
			}
		}
	}
	knowledgeBaseID, err := strconv.Atoi(knowledgeBaseIDStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid knowledge_base_id: %v", err))
		return
	}

	// 解析 multipart/form-data
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("parse multipart form: %v", err))
		return
	}

	// 获取文件
	file, header, err := r.FormFile("file")
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("get file: %v", err))
		return
	}
	defer file.Close()

	extractGraph := false
	if flag := strings.ToLower(r.FormValue("extractGraph")); flag != "" {
		extractGraph = flag == "true" || flag == "1" || flag == "on"
	}

	// 上传并处理文档
	doc, err := s.knowledgeUsecase.UploadAndProcessDocument(
		r.Context(),
		knowledgeBaseID,
		header.Filename,
		file,
		header.Size,
		extractGraph,
	)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("upload document: %v", err))
		return
	}

	// 返回成功响应
	response := &pb.DocumentResponse{
		Ret: &pb.BaseResponse{
			Code:    0,
			Message: "Document uploaded successfully",
		},
		Document: documentToProto(doc),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := &pb.DocumentResponse{
		Ret: &pb.BaseResponse{
			Code:    1,
			Message: message,
		},
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
