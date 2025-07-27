package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestBulkEnrichFieldsUseCase_Execute(t *testing.T) {
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
				FilterBy:      "all",
				MaxConcurrent: 1,
				DryRun:        true,
			},
			setupMocks: func(repo *TestMockAssetRepository, _ *TestMockLlamaClient) {
				assets := []*domain.Asset{
					{ID: "1", Name: "Test Asset", Description: ""},
				}
				repo.On("FindAll").Return(assets, nil)
				// No Save or LLaMA calls should be made in dry run
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 1, result.TotalProcessed)
				assert.Equal(t, 1, result.TotalSuccessful) // Dry run simulates success
			},
		},
		{
			name: "validation error - no fields",
			input: BulkEnrichFieldsInput{
				Fields: []string{}, // Empty fields
			},
			setupMocks: func(_ *TestMockAssetRepository, _ *TestMockLlamaClient) {
				// No mocks needed - validation fails early
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
				Fields: []string{"invalid_field"},
			},
			setupMocks: func(_ *TestMockAssetRepository, _ *TestMockLlamaClient) {
				// No mocks needed - validation fails early
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid field 'invalid_field'")
				assert.Nil(t, result)
			},
		},
		{
			name: "specific asset names",
			input: BulkEnrichFieldsInput{
				AssetNames: []string{"Asset1"},
				Fields:     []string{"description"},
			},
			setupMocks: func(repo *TestMockAssetRepository, llama *TestMockLlamaClient) {
				asset := &domain.Asset{ID: "1", Name: "Asset1", Description: ""}
				repo.On("FindByName", "Asset1").Return(asset, nil)
				llama.On("EnrichContent", mock.AnythingOfType("string"), "description", asset).Return("New description", nil)
				repo.On("Save", asset).Return(nil)
			},
			expectedResult: func(t *testing.T, result *BulkEnrichFieldsResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 1, result.TotalProcessed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(TestMockAssetRepository)
			llama := new(TestMockLlamaClient)

			tt.setupMocks(repo, llama)

			useCase := NewBulkEnrichFieldsUseCase(repo, llama)
			result, err := useCase.Execute(context.Background(), tt.input)

			tt.expectedResult(t, result, err)

			repo.AssertExpectations(t)
			llama.AssertExpectations(t)
		})
	}
}

func TestBulkEnrichFieldsUseCase_validateInput(t *testing.T) {
	useCase := &BulkEnrichFieldsUseCase{}

	tests := []struct {
		name        string
		input       BulkEnrichFieldsInput
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid fields",
			input: BulkEnrichFieldsInput{
				Fields: []string{"description", "why", "benefits"},
			},
			expectError: false,
		},
		{
			name:        "no fields",
			input:       BulkEnrichFieldsInput{Fields: []string{}},
			expectError: true,
			errorMsg:    "at least one field must be specified",
		},
		{
			name: "invalid field",
			input: BulkEnrichFieldsInput{
				Fields: []string{"description", "invalid"},
			},
			expectError: true,
			errorMsg:    "invalid field 'invalid'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := useCase.validateInput(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBulkEnrichFieldsUseCase_isFieldEmpty(t *testing.T) {
	useCase := &BulkEnrichFieldsUseCase{}

	asset := &domain.Asset{
		Description: "Has description",
		Why:         "",
		Benefits:    "Has benefits",
		How:         "",
		Metrics:     "Has metrics",
	}

	tests := []struct {
		field    string
		expected bool
	}{
		{"description", false}, // Has content
		{"why", true},          // Empty
		{"benefits", false},    // Has content
		{"how", true},          // Empty
		{"metrics", false},     // Has content
		{"unknown", false},     // Unknown field
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			result := useCase.isFieldEmpty(asset, tt.field)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBulkEnrichFieldsUseCase_updateAssetField(t *testing.T) {
	useCase := &BulkEnrichFieldsUseCase{}
	asset := &domain.Asset{}

	tests := []struct {
		field   string
		content string
		verify  func(*domain.Asset)
	}{
		{
			field:   "description",
			content: "New description",
			verify:  func(a *domain.Asset) { assert.Equal(t, "New description", a.Description) },
		},
		{
			field:   "why",
			content: "New why",
			verify:  func(a *domain.Asset) { assert.Equal(t, "New why", a.Why) },
		},
		{
			field:   "benefits",
			content: "New benefits",
			verify:  func(a *domain.Asset) { assert.Equal(t, "New benefits", a.Benefits) },
		},
		{
			field:   "how",
			content: "New how",
			verify:  func(a *domain.Asset) { assert.Equal(t, "New how", a.How) },
		},
		{
			field:   "metrics",
			content: "New metrics",
			verify:  func(a *domain.Asset) { assert.Equal(t, "New metrics", a.Metrics) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			err := useCase.updateAssetField(asset, tt.field, tt.content)
			assert.NoError(t, err)
			tt.verify(asset)
		})
	}

	// Test unknown field
	err := useCase.updateAssetField(asset, "unknown", "content")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}

func TestBulkEnrichFieldsUseCase_processAsset(t *testing.T) {
	t.Run("dry run", func(t *testing.T) {
		repo := new(TestMockAssetRepository)
		llama := new(TestMockLlamaClient)

		useCase := NewBulkEnrichFieldsUseCase(repo, llama)
		asset := &domain.Asset{Name: "Test Asset"}

		fieldsEnriched, err := useCase.processAsset(context.Background(), asset, []string{"description"}, true)

		assert.NoError(t, err)
		assert.True(t, fieldsEnriched["description"]) // Dry run simulates success

		repo.AssertExpectations(t)
		llama.AssertExpectations(t)
	})

	t.Run("skip field with existing content", func(t *testing.T) {
		repo := new(TestMockAssetRepository)
		llama := new(TestMockLlamaClient)

		useCase := NewBulkEnrichFieldsUseCase(repo, llama)
		asset := &domain.Asset{
			Name:        "Test Asset",
			Description: "Existing description", // Not empty - should be skipped
		}

		fieldsEnriched, err := useCase.processAsset(context.Background(), asset, []string{"description"}, false)

		assert.NoError(t, err)
		assert.False(t, fieldsEnriched["description"]) // Should be skipped

		repo.AssertExpectations(t)
		llama.AssertExpectations(t)
	})

	t.Run("LLM enrichment failure", func(t *testing.T) {
		repo := new(TestMockAssetRepository)
		llama := new(TestMockLlamaClient)

		asset := &domain.Asset{
			Name:        "Test Asset",
			Description: "", // Empty - should try to enrich
		}

		llama.On("EnrichContent", mock.AnythingOfType("string"), "description", asset).Return("", errors.New("LLM error"))

		useCase := NewBulkEnrichFieldsUseCase(repo, llama)
		fieldsEnriched, err := useCase.processAsset(context.Background(), asset, []string{"description"}, false)

		assert.NoError(t, err)                         // Function doesn't fail on individual field errors
		assert.False(t, fieldsEnriched["description"]) // Should be marked as failed

		repo.AssertExpectations(t)
		llama.AssertExpectations(t)
	})
}
