package loader

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type textLoader struct{}

func init() {
	registerLoader(textLoader{})
}

func (textLoader) Extensions() []string {
	return []string{".txt", ".log"}
}

func (textLoader) Load(ctx context.Context, path string, opts Options) ([]Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read text file %s: %w", path, err)
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	content := normalizeWhitespace(strings.ReplaceAll(string(data), "\r\n", "\n"))
	chunks, err := chunkContent(ctx, content, opts)
	if err != nil {
		return nil, fmt.Errorf("chunk text: %w", err)
	}

	return buildDocuments(path, chunks, opts, map[string]string{
		"source_type": "text",
	}), nil
}
