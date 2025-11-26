package loader

import (
	"context"
)

// chunkContent 通用的内容分块函数
func chunkContent(ctx context.Context, content string, opts Options) ([]string, error) {
	if opts.ChunkingConfig != nil {
		chunker := NewChunker(opts.ChunkingConfig)
		return chunker.ChunkText(ctx, content)
	}
	// 使用默认固定大小分块
	return chunkText(content, opts.ChunkSize, opts.ChunkOverlap), nil
}
