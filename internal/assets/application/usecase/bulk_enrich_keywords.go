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
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/keywords"
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
	repository  ports.AssetRepository
	llamaClient application.LlamaClient
}

// NewBulkEnrichKeywordsUseCase creates a new bulk enrich keywords use case
func NewBulkEnrichKeywordsUseCase(repository ports.AssetRepository, llamaClient application.LlamaClient) *BulkEnrichKeywordsUseCase {
	return &BulkEnrichKeywordsUseCase{
		repository:  repository,
		llamaClient: llamaClient,
	}
}

// Execute performs bulk keywords enrichment
func (uc *BulkEnrichKeywordsUseCase) Execute(ctx context.Context, input BulkEnrichKeywordsInput) (*BulkEnrichKeywordsResult, error) {
	startTime := time.Now()

	// Set defaults
	if input.MaxConcurrent <= 0 {
		input.MaxConcurrent = 3 // Conservative default to avoid overwhelming LLM service
	}

	// Get assets to process
	assetsToProcess, err := uc.getAssetsToProcess(input)
	if err != nil {
		return nil, fmt.Errorf("failed to get assets to process: %w", err)
	}

	if len(assetsToProcess) == 0 {
		return &BulkEnrichKeywordsResult{
			TotalProcessed: 0,
			Duration:       time.Since(startTime),
		}, nil
	}

	log.Printf("Starting bulk keywords enrichment for %d assets (max concurrent: %d, dry-run: %v)",
		len(assetsToProcess), input.MaxConcurrent, input.DryRun)

	result := &BulkEnrichKeywordsResult{
		FailedAssets: make(map[string]string),
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
			err := uc.processAsset(ctx, a, input.DryRun)

			// Update results
			mu.Lock()
			result.ProcessedAssets = append(result.ProcessedAssets, a.Name)
			if err != nil {
				result.FailedAssets[a.Name] = err.Error()
				result.TotalFailed++
			} else {
				result.SuccessfulAssets = append(result.SuccessfulAssets, a.Name)
				result.TotalSuccessful++
			}
			mu.Unlock()
		}(asset)
	}

	wg.Wait()

	result.TotalProcessed = len(assetsToProcess)
	result.Duration = time.Since(startTime)

	log.Printf("Bulk keywords enrichment completed: %d processed, %d successful, %d failed in %v",
		result.TotalProcessed, result.TotalSuccessful, result.TotalFailed, result.Duration)

	return result, nil
}

// getAssetsToProcess determines which assets need keywords enrichment
func (uc *BulkEnrichKeywordsUseCase) getAssetsToProcess(input BulkEnrichKeywordsInput) ([]*domain.Asset, error) {
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
	case "missing-keywords":
		for _, asset := range allAssets {
			if len(asset.Keywords) == 0 {
				assets = append(assets, asset)
			}
		}
	case "outdated":
		// Consider keywords outdated if asset was updated after keywords were last generated
		// or if keywords are older than 30 days
		cutoff := time.Now().AddDate(0, 0, -30)
		for _, asset := range allAssets {
			if len(asset.Keywords) == 0 || asset.UpdatedAt.After(cutoff) {
				assets = append(assets, asset)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported filter: %s. Supported filters: all, missing-keywords, outdated", input.FilterBy)
	}

	return assets, nil
}

// processAsset generates keywords for a single asset
func (uc *BulkEnrichKeywordsUseCase) processAsset(_ context.Context, asset *domain.Asset, dryRun bool) error {
	if dryRun {
		log.Printf("DRY RUN: Would generate keywords for asset '%s'", asset.Name)
		return nil
	}

	log.Printf("Generating keywords for asset '%s'...", asset.Name)

	// Create keyword generator and generate keywords
	generator := keywords.NewGenerator(uc.llamaClient)
	keywordList, err := generator.GenerateKeywords(asset)
	if err != nil {
		return fmt.Errorf("failed to generate keywords: %w", err)
	}

	// Update asset with new keywords
	asset.Keywords = keywordList
	asset.UpdatedAt = time.Now()

	// Save the updated asset
	if err := uc.repository.Save(asset); err != nil {
		return fmt.Errorf("failed to save asset: %w", err)
	}

	log.Printf("Successfully generated %d keywords for asset '%s'", len(keywordList), asset.Name)
	return nil
}
