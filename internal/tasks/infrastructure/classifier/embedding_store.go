package classifier

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// EmbeddingEntry stores a cached embedding vector with its source text hash for invalidation
type EmbeddingEntry struct {
	Vector   []float64 `json:"vector"`
	TextHash string    `json:"text_hash"`
}

// EmbeddingStore manages cached embedding vectors on disk
type EmbeddingStore struct {
	path    string
	entries map[string]*EmbeddingEntry
	mu      sync.RWMutex
	dirty   bool
}

// NewEmbeddingStore creates or loads an embedding store from the given file path
func NewEmbeddingStore(path string) (*EmbeddingStore, error) {
	store := &EmbeddingStore{
		path:    path,
		entries: make(map[string]*EmbeddingEntry),
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

// load reads the store from disk, ignoring file-not-found errors
func (s *EmbeddingStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read embedding store: %w", err)
	}

	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, &s.entries); err != nil {
		return fmt.Errorf("failed to parse embedding store: %w", err)
	}

	return nil
}

// Save persists the store to disk
func (s *EmbeddingStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.dirty {
		return nil
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create embedding store directory: %w", err)
	}

	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal embedding store: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write embedding store: %w", err)
	}

	s.dirty = false
	return nil
}

// Get returns the cached embedding for a key if it exists and the text hash matches
func (s *EmbeddingStore) Get(key, textHash string) ([]float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.entries[key]
	if !exists || entry.TextHash != textHash {
		return nil, false
	}

	return entry.Vector, true
}

// Set stores an embedding vector for a key with its text hash
func (s *EmbeddingStore) Set(key string, vector []float64, textHash string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[key] = &EmbeddingEntry{
		Vector:   vector,
		TextHash: textHash,
	}
	s.dirty = true
}

// Len returns the number of cached entries
func (s *EmbeddingStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// HashText produces a SHA-256 hex digest for cache invalidation
func HashText(text string) string {
	h := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", h)
}
