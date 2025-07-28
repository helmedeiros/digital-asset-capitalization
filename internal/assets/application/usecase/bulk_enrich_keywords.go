package usecase

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// BulkEnrichKeywordsInput represents the input for bulk keywords enrichment
type BulkEnrichKeywordsInput struct {
	AssetNames    []string // Specific asset names to enrich (optional)
	FilterBy      string   // Filter criteria: "all", "missing-keywords", "outdated"
	MaxConcurrent int      // Maximum concurrent enrichment operations
	DryRun        bool     // If true, only shows what would be enriched
}

// BulkEnrichKeywordsResult represents the result of bulk keywords enrichment
type BulkEnrichKeywordsResult struct {
	ProcessedAssets  []string          // Assets that were processed
	SuccessfulAssets []string          // Assets that were successfully enriched
	FailedAssets     map[string]string // Assets that failed with error messages
	SkippedAssets    []string          // Assets that were skipped
	TotalProcessed   int               // Total number of assets processed
	TotalSuccessful  int               // Total number of successful enrichments
	TotalFailed      int               // Total number of failed enrichments
	TotalSkipped     int               // Total number of skipped assets
	Duration         time.Duration     // Total time taken
}

// BulkEnrichKeywordsUseCase handles bulk keywords generation for multiple assets
type BulkEnrichKeywordsUseCase struct {
	repository      ports.AssetRepository
	keywordsService ports.KeywordsEnrichmentService
}

// NewBulkEnrichKeywordsUseCase creates a new bulk enrich keywords use case
func NewBulkEnrichKeywordsUseCase(repository ports.AssetRepository, keywordsService ports.KeywordsEnrichmentService) *BulkEnrichKeywordsUseCase {
	return &BulkEnrichKeywordsUseCase{
		repository:      repository,
		keywordsService: keywordsService,
	}
}

// Execute performs bulk keywords enrichment
func (uc *BulkEnrichKeywordsUseCase) Execute(ctx context.Context, input BulkEnrichKeywordsInput) (*BulkEnrichKeywordsResult, error) {
	startTime := time.Now()

	result := &BulkEnrichKeywordsResult{
		FailedAssets: make(map[string]string),
	}

	// Get assets to process
	assets, err := uc.getAssetsToProcess(input)
	if err != nil {
		return nil, fmt.Errorf("failed to get assets: %w", err)
	}

	// Apply filtering
	filteredAssets := uc.applyFilter(assets, input.FilterBy)

	if len(filteredAssets) == 0 {
		log.Printf("No assets matched the filter criteria")
		result.Duration = time.Since(startTime)
		return result, nil
	}

	log.Printf("Processing %d assets for keywords enrichment...", len(filteredAssets))

	// Process assets concurrently
	uc.processAssetsConcurrently(ctx, filteredAssets, input, result)

	result.Duration = time.Since(startTime)
	return result, nil
}

// getAssetsToProcess retrieves the assets that need to be processed
func (uc *BulkEnrichKeywordsUseCase) getAssetsToProcess(input BulkEnrichKeywordsInput) ([]*domain.Asset, error) {
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
func (uc *BulkEnrichKeywordsUseCase) applyFilter(assets []*domain.Asset, filterBy string) []*domain.Asset {
	if filterBy == "" || filterBy == "all" {
		return assets
	}

	var filtered []*domain.Asset
	for _, asset := range assets {
		switch filterBy {
		case "missing-keywords":
			if len(asset.Keywords) == 0 {
				filtered = append(filtered, asset)
			}
		case "outdated":
			// Simple heuristic: if keywords exist but were created more than 30 days ago
			if len(asset.Keywords) > 0 && time.Since(asset.UpdatedAt) > 30*24*time.Hour {
				filtered = append(filtered, asset)
			}
		case "empty-description":
			// Specific filter for empty description field
			if asset.Description == "" {
				filtered = append(filtered, asset)
			}
		default:
			// Unknown filter, return all assets
			return assets
		}
	}

	return filtered
}

// processAssetsConcurrently processes assets concurrently with semaphore control
func (uc *BulkEnrichKeywordsUseCase) processAssetsConcurrently(ctx context.Context, assets []*domain.Asset, input BulkEnrichKeywordsInput, result *BulkEnrichKeywordsResult) {
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
			mu.Unlock()

			if input.DryRun {
				log.Printf("DRY RUN: Would generate keywords for asset '%s'", asset.Name)
				mu.Lock()
				result.SuccessfulAssets = append(result.SuccessfulAssets, asset.Name)
				result.TotalSuccessful++
				mu.Unlock()
				return
			}

			if err := uc.processAsset(ctx, asset); err != nil {
				mu.Lock()
				result.FailedAssets[asset.Name] = err.Error()
				result.TotalFailed++
				mu.Unlock()
				return
			}

			mu.Lock()
			result.SuccessfulAssets = append(result.SuccessfulAssets, asset.Name)
			result.TotalSuccessful++
			mu.Unlock()
		}(asset)
	}

	wg.Wait()
}

// processAsset processes a single asset for keywords enrichment
func (uc *BulkEnrichKeywordsUseCase) processAsset(ctx context.Context, asset *domain.Asset) error {
	// Generate keywords using the domain service
	keywords, err := uc.keywordsService.GenerateKeywords(ctx, asset)
	if err != nil {
		return fmt.Errorf("failed to generate keywords: %w", err)
	}

	// Update asset with new keywords
	asset.Keywords = keywords
	asset.UpdatedAt = time.Now()

	// Save the updated asset
	if err := uc.repository.Save(asset); err != nil {
		return fmt.Errorf("failed to save asset: %w", err)
	}

	return nil
}
