package classifier

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/llama"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// OllamaEmbeddingAdapter adapts the llama.Client to the ports.EmbeddingService interface
type OllamaEmbeddingAdapter struct {
	client *llama.Client
	model  string
}

// NewOllamaEmbeddingAdapter creates an adapter that implements EmbeddingService using Ollama
func NewOllamaEmbeddingAdapter(client *llama.Client, model string) ports.EmbeddingService {
	return &OllamaEmbeddingAdapter{
		client: client,
		model:  model,
	}
}

// Embed generates embedding vectors using Ollama's /api/embed endpoint
func (a *OllamaEmbeddingAdapter) Embed(texts []string) ([][]float64, error) {
	return a.client.GenerateEmbeddings(a.model, texts)
}
