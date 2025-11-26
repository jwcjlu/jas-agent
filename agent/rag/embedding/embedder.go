package embedding

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// Embedder 生成文本向量的接口
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	Dimensions() int
}

// Config embedding 配置
type Config struct {
	ApiKey  string
	BaseURL string
	Model   string
}

// DefaultConfig 返回默认配置
func DefaultConfig(apiKey string) Config {
	return Config{
		ApiKey:  apiKey,
		BaseURL: "https://api.openai.com/v1",
		Model:   "text-embedding-3-small", // 1536 维度
	}
}

// openaiEmbedder OpenAI embedding 实现
type openaiEmbedder struct {
	client *openai.Client
	model  string
}

// NewOpenAIEmbedder 创建 OpenAI embedder
func NewOpenAIEmbedder(cfg Config) Embedder {
	config := openai.DefaultConfig(cfg.ApiKey)
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	}
	return &openaiEmbedder{
		client: openai.NewClientWithConfig(config),
		model:  cfg.Model,
	}
}

func (e *openaiEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	model := openai.SmallEmbedding3
	if e.model != "" {
		model = openai.EmbeddingModel(e.model)
	}

	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: model,
	}

	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	return resp.Data[0].Embedding, nil
}

func (e *openaiEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	model := openai.SmallEmbedding3
	if e.model != "" {
		model = openai.EmbeddingModel(e.model)
	}

	req := openai.EmbeddingRequest{
		Input: texts,
		Model: model,
	}

	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create batch embeddings: %w", err)
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

func (e *openaiEmbedder) Dimensions() int {
	// text-embedding-3-small 是 1536 维度
	if e.model == "text-embedding-3-small" {
		return 1024
	}
	if e.model == "text-embedding-3-large" {
		return 1024
	}
	// text-embedding-ada-002 是 1536 维度
	return 1024
}
