package testutil

import (
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// MockLlamaClient provides a mock implementation of the LlamaClient interface for testing
type MockLlamaClient struct {
	mock.Mock
}

// EnrichContent mocks the content enrichment functionality
func (m *MockLlamaClient) EnrichContent(content, field string, asset *domain.Asset) (string, error) {
	args := m.Called(content, field, asset)
	return args.String(0), args.Error(1)
}

// Close mocks the close functionality
func (m *MockLlamaClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockAssetService provides a mock implementation of the AssetService interface for testing
type MockAssetService struct {
	mock.Mock
}

// SyncResult represents the result of a sync operation (duplicated here to avoid circular imports)
type SyncResult struct {
	Assets []string
	Errors []string
}

// SyncFromConfluence mocks the sync from Confluence functionality
func (m *MockAssetService) SyncFromConfluence(spaceKey, label string, debug bool) (*SyncResult, error) {
	args := m.Called(spaceKey, label, debug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SyncResult), args.Error(1)
}