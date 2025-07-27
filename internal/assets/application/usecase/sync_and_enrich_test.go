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

// MockAssetService for testing sync-and-enrich use case
type MockAssetService struct {
	mock.Mock
}

func (m *MockAssetService) SyncFromConfluence(spaceKey, label string, debug bool) (*SyncResult, error) {
	args := m.Called(spaceKey, label, debug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SyncResult), args.Error(1)
}

func TestSyncAndEnrichUseCase_Execute(t *testing.T) {
	tests := []struct {
		name           string
		input          SyncAndEnrichInput
		setupMocks     func(*TestMockAssetRepository, *TestMockLlamaClient, *MockAssetService)
		expectedResult func(*testing.T, *SyncAndEnrichResult, error)
	}{
		{
			name: "successful sync and enrich workflow",
			input: SyncAndEnrichInput{
				SpaceKey:       "TEST",
				Label:          "cap-asset",
				EnrichKeywords: true,
				EnrichFields:   []string{"description", "why"},
				MaxConcurrent:  2,
				DryRun:         false,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient, assetSvc *MockAssetService) {
				// Mock sync operation
				syncResult := &SyncResult{
					Assets: []string{"Asset1", "Asset2"},
					Errors: []string{},
				}
				assetSvc.On("SyncFromConfluence", "TEST", "cap-asset", false).Return(syncResult, nil)

				// Mock asset retrieval for enrichment
				asset1 := &domain.Asset{ID: "1", Name: "Asset1", Keywords: []string{}, Description: ""}
				asset2 := &domain.Asset{ID: "2", Name: "Asset2", Keywords: []string{}, Description: ""}

				repo.On("FindByName", "Asset1").Return(asset1, nil)
				repo.On("FindByName", "Asset2").Return(asset2, nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)

				// Mock LLM enrichment
				llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.AnythingOfType("*domain.Asset")).Return("keyword1, keyword2", nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "description", mock.AnythingOfType("*domain.Asset")).Return("Generated description", nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "why", mock.AnythingOfType("*domain.Asset")).Return("Generated why", nil)
			},
			expectedResult: func(t *testing.T, result *SyncAndEnrichResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 2, result.TotalSynced)
				assert.Len(t, result.SyncedAssets, 2)
				assert.NotNil(t, result.KeywordsResult)
				assert.NotNil(t, result.FieldsResult)
				assert.Greater(t, result.Duration, time.Duration(0))
				assert.Contains(t, result.Summary, "Synced 2 assets")
			},
		},
		{
			name: "dry run mode",
			input: SyncAndEnrichInput{
				SpaceKey:       "TEST",
				Label:          "cap-asset",
				EnrichKeywords: true,
				EnrichFields:   []string{"description"},
				DryRun:         true,
			},
			setupMocks: func(repo *TestMockAssetRepository, _ *TestMockLlamaClient, _ *MockAssetService) {
				// Mock existing assets for dry run simulation
				assets := []*domain.Asset{
					{ID: "1", Name: "Asset1", Keywords: []string{}},
					{ID: "2", Name: "Asset2", Keywords: []string{}},
				}
				repo.On("FindAll").Return(assets, nil)
				repo.On("FindByName", "Asset1").Return(assets[0], nil)
				repo.On("FindByName", "Asset2").Return(assets[1], nil)
				// No Save or LLaMA calls should be made in dry run
			},
			expectedResult: func(t *testing.T, result *SyncAndEnrichResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 2, result.TotalSynced)
				assert.NotNil(t, result.KeywordsResult)
				assert.NotNil(t, result.FieldsResult)
			},
		},
		{
			name: "sync failure",
			input: SyncAndEnrichInput{
				SpaceKey:       "INVALID",
				Label:          "cap-asset",
				EnrichKeywords: true,
			},
			setupMocks: func(_ *TestMockAssetRepository, _ *TestMockLlamaClient, assetSvc *MockAssetService) {
				assetSvc.On("SyncFromConfluence", "INVALID", "cap-asset", false).Return(nil, errors.New("confluence sync failed"))
			},
			expectedResult: func(t *testing.T, result *SyncAndEnrichResult, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "sync phase failed")
				assert.Nil(t, result)
			},
		},
		{
			name: "no assets synced",
			input: SyncAndEnrichInput{
				SpaceKey:       "EMPTY",
				Label:          "cap-asset",
				EnrichKeywords: true,
			},
			setupMocks: func(_ *TestMockAssetRepository, _ *TestMockLlamaClient, assetSvc *MockAssetService) {
				syncResult := &SyncResult{
					Assets: []string{}, // No assets synced
					Errors: []string{},
				}
				assetSvc.On("SyncFromConfluence", "EMPTY", "cap-asset", false).Return(syncResult, nil)
			},
			expectedResult: func(t *testing.T, result *SyncAndEnrichResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 0, result.TotalSynced)
				assert.Equal(t, "No assets synced from Confluence", result.Summary)
			},
		},
		{
			name: "keywords only enrichment",
			input: SyncAndEnrichInput{
				SpaceKey:       "TEST",
				Label:          "cap-asset",
				EnrichKeywords: true,
				EnrichFields:   []string{}, // No field enrichment
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient, assetSvc *MockAssetService) {
				syncResult := &SyncResult{
					Assets: []string{"Asset1"},
					Errors: []string{},
				}
				assetSvc.On("SyncFromConfluence", "TEST", "cap-asset", false).Return(syncResult, nil)

				asset := &domain.Asset{ID: "1", Name: "Asset1", Keywords: []string{}}
				repo.On("FindByName", "Asset1").Return(asset, nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.AnythingOfType("*domain.Asset")).Return("keyword1, keyword2", nil)
			},
			expectedResult: func(t *testing.T, result *SyncAndEnrichResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 1, result.TotalSynced)
				assert.NotNil(t, result.KeywordsResult)
				assert.Nil(t, result.FieldsResult)
				assert.Contains(t, result.Summary, "generated keywords for")
			},
		},
		{
			name: "fields only enrichment",
			input: SyncAndEnrichInput{
				SpaceKey:       "TEST",
				Label:          "cap-asset",
				EnrichKeywords: false,
				EnrichFields:   []string{"description"},
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient, assetSvc *MockAssetService) {
				syncResult := &SyncResult{
					Assets: []string{"Asset1"},
					Errors: []string{},
				}
				assetSvc.On("SyncFromConfluence", "TEST", "cap-asset", false).Return(syncResult, nil)

				asset := &domain.Asset{ID: "1", Name: "Asset1", Description: ""}
				repo.On("FindByName", "Asset1").Return(asset, nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "description", mock.AnythingOfType("*domain.Asset")).Return("Generated description", nil)
			},
			expectedResult: func(t *testing.T, result *SyncAndEnrichResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 1, result.TotalSynced)
				assert.Nil(t, result.KeywordsResult)
				assert.NotNil(t, result.FieldsResult)
				assert.Contains(t, result.Summary, "enriched")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(TestMockAssetRepository)
			llama := new(TestMockLlamaClient)
			assetSvc := new(MockAssetService)

			tt.setupMocks(repo, llama, assetSvc)

			useCase := NewSyncAndEnrichUseCase(repo, llama, assetSvc)
			result, err := useCase.Execute(context.Background(), tt.input)

			tt.expectedResult(t, result, err)

			repo.AssertExpectations(t)
			llama.AssertExpectations(t)
			assetSvc.AssertExpectations(t)
		})
	}
}

func TestSyncAndEnrichUseCase_generateSummary(t *testing.T) {
	useCase := &SyncAndEnrichUseCase{}

	tests := []struct {
		name     string
		result   *SyncAndEnrichResult
		expected string
	}{
		{
			name: "sync only",
			result: &SyncAndEnrichResult{
				TotalSynced: 3,
			},
			expected: "Synced 3 assets",
		},
		{
			name: "sync and keywords",
			result: &SyncAndEnrichResult{
				TotalSynced: 2,
				KeywordsResult: &BulkEnrichKeywordsResult{
					TotalSuccessful: 2,
				},
			},
			expected: "Synced 2 assets, generated keywords for 2 assets",
		},
		{
			name: "sync, keywords, and fields",
			result: &SyncAndEnrichResult{
				TotalSynced: 1,
				KeywordsResult: &BulkEnrichKeywordsResult{
					TotalSuccessful: 1,
				},
				FieldsResult: &BulkEnrichFieldsResult{
					TotalFieldsEnriched: 3,
					TotalSuccessful:     1,
				},
			},
			expected: "Synced 1 assets, generated keywords for 1 assets, enriched 3 fields across 1 assets",
		},
		{
			name: "with failures",
			result: &SyncAndEnrichResult{
				TotalSynced: 2,
				KeywordsResult: &BulkEnrichKeywordsResult{
					TotalSuccessful: 1,
					TotalFailed:     1,
				},
			},
			expected: "Synced 2 assets, generated keywords for 1 assets (1 failed operations)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := useCase.generateSummary(tt.result)
			assert.Contains(t, summary, tt.expected)
		})
	}
}
