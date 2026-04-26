package embed

import (
	"math"
	"testing"
)

func TestHashEmbedder(t *testing.T) {
	e := NewHashEmbedder(128)

	v1, err := e.Embed("hello world")
	if err != nil {
		t.Fatalf("embed failed: %v", err)
	}
	if len(v1) != 128 {
		t.Fatalf("expected 128 dims, got %d", len(v1))
	}

	v2, _ := e.Embed("hello world")
	// Deterministic
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("embeddings not deterministic at index %d", i)
		}
	}

	// L2-normalized
	norm := float64(0)
	for _, v := range v1 {
		norm += float64(v) * float64(v)
	}
	if math.Abs(norm-1.0) > 0.01 && norm != 0 {
		t.Fatalf("expected ~L2 norm 1.0, got %f", norm)
	}
}

func TestHashEmbedder_Similarity(t *testing.T) {
	e := NewHashEmbedder(128)
	v1, _ := e.Embed("bonsai brain go agent")
	v2, _ := e.Embed("bonsai brain python agent")
	v3, _ := e.Embed("completely unrelated text about cars")

	sim12 := cosine(v1, v2)
	sim13 := cosine(v1, v3)

	if sim12 < sim13 {
		t.Fatalf("expected similar texts to have higher similarity: sim12=%f sim13=%f", sim12, sim13)
	}
}

func TestTFIDFEmbedder(t *testing.T) {
	vocab := []string{"bonsai", "brain", "agent", "go", "python"}
	e := NewTFIDFEmbedder(vocab)

	v, err := e.Embed("bonsai brain is written in go")
	if err != nil {
		t.Fatalf("embed failed: %v", err)
	}
	if len(v) != len(vocab) {
		t.Fatalf("expected %d dims, got %d", len(vocab), len(v))
	}
	if v[0] == 0 || v[1] == 0 || v[3] == 0 {
		t.Fatal("expected non-zero values for present terms")
	}
	if v[4] != 0 {
		t.Fatal("expected zero for absent term 'python'")
	}
}

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
