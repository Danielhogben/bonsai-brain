// Package embed provides lightweight, dependency-free text embedders for
// the vector store. These are not state-of-the-art — they are tiny, fast,
// and require zero external models or network access.
//
// Use these for embedded deployments where you cannot load a 100 MB
// sentence-transformer model. For production quality, swap in a real
// embedder that calls an API or a local ONNX model.
package embed

import (
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

// ---------------------------------------------------------------------------
// HashEmbedder — deterministic, zero-allocation-ish, 128-dim
// ---------------------------------------------------------------------------

// HashEmbedder creates fixed-size vectors by hashing character n-grams.
// It is deterministic, thread-safe, and has no dependencies.
// Dimension is always 128.
type HashEmbedder struct {
	dim int
}

// NewHashEmbedder creates a HashEmbedder. The dim parameter is ignored
// (always 128) but accepted for API compatibility.
func NewHashEmbedder(dim int) *HashEmbedder {
	return &HashEmbedder{dim: 128}
}

// Embed returns a 128-dim float32 vector for the given text.
func (e *HashEmbedder) Embed(text string) ([]float32, error) {
	vec := make([]float32, e.dim)
	// Normalize: lowercase, strip extra spaces.
	text = strings.ToLower(text)
	// Character bi-gram hashing.
	for i := 0; i < len(text)-1; i++ {
		a, b := text[i], text[i+1]
		if !isRelevant(a) || !isRelevant(b) {
			continue
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte{a, b})
		idx := int(h.Sum32()) % e.dim
		vec[idx]++
	}
	// L2-normalize.
	norm := float32(0)
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		nf := float32(math.Sqrt(float64(norm)))
		for i := range vec {
			vec[i] /= nf
		}
	}
	return vec, nil
}

func isRelevant(b byte) bool {
	r := rune(b)
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// ---------------------------------------------------------------------------
// TFIDFEmbedder — term-frequency based, vocabulary required
// ---------------------------------------------------------------------------

// TFIDFEmbedder builds vectors from a fixed vocabulary. Each dimension
// corresponds to one vocabulary term; the value is the term's normalized
// frequency in the input text.
type TFIDFEmbedder struct {
	vocab map[string]int // term -> dimension index
}

// NewTFIDFEmbedder creates an embedder from a vocabulary slice.
func NewTFIDFEmbedder(vocab []string) *TFIDFEmbedder {
	m := make(map[string]int, len(vocab))
	for i, v := range vocab {
		m[strings.ToLower(v)] = i
	}
	return &TFIDFEmbedder{vocab: m}
}

// Embed returns a term-frequency vector.
func (e *TFIDFEmbedder) Embed(text string) ([]float32, error) {
	vec := make([]float32, len(e.vocab))
	terms := tokenize(text)
	for _, t := range terms {
		if idx, ok := e.vocab[t]; ok {
			vec[idx]++
		}
	}
	// L2-normalize.
	norm := float32(0)
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		nf := float32(math.Sqrt(float64(norm)))
		for i := range vec {
			vec[i] /= nf
		}
	}
	return vec, nil
}

func tokenize(text string) []string {
	var terms []string
	var buf strings.Builder
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf.WriteRune(r)
		} else if buf.Len() > 0 {
			terms = append(terms, buf.String())
			buf.Reset()
		}
	}
	if buf.Len() > 0 {
		terms = append(terms, buf.String())
	}
	return terms
}
