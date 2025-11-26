package loader

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"
)

type pdfLoader struct{}

func init() {
	registerLoader(pdfLoader{})
}

func (pdfLoader) Extensions() []string {
	return []string{".pdf"}
}

func (pdfLoader) Load(ctx context.Context, path string, opts Options) ([]Document, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open pdf %s: %w", path, err)
	}
	defer file.Close()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	plain, err := reader.GetPlainText()
	if err != nil {
		return nil, fmt.Errorf("extract pdf text %s: %w", path, err)
	}
	var buf bytes.Buffer
	if _, err = io.Copy(&buf, plain); err != nil {
		return nil, fmt.Errorf("read pdf text %s: %w", path, err)
	}
	text := normalizeWhitespace(buf.String())
	chunks, err := chunkContent(ctx, text, opts)
	if err != nil {
		return nil, fmt.Errorf("chunk text: %w", err)
	}
	return buildDocuments(path, chunks, opts, map[string]string{
		"source_type": "pdf",
	}), nil
}
