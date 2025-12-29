package loader

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"

	"jas-agent/agent/rag/embedding"
)

// ChunkingStrategy 分块策略类型
type ChunkingStrategy int

const (
	// ChunkingStrategyFixed 固定大小分块（按字符数）
	ChunkingStrategyFixed ChunkingStrategy = iota
	// ChunkingStrategySemantic 语义分块（基于嵌入模型）
	ChunkingStrategySemantic
	// ChunkingStrategyRecursive 递归分块（先按段落，再按句子）
	ChunkingStrategyRecursive
)

// ChunkingConfig 分块配置
type ChunkingConfig struct {
	Strategy  ChunkingStrategy   // 分块策略
	ChunkSize int                // 块大小（字符数）
	Overlap   int                // 重叠大小（字符数）
	Embedder  embedding.Embedder // 嵌入模型（用于语义分块）

	// 多粒度配置
	MultiGranularity bool // 是否启用多粒度分块
	CoarseSize       int  // 粗粒度块大小（如整个章节）
	FineSize         int  // 细粒度块大小（如单个段落）
}

// DefaultChunkingConfig 返回默认分块配置
func DefaultChunkingConfig() *ChunkingConfig {
	return &ChunkingConfig{
		Strategy:         ChunkingStrategyFixed,
		ChunkSize:        800,
		Overlap:          120,
		MultiGranularity: false,
		CoarseSize:       2000,
		FineSize:         400,
	}
}

// WithSemanticChunking 使用语义分块
func (c *ChunkingConfig) WithSemanticChunking(embedder embedding.Embedder) *ChunkingConfig {
	c.Strategy = ChunkingStrategySemantic
	c.Embedder = embedder
	return c
}

// WithMultiGranularity 启用多粒度分块
func (c *ChunkingConfig) WithMultiGranularity() *ChunkingConfig {
	c.MultiGranularity = true
	return c
}

// Chunker 分块器
type Chunker struct {
	config *ChunkingConfig
}

// NewChunker 创建分块器
func NewChunker(config *ChunkingConfig) *Chunker {
	if config == nil {
		config = DefaultChunkingConfig()
	}
	return &Chunker{
		config: config,
	}
}

// ChunkText 对文本进行分块
func (c *Chunker) ChunkText(ctx context.Context, text string) ([]string, error) {
	if c.config.MultiGranularity {
		return c.chunkMultiGranularity(ctx, text)
	}

	switch c.config.Strategy {
	case ChunkingStrategySemantic:
		if c.config.Embedder == nil {
			return nil, fmt.Errorf("embedder is required for semantic chunking")
		}
		return c.chunkSemantic(ctx, text)
	case ChunkingStrategyRecursive:
		return c.chunkRecursive(ctx, text), nil
	default:
		return chunkText(text, c.config.ChunkSize, c.config.Overlap), nil
	}
}

// chunkSemantic 语义分块（基于嵌入相似度）
func (c *Chunker) chunkSemantic(ctx context.Context, text string) ([]string, error) {
	// 首先按段落分割
	paragraphs := splitIntoParagraphs(text)
	if len(paragraphs) == 0 {
		return nil, nil
	}

	if len(paragraphs) == 1 {
		// 只有一个段落，使用固定大小分块
		return chunkText(text, c.config.ChunkSize, c.config.Overlap), nil
	}

	// 为每个段落生成嵌入
	embeddings, err := c.config.Embedder.EmbedBatch(ctx, paragraphs)
	if err != nil {
		// 如果嵌入失败，回退到固定大小分块
		return chunkText(text, c.config.ChunkSize, c.config.Overlap), nil
	}

	if len(embeddings) != len(paragraphs) {
		return chunkText(text, c.config.ChunkSize, c.config.Overlap), nil
	}

	// 基于嵌入相似度合并段落
	chunks := mergeParagraphsBySimilarity(paragraphs, embeddings, c.config.ChunkSize, c.config.Overlap)
	return chunks, nil
}

// chunkRecursive 递归分块
func (c *Chunker) chunkRecursive(ctx context.Context, text string) []string {
	// 首先按段落分割
	paragraphs := splitIntoParagraphs(text)

	chunks := make([]string, 0)
	for _, para := range paragraphs {
		// 如果段落小于块大小，直接作为一个块
		if len([]rune(para)) <= c.config.ChunkSize {
			chunks = append(chunks, para)
			continue
		}

		// 否则按句子分割后再分块
		sentences := splitIntoSentences(para)
		currentChunk := ""

		for _, sentence := range sentences {
			sentence = strings.TrimSpace(sentence)
			if sentence == "" {
				continue
			}

			// 如果当前块加上新句子不超过大小限制，则合并
			if len([]rune(currentChunk))+len([]rune(sentence)) <= c.config.ChunkSize {
				if currentChunk != "" {
					currentChunk += " " + sentence
				} else {
					currentChunk = sentence
				}
			} else {
				// 保存当前块，开始新块
				if currentChunk != "" {
					chunks = append(chunks, currentChunk)
				}
				currentChunk = sentence
			}
		}

		// 保存最后一个块
		if currentChunk != "" {
			chunks = append(chunks, currentChunk)
		}
	}

	return chunks
}

// chunkMultiGranularity 多粒度分块
func (c *Chunker) chunkMultiGranularity(ctx context.Context, text string) ([]string, error) {
	// 生成粗粒度块（大块）
	coarseChunks := chunkText(text, c.config.CoarseSize, c.config.Overlap)

	// 生成细粒度块（小块）
	fineChunks := make([]string, 0)
	for _, coarseChunk := range coarseChunks {
		chunks := chunkText(coarseChunk, c.config.FineSize, c.config.Overlap)
		fineChunks = append(fineChunks, chunks...)
	}

	// 合并粗粒度和细粒度块
	allChunks := make([]string, 0, len(coarseChunks)+len(fineChunks))
	allChunks = append(allChunks, coarseChunks...)
	allChunks = append(allChunks, fineChunks...)

	return allChunks, nil
}

// splitIntoParagraphs 将文本分割为段落
func splitIntoParagraphs(text string) []string {
	// 按双换行符分割段落
	parts := strings.Split(text, "\n\n")
	paragraphs := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			paragraphs = append(paragraphs, part)
		}
	}

	// 如果没有双换行符，按单换行符分割
	if len(paragraphs) <= 1 {
		lines := strings.Split(text, "\n")
		paragraphs = make([]string, 0, len(lines))
		currentPara := ""

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				if currentPara != "" {
					paragraphs = append(paragraphs, currentPara)
					currentPara = ""
				}
			} else {
				if currentPara != "" {
					currentPara += " " + line
				} else {
					currentPara = line
				}
			}
		}

		if currentPara != "" {
			paragraphs = append(paragraphs, currentPara)
		}
	}

	return paragraphs
}

// splitIntoSentences 将文本分割为句子
func splitIntoSentences(text string) []string {
	// 使用句子结束符分割
	re := regexp.MustCompile(`([。！？.!?]+)`)
	parts := re.Split(text, -1)
	separators := re.FindAllString(text, -1)

	sentences := make([]string, 0)
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 添加标点符号
		if i < len(separators) {
			part += separators[i]
		}

		sentences = append(sentences, part)
	}

	return sentences
}

// mergeParagraphsBySimilarity 基于相似度合并段落
func mergeParagraphsBySimilarity(paragraphs []string, embeddings [][]float32, maxSize, overlap int) []string {
	if len(paragraphs) == 0 {
		return nil
	}

	chunks := make([]string, 0)
	currentChunk := paragraphs[0]
	currentSize := len([]rune(paragraphs[0]))
	currentEmbedding := embeddings[0]

	for i := 1; i < len(paragraphs); i++ {
		para := paragraphs[i]
		paraSize := len([]rune(para))
		paraEmbedding := embeddings[i]

		// 计算相似度
		similarity := cosineSimilarity(currentEmbedding, paraEmbedding)

		// 如果相似度高且合并后不超过大小限制，则合并
		if similarity > 0.85 && currentSize+paraSize <= maxSize {
			currentChunk += "\n\n" + para
			currentSize += paraSize
			// 更新嵌入为平均值
			currentEmbedding = averageEmbedding(currentEmbedding, paraEmbedding)
		} else {
			// 保存当前块，开始新块
			chunks = append(chunks, currentChunk)
			currentChunk = para
			currentSize = paraSize
			currentEmbedding = paraEmbedding
		}
	}

	// 保存最后一个块
	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct float64
	var normA, normB float64

	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// averageEmbedding 计算两个嵌入的平均值
func averageEmbedding(a, b []float32) []float32 {
	if len(a) != len(b) {
		return a
	}

	result := make([]float32, len(a))
	for i := range a {
		result[i] = (a[i] + b[i]) / 2.0
	}

	return result
}
