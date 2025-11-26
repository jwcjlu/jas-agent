package loader

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type markdownLoader struct{}

func init() {
	registerLoader(markdownLoader{})
}

func (markdownLoader) Extensions() []string {
	return []string{".md", ".markdown"}
}

func (markdownLoader) Load(ctx context.Context, path string, opts Options) ([]Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read markdown file %s: %w", path, err)
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	// Markdown 保留原始格式，只做基本的换行规范化
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	chunks, err := chunkContent(ctx, content, opts)
	if err != nil {
		return nil, fmt.Errorf("chunk text: %w", err)
	}
	return buildDocuments(path, chunks, opts, map[string]string{
		"source_type": "markdown",
	}), nil
}
