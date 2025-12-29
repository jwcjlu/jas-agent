package graphrag

import "testing"

func TestTokenizeAndSimilarity(t *testing.T) {
	textA := "GraphRAG builds a knowledge graph for retrieval augmented generation."
	textB := "Knowledge graph retrieval helps GraphRAG answer complex questions."

	tokensA := tokenize(textA)
	tokensB := tokenize(textB)
	if len(tokensA) == 0 || len(tokensB) == 0 {
		t.Fatalf("tokenize failed")
	}

	score := semanticSimilarity(tokensA, tokensB)
	if score <= 0 {
		t.Fatalf("expected positive similarity, got %f", score)
	}
}
