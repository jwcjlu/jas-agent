package loader

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestLoadDocumentsMixedSources(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(txtPath, []byte("Hello RAG!\nThis is a plain text file."), 0o644); err != nil {
		t.Fatalf("write txt: %v", err)
	}

	htmlPath := filepath.Join(dir, "index.html")
	html := `<html><head><title>Doc</title><script>const a = 1;</script></head><body><h1>GraphRAG</h1><p>HTML content.</p></body></html>`
	if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
		t.Fatalf("write html: %v", err)
	}

	excelPath := filepath.Join(dir, "data.xlsx")
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "Name")
	f.SetCellValue("Sheet1", "B1", "Score")
	f.SetCellValue("Sheet1", "A2", "Alice")
	f.SetCellValue("Sheet1", "B2", 95)
	if err := f.SaveAs(excelPath); err != nil {
		t.Fatalf("save excel: %v", err)
	}

	// 测试 Markdown
	mdPath := filepath.Join(dir, "readme.md")
	mdContent := "# Title\n\nThis is **markdown** content.\n\n- Item 1\n- Item 2"
	if err := os.WriteFile(mdPath, []byte(mdContent), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	// 测试 CSV
	csvPath := filepath.Join(dir, "data.csv")
	csvContent := "Name,Age,City\nAlice,30,Beijing\nBob,25,Shanghai"
	if err := os.WriteFile(csvPath, []byte(csvContent), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	// 测试 JSON
	jsonPath := filepath.Join(dir, "data.json")
	jsonContent := `{"name": "Test", "value": 123, "items": ["a", "b"]}`
	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0o644); err != nil {
		t.Fatalf("write json: %v", err)
	}

	ctx := context.Background()
	docs, err := LoadDocuments(ctx, []string{dir}, WithChunkSize(128))
	if err != nil {
		t.Fatalf("load documents: %v", err)
	}
	if len(docs) < 6 {
		t.Fatalf("expected >=6 documents, got %d", len(docs))
	}
	assertContainsText(t, docs, "Hello RAG!")
	assertContainsText(t, docs, "GraphRAG")
	assertContainsText(t, docs, "Alice")
	assertContainsText(t, docs, "markdown")
	assertContainsText(t, docs, "Beijing")
	assertContainsText(t, docs, "Test")
}

func assertContainsText(t *testing.T, docs []Document, needle string) {
	t.Helper()
	for _, doc := range docs {
		if strings.Contains(doc.Text, needle) {
			return
		}
	}
	t.Fatalf("expected document containing %q", needle)
}
