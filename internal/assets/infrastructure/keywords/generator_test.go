package keywords

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// MockLlamaClient is a mock implementation of the LLaMA client
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

func TestGenerateKeywords(t *testing.T) {
	tests := []struct {
		name          string
		asset         *domain.Asset
		mockResponse  string
		expectedError string
		expectedCount int
		expectedFirst string
		expectedLast  string
	}{
		{
			name: "successful keyword generation",
			asset: &domain.Asset{
				Name:        "Test Asset",
				Description: "A test asset for keyword generation",
				Why:         "To test keyword generation functionality",
				Benefits:    "Improved asset discoverability",
				How:         "Using LLaMA for NLP-based keyword extraction",
				Metrics:     "Keyword relevance and coverage",
			},
			mockResponse:  "test-asset, keyword-generation, nlp-extraction, asset-discovery, technical-testing",
			expectedCount: 5,
			expectedFirst: "test-asset",
			expectedLast:  "technical-testing",
		},
		{
			name: "empty response",
			asset: &domain.Asset{
				Name:        "Empty Asset",
				Description: "An asset with empty response",
			},
			mockResponse:  "",
			expectedCount: 0,
		},
		{
			name: "response with invalid keywords",
			asset: &domain.Asset{
				Name:        "Invalid Asset",
				Description: "An asset with invalid keywords",
			},
			mockResponse:  "a, very-long-keyword-that-should-be-filtered-out, !@#$%^&*(), valid-keyword",
			expectedCount: 1,
			expectedFirst: "valid-keyword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock LLaMA client
			mockLlama := new(MockLlamaClient)
			mockLlama.On("EnrichContent", mock.Anything, "keywords", tt.asset).Return(tt.mockResponse, nil)

			// Create generator with mock client
			generator := NewGenerator(mockLlama)

			// Generate keywords
			keywords, err := generator.GenerateKeywords(tt.asset)

			// Check error
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			// Check keywords
			assert.NoError(t, err)
			assert.Len(t, keywords, tt.expectedCount)

			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectedFirst, keywords[0])
				if tt.expectedLast != "" {
					assert.Equal(t, tt.expectedLast, keywords[len(keywords)-1])
				}
			}

			// Verify all keywords are lowercase and clean
			for _, keyword := range keywords {
				assert.Equal(t, strings.ToLower(keyword), keyword)
				assert.Regexp(t, `^[a-z0-9\s-]+$`, keyword)
				assert.GreaterOrEqual(t, len(keyword), 2)
				assert.LessOrEqual(t, len(keyword), 50)
			}
		})
	}
}

func TestProcessKeywords(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		expectedFirst string
		expectedLast  string
	}{
		{
			name:          "clean keywords",
			input:         "keyword1, keyword2, keyword3",
			expectedCount: 3,
			expectedFirst: "keyword1",
			expectedLast:  "keyword3",
		},
		{
			name:          "keywords with special characters",
			input:         "key@word1, key#word2, key$word3",
			expectedCount: 3,
			expectedFirst: "keyword1",
			expectedLast:  "keyword3",
		},
		{
			name:          "keywords with mixed case",
			input:         "KeyWord1, KEYWORD2, keyword3",
			expectedCount: 3,
			expectedFirst: "keyword1",
			expectedLast:  "keyword3",
		},
		{
			name:          "keywords with extra spaces",
			input:         "  keyword1  ,  keyword2  ,  keyword3  ",
			expectedCount: 3,
			expectedFirst: "keyword1",
			expectedLast:  "keyword3",
		},
		{
			name:          "empty input",
			input:         "",
			expectedCount: 0,
		},
		{
			name:          "input with empty keywords",
			input:         "keyword1, , keyword3",
			expectedCount: 2,
			expectedFirst: "keyword1",
			expectedLast:  "keyword3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keywords := processKeywords(tt.input)

			assert.Len(t, keywords, tt.expectedCount)

			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectedFirst, keywords[0])
				if tt.expectedLast != "" {
					assert.Equal(t, tt.expectedLast, keywords[len(keywords)-1])
				}
			}

			// Verify all keywords are lowercase and clean
			for _, keyword := range keywords {
				assert.Equal(t, strings.ToLower(keyword), keyword)
				assert.Regexp(t, `^[a-z0-9\s-]+$`, keyword)
				assert.GreaterOrEqual(t, len(keyword), 2)
				assert.LessOrEqual(t, len(keyword), 50)
			}
		})
	}
}
