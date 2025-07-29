package usecase

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

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
	repository    ports.AssetRepository
	fieldsService ports.FieldsEnrichmentService
	validFields   map[string]bool
}

// NewBulkEnrichFieldsUseCase creates a new bulk enrich fields use case
func NewBulkEnrichFieldsUseCase(repository ports.AssetRepository, fieldsService ports.FieldsEnrichmentService) *BulkEnrichFieldsUseCase {
	return &BulkEnrichFieldsUseCase{
		repository:    repository,
		fieldsService: fieldsService,
		validFields: map[string]bool{
			"description": true,
			"why":         true,
			"benefits":    true,
			"how":         true,
			"metrics":     true,
		},
	}
}

// Execute performs bulk field enrichment
func (uc *BulkEnrichFieldsUseCase) Execute(ctx context.Context, input BulkEnrichFieldsInput) (*BulkEnrichFieldsResult, error) {
	startTime := time.Now()

	// Validate input fields
	if err := uc.validateFields(input.Fields); err != nil {
		return nil, err
	}

	result := &BulkEnrichFieldsResult{
		FailedAssets:   make(map[string]string),
		FieldsEnriched: make(map[string]map[string]bool),
	}

	// Get assets to process
	assets, err := uc.getAssetsToProcess(input)
	if err != nil {
		return nil, fmt.Errorf("failed to get assets: %w", err)
	}

	// Apply filtering
	filteredAssets := uc.applyFilter(assets, input.FilterBy, input.Fields)

	if len(filteredAssets) == 0 {
		log.Printf("No assets matched the filter criteria")
		result.Duration = time.Since(startTime)
		return result, nil
	}

	log.Printf("Processing %d assets for field enrichment...", len(filteredAssets))

	// Process assets concurrently
	uc.processAssetsConcurrently(ctx, filteredAssets, input, result)

	result.Duration = time.Since(startTime)
	return result, nil
}

// validateFields validates that all specified fields are valid
func (uc *BulkEnrichFieldsUseCase) validateFields(fields []string) error {
	if len(fields) == 0 {
		return fmt.Errorf("at least one field must be specified")
	}

	for _, field := range fields {
		if !uc.validFields[field] {
			return fmt.Errorf("invalid field '%s'. Valid fields: description, why, benefits, how, metrics", field)
		}
	}

	return nil
}

// getAssetsToProcess retrieves the assets that need to be processed
func (uc *BulkEnrichFieldsUseCase) getAssetsToProcess(input BulkEnrichFieldsInput) ([]*domain.Asset, error) {
	if len(input.AssetNames) > 0 {
		// Process specific assets
		var assets []*domain.Asset
		for _, name := range input.AssetNames {
			asset, err := uc.repository.FindByName(name)
			if err != nil {
				log.Printf("Asset '%s' not found: %v", name, err)
				continue
			}
			assets = append(assets, asset)
		}
		return assets, nil
	}

	// Process all assets
	return uc.repository.FindAll()
}

// applyFilter applies filtering logic to assets
func (uc *BulkEnrichFieldsUseCase) applyFilter(assets []*domain.Asset, filterBy string, fields []string) []*domain.Asset {
	if filterBy == "" || filterBy == "all" {
		return assets
	}

	var filtered []*domain.Asset
	for _, asset := range assets {
		switch filterBy {
		case "missing-fields", "empty-fields":
			if uc.hasEmptyFields(asset, fields) {
				filtered = append(filtered, asset)
			}
		case "empty-description":
			// Specific filter for empty description field
			if uc.isFieldEmpty(asset, "description") {
				filtered = append(filtered, asset)
			}
		default:
			// Unknown filter, return all assets
			return assets
		}
	}

	return filtered
}

// hasEmptyFields checks if any of the specified fields are empty for an asset
func (uc *BulkEnrichFieldsUseCase) hasEmptyFields(asset *domain.Asset, fields []string) bool {
	for _, field := range fields {
		if uc.isFieldEmpty(asset, field) {
			return true
		}
	}
	return false
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

// processAssetsConcurrently processes assets concurrently with semaphore control
func (uc *BulkEnrichFieldsUseCase) processAssetsConcurrently(ctx context.Context, assets []*domain.Asset, input BulkEnrichFieldsInput, result *BulkEnrichFieldsResult) {
	maxConcurrent := input.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 5 // Default concurrency
	}

	sem := semaphore.NewWeighted(int64(maxConcurrent))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, asset := range assets {
		wg.Add(1)
		go func(asset *domain.Asset) {
			defer wg.Done()

			if err := sem.Acquire(ctx, 1); err != nil {
				mu.Lock()
				result.FailedAssets[asset.Name] = fmt.Sprintf("failed to acquire semaphore: %v", err)
				result.TotalFailed++
				mu.Unlock()
				return
			}
			defer sem.Release(1)

			mu.Lock()
			result.ProcessedAssets = append(result.ProcessedAssets, asset.Name)
			result.TotalProcessed++
			result.FieldsEnriched[asset.Name] = make(map[string]bool)
			mu.Unlock()

			if input.DryRun {
				log.Printf("DRY RUN: Would enrich fields %v for asset '%s'", input.Fields, asset.Name)
				mu.Lock()
				result.SuccessfulAssets = append(result.SuccessfulAssets, asset.Name)
				result.TotalSuccessful++
				for _, field := range input.Fields {
					result.FieldsEnriched[asset.Name][field] = true
				}
				result.TotalFieldsEnriched += len(input.Fields)
				mu.Unlock()
				return
			}

			enrichedCount, err := uc.processAsset(ctx, asset, input.Fields)
			if err != nil {
				mu.Lock()
				result.FailedAssets[asset.Name] = err.Error()
				result.TotalFailed++
				mu.Unlock()
				return
			}

			mu.Lock()
			result.SuccessfulAssets = append(result.SuccessfulAssets, asset.Name)
			result.TotalSuccessful++
			result.TotalFieldsEnriched += enrichedCount
			mu.Unlock()
		}(asset)
	}

	wg.Wait()
}

// processAsset processes a single asset for field enrichment
func (uc *BulkEnrichFieldsUseCase) processAsset(ctx context.Context, asset *domain.Asset, fields []string) (int, error) {
	hasChanges := false
	enrichedCount := 0
	processingAttempts := 0
	errors := []string{}

	// Enrich each field
	for _, field := range fields {
		// Skip if field already has content
		if !uc.isFieldEmpty(asset, field) {
			log.Printf("Skipping field '%s' for asset '%s' - already has content", field, asset.Name)
			continue
		}

		processingAttempts++

		// Generate content for this field using the domain service
		enrichedContent, err := uc.fieldsService.EnrichField(ctx, asset, field, "")
		if err != nil {
			log.Printf("Failed to enrich field '%s' for asset '%s': %v", field, asset.Name, err)
			errors = append(errors, fmt.Sprintf("field '%s': %v", field, err))
			continue
		}

		// Update the asset field
		if err := uc.updateAssetField(asset, field, enrichedContent); err != nil {
			log.Printf("Failed to update field '%s' for asset '%s': %v", field, asset.Name, err)
			errors = append(errors, fmt.Sprintf("field '%s' update: %v", field, err))
			continue
		}

		hasChanges = true
		enrichedCount++
		log.Printf("Successfully enriched field '%s' for asset '%s'", field, asset.Name)
	}

	// If we attempted to process fields but all failed, return an error
	if processingAttempts > 0 && enrichedCount == 0 {
		return 0, fmt.Errorf("all field enrichment attempts failed: %s", strings.Join(errors, "; "))
	}

	// Save the asset if any changes were made
	if hasChanges {
		asset.UpdatedAt = time.Now()
		if err := uc.repository.Save(asset); err != nil {
			return 0, fmt.Errorf("failed to save asset: %w", err)
		}
	}

	return enrichedCount, nil
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
