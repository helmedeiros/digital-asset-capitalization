package ports

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// KeywordsEnrichmentService defines the interface for keywords generation
type KeywordsEnrichmentService interface {
	// GenerateKeywords generates keywords for an asset
	GenerateKeywords(ctx context.Context, asset *domain.Asset) ([]string, error)
}

// FieldsEnrichmentService defines the interface for field enrichment
type FieldsEnrichmentService interface {
	// EnrichField enriches a specific field of an asset
	EnrichField(ctx context.Context, asset *domain.Asset, field string, content string) (string, error)
}

// BulkEnrichmentService defines the interface for bulk enrichment operations
type BulkEnrichmentService interface {
	// ProcessAssets processes multiple assets with the given filter and configuration
	ProcessAssets(ctx context.Context, input BulkEnrichmentInput) (*BulkEnrichmentResult, error)
}

// BulkEnrichmentInput represents input for bulk enrichment operations
type BulkEnrichmentInput struct {
	AssetNames    []string
	FilterBy      string
	MaxConcurrent int
	DryRun        bool
}

// BulkEnrichmentResult represents the result of bulk enrichment operations
type BulkEnrichmentResult struct {
	ProcessedAssets  []string
	SuccessfulAssets []string
	FailedAssets     map[string]string // asset name -> error message
	TotalProcessed   int
	TotalSuccessful  int
	TotalFailed      int
}

// AddProcessedAsset adds an asset to the processed list
func (r *BulkEnrichmentResult) AddProcessedAsset(assetName string) {
	r.ProcessedAssets = append(r.ProcessedAssets, assetName)
	r.TotalProcessed++
}

// AddSuccessfulAsset adds an asset to the successful list
func (r *BulkEnrichmentResult) AddSuccessfulAsset(assetName string) {
	r.SuccessfulAssets = append(r.SuccessfulAssets, assetName)
	r.TotalSuccessful++
}

// AddFailedAsset adds an asset to the failed list with error message
func (r *BulkEnrichmentResult) AddFailedAsset(assetName, errorMsg string) {
	if r.FailedAssets == nil {
		r.FailedAssets = make(map[string]string)
	}
	// Only increment if asset is not already in failed list
	if _, exists := r.FailedAssets[assetName]; !exists {
		r.TotalFailed++
	}
	r.FailedAssets[assetName] = errorMsg
}

// NewBulkEnrichmentResult creates a new bulk enrichment result
func NewBulkEnrichmentResult() *BulkEnrichmentResult {
	return &BulkEnrichmentResult{
		ProcessedAssets:  make([]string, 0),
		SuccessfulAssets: make([]string, 0),
		FailedAssets:     make(map[string]string),
	}
}
