package enrichment

import (
	"context"
	"fmt"
	"log"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// FieldsEnrichmentAdapter implements the FieldsEnrichmentService interface
type FieldsEnrichmentAdapter struct {
	llamaClient application.LlamaClient
	validFields map[string]bool
}

// NewFieldsEnrichmentAdapter creates a new fields enrichment adapter
func NewFieldsEnrichmentAdapter(llamaClient application.LlamaClient) ports.FieldsEnrichmentService {
	return &FieldsEnrichmentAdapter{
		llamaClient: llamaClient,
		validFields: map[string]bool{
			"description": true,
			"why":         true,
			"benefits":    true,
			"how":         true,
			"metrics":     true,
		},
	}
}

// EnrichField enriches a specific field of an asset
func (a *FieldsEnrichmentAdapter) EnrichField(_ context.Context, asset *domain.Asset, field string, content string) (string, error) {
	// Validate field
	if !a.validFields[field] {
		return "", fmt.Errorf("invalid field '%s'. Valid fields: description, why, benefits, how, metrics", field)
	}

	log.Printf("Enriching field '%s' for asset '%s'...", field, asset.Name)

	// Create content from existing asset information if content is empty
	if content == "" {
		content = fmt.Sprintf("Asset: %s\nDescription: %s\nWhy: %s\nBenefits: %s\nHow: %s\nMetrics: %s",
			asset.Name, asset.Description, asset.Why, asset.Benefits, asset.How, asset.Metrics)
	}

	// Generate enriched content using LLM
	enrichedContent, err := a.llamaClient.EnrichContent(content, field, asset)
	if err != nil {
		return "", fmt.Errorf("LLM enrichment failed: %w", err)
	}

	log.Printf("Successfully enriched field '%s' for asset '%s'", field, asset.Name)
	return enrichedContent, nil
}
