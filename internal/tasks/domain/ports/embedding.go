package ports

// EmbeddingService defines the interface for generating text embeddings
type EmbeddingService interface {
	// Embed generates embedding vectors for the given texts
	Embed(texts []string) ([][]float64, error)
}
