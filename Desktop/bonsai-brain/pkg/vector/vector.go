// Package vector provides a lightweight, in-process vector store for RAG
// (Retrieval-Augmented Generation). No external database required — everything
// lives in Go slices and maps. Documents are stored as raw text alongside
// pre-computed embedding vectors. Search uses cosine similarity.
//
// This is intentionally simple: add documents, search by vector, get top-K
// results. For production scale, swap this out for a dedicated vector DB.
package vector

import (
	"fmt"
	"math"
	"sort"
	"sync"
)

// ---------------------------------------------------------------------------
// Document
// ---------------------------------------------------------------------------

// Document is a chunk of text with its embedding vector and metadata.
type Document struct {
	ID       string
	Text     string
	Vector   []float32
	Metadata map[string]any
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

// Store is an in-memory vector database.
type Store struct {
	mu        sync.RWMutex
	docs      map[string]Document
	dim       int // expected embedding dimension; 0 = unrestricted
	embedder  Embedder
}

// Embedder turns text into a float32 embedding vector.
// Implementations can call a local model, an API, or use a simple
// bag-of-words / TF-IDF fallback.
type Embedder interface {
	Embed(text string) ([]float32, error)
}

// NewStore creates an empty vector store.
func NewStore(embedder Embedder) *Store {
	return &Store{
		docs:     make(map[string]Document),
		embedder: embedder,
	}
}

// Add inserts or overwrites a document. If doc.Vector is nil, the store
// calls its embedder to generate one.
func (s *Store) Add(doc Document) error {
	if doc.Vector == nil && s.embedder != nil {
		vec, err := s.embedder.Embed(doc.Text)
		if err != nil {
			return fmt.Errorf("vector: embed failed for doc %q: %w", doc.ID, err)
		}
		doc.Vector = vec
	}
	if len(doc.Vector) == 0 {
		return fmt.Errorf("vector: doc %q has empty vector and no embedder", doc.ID)
	}
	if s.dim == 0 {
		s.dim = len(doc.Vector)
	} else if len(doc.Vector) != s.dim {
		return fmt.Errorf("vector: dimension mismatch for doc %q: got %d, want %d", doc.ID, len(doc.Vector), s.dim)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[doc.ID] = doc
	return nil
}

// AddText is a convenience wrapper that creates a Document from plain text.
func (s *Store) AddText(id, text string, metadata map[string]any) error {
	return s.Add(Document{ID: id, Text: text, Metadata: metadata})
}

// Delete removes a document by ID. Returns true if it existed.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.docs[id]
	if ok {
		delete(s.docs, id)
	}
	return ok
}

// Get retrieves a document by ID.
func (s *Store) Get(id string) (Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.docs[id]
	return d, ok
}

// Count returns the number of stored documents.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.docs)
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

// Result pairs a document with its similarity score.
type Result struct {
	Document   Document
	Similarity float64 // cosine similarity, 0.0–1.0
}

// Search finds the top-K most similar documents to the given query vector.
func (s *Store) Search(query []float32, topK int) []Result {
	if topK <= 0 {
		topK = 3
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []Result
	for _, doc := range s.docs {
		sim := cosineSimilarity(query, doc.Vector)
		results = append(results, Result{Document: doc, Similarity: sim})
	}

	// Sort descending by similarity.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > topK {
		results = results[:topK]
	}
	return results
}

// SearchText embeds the query text then performs a vector search.
func (s *Store) SearchText(queryText string, topK int) ([]Result, error) {
	if s.embedder == nil {
		return nil, fmt.Errorf("vector: no embedder configured")
	}
	vec, err := s.embedder.Embed(queryText)
	if err != nil {
		return nil, fmt.Errorf("vector: embed query failed: %w", err)
	}
	return s.Search(vec, topK), nil
}

// ---------------------------------------------------------------------------
// Cosine similarity
// ---------------------------------------------------------------------------

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		x := float64(a[i])
		y := float64(b[i])
		dot += x * y
		normA += x * x
		normB += y * y
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
