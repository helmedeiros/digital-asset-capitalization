package enrichment

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

// BulkEnrichmentAdapter implements bulk enrichment operations
type BulkEnrichmentAdapter struct {
	repository      ports.AssetRepository
	keywordsService ports.KeywordsEnrichmentService
	fieldsService   ports.FieldsEnrichmentService
}

// NewBulkEnrichmentAdapter creates a new bulk enrichment adapter
func NewBulkEnrichmentAdapter(
	repository ports.AssetRepository,
	keywordsService ports.KeywordsEnrichmentService,
	fieldsService ports.FieldsEnrichmentService,
) ports.BulkEnrichmentService {
	return &BulkEnrichmentAdapter{
		repository:      repository,
		keywordsService: keywordsService,
		fieldsService:   fieldsService,
	}
}

// ProcessAssets processes multiple assets with the given filter and configuration
func (a *BulkEnrichmentAdapter) ProcessAssets(ctx context.Context, input ports.BulkEnrichmentInput) (*ports.BulkEnrichmentResult, error) {
	result := ports.NewBulkEnrichmentResult()

	// Get assets to process
	var assets []*domain.Asset
	var err error

	if len(input.AssetNames) > 0 {
		// Process specific assets
		for _, name := range input.AssetNames {
			asset, err := a.repository.FindByName(name)
			if err != nil {
				result.AddFailedAsset(name, fmt.Sprintf("asset not found: %v", err))
				continue
			}
			assets = append(assets, asset)
		}
	} else {
		// Process all assets
		assets, err = a.repository.FindAll()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve assets: %w", err)
		}
	}

	// Apply filtering if specified
	filteredAssets := a.applyFilter(assets, input.FilterBy)

	if len(filteredAssets) == 0 {
		log.Printf("No assets matched the filter criteria")
		return result, nil
	}

	log.Printf("Processing %d assets...", len(filteredAssets))

	// Process assets concurrently
	sem := semaphore.NewWeighted(int64(input.MaxConcurrent))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, asset := range filteredAssets {
		wg.Add(1)
		go func(asset *domain.Asset) {
			defer wg.Done()

			if err := sem.Acquire(ctx, 1); err != nil {
				mu.Lock()
				result.AddFailedAsset(asset.Name, fmt.Sprintf("failed to acquire semaphore: %v", err))
				mu.Unlock()
				return
			}
			defer sem.Release(1)

			mu.Lock()
			result.AddProcessedAsset(asset.Name)
			mu.Unlock()

			if input.DryRun {
				log.Printf("DRY RUN: Would process asset '%s'", asset.Name)
				mu.Lock()
				result.AddSuccessfulAsset(asset.Name)
				mu.Unlock()
				return
			}

			// This is a simplified implementation - in practice, you'd specify
			// what type of enrichment to perform in the input
			a.processAsset(ctx, asset)

			mu.Lock()
			result.AddSuccessfulAsset(asset.Name)
			mu.Unlock()
		}(asset)
	}

	wg.Wait()
	return result, nil
}

// processAsset processes a single asset - simplified version
func (a *BulkEnrichmentAdapter) processAsset(_ context.Context, _ *domain.Asset) {
	// This is a placeholder implementation
	// In practice, you'd determine what type of enrichment to perform
	// based on the input parameters

	// For now, just return success to demonstrate the pattern
	time.Sleep(100 * time.Millisecond) // Simulate processing time
}

// applyFilter applies filtering logic to assets
func (a *BulkEnrichmentAdapter) applyFilter(assets []*domain.Asset, filterBy string) []*domain.Asset {
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
		case "empty-description":
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
