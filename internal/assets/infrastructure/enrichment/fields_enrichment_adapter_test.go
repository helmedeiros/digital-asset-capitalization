package enrichment

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestFieldsEnrichmentAdapter_EnrichField(t *testing.T) {
	tests := []struct {
		name           string
		asset          *domain.Asset
		field          string
		content        string
		mockSetup      func(*MockLlamaClient)
		expectedResult string
		expectError    bool
		errorMessage   string
	}{
		{
			name: "successful field enrichment",
			asset: &domain.Asset{
				Name:        "Test Asset",
				Description: "A test asset",
			},
			field:   "description",
			content: "Asset: Test Asset Description: A test asset",
			mockSetup: func(mockClient *MockLlamaClient) {
				mockClient.On("EnrichContent", "Asset: Test Asset Description: A test asset", "description", mock.AnythingOfType("*domain.Asset")).
					Return("Enhanced description content", nil)
			},
			expectedResult: "Enhanced description content",
			expectError:    false,
		},
		{
			name: "empty content uses asset information",
			asset: &domain.Asset{
				Name:        "Test Asset",
				Description: "A test asset",
				Why:         "For testing",
			},
			field:   "why",
			content: "", // Empty content should trigger auto-generation
			mockSetup: func(mockClient *MockLlamaClient) {
				expectedContent := "Asset: Test Asset\nDescription: A test asset\nWhy: For testing\nBenefits: \nHow: \nMetrics: "
				mockClient.On("EnrichContent", expectedContent, "why", mock.AnythingOfType("*domain.Asset")).
					Return("Enhanced why content", nil)
			},
			expectedResult: "Enhanced why content",
			expectError:    false,
		},
		{
			name: "invalid field",
			asset: &domain.Asset{
				Name: "Test Asset",
			},
			field:   "invalid_field",
			content: "some content",
			mockSetup: func(_ *MockLlamaClient) {
				// No mock setup needed as it should fail before calling LLM
			},
			expectedResult: "",
			expectError:    true,
			errorMessage:   "invalid field 'invalid_field'",
		},
		{
			name: "LLM client error",
			asset: &domain.Asset{
				Name: "Test Asset",
			},
			field:   "description",
			content: "test content",
			mockSetup: func(mockClient *MockLlamaClient) {
				mockClient.On("EnrichContent", "test content", "description", mock.AnythingOfType("*domain.Asset")).
					Return("", errors.New("LLM error"))
			},
			expectedResult: "",
			expectError:    true,
			errorMessage:   "LLM enrichment failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockLlamaClient)
			tt.mockSetup(mockClient)

			adapter := NewFieldsEnrichmentAdapter(mockClient)
			result, err := adapter.EnrichField(context.Background(), tt.asset, tt.field, tt.content)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestFieldsEnrichmentAdapter_ValidFields(t *testing.T) {
	mockClient := new(MockLlamaClient)
	adapter := NewFieldsEnrichmentAdapter(mockClient)

	validFields := []string{"description", "why", "benefits", "how", "metrics"}
	invalidFields := []string{"invalid", "unknown", "bad_field"}

	asset := &domain.Asset{Name: "Test Asset"}

	// Test valid fields
	for _, field := range validFields {
		mockClient.On("EnrichContent", mock.AnythingOfType("string"), field, mock.AnythingOfType("*domain.Asset")).
			Return("enriched content", nil).Once()

		result, err := adapter.EnrichField(context.Background(), asset, field, "test content")
		assert.NoError(t, err)
		assert.Equal(t, "enriched content", result)
	}

	// Test invalid fields
	for _, field := range invalidFields {
		result, err := adapter.EnrichField(context.Background(), asset, field, "test content")
		assert.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "invalid field")
	}

	mockClient.AssertExpectations(t)
}
