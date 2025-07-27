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
						ID:          "1",
						Name:        "Test Asset 1",
						Keywords:    []string{}, // No keywords
						Description: "Test description",
						UpdatedAt:   time.Now(),
					},
					{
						ID:          "2",
						Name:        "Test Asset 2",
						Keywords:    []string{"existing"}, // Has keywords
						Description: "Test description 2",
						UpdatedAt:   time.Now(),
					},
				}
				repo.On("FindAll").Return(assets, nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.AnythingOfType("*domain.Asset")).Return("keyword1, keyword2, keyword3", nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed) // Only assets without keywords
				assert.GreaterOrEqual(t, result.TotalSuccessful, 0)
				assert.Greater(t, result.Duration, time.Duration(0))
			},
		},
		{
			name: "dry run mode",
			input: BulkEnrichKeywordsInput{
				FilterBy:      "all",
				MaxConcurrent: 1,
				DryRun:        true,
			},
			setupMocks: func(repo *TestMockAssetRepository, _ *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{
						ID:   "1",
						Name: "Test Asset",
					},
				}
				repo.On("FindAll").Return(assets, nil)
				// No Save or LLaMA calls should be made in dry run
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed)
				assert.Equal(t, 1, result.TotalSuccessful) // Dry run simulates success
			},
		},
		{
			name: "specific asset names",
			input: BulkEnrichKeywordsInput{
				AssetNames:    []string{"Asset1", "Asset2"},
				MaxConcurrent: 1,
				DryRun:        false,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				asset1 := &domain.Asset{ID: "1", Name: "Asset1", Keywords: []string{}}
				asset2 := &domain.Asset{ID: "2", Name: "Asset2", Keywords: []string{}}

				repo.On("FindByName", "Asset1").Return(asset1, nil)
				repo.On("FindByName", "Asset2").Return(asset2, nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.AnythingOfType("*domain.Asset")).Return("keyword1, keyword2", nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 2, result.TotalProcessed)
			},
		},
		{
			name: "repository error",
			input: BulkEnrichKeywordsInput{
				FilterBy: "all",
			},
			setupMocks: func(repo *TestMockAssetRepository, _ *TestMockLlamaClient) {
				repo.On("FindAll").Return(nil, errors.New("repository error"))
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get assets to process")
				assert.Nil(t, result)
			},
		},
		{
			name: "no assets to process",
			input: BulkEnrichKeywordsInput{
				FilterBy: "missing-keywords",
			},
			setupMocks: func(repo *TestMockAssetRepository, _ *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{
						ID:       "1",
						Name:     "Asset with keywords",
						Keywords: []string{"keyword1", "keyword2"},
					},
				}
				repo.On("FindAll").Return(assets, nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichKeywordsResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 0, result.TotalProcessed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(TestMockAssetRepository)
			llama := new(TestMockLlamaClient)

			tt.setupMocks(repo, llama)

			useCase := NewBulkEnrichKeywordsUseCase(repo, llama)
			result, err := useCase.Execute(context.Background(), tt.input)

			tt.expectedResult(t, result, err)

			repo.AssertExpectations(t)
			llama.AssertExpectations(t)
		})
	}
}

func TestBulkEnrichKeywordsUseCase_getAssetsToProcess(t *testing.T) {
	tests := []struct {
		name        string
		input       BulkEnrichKeywordsInput
		setupMocks  func(*TestMockAssetRepository)
		expectCount int
		expectError bool
	}{
		{
			name: "filter by missing keywords",
			input: BulkEnrichKeywordsInput{
				FilterBy: "missing-keywords",
			},
			setupMocks: func(repo *TestMockAssetRepository) {
				assets := []*domain.Asset{
					{Name: "Asset1", Keywords: []string{}},          // No keywords - should be included
					{Name: "Asset2", Keywords: []string{"keyword"}}, // Has keywords - should be excluded
					{Name: "Asset3", Keywords: nil},                 // Nil keywords - should be included
				}
				repo.On("FindAll").Return(assets, nil)
			},
			expectCount: 2,
			expectError: false,
		},
		{
			name: "filter by outdated",
			input: BulkEnrichKeywordsInput{
				FilterBy: "outdated",
			},
			setupMocks: func(repo *TestMockAssetRepository) {
				oldDate := time.Now().AddDate(0, 0, -31) // 31 days ago
				recentDate := time.Now()

				assets := []*domain.Asset{
					{Name: "Asset1", Keywords: []string{}, UpdatedAt: oldDate},       // No keywords - should be included
					{Name: "Asset2", Keywords: []string{"k"}, UpdatedAt: recentDate}, // Recent with keywords - should be included
					{Name: "Asset3", Keywords: []string{"k"}, UpdatedAt: oldDate},    // Old with keywords - should be excluded
				}
				repo.On("FindAll").Return(assets, nil)
			},
			expectCount: 2,
			expectError: false,
		},
		{
			name: "invalid filter",
			input: BulkEnrichKeywordsInput{
				FilterBy: "invalid",
			},
			setupMocks: func(repo *TestMockAssetRepository) {
				repo.On("FindAll").Return([]*domain.Asset{}, nil)
			},
			expectCount: 0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(TestMockAssetRepository)
			llama := new(TestMockLlamaClient)

			tt.setupMocks(repo)

			useCase := NewBulkEnrichKeywordsUseCase(repo, llama)
			assets, err := useCase.getAssetsToProcess(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, assets, tt.expectCount)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestBulkEnrichKeywordsUseCase_processAsset(t *testing.T) {
	t.Run("dry run", func(t *testing.T) {
		repo := new(TestMockAssetRepository)
		llama := new(TestMockLlamaClient)

		useCase := NewBulkEnrichKeywordsUseCase(repo, llama)
		asset := &domain.Asset{Name: "Test Asset"}

		err := useCase.processAsset(context.Background(), asset, true)

		assert.NoError(t, err)
		// No mocks should be called in dry run
		repo.AssertExpectations(t)
		llama.AssertExpectations(t)
	})

	t.Run("successful processing", func(t *testing.T) {
		repo := new(TestMockAssetRepository)
		llama := new(TestMockLlamaClient)

		asset := &domain.Asset{
			Name:        "Test Asset",
			Description: "Test description",
			Keywords:    []string{}, // Start with no keywords
		}

		// Mock the keyword generator workflow
		llama.On("EnrichContent", mock.AnythingOfType("string"), "keywords", asset).Return("keyword1, keyword2, keyword3", nil)
		repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)

		useCase := NewBulkEnrichKeywordsUseCase(repo, llama)
		err := useCase.processAsset(context.Background(), asset, false)

		assert.NoError(t, err)
		assert.Len(t, asset.Keywords, 3)
		assert.Contains(t, asset.Keywords, "keyword1")

		repo.AssertExpectations(t)
		llama.AssertExpectations(t)
	})
}
