package loader

import (
	"context"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

type excelLoader struct{}

func init() {
	registerLoader(excelLoader{})
}

func (excelLoader) Extensions() []string {
	return []string{".xlsx", ".xlsm", ".xls"}
}

func (excelLoader) Load(ctx context.Context, path string, opts Options) ([]Document, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open excel %s: %w", path, err)
	}
	defer file.Close()

	var documents []Document
	sheets := file.GetSheetList()
	for _, sheet := range sheets {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		rows, err := file.GetRows(sheet)
		if err != nil {
			return nil, fmt.Errorf("read sheet %s: %w", sheet, err)
		}
		if len(rows) == 0 {
			continue
		}
		var builder strings.Builder
		for rowIdx, row := range rows {
			builder.WriteString(strings.Join(row, "\t"))
			if rowIdx != len(rows)-1 {
				builder.WriteRune('\n')
			}
		}
		text := normalizeWhitespace(builder.String())
		chunks, err := chunkContent(ctx, text, opts)
		if err != nil {
			return nil, fmt.Errorf("chunk text: %w", err)
		}
		meta := map[string]string{
			"source_type":  "excel",
			"source_sheet": sheet,
		}
		documents = append(documents, buildDocuments(path, chunks, opts, meta)...)
	}
	return documents, nil
}
