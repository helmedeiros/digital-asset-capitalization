package classifier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddingStore_NewEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "embeddings.json")
	store, err := NewEmbeddingStore(path)
	require.NoError(t, err)
	assert.Equal(t, 0, store.Len())
}

func TestEmbeddingStore_SetAndGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "embeddings.json")
	store, err := NewEmbeddingStore(path)
	require.NoError(t, err)

	vector := []float64{0.1, 0.2, 0.3}
	hash := HashText("test text")

	store.Set("asset-1", vector, hash)

	got, ok := store.Get("asset-1", hash)
	assert.True(t, ok)
	assert.Equal(t, vector, got)
	assert.Equal(t, 1, store.Len())
}

func TestEmbeddingStore_GetMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "embeddings.json")
	store, err := NewEmbeddingStore(path)
	require.NoError(t, err)

	_, ok := store.Get("nonexistent", "hash")
	assert.False(t, ok)
}

func TestEmbeddingStore_GetStaleHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "embeddings.json")
	store, err := NewEmbeddingStore(path)
	require.NoError(t, err)

	store.Set("asset-1", []float64{0.1}, HashText("old text"))

	_, ok := store.Get("asset-1", HashText("new text"))
	assert.False(t, ok, "should not return entry with mismatched hash")
}

func TestEmbeddingStore_SaveAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "embeddings.json")

	// Create and populate store
	store, err := NewEmbeddingStore(path)
	require.NoError(t, err)

	hash := HashText("my asset text")
	store.Set("asset-1", []float64{1.0, 2.0, 3.0}, hash)
	store.Set("asset-2", []float64{4.0, 5.0, 6.0}, HashText("other text"))

	err = store.Save()
	require.NoError(t, err)

	// Reload from disk
	store2, err := NewEmbeddingStore(path)
	require.NoError(t, err)
	assert.Equal(t, 2, store2.Len())

	got, ok := store2.Get("asset-1", hash)
	assert.True(t, ok)
	assert.Equal(t, []float64{1.0, 2.0, 3.0}, got)
}

func TestEmbeddingStore_SaveSkipsWhenNotDirty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "embeddings.json")

	store, err := NewEmbeddingStore(path)
	require.NoError(t, err)

	// Save without setting anything - file should not be created
	err = store.Save()
	require.NoError(t, err)

	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err), "file should not be created when not dirty")
}

func TestEmbeddingStore_SaveCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	path := filepath.Join(dir, "embeddings.json")

	store, err := NewEmbeddingStore(path)
	require.NoError(t, err)

	store.Set("key", []float64{1.0}, "hash")
	err = store.Save()
	require.NoError(t, err)

	_, err = os.Stat(path)
	assert.NoError(t, err, "file should exist after save")
}

func TestHashText(t *testing.T) {
	hash1 := HashText("hello")
	hash2 := HashText("hello")
	hash3 := HashText("world")

	assert.Equal(t, hash1, hash2, "same input should produce same hash")
	assert.NotEqual(t, hash1, hash3, "different input should produce different hash")
	assert.Len(t, hash1, 64, "SHA-256 hex should be 64 chars")
}

func TestEmbeddingStore_CorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "embeddings.json")
	err := os.WriteFile(path, []byte("not valid json"), 0o644)
	require.NoError(t, err)

	_, err = NewEmbeddingStore(path)
	assert.Error(t, err, "should error on corrupt file")
}
