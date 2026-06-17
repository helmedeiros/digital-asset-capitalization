package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/enrichment"
)

// TestMockAssetRepository for testing bulk keywords use case
type TestMockAssetRepository struct {
	mock.Mock
}

func (m *TestMockAssetRepository) Save(asset *domain.Asset) error {
	args := m.Called(asset)
	return args.Error(0)
}

func (m *TestMockAssetRepository) FindByName(name string) (*domain.Asset, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *TestMockAssetRepository) FindByID(id string) (*domain.Asset, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *TestMockAssetRepository) FindAll() ([]*domain.Asset, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Asset), args.Error(1)
}

func (m *TestMockAssetRepository) Delete(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// TestMockLlamaClient for testing bulk keywords use case
type TestMockLlamaClient struct {
	mock.Mock
}

func (m *TestMockLlamaClient) EnrichContent(content, field string, asset *domain.Asset) (string, error) {
	args := m.Called(content, field, asset)
	return args.String(0), args.Error(1)
}

func (m *TestMockLlamaClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestBulkEnrichKeywordsUseCase_Execute(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          BulkEnrichKeywordsInput
		setupMocks     func(*TestMockAssetRepository, *TestMockLlamaClient)
		expectedResult func(*testing.T, *BulkEnrichKeywordsResult, error)
	}{
		{
			name: "successful bulk keywords enrichment",
			input: BulkEnrichKeywordsInput{
				FilterBy:      "missing-keywords",
				MaxConcurrent: 2,
				DryRun:        false,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{
						ID:       "1",
						Name:     "Test Asset 1",
						Keywords: []string{}, // Empty keywords - should be enriched
					},
				}
				repo.On("FindAll").Return(assets, nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.AnythingOfType("*domain.Asset")).Return("keyword1, keyword2, keyword3", nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed)
				assert.Equal(t, 1, result.TotalSuccessful)
			},
		},
		{
			name: "dry run mode",
			input: BulkEnrichKeywordsInput{
				FilterBy:      "missing-keywords",
				MaxConcurrent: 1,
				DryRun:        true,
			},
			setupMocks: func(repo *TestMockAssetRepository, _ *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{ID: "1", Name: "Test Asset", Keywords: []string{}},
				}
				repo.On("FindAll").Return(assets, nil)
				// No enrichment calls should be made in dry run
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed)
			},
		},
		{
			name: "specific asset names",
			input: BulkEnrichKeywordsInput{
				AssetNames:    []string{"Asset1", "Asset2"},
				MaxConcurrent: 2,
				DryRun:        false,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				asset1 := &domain.Asset{ID: "1", Name: "Asset1", Keywords: []string{}}
				asset2 := &domain.Asset{ID: "2", Name: "Asset2", Keywords: []string{}}
				repo.On("FindByName", "Asset1").Return(asset1, nil)
				repo.On("FindByName", "Asset2").Return(asset2, nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.AnythingOfType("*domain.Asset")).Return("keyword1, keyword2", nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 2, result.TotalProcessed) // Only Asset1 and Asset2 should be processed
			},
		},
		{
			name: "repository error",
			input: BulkEnrichKeywordsInput{
				MaxConcurrent: 1,
			},
			setupMocks: func(repo *TestMockAssetRepository, _ *TestMockLlamaClient) {
				repo.On("FindAll").Return(nil, fmt.Errorf("repository error"))
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "repository error")
				assert.Nil(t, result)
			},
		},
		{
			name: "no assets to process",
			input: BulkEnrichKeywordsInput{
				FilterBy:      "missing-keywords",
				MaxConcurrent: 1,
			},
			setupMocks: func(repo *TestMockAssetRepository, _ *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{ID: "1", Name: "Asset1", Keywords: []string{"existing"}}, // Should be filtered out
				}
				repo.On("FindAll").Return(assets, nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 0, result.TotalProcessed) // No assets should be processed
			},
		},
		{
			name: "enrichment service error",
			input: BulkEnrichKeywordsInput{
				MaxConcurrent: 1,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{ID: "1", Name: "Test Asset", Keywords: []string{}},
				}
				repo.On("FindAll").Return(assets, nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.AnythingOfType("*domain.Asset")).Return("", fmt.Errorf("enrichment error"))
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.NoError(t, err) // Use case should not fail, but track errors
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed)
				assert.Equal(t, 0, result.TotalSuccessful) // Asset should fail enrichment
				assert.Equal(t, 1, result.TotalFailed)
			},
		},
		{
			name: "save error after enrichment",
			input: BulkEnrichKeywordsInput{
				MaxConcurrent: 1,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{ID: "1", Name: "Test Asset", Keywords: []string{}},
				}
				repo.On("FindAll").Return(assets, nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.AnythingOfType("*domain.Asset")).Return("keyword1, keyword2", nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(fmt.Errorf("save error"))
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.NoError(t, err) // Use case should not fail, but track errors
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed)
				assert.Equal(t, 0, result.TotalSuccessful) // Asset should fail due to save error
				assert.Equal(t, 1, result.TotalFailed)
			},
		},
		{
			name: "filter by empty-description",
			input: BulkEnrichKeywordsInput{
				FilterBy:      "empty-description",
				MaxConcurrent: 1,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{ID: "1", Name: "Asset1", Description: "", Keywords: []string{}},        // Should be included
					{ID: "2", Name: "Asset2", Description: "Content", Keywords: []string{}}, // Should be excluded
				}
				repo.On("FindAll").Return(assets, nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.AnythingOfType("*domain.Asset")).Return("keyword1, keyword2", nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed) // Only Asset1 should be processed
			},
		},
		{
			name: "unknown filter type",
			input: BulkEnrichKeywordsInput{
				FilterBy:      "unknown-filter",
				MaxConcurrent: 1,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{ID: "1", Name: "Asset1", Keywords: []string{}},
				}
				repo.On("FindAll").Return(assets, nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.AnythingOfType("*domain.Asset")).Return("keyword1, keyword2", nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed) // Should process all assets for unknown filter
			},
		},
		{
			name: "multiple assets with mixed success and failure",
			input: BulkEnrichKeywordsInput{
				MaxConcurrent: 2,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{ID: "1", Name: "Asset1", Keywords: []string{}},
					{ID: "2", Name: "Asset2", Keywords: []string{}},
				}
				repo.On("FindAll").Return(assets, nil)
				// First call succeeds
				llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.MatchedBy(func(asset *domain.Asset) bool {
					return asset.Name == "Asset1"
				})).Return("keyword1, keyword2", nil)
				// Second call fails
				llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.MatchedBy(func(asset *domain.Asset) bool {
					return asset.Name == "Asset2"
				})).Return("", fmt.Errorf("enrichment error"))
				// Only one save call should be made (for successful asset)
				repo.On("Save", mock.MatchedBy(func(asset *domain.Asset) bool {
					return asset.Name == "Asset1"
				})).Return(nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 2, result.TotalProcessed)
				assert.Equal(t, 1, result.TotalSuccessful)
				assert.Equal(t, 1, result.TotalFailed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &TestMockAssetRepository{}
			mockLlama := &TestMockLlamaClient{}

			enrichmentService := enrichment.NewKeywordsEnrichmentAdapter(mockLlama)
			useCase := NewBulkEnrichKeywordsUseCase(mockRepo, enrichmentService)

			tt.setupMocks(mockRepo, mockLlama)

			result, err := useCase.Execute(context.Background(), tt.input)

			tt.expectedResult(t, result, err)

			mockRepo.AssertExpectations(t)
			mockLlama.AssertExpectations(t)
		})
	}
}

// Note: getAssetsToProcess is now handled internally by the bulk processor
// This test has been replaced with integration tests that verify the overall behavior

// Note: processAsset is now handled internally by the processors
// Individual asset processing is tested at the processor level
