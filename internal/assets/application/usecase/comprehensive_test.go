package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// Additional comprehensive tests to improve coverage

func TestBulkEnrichKeywordsUseCase_processAsset_errorHandling(t *testing.T) {
	repo := new(TestMockAssetRepository)
	llama := new(TestMockLlamaClient)

	asset := &domain.Asset{
		Name:        "Test Asset",
		Description: "Test description",
		Keywords:    []string{},
	}

	// Test LLM enrichment error
	llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", asset).Return("", errors.New("LLM service unavailable"))

	useCase := NewBulkEnrichKeywordsUseCase(repo, llama)
	err := useCase.processAsset(context.Background(), asset, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate keywords")

	llama.AssertExpectations(t)
}

func TestBulkEnrichKeywordsUseCase_processAsset_saveError(t *testing.T) {
	repo := new(TestMockAssetRepository)
	llama := new(TestMockLlamaClient)

	asset := &domain.Asset{
		Name:        "Test Asset",
		Description: "Test description",
		Keywords:    []string{},
	}

	// Mock successful keyword generation but failed save
	llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", asset).Return("keyword1, keyword2", nil)
	repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(errors.New("database error"))

	useCase := NewBulkEnrichKeywordsUseCase(repo, llama)
	err := useCase.processAsset(context.Background(), asset, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save asset")

	repo.AssertExpectations(t)
	llama.AssertExpectations(t)
}

func TestBulkEnrichFieldsUseCase_generateFieldContent_error(t *testing.T) {
	repo := new(TestMockAssetRepository)
	llama := new(TestMockLlamaClient)

	asset := &domain.Asset{
		Name:        "Test Asset",
		Description: "Test description",
	}

	// Mock LLM error
	llama.On("EnrichContent", mock.AnythingOfType("string"), "description", asset).Return("", errors.New("LLM timeout"))

	useCase := NewBulkEnrichFieldsUseCase(repo, llama)
	content, err := useCase.generateFieldContent(context.Background(), asset, "description")

	assert.Error(t, err)
	assert.Empty(t, content)
	assert.Contains(t, err.Error(), "LLM enrichment failed")

	llama.AssertExpectations(t)
}

func TestBulkEnrichFieldsUseCase_updateAssetField_unknownField(t *testing.T) {
	useCase := &BulkEnrichFieldsUseCase{}
	asset := &domain.Asset{}

	err := useCase.updateAssetField(asset, "unknown_field", "content")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}

func TestBulkEnrichFieldsUseCase_getAssetsToProcess_error(t *testing.T) {
	repo := new(TestMockAssetRepository)
	llama := new(TestMockLlamaClient)

	repo.On("FindAll").Return(nil, errors.New("database connection failed"))

	useCase := NewBulkEnrichFieldsUseCase(repo, llama)
	input := BulkEnrichFieldsInput{
		Fields:   []string{"description"},
		FilterBy: "all",
	}

	assets, err := useCase.getAssetsToProcess(input)

	assert.Error(t, err)
	assert.Nil(t, assets)
	assert.Contains(t, err.Error(), "failed to get all assets")

	repo.AssertExpectations(t)
}

func TestBulkEnrichFieldsUseCase_getAssetsToProcess_assetNotFound(t *testing.T) {
	repo := new(TestMockAssetRepository)
	llama := new(TestMockLlamaClient)

	repo.On("FindByName", "NonExistentAsset").Return(nil, errors.New("asset not found"))

	useCase := NewBulkEnrichFieldsUseCase(repo, llama)
	input := BulkEnrichFieldsInput{
		AssetNames: []string{"NonExistentAsset"},
		Fields:     []string{"description"},
	}

	assets, err := useCase.getAssetsToProcess(input)

	assert.Error(t, err)
	assert.Nil(t, assets)
	assert.Contains(t, err.Error(), "failed to find asset 'NonExistentAsset'")

	repo.AssertExpectations(t)
}

func TestBulkEnrichFieldsUseCase_getAssetsToProcess_unsupportedFilter(t *testing.T) {
	repo := new(TestMockAssetRepository)
	llama := new(TestMockLlamaClient)

	repo.On("FindAll").Return([]*domain.Asset{}, nil)

	useCase := NewBulkEnrichFieldsUseCase(repo, llama)
	input := BulkEnrichFieldsInput{
		Fields:   []string{"description"},
		FilterBy: "unsupported-filter",
	}

	assets, err := useCase.getAssetsToProcess(input)

	assert.Error(t, err)
	assert.Nil(t, assets)
	assert.Contains(t, err.Error(), "unsupported filter")

	repo.AssertExpectations(t)
}

func TestBulkEnrichKeywordsUseCase_getAssetsToProcess_assetNotFound(t *testing.T) {
	repo := new(TestMockAssetRepository)
	llama := new(TestMockLlamaClient)

	repo.On("FindByName", "NonExistentAsset").Return(nil, errors.New("asset not found"))

	useCase := NewBulkEnrichKeywordsUseCase(repo, llama)
	input := BulkEnrichKeywordsInput{
		AssetNames: []string{"NonExistentAsset"},
	}

	assets, err := useCase.getAssetsToProcess(input)

	assert.Error(t, err)
	assert.Nil(t, assets)
	assert.Contains(t, err.Error(), "failed to find asset 'NonExistentAsset'")

	repo.AssertExpectations(t)
}

func TestBulkEnrichKeywordsUseCase_getAssetsToProcess_outdatedFilter(t *testing.T) {
	repo := new(TestMockAssetRepository)
	llama := new(TestMockLlamaClient)

	oldDate := time.Now().AddDate(0, 0, -31) // 31 days ago
	recentDate := time.Now()

	assets := []*domain.Asset{
		{
			Name:      "OldAsset",
			Keywords:  []string{"old"},
			UpdatedAt: oldDate, // Old asset with keywords - should be included
		},
		{
			Name:      "RecentAsset",
			Keywords:  []string{"recent"},
			UpdatedAt: recentDate, // Recent asset with keywords - should be included
		},
		{
			Name:      "NoKeywordsAsset",
			Keywords:  []string{},
			UpdatedAt: recentDate, // No keywords - should be included
		},
	}

	repo.On("FindAll").Return(assets, nil)

	useCase := NewBulkEnrichKeywordsUseCase(repo, llama)
	input := BulkEnrichKeywordsInput{
		FilterBy: "outdated",
	}

	filteredAssets, err := useCase.getAssetsToProcess(input)

	assert.NoError(t, err)
	assert.Len(t, filteredAssets, 2) // Only recent asset with keywords and no-keywords asset should be included

	repo.AssertExpectations(t)
}

func TestSyncAndEnrichUseCase_performSync_repositoryError(t *testing.T) {
	repo := new(TestMockAssetRepository)
	llama := new(TestMockLlamaClient)
	assetSvc := new(MockAssetService)

	// Mock repository error during dry run
	repo.On("FindAll").Return(nil, errors.New("repository error"))

	useCase := NewSyncAndEnrichUseCase(repo, llama, assetSvc)
	input := SyncAndEnrichInput{
		SpaceKey: "TEST",
		Label:    "cap-asset",
		DryRun:   true,
	}
	result := &SyncAndEnrichResult{
		SyncErrors: make(map[string]string),
	}

	err := useCase.performSync(context.Background(), input, result)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get existing assets for dry run")

	repo.AssertExpectations(t)
}

func TestSyncAndEnrichUseCase_performKeywordsEnrichment_error(t *testing.T) {
	repo := new(TestMockAssetRepository)
	llama := new(TestMockLlamaClient)
	assetSvc := new(MockAssetService)

	// Mock repository error
	repo.On("FindByName", "Asset1").Return(nil, errors.New("asset not found"))

	useCase := NewSyncAndEnrichUseCase(repo, llama, assetSvc)
	input := SyncAndEnrichInput{
		SpaceKey:       "TEST",
		Label:          "cap-asset",
		EnrichKeywords: true,
		MaxConcurrent:  1,
	}
	result := &SyncAndEnrichResult{
		SyncErrors: make(map[string]string),
	}

	err := useCase.performKeywordsEnrichment(context.Background(), input, []string{"Asset1"}, result)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bulk keywords enrichment failed")

	repo.AssertExpectations(t)
}

func TestSyncAndEnrichUseCase_performFieldsEnrichment_error(t *testing.T) {
	repo := new(TestMockAssetRepository)
	llama := new(TestMockLlamaClient)
	assetSvc := new(MockAssetService)

	// Mock repository error
	repo.On("FindByName", "Asset1").Return(nil, errors.New("asset not found"))

	useCase := NewSyncAndEnrichUseCase(repo, llama, assetSvc)
	input := SyncAndEnrichInput{
		SpaceKey:      "TEST",
		Label:         "cap-asset",
		EnrichFields:  []string{"description"},
		MaxConcurrent: 1,
	}
	result := &SyncAndEnrichResult{
		SyncErrors: make(map[string]string),
	}

	err := useCase.performFieldsEnrichment(context.Background(), input, []string{"Asset1"}, result)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bulk fields enrichment failed")

	repo.AssertExpectations(t)
}

func TestSyncAndEnrichUseCase_Execute_syncErrorHandling(t *testing.T) {
	repo := new(TestMockAssetRepository)
	llama := new(TestMockLlamaClient)
	assetSvc := new(MockAssetService)

	// Mock sync with errors
	syncResult := &SyncResult{
		Assets: []string{"Asset1"},
		Errors: []string{"Warning: Asset2 validation failed"},
	}
	assetSvc.On("SyncFromConfluence", "TEST", "cap-asset", false).Return(syncResult, nil)

	// Mock successful enrichment
	asset := &domain.Asset{ID: "1", Name: "Asset1", Keywords: []string{}}
	repo.On("FindByName", "Asset1").Return(asset, nil)
	repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
	llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.AnythingOfType("*domain.Asset")).Return("keyword1", nil)

	useCase := NewSyncAndEnrichUseCase(repo, llama, assetSvc)
	input := SyncAndEnrichInput{
		SpaceKey:       "TEST",
		Label:          "cap-asset",
		EnrichKeywords: true,
		MaxConcurrent:  1,
	}

	result, err := useCase.Execute(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalSynced)
	assert.Contains(t, result.SyncErrors, "sync") // Error should be mapped

	repo.AssertExpectations(t)
	llama.AssertExpectations(t)
	assetSvc.AssertExpectations(t)
}

func TestSyncAndEnrichUseCase_Execute_enrichmentFailureDoesNotStopWorkflow(t *testing.T) {
	repo := new(TestMockAssetRepository)
	llama := new(TestMockLlamaClient)
	assetSvc := new(MockAssetService)

	// Mock successful sync
	syncResult := &SyncResult{
		Assets: []string{"Asset1"},
		Errors: []string{},
	}
	assetSvc.On("SyncFromConfluence", "TEST", "cap-asset", false).Return(syncResult, nil)

	// Mock failed keywords enrichment but successful field enrichment
	repo.On("FindByName", "Asset1").Return(nil, errors.New("asset not found during keywords")).Once()
	asset := &domain.Asset{ID: "1", Name: "Asset1", Description: ""}
	repo.On("FindByName", "Asset1").Return(asset, nil).Once()
	repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
	llama.On("EnrichContent", mock.AnythingOfType("string"), "description", mock.AnythingOfType("*domain.Asset")).Return("Generated description", nil)

	useCase := NewSyncAndEnrichUseCase(repo, llama, assetSvc)
	input := SyncAndEnrichInput{
		SpaceKey:       "TEST",
		Label:          "cap-asset",
		EnrichKeywords: true,
		EnrichFields:   []string{"description"},
		MaxConcurrent:  1,
	}

	result, err := useCase.Execute(context.Background(), input)

	// Should succeed overall even if keywords enrichment fails
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalSynced)
	assert.Nil(t, result.KeywordsResult)  // Should be nil due to failure
	assert.NotNil(t, result.FieldsResult) // Should succeed

	assetSvc.AssertExpectations(t)
}
