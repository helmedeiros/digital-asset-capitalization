package classifier

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/llama"
)

// TestOllamaEmbeddingAdapter_Embed_PassesThrough drives the
// previously-uncovered Embed method by pointing a real llama.Client
// at an httptest server. The adapter is a thin pass-through to
// GenerateEmbeddings so a single happy path is enough.
func TestOllamaEmbeddingAdapter_Embed_PassesThrough(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/embed", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"embeddings": [[0.1, 0.2], [0.3, 0.4]]}`))
	}))
	defer server.Close()

	client, err := llama.NewClient(llama.Config{BaseURL: server.URL, BackoffBase: time.Microsecond})
	require.NoError(t, err)

	adapter := NewOllamaEmbeddingAdapter(client, "nomic-embed-text")
	got, err := adapter.Embed([]string{"alpha", "beta"})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.InDelta(t, 0.1, got[0][0], 0.0001)
	assert.InDelta(t, 0.4, got[1][1], 0.0001)
}
