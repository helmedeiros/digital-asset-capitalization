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

func TestBulkEnrichFieldsUseCase_Execute(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          BulkEnrichFieldsInput
		setupMocks     func(*TestMockAssetRepository, *TestMockLlamaClient)
		expectedResult func(*testing.T, *BulkEnrichFieldsResult, error)
	}{
		{
			name: "successful bulk field enrichment",
			input: BulkEnrichFieldsInput{
				Fields:        []string{"description", "benefits"},
				FilterBy:      "missing-fields",
				MaxConcurrent: 2,
				DryRun:        false,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{
						ID:          "1",
						Name:        "Test Asset 1",
						Description: "",                  // Empty field - should be enriched
						Benefits:    "Existing benefits", // Not empty - should be skipped
					},
				}
				repo.On("FindAll").Return(assets, nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "description", mock.AnythingOfType("*domain.Asset")).Return("Generated description", nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed)
				assert.Equal(t, 1, result.TotalSuccessful)
				assert.Equal(t, 1, result.TotalFieldsEnriched) // Only description was enriched
			},
		},
		{
			name: "dry run mode",
			input: BulkEnrichFieldsInput{
				Fields:        []string{"description"},
				FilterBy:      "missing-fields",
				MaxConcurrent: 1,
				DryRun:        true,
			},
			setupMocks: func(repo *TestMockAssetRepository, _ *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{ID: "1", Name: "Test Asset", Description: ""},
				}
				repo.On("FindAll").Return(assets, nil)
				// No enrichment calls should be made in dry run
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed)
				assert.Equal(t, 1, result.TotalFieldsEnriched) // Dry run still counts potential enrichments
			},
		},
		{
			name: "validation error - no fields",
			input: BulkEnrichFieldsInput{
				Fields:        []string{},
				MaxConcurrent: 1,
			},
			setupMocks: func(_ *TestMockAssetRepository, _ *TestMockLlamaClient) {
				// No mocks needed for validation error
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "at least one field must be specified")
				assert.Nil(t, result)
			},
		},
		{
			name: "validation error - invalid field",
			input: BulkEnrichFieldsInput{
				Fields:        []string{"invalid_field"},
				MaxConcurrent: 1,
			},
			setupMocks: func(_ *TestMockAssetRepository, _ *TestMockLlamaClient) {
				// No mocks needed for validation error
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid field")
				assert.Nil(t, result)
			},
		},
		{
			name: "specific asset names",
			input: BulkEnrichFieldsInput{
				Fields:        []string{"description"},
				AssetNames:    []string{"Asset1"},
				MaxConcurrent: 1,
				DryRun:        false,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				asset1 := &domain.Asset{ID: "1", Name: "Asset1", Description: ""}
				repo.On("FindByName", "Asset1").Return(asset1, nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "description", mock.AnythingOfType("*domain.Asset")).Return("Generated description", nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed) // Only Asset1 should be processed
			},
		},
		{
			name: "repository error",
			input: BulkEnrichFieldsInput{
				Fields:        []string{"description"},
				MaxConcurrent: 1,
			},
			setupMocks: func(repo *TestMockAssetRepository, _ *TestMockLlamaClient) {
				repo.On("FindAll").Return(nil, fmt.Errorf("repository error"))
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "repository error")
				assert.Nil(t, result)
			},
		},
		{
			name: "enrichment service error",
			input: BulkEnrichFieldsInput{
				Fields:        []string{"description"},
				MaxConcurrent: 1,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{ID: "1", Name: "Test Asset", Description: ""},
				}
				repo.On("FindAll").Return(assets, nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "description", mock.AnythingOfType("*domain.Asset")).Return("", fmt.Errorf("enrichment error"))
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.NoError(t, err) // Use case should not fail, but track errors
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed)
				assert.Equal(t, 0, result.TotalSuccessful) // Asset should fail enrichment
				assert.Equal(t, 1, result.TotalFailed)
			},
		},
		{
			name: "save error after enrichment",
			input: BulkEnrichFieldsInput{
				Fields:        []string{"description"},
				MaxConcurrent: 1,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{ID: "1", Name: "Test Asset", Description: ""},
				}
				repo.On("FindAll").Return(assets, nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "description", mock.AnythingOfType("*domain.Asset")).Return("Generated description", nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(fmt.Errorf("save error"))
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.NoError(t, err) // Use case should not fail, but track errors
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed)
				assert.Equal(t, 0, result.TotalSuccessful) // Asset should fail due to save error
				assert.Equal(t, 1, result.TotalFailed)
			},
		},
		{
			name: "all field types coverage",
			input: BulkEnrichFieldsInput{
				Fields:        []string{"description", "why", "benefits", "how", "metrics"},
				MaxConcurrent: 1,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{
						ID:          "1",
						Name:        "Test Asset",
						Description: "",
						Why:         "",
						Benefits:    "",
						How:         "",
						Metrics:     "",
					},
				}
				repo.On("FindAll").Return(assets, nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("*domain.Asset")).Return("Generated content", nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed)
				assert.Equal(t, 1, result.TotalSuccessful)
				assert.Equal(t, 5, result.TotalFieldsEnriched) // All 5 fields should be enriched
			},
		},
		{
			name: "filter by empty-description",
			input: BulkEnrichFieldsInput{
				Fields:        []string{"description"},
				FilterBy:      "empty-description",
				MaxConcurrent: 1,
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{ID: "1", Name: "Asset1", Description: ""},        // Should be included
					{ID: "2", Name: "Asset2", Description: "Content"}, // Should be excluded
				}
				repo.On("FindAll").Return(assets, nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "description", mock.AnythingOfType("*domain.Asset")).Return("Generated description", nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.TotalProcessed) // Only Asset1 should be processed
			},
		},
		{
			name: "no assets match filter",
			input: BulkEnrichFieldsInput{
				Fields:        []string{"description"},
				FilterBy:      "empty-description",
				MaxConcurrent: 1,
			},
			setupMocks: func(repo *TestMockAssetRepository, _ *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{ID: "1", Name: "Asset1", Description: "Has content"}, // Should be excluded
				}
				repo.On("FindAll").Return(assets, nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 0, result.TotalProcessed) // No assets should be processed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &TestMockAssetRepository{}
			mockLlama := &TestMockLlamaClient{}

			enrichmentService := enrichment.NewFieldsEnrichmentAdapter(mockLlama)
			useCase := NewBulkEnrichFieldsUseCase(mockRepo, enrichmentService)

			tt.setupMocks(mockRepo, mockLlama)

			result, err := useCase.Execute(context.Background(), tt.input)

			tt.expectedResult(t, result, err)

			mockRepo.AssertExpectations(t)
			mockLlama.AssertExpectations(t)
		})
	}
}
