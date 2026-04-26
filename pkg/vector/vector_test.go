package vector

import (
	"math"
	"testing"
)

type mockEmbedder struct{}

func (m *mockEmbedder) Embed(text string) ([]float32, error) {
	// Deterministic fake embeddings: each char contributes to dimensions.
	vec := make([]float32, 4)
	for i, c := range text {
		vec[i%4] += float32(c)
	}
	return vec, nil
}

func TestStore_AddAndSearch(t *testing.T) {
	store := NewStore(&mockEmbedder{})

	_ = store.AddText("doc1", "hello world", nil)
	_ = store.AddText("doc2", "goodbye world", nil)
	_ = store.AddText("doc3", "foo bar baz", nil)

	if store.Count() != 3 {
		t.Fatalf("expected 3 docs, got %d", store.Count())
	}

	results, err := store.SearchText("hello", 2)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Top result should have reasonable similarity.
	if results[0].Similarity < 0 || results[0].Similarity > 1.0001 {
		t.Fatalf("similarity out of range: %f", results[0].Similarity)
	}
	// All results should be sorted descending.
	for i := 1; i < len(results); i++ {
		if results[i].Similarity > results[i-1].Similarity {
			t.Fatalf("results not sorted descending at index %d", i)
		}
	}
}

func TestStore_Delete(t *testing.T) {
	store := NewStore(&mockEmbedder{})
	_ = store.AddText("a", "alpha", nil)

	if !store.Delete("a") {
		t.Fatal("expected delete to return true")
	}
	if store.Count() != 0 {
		t.Fatalf("expected 0 docs after delete, got %d", store.Count())
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	if math.Abs(cosineSimilarity(a, b)-1.0) > 0.0001 {
		t.Fatalf("expected 1.0 for identical vectors, got %f", cosineSimilarity(a, b))
	}

	c := []float32{0, 1, 0}
	if math.Abs(cosineSimilarity(a, c)) > 0.0001 {
		t.Fatalf("expected 0.0 for orthogonal vectors, got %f", cosineSimilarity(a, c))
	}
}
