package loader

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

type csvLoader struct{}

func init() {
	registerLoader(csvLoader{})
}

func (csvLoader) Extensions() []string {
	return []string{".csv"}
}

func (csvLoader) Load(ctx context.Context, path string, opts Options) ([]Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open csv file %s: %w", path, err)
	}
	defer file.Close()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv %s: %w", path, err)
	}

	if len(records) == 0 {
		return nil, nil
	}

	var builder strings.Builder
	// 第一行作为表头
	if len(records) > 0 {
		builder.WriteString("表头: ")
		builder.WriteString(strings.Join(records[0], " | "))
		builder.WriteString("\n\n")
	}

	// 数据行
	for i := 1; i < len(records); i++ {
		builder.WriteString(fmt.Sprintf("行 %d: ", i))
		builder.WriteString(strings.Join(records[i], " | "))
		if i < len(records)-1 {
			builder.WriteString("\n")
		}
	}

	content := normalizeWhitespace(builder.String())
	chunks, err := chunkContent(ctx, content, opts)
	if err != nil {
		return nil, fmt.Errorf("chunk text: %w", err)
	}
	return buildDocuments(path, chunks, opts, map[string]string{
		"source_type":  "csv",
		"row_count":    fmt.Sprintf("%d", len(records)),
		"column_count": fmt.Sprintf("%d", len(records[0])),
	}), nil
}
