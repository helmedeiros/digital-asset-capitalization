package enrichment

import (
	"context"
	"fmt"
	"log"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/keywords"
)

// KeywordsEnrichmentAdapter implements the KeywordsEnrichmentService interface
type KeywordsEnrichmentAdapter struct {
	llamaClient application.LlamaClient
}

// NewKeywordsEnrichmentAdapter creates a new keywords enrichment adapter
func NewKeywordsEnrichmentAdapter(llamaClient application.LlamaClient) ports.KeywordsEnrichmentService {
	return &KeywordsEnrichmentAdapter{
		llamaClient: llamaClient,
	}
}

// GenerateKeywords generates keywords for an asset
func (a *KeywordsEnrichmentAdapter) GenerateKeywords(_ context.Context, asset *domain.Asset) ([]string, error) {
	log.Printf("Generating keywords for asset '%s'...", asset.Name)

	// Create keyword generator and generate keywords
	generator := keywords.NewGenerator(a.llamaClient)
	keywordList, err := generator.GenerateKeywords(asset)
	if err != nil {
		return nil, fmt.Errorf("failed to generate keywords: %w", err)
	}

	log.Printf("Successfully generated %d keywords for asset '%s'", len(keywordList), asset.Name)
	return keywordList, nil
}
