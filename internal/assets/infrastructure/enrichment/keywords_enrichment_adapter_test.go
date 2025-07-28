package enrichment

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// MockLlamaClient is a mock implementation of the LlamaClient interface for testing
type MockLlamaClient struct {
	mock.Mock
}

func (m *MockLlamaClient) EnrichContent(content, field string, asset *domain.Asset) (string, error) {
	args := m.Called(content, field, asset)
	return args.String(0), args.Error(1)
}

func (m *MockLlamaClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestKeywordsEnrichmentAdapter_GenerateKeywords(t *testing.T) {
	tests := []struct {
		name           string
		asset          *domain.Asset
		mockSetup      func(*MockLlamaClient)
		expectedResult []string
		expectError    bool
	}{
		{
			name: "successful keywords generation",
			asset: &domain.Asset{
				Name:        "Test Asset",
				Description: "A test asset for testing",
			},
			mockSetup: func(mockClient *MockLlamaClient) {
				mockClient.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.AnythingOfType("*domain.Asset")).
					Return("keyword1, keyword2, keyword3", nil)
			},
			expectedResult: []string{"keyword1", "keyword2", "keyword3"},
			expectError:    false,
		},
		{
			name: "LLM client error",
			asset: &domain.Asset{
				Name:        "Test Asset",
				Description: "A test asset for testing",
			},
			mockSetup: func(mockClient *MockLlamaClient) {
				mockClient.On("EnrichContent", mock.AnythingOfType("string"), "keywords", mock.AnythingOfType("*domain.Asset")).
					Return("", errors.New("LLM error"))
			},
			expectedResult: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockLlamaClient)
			tt.mockSetup(mockClient)

			adapter := NewKeywordsEnrichmentAdapter(mockClient)
			result, err := adapter.GenerateKeywords(context.Background(), tt.asset)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockClient.AssertExpectations(t)
		})
	}
}
