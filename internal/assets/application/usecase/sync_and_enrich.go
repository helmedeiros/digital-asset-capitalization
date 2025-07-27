package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// SyncAndEnrichInput represents the input for the sync-and-enrich operation
type SyncAndEnrichInput struct {
	// Sync parameters
	SpaceKey string // Confluence space key (empty for all spaces)
	Label    string // Confluence label to filter by
	Debug    bool   // Enable debug output

	// Enrichment parameters
	EnrichKeywords bool     // Whether to generate keywords
	EnrichFields   []string // Fields to enrich: "description", "why", "benefits", "how", "metrics"
	FieldFilter    string   // Filter for field enrichment: "all", "missing-fields", "empty-fields"
	MaxConcurrent  int      // Maximum concurrent enrichment operations

	// General parameters
	DryRun bool // If true, only shows what would be done
}

// SyncAndEnrichResult represents the result of the sync-and-enrich operation
type SyncAndEnrichResult struct {
	// Sync results
	SyncedAssets []string          // Assets synced from Confluence
	SyncErrors   map[string]string // Sync errors by asset name
	TotalSynced  int               // Total assets synced

	// Keywords enrichment results
	KeywordsResult *BulkEnrichKeywordsResult // Keywords enrichment result

	// Fields enrichment results
	FieldsResult *BulkEnrichFieldsResult // Fields enrichment result

	// Overall results
	TotalProcessed  int           // Total assets processed
	TotalSuccessful int           // Total successful operations
	TotalFailed     int           // Total failed operations
	Duration        time.Duration // Total operation time
	Summary         string        // Human-readable summary
}

// AssetService defines the interface for asset operations
type AssetService interface {
	SyncFromConfluence(spaceKey, label string, debug bool) (*SyncResult, error)
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	Assets []string
	Errors []string
}

// SyncAndEnrichUseCase orchestrates the sync-and-enrich workflow
type SyncAndEnrichUseCase struct {
	repository          ports.AssetRepository
	llamaClient         application.LlamaClient
	assetService        AssetService
	bulkKeywordsUseCase *BulkEnrichKeywordsUseCase
	bulkFieldsUseCase   *BulkEnrichFieldsUseCase
}

// NewSyncAndEnrichUseCase creates a new sync-and-enrich use case
func NewSyncAndEnrichUseCase(
	repository ports.AssetRepository,
	llamaClient application.LlamaClient,
	assetService AssetService,
) *SyncAndEnrichUseCase {
	return &SyncAndEnrichUseCase{
		repository:          repository,
		llamaClient:         llamaClient,
		assetService:        assetService,
		bulkKeywordsUseCase: NewBulkEnrichKeywordsUseCase(repository, llamaClient),
		bulkFieldsUseCase:   NewBulkEnrichFieldsUseCase(repository, llamaClient),
	}
}

// Execute performs the complete sync-and-enrich workflow
func (uc *SyncAndEnrichUseCase) Execute(ctx context.Context, input SyncAndEnrichInput) (*SyncAndEnrichResult, error) {
	startTime := time.Now()

	result := &SyncAndEnrichResult{
		SyncErrors: make(map[string]string),
	}

	log.Printf("Starting sync-and-enrich workflow: space=%s, label=%s, keywords=%v, fields=%v, dry-run=%v",
		input.SpaceKey, input.Label, input.EnrichKeywords, input.EnrichFields, input.DryRun)

	// Step 1: Sync assets from Confluence
	if err := uc.performSync(ctx, input, result); err != nil {
		return nil, fmt.Errorf("sync phase failed: %w", err)
	}

	// Get list of synced assets for enrichment
	syncedAssetNames := result.SyncedAssets
	if len(syncedAssetNames) == 0 {
		log.Printf("No assets were synced, skipping enrichment phase")
		result.Duration = time.Since(startTime)
		result.Summary = "No assets synced from Confluence"
		return result, nil
	}

	log.Printf("Synced %d assets, proceeding with enrichment phase", len(syncedAssetNames))

	// Step 2: Enrich keywords if requested
	if input.EnrichKeywords {
		if err := uc.performKeywordsEnrichment(ctx, input, syncedAssetNames, result); err != nil {
			log.Printf("Keywords enrichment failed: %v", err)
			// Continue with field enrichment even if keywords fail
		}
	}

	// Step 3: Enrich fields if requested
	if len(input.EnrichFields) > 0 {
		if err := uc.performFieldsEnrichment(ctx, input, syncedAssetNames, result); err != nil {
			log.Printf("Fields enrichment failed: %v", err)
			// Log but don't fail the entire operation
		}
	}

	// Step 4: Generate summary
	result.Duration = time.Since(startTime)
	result.Summary = uc.generateSummary(result)

	log.Printf("Sync-and-enrich workflow completed in %v: %s", result.Duration, result.Summary)

	return result, nil
}

// performSync handles the sync phase
func (uc *SyncAndEnrichUseCase) performSync(_ context.Context, input SyncAndEnrichInput, result *SyncAndEnrichResult) error {
	if input.DryRun {
		log.Printf("DRY RUN: Would sync assets from Confluence with space=%s, label=%s", input.SpaceKey, input.Label)

		// For dry run, get current assets to simulate what would be synced
		allAssets, err := uc.repository.FindAll()
		if err != nil {
			return fmt.Errorf("failed to get existing assets for dry run: %w", err)
		}

		for _, asset := range allAssets {
			result.SyncedAssets = append(result.SyncedAssets, asset.Name)
		}
		result.TotalSynced = len(result.SyncedAssets)
		return nil
	}

	// Perform actual sync
	syncResult, err := uc.assetService.SyncFromConfluence(input.SpaceKey, input.Label, input.Debug)
	if err != nil {
		return fmt.Errorf("failed to sync from Confluence: %w", err)
	}

	result.SyncedAssets = syncResult.Assets
	result.TotalSynced = len(syncResult.Assets)

	// Map sync errors
	for _, errMsg := range syncResult.Errors {
		result.SyncErrors["sync"] = errMsg
	}

	return nil
}

// performKeywordsEnrichment handles the keywords enrichment phase
func (uc *SyncAndEnrichUseCase) performKeywordsEnrichment(ctx context.Context, input SyncAndEnrichInput, assetNames []string, result *SyncAndEnrichResult) error {
	log.Printf("Starting keywords enrichment for %d assets", len(assetNames))

	keywordsInput := BulkEnrichKeywordsInput{
		AssetNames:    assetNames,
		FilterBy:      "missing-keywords", // Only enrich assets without keywords
		MaxConcurrent: input.MaxConcurrent,
		DryRun:        input.DryRun,
	}

	keywordsResult, err := uc.bulkKeywordsUseCase.Execute(ctx, keywordsInput)
	if err != nil {
		return fmt.Errorf("bulk keywords enrichment failed: %w", err)
	}

	result.KeywordsResult = keywordsResult
	return nil
}

// performFieldsEnrichment handles the fields enrichment phase
func (uc *SyncAndEnrichUseCase) performFieldsEnrichment(ctx context.Context, input SyncAndEnrichInput, assetNames []string, result *SyncAndEnrichResult) error {
	log.Printf("Starting fields enrichment for %d assets, fields: %v", len(assetNames), input.EnrichFields)

	filterBy := input.FieldFilter
	if filterBy == "" {
		filterBy = "missing-fields" // Default to only enrich missing fields
	}

	fieldsInput := BulkEnrichFieldsInput{
		AssetNames:    assetNames,
		Fields:        input.EnrichFields,
		FilterBy:      filterBy,
		MaxConcurrent: input.MaxConcurrent,
		DryRun:        input.DryRun,
	}

	fieldsResult, err := uc.bulkFieldsUseCase.Execute(ctx, fieldsInput)
	if err != nil {
		return fmt.Errorf("bulk fields enrichment failed: %w", err)
	}

	result.FieldsResult = fieldsResult
	return nil
}

// generateSummary creates a human-readable summary of the operation
func (uc *SyncAndEnrichUseCase) generateSummary(result *SyncAndEnrichResult) string {
	summary := fmt.Sprintf("Synced %d assets", result.TotalSynced)

	if result.KeywordsResult != nil {
		summary += fmt.Sprintf(", generated keywords for %d assets", result.KeywordsResult.TotalSuccessful)
	}

	if result.FieldsResult != nil {
		summary += fmt.Sprintf(", enriched %d fields across %d assets", result.FieldsResult.TotalFieldsEnriched, result.FieldsResult.TotalSuccessful)
	}

	// Calculate overall totals
	result.TotalProcessed = result.TotalSynced
	result.TotalSuccessful = result.TotalSynced
	result.TotalFailed = 0

	if result.KeywordsResult != nil {
		result.TotalSuccessful += result.KeywordsResult.TotalSuccessful
		result.TotalFailed += result.KeywordsResult.TotalFailed
	}

	if result.FieldsResult != nil {
		result.TotalSuccessful += result.FieldsResult.TotalSuccessful
		result.TotalFailed += result.FieldsResult.TotalFailed
	}

	if result.TotalFailed > 0 {
		summary += fmt.Sprintf(" (%d failed operations)", result.TotalFailed)
	}

	return summary
}
