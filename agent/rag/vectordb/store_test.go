package vectordb

import (
	"context"
	"jas-agent/agent/rag/loader"
	"testing"
)

func TestInMemoryStore(t *testing.T) {
	store := NewInMemoryStore(1536)
	ctx := context.Background()

	// 创建测试向量（简化版，实际应该是真实 embedding）
	vectors := []Vector{
		{
			ID: "doc-1",
			Document: &loader.Document{
				ID:   "doc-1",
				Text: "这是第一个文档",
				Metadata: map[string]string{
					"source": "test",
				},
			},
			Vector: make([]float32, 1536), // 简化：实际应该是真实 embedding
		},
		{
			ID: "doc-2",
			Document: &loader.Document{
				ID:   "doc-2",
				Text: "这是第二个文档",
				Metadata: map[string]string{
					"source": "test",
				},
			},
			Vector: make([]float32, 1536),
		},
	}

	// 设置一些非零向量值用于测试
	for i := 0; i < 10; i++ {
		vectors[0].Vector[i] = 0.1
		vectors[1].Vector[i] = 0.2
	}

	// 插入向量
	if err := store.Insert(ctx, vectors); err != nil {
		t.Fatalf("insert vectors: %v", err)
	}

	// 测试统计信息
	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if stats.Count != 2 {
		t.Errorf("expected 2 vectors, got %d", stats.Count)
	}

	// 测试搜索
	queryVector := make([]float32, 1536)
	for i := 0; i < 10; i++ {
		queryVector[i] = 0.15 // 介于两个向量之间
	}

	results, err := store.Search(ctx, queryVector, 2, nil)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// 验证搜索结果包含完整的元数据
	if len(results) > 0 {
		r := results[0]
		if r.ID == "" {
			t.Error("SearchResult.ID should not be empty")
		}
		if r.Text == "" {
			t.Error("SearchResult.Text should not be empty")
		}
		if r.Metadata == nil {
			t.Error("SearchResult.Metadata should not be nil")
		} else {
			if source, ok := r.Metadata["source"]; !ok || source == "" {
				t.Error("SearchResult.Metadata should contain 'source'")
			}
		}

		// 测试便捷方法
		if r.GetDocument() == nil {
			t.Error("GetDocument() should not return nil")
		}
		if r.GetMetadata("source") == "" {
			t.Error("GetMetadata('source') should not return empty")
		}
	}

	// 测试 GetByID
	doc, err := store.GetByID(ctx, "doc-1")
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if doc.ID != "doc-1" {
		t.Errorf("expected doc-1, got %s", doc.ID)
	}

	// 测试删除
	if err := store.Delete(ctx, []string{"doc-1"}); err != nil {
		t.Fatalf("delete: %v", err)
	}

	stats, _ = store.Stats(ctx)
	if stats.Count != 1 {
		t.Errorf("expected 1 vector after deletion, got %d", stats.Count)
	}
}
