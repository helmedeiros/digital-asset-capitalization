package ports

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestBulkEnrichmentResult_Methods(t *testing.T) {
	result := NewBulkEnrichmentResult()

	// Test initial state
	assert.Equal(t, 0, result.TotalProcessed)
	assert.Equal(t, 0, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)
	assert.Empty(t, result.ProcessedAssets)
	assert.Empty(t, result.SuccessfulAssets)
	assert.Empty(t, result.FailedAssets)

	// Test AddProcessedAsset
	result.AddProcessedAsset("asset1")
	assert.Equal(t, 1, result.TotalProcessed)
	assert.Contains(t, result.ProcessedAssets, "asset1")

	// Test AddSuccessfulAsset
	result.AddSuccessfulAsset("asset1")
	assert.Equal(t, 1, result.TotalSuccessful)
	assert.Contains(t, result.SuccessfulAssets, "asset1")

	// Test AddFailedAsset with error message
	result.AddFailedAsset("asset2", "test error")
	assert.Equal(t, 1, result.TotalFailed)
	assert.Contains(t, result.FailedAssets, "asset2")
	assert.Equal(t, "test error", result.FailedAssets["asset2"])

	// Test multiple failures with different error messages
	result.AddFailedAsset("asset3", "another error")
	assert.Equal(t, 2, result.TotalFailed)
	assert.Contains(t, result.FailedAssets, "asset3")
	assert.Equal(t, "another error", result.FailedAssets["asset3"])

	// Test empty error message (covers the condition that was at 75%)
	result.AddFailedAsset("asset4", "")
	assert.Equal(t, 3, result.TotalFailed)
	assert.Contains(t, result.FailedAssets, "asset4")
	assert.Equal(t, "", result.FailedAssets["asset4"])

	// Test error message overwrite (additional coverage)
	result.AddFailedAsset("asset2", "updated error")
	assert.Equal(t, 3, result.TotalFailed) // Count should not change
	assert.Equal(t, "updated error", result.FailedAssets["asset2"])

	// Test nil error handling path
	result.AddFailedAsset("", "empty asset name")
	assert.Equal(t, 4, result.TotalFailed)
	assert.Contains(t, result.FailedAssets, "")
}

func TestInterfaceContracts(_ *testing.T) {
	// Test that our interfaces can be implemented
	var _ KeywordsEnrichmentService = (*dummyKeywordsService)(nil)
	var _ FieldsEnrichmentService = (*dummyFieldsService)(nil)
	var _ BulkEnrichmentService = (*dummyBulkEnrichmentService)(nil)
	var _ IDGenerator = (*dummyIDGenerator)(nil)
}

func TestBulkEnrichmentInput(t *testing.T) {
	input := BulkEnrichmentInput{
		AssetNames:    []string{"asset1", "asset2"},
		FilterBy:      "test-filter",
		MaxConcurrent: 5,
		DryRun:        true,
	}

	assert.Equal(t, []string{"asset1", "asset2"}, input.AssetNames)
	assert.Equal(t, "test-filter", input.FilterBy)
	assert.Equal(t, 5, input.MaxConcurrent)
	assert.True(t, input.DryRun)

	// Test with empty values
	emptyInput := BulkEnrichmentInput{}
	assert.Empty(t, emptyInput.AssetNames)
	assert.Empty(t, emptyInput.FilterBy)
	assert.Equal(t, 0, emptyInput.MaxConcurrent)
	assert.False(t, emptyInput.DryRun)
}

func TestDummyImplementations(t *testing.T) {
	// Test dummy implementations don't panic and return expected values
	keywordsService := &dummyKeywordsService{}
	keywords, err := keywordsService.GenerateKeywords(context.Background(), &domain.Asset{})
	assert.Equal(t, []string{"dummy"}, keywords)
	assert.NoError(t, err)

	fieldsService := &dummyFieldsService{}
	result, err := fieldsService.EnrichField(context.Background(), &domain.Asset{}, "test", "content")
	assert.Equal(t, "dummy", result)
	assert.NoError(t, err)

	bulkService := &dummyBulkEnrichmentService{}
	bulkResult, err := bulkService.ProcessAssets(context.Background(), BulkEnrichmentInput{})
	assert.NotNil(t, bulkResult)
	assert.NoError(t, err)

	idGen := &dummyIDGenerator{}
	id := idGen.GenerateID("test")
	assert.Equal(t, "dummy-id", id)
}

// Dummy implementations for interface testing
type dummyKeywordsService struct{}

func (d *dummyKeywordsService) GenerateKeywords(_ context.Context, _ *domain.Asset) ([]string, error) {
	return []string{"dummy"}, nil
}

type dummyFieldsService struct{}

func (d *dummyFieldsService) EnrichField(_ context.Context, _ *domain.Asset, _ string, _ string) (string, error) {
	return "dummy", nil
}

type dummyBulkEnrichmentService struct{}

func (d *dummyBulkEnrichmentService) ProcessAssets(_ context.Context, _ BulkEnrichmentInput) (*BulkEnrichmentResult, error) {
	return NewBulkEnrichmentResult(), nil
}

type dummyIDGenerator struct{}

func (d *dummyIDGenerator) GenerateID(_ string) string {
	return "dummy-id"
}
