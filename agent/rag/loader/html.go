package loader

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type htmlLoader struct{}

func init() {
	registerLoader(htmlLoader{})
}

func (htmlLoader) Extensions() []string {
	return []string{".html", ".htm"}
}

func (htmlLoader) Load(ctx context.Context, path string, opts Options) ([]Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open html file %s: %w", path, err)
	}
	defer file.Close()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	doc, err := goquery.NewDocumentFromReader(file)
	if err != nil {
		return nil, fmt.Errorf("parse html %s: %w", path, err)
	}
	doc.Find("script, style, noscript").Each(func(i int, selection *goquery.Selection) {
		selection.Remove()
	})
	text := doc.Text()
	text = strings.TrimSpace(text)
	text = normalizeWhitespace(text)
	chunks, err := chunkContent(ctx, text, opts)
	if err != nil {
		return nil, fmt.Errorf("chunk text: %w", err)
	}
	return buildDocuments(path, chunks, opts, map[string]string{
		"source_type": "html",
	}), nil
}
