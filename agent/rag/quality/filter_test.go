package quality

import (
	"jas-agent/agent/rag/loader"
	"testing"
)

func TestDocumentFilter(t *testing.T) {
	config := DefaultFilterConfig()
	filter := NewDocumentFilter(config)

	docs := []loader.Document{
		{ID: "1", Text: "This is a valid document with enough content to pass the filter."},
		{ID: "2", Text: "Short"}, // 太短
		{ID: "3", Text: ""},      // 空文档
		{ID: "4", Text: "This is another valid document with sufficient content."},
		{ID: "5", Text: "This is a valid document with enough content to pass the filter."}, // 重复
	}

	valid, result := filter.FilterDocuments(docs)

	t.Logf("Total: %d, Filtered: %d, Valid: %d, Duplicates: %d",
		result.Total, result.Filtered, result.Valid, result.Duplicates)

	if result.Valid != 2 {
		t.Errorf("Expected 2 valid documents, got %d", result.Valid)
	}

	if result.Duplicates == 0 && config.DedupByHash {
		t.Error("Expected duplicates to be detected")
	}

	if len(valid) != 2 {
		t.Errorf("Expected 2 valid documents, got %d", len(valid))
	}
}

func TestScoreDocument(t *testing.T) {
	tests := []struct {
		name     string
		doc      loader.Document
		minScore float64
	}{
		{
			name:     "good document",
			doc:      loader.Document{ID: "1", Text: "This is a well-formed document with proper punctuation and sufficient length to score well."},
			minScore: 0.7,
		},
		{
			name:     "short document",
			doc:      loader.Document{ID: "2", Text: "Short"},
			minScore: 0.3, // 短文档得分应该较低
		},
		{
			name:     "empty document",
			doc:      loader.Document{ID: "3", Text: ""},
			minScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := ScoreDocument(tt.doc)
			t.Logf("Document score: %.3f", score)
			if score < tt.minScore {
				t.Errorf("Expected score >= %.2f, got %.2f", tt.minScore, score)
			}
		})
	}
}

func TestCleanDocument(t *testing.T) {
	config := DefaultFilterConfig()
	filter := NewDocumentFilter(config)

	doc := loader.Document{
		ID:   "1",
		Text: "This   has    extra    spaces   and\n\n\n\nmultiple\n\nnewlines",
	}

	cleaned := filter.cleanDocument(doc)
	t.Logf("Original: %q", doc.Text)
	t.Logf("Cleaned:  %q", cleaned.Text)

	if len(cleaned.Text) > len(doc.Text) {
		t.Error("Cleaned text should not be longer than original")
	}
}
