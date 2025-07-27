package usecase

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// BulkEnrichFieldsInput represents the input for bulk field enrichment
type BulkEnrichFieldsInput struct {
	AssetNames    []string // Specific asset names to enrich (optional)
	Fields        []string // Fields to enrich: "description", "why", "benefits", "how", "metrics"
	FilterBy      string   // Filter criteria: "all", "missing-fields", "empty-fields"
	MaxConcurrent int      // Maximum concurrent enrichment operations
	DryRun        bool     // If true, only shows what would be enriched
}

// BulkEnrichFieldsResult represents the result of bulk field enrichment
type BulkEnrichFieldsResult struct {
	ProcessedAssets     []string                   // Assets that were processed
	SuccessfulAssets    []string                   // Assets that were successfully enriched
	FailedAssets        map[string]string          // Assets that failed with error messages
	SkippedAssets       []string                   // Assets that were skipped
	FieldsEnriched      map[string]map[string]bool // [assetName][field] = success
	TotalProcessed      int                        // Total number of assets processed
	TotalSuccessful     int                        // Total number of successful enrichments
	TotalFailed         int                        // Total number of failed enrichments
	TotalSkipped        int                        // Total number of skipped assets
	TotalFieldsEnriched int                        // Total number of fields enriched
	Duration            time.Duration              // Total time taken
}

// BulkEnrichFieldsUseCase handles bulk field enrichment for multiple assets
type BulkEnrichFieldsUseCase struct {
	repository  ports.AssetRepository
	llamaClient application.LlamaClient
}

// NewBulkEnrichFieldsUseCase creates a new bulk enrich fields use case
func NewBulkEnrichFieldsUseCase(repository ports.AssetRepository, llamaClient application.LlamaClient) *BulkEnrichFieldsUseCase {
	return &BulkEnrichFieldsUseCase{
		repository:  repository,
		llamaClient: llamaClient,
	}
}

// Execute performs bulk field enrichment
func (uc *BulkEnrichFieldsUseCase) Execute(ctx context.Context, input BulkEnrichFieldsInput) (*BulkEnrichFieldsResult, error) {
	startTime := time.Now()

	// Validate input
	if err := uc.validateInput(input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Set defaults
	if input.MaxConcurrent <= 0 {
		input.MaxConcurrent = 2 // Conservative for LLM calls which can be slow
	}

	// Get assets to process
	assetsToProcess, err := uc.getAssetsToProcess(input)
	if err != nil {
		return nil, fmt.Errorf("failed to get assets to process: %w", err)
	}

	if len(assetsToProcess) == 0 {
		return &BulkEnrichFieldsResult{
			TotalProcessed: 0,
			Duration:       time.Since(startTime),
			FailedAssets:   make(map[string]string),
			FieldsEnriched: make(map[string]map[string]bool),
		}, nil
	}

	log.Printf("Starting bulk field enrichment for %d assets, %d fields (max concurrent: %d, dry-run: %v)",
		len(assetsToProcess), len(input.Fields), input.MaxConcurrent, input.DryRun)

	result := &BulkEnrichFieldsResult{
		FailedAssets:   make(map[string]string),
		FieldsEnriched: make(map[string]map[string]bool),
	}

	// Process assets concurrently with limit
	sem := make(chan struct{}, input.MaxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, asset := range assetsToProcess {
		wg.Add(1)
		go func(a *domain.Asset) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Process asset
			fieldsEnriched, err := uc.processAsset(ctx, a, input.Fields, input.DryRun)

			// Update results
			mu.Lock()
			result.ProcessedAssets = append(result.ProcessedAssets, a.Name)
			result.FieldsEnriched[a.Name] = fieldsEnriched

			if err != nil {
				result.FailedAssets[a.Name] = err.Error()
				result.TotalFailed++
			} else {
				result.SuccessfulAssets = append(result.SuccessfulAssets, a.Name)
				result.TotalSuccessful++

				// Count successful field enrichments
				for _, success := range fieldsEnriched {
					if success {
						result.TotalFieldsEnriched++
					}
				}
			}
			mu.Unlock()
		}(asset)
	}

	wg.Wait()

	result.TotalProcessed = len(assetsToProcess)
	result.Duration = time.Since(startTime)

	log.Printf("Bulk field enrichment completed: %d processed, %d successful, %d failed, %d fields enriched in %v",
		result.TotalProcessed, result.TotalSuccessful, result.TotalFailed, result.TotalFieldsEnriched, result.Duration)

	return result, nil
}

// validateInput validates the input parameters
func (uc *BulkEnrichFieldsUseCase) validateInput(input BulkEnrichFieldsInput) error {
	if len(input.Fields) == 0 {
		return fmt.Errorf("at least one field must be specified")
	}

	validFields := map[string]bool{
		"description": true,
		"why":         true,
		"benefits":    true,
		"how":         true,
		"metrics":     true,
	}

	for _, field := range input.Fields {
		if !validFields[field] {
			return fmt.Errorf("invalid field '%s'. Valid fields: description, why, benefits, how, metrics", field)
		}
	}

	return nil
}

// getAssetsToProcess determines which assets need field enrichment
func (uc *BulkEnrichFieldsUseCase) getAssetsToProcess(input BulkEnrichFieldsInput) ([]*domain.Asset, error) {
	var assets []*domain.Asset

	// If specific asset names provided, get those
	if len(input.AssetNames) > 0 {
		for _, name := range input.AssetNames {
			asset, err := uc.repository.FindByName(name)
			if err != nil {
				return nil, fmt.Errorf("failed to find asset '%s': %w", name, err)
			}
			assets = append(assets, asset)
		}
		return assets, nil
	}

	// Otherwise get all assets and apply filtering
	allAssets, err := uc.repository.FindAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all assets: %w", err)
	}

	// Apply filtering based on FilterBy
	switch input.FilterBy {
	case "all", "":
		assets = allAssets
	case "missing-fields":
		// Assets that have any of the target fields missing/empty
		for _, asset := range allAssets {
			needsEnrichment := false
			for _, field := range input.Fields {
				if uc.isFieldEmpty(asset, field) {
					needsEnrichment = true
					break
				}
			}
			if needsEnrichment {
				assets = append(assets, asset)
			}
		}
	case "empty-fields":
		// Same as missing-fields for backward compatibility
		for _, asset := range allAssets {
			needsEnrichment := false
			for _, field := range input.Fields {
				if uc.isFieldEmpty(asset, field) {
					needsEnrichment = true
					break
				}
			}
			if needsEnrichment {
				assets = append(assets, asset)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported filter: %s. Supported filters: all, missing-fields, empty-fields", input.FilterBy)
	}

	return assets, nil
}

// isFieldEmpty checks if a specific field is empty for an asset
func (uc *BulkEnrichFieldsUseCase) isFieldEmpty(asset *domain.Asset, field string) bool {
	switch field {
	case "description":
		return asset.Description == ""
	case "why":
		return asset.Why == ""
	case "benefits":
		return asset.Benefits == ""
	case "how":
		return asset.How == ""
	case "metrics":
		return asset.Metrics == ""
	default:
		return false
	}
}

// processAsset enriches specified fields for a single asset
func (uc *BulkEnrichFieldsUseCase) processAsset(ctx context.Context, asset *domain.Asset, fields []string, dryRun bool) (map[string]bool, error) {
	fieldsEnriched := make(map[string]bool)

	if dryRun {
		log.Printf("DRY RUN: Would enrich fields %v for asset '%s'", fields, asset.Name)
		for _, field := range fields {
			fieldsEnriched[field] = true // Simulate success
		}
		return fieldsEnriched, nil
	}

	log.Printf("Enriching fields %v for asset '%s'...", fields, asset.Name)

	hasChanges := false

	// Enrich each field
	for _, field := range fields {
		// Skip if field already has content (unless we want to re-enrich)
		if !uc.isFieldEmpty(asset, field) {
			log.Printf("Skipping field '%s' for asset '%s' - already has content", field, asset.Name)
			fieldsEnriched[field] = false
			continue
		}

		// Generate content for this field
		content, err := uc.generateFieldContent(ctx, asset, field)
		if err != nil {
			log.Printf("Failed to enrich field '%s' for asset '%s': %v", field, asset.Name, err)
			fieldsEnriched[field] = false
			continue
		}

		// Update the asset field
		if err := uc.updateAssetField(asset, field, content); err != nil {
			log.Printf("Failed to update field '%s' for asset '%s': %v", field, asset.Name, err)
			fieldsEnriched[field] = false
			continue
		}

		fieldsEnriched[field] = true
		hasChanges = true
		log.Printf("Successfully enriched field '%s' for asset '%s'", field, asset.Name)
	}

	// Save the asset if any changes were made
	if hasChanges {
		asset.UpdatedAt = time.Now()
		if err := uc.repository.Save(asset); err != nil {
			return fieldsEnriched, fmt.Errorf("failed to save asset: %w", err)
		}
	}

	return fieldsEnriched, nil
}

// generateFieldContent generates content for a specific field using LLM
func (uc *BulkEnrichFieldsUseCase) generateFieldContent(_ context.Context, asset *domain.Asset, field string) (string, error) {
	// Create content from existing asset information
	existingContent := fmt.Sprintf("Asset: %s\nDescription: %s\nWhy: %s\nBenefits: %s\nHow: %s\nMetrics: %s",
		asset.Name, asset.Description, asset.Why, asset.Benefits, asset.How, asset.Metrics)

	// Generate enriched content using LLM
	enrichedContent, err := uc.llamaClient.EnrichContent(existingContent, field, asset)
	if err != nil {
		return "", fmt.Errorf("LLM enrichment failed: %w", err)
	}

	return enrichedContent, nil
}

// updateAssetField updates a specific field in the asset
func (uc *BulkEnrichFieldsUseCase) updateAssetField(asset *domain.Asset, field, content string) error {
	switch field {
	case "description":
		asset.Description = content
	case "why":
		asset.Why = content
	case "benefits":
		asset.Benefits = content
	case "how":
		asset.How = content
	case "metrics":
		asset.Metrics = content
	default:
		return fmt.Errorf("unknown field: %s", field)
	}

	return nil
}
