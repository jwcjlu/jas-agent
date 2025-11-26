package loader

import (
	"context"
	"fmt"
	"strings"

	"github.com/unidoc/unioffice/document"
)

type docxLoader struct{}

func init() {
	registerLoader(docxLoader{})
}

func (docxLoader) Extensions() []string {
	return []string{".docx"}
}

func (docxLoader) Load(ctx context.Context, path string, opts Options) ([]Document, error) {
	doc, err := document.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open docx file %s: %w", path, err)
	}
	defer doc.Close()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var builder strings.Builder
	for _, para := range doc.Paragraphs() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		// 提取段落中的所有文本运行
		var paraText strings.Builder
		for _, run := range para.Runs() {
			paraText.WriteString(run.Text())
		}
		text := paraText.String()
		if strings.TrimSpace(text) != "" {
			builder.WriteString(text)
			builder.WriteString("\n")
		}
	}

	// 提取表格内容
	for _, tbl := range doc.Tables() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		builder.WriteString("\n[表格]\n")
		for _, row := range tbl.Rows() {
			var cells []string
			for _, cell := range row.Cells() {
				var cellText strings.Builder
				for _, para := range cell.Paragraphs() {
					for _, run := range para.Runs() {
						cellText.WriteString(run.Text())
					}
				}
				cells = append(cells, strings.TrimSpace(cellText.String()))
			}
			if len(cells) > 0 {
				builder.WriteString(strings.Join(cells, " | "))
				builder.WriteString("\n")
			}
		}
	}

	content := normalizeWhitespace(builder.String())
	chunks, err := chunkContent(ctx, content, opts)
	if err != nil {
		return nil, fmt.Errorf("chunk text: %w", err)
	}
	return buildDocuments(path, chunks, opts, map[string]string{
		"source_type": "docx",
	}), nil
}
