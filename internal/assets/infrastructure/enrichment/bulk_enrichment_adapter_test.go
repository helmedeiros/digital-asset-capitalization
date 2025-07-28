package enrichment

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// Mock services for testing
type MockAssetRepository struct {
	mock.Mock
}

func (m *MockAssetRepository) Save(asset *domain.Asset) error {
	args := m.Called(asset)
	return args.Error(0)
}

func (m *MockAssetRepository) FindByName(name string) (*domain.Asset, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *MockAssetRepository) FindAll() ([]*domain.Asset, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Asset), args.Error(1)
}

func (m *MockAssetRepository) FindByID(id string) (*domain.Asset, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *MockAssetRepository) Delete(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

type MockKeywordsService struct {
	mock.Mock
}

func (m *MockKeywordsService) GenerateKeywords(ctx context.Context, asset *domain.Asset) ([]string, error) {
	args := m.Called(ctx, asset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

type MockFieldsService struct {
	mock.Mock
}

func (m *MockFieldsService) EnrichField(ctx context.Context, asset *domain.Asset, field string, content string) (string, error) {
	args := m.Called(ctx, asset, field, content)
	return args.String(0), args.Error(1)
}

func TestBulkEnrichmentAdapter_NewBulkEnrichmentAdapter(t *testing.T) {
	mockRepo := &MockAssetRepository{}
	mockKeywords := &MockKeywordsService{}
	mockFields := &MockFieldsService{}

	adapter := NewBulkEnrichmentAdapter(mockRepo, mockKeywords, mockFields)
	assert.NotNil(t, adapter)
}

func TestBulkEnrichmentAdapter_ProcessAssets(t *testing.T) {
	t.Run("should process all assets successfully", func(t *testing.T) {
		mockRepo := &MockAssetRepository{}
		mockKeywords := &MockKeywordsService{}
		mockFields := &MockFieldsService{}

		adapter := NewBulkEnrichmentAdapter(mockRepo, mockKeywords, mockFields)

		// Create test assets
		assets := []*domain.Asset{
			{Name: "asset1", Description: "Test asset 1"},
			{Name: "asset2", Description: "Test asset 2"},
		}

		// Set up mock expectations
		mockRepo.On("FindAll").Return(assets, nil)

		input := ports.BulkEnrichmentInput{
			AssetNames:    []string{},
			FilterBy:      "all",
			MaxConcurrent: 1,
			DryRun:        true, // Use dry run to avoid actual processing
		}

		ctx := context.Background()
		result, err := adapter.ProcessAssets(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.TotalProcessed)
		assert.Equal(t, 2, result.TotalSuccessful)
		assert.Equal(t, 0, result.TotalFailed)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should process specific assets by name", func(t *testing.T) {
		mockRepo := &MockAssetRepository{}
		mockKeywords := &MockKeywordsService{}
		mockFields := &MockFieldsService{}

		adapter := NewBulkEnrichmentAdapter(mockRepo, mockKeywords, mockFields)

		asset := &domain.Asset{Name: "asset1", Description: "Test asset 1"}
		mockRepo.On("FindByName", "asset1").Return(asset, nil)

		input := ports.BulkEnrichmentInput{
			AssetNames:    []string{"asset1"},
			FilterBy:      "all",
			MaxConcurrent: 1,
			DryRun:        true,
		}

		ctx := context.Background()
		result, err := adapter.ProcessAssets(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalProcessed)
		assert.Equal(t, 1, result.TotalSuccessful)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should handle asset not found", func(t *testing.T) {
		mockRepo := &MockAssetRepository{}
		mockKeywords := &MockKeywordsService{}
		mockFields := &MockFieldsService{}

		adapter := NewBulkEnrichmentAdapter(mockRepo, mockKeywords, mockFields)

		mockRepo.On("FindByName", "nonexistent").Return(nil, assert.AnError)

		input := ports.BulkEnrichmentInput{
			AssetNames:    []string{"nonexistent"},
			FilterBy:      "all",
			MaxConcurrent: 1,
			DryRun:        true,
		}

		ctx := context.Background()
		result, err := adapter.ProcessAssets(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, 0, result.TotalProcessed)
		assert.Equal(t, 0, result.TotalSuccessful)
		assert.Equal(t, 1, result.TotalFailed)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should handle repository error for FindAll", func(t *testing.T) {
		mockRepo := &MockAssetRepository{}
		mockKeywords := &MockKeywordsService{}
		mockFields := &MockFieldsService{}

		adapter := NewBulkEnrichmentAdapter(mockRepo, mockKeywords, mockFields)

		mockRepo.On("FindAll").Return(nil, assert.AnError)

		input := ports.BulkEnrichmentInput{
			AssetNames:    []string{},
			FilterBy:      "all",
			MaxConcurrent: 1,
			DryRun:        true,
		}

		ctx := context.Background()
		_, err := adapter.ProcessAssets(ctx, input)

		assert.Error(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should apply missing-keywords filter", func(t *testing.T) {
		mockRepo := &MockAssetRepository{}
		mockKeywords := &MockKeywordsService{}
		mockFields := &MockFieldsService{}

		adapter := NewBulkEnrichmentAdapter(mockRepo, mockKeywords, mockFields)

		// Create assets with and without keywords
		asset1 := &domain.Asset{Name: "asset1", Description: "Test asset 1", Keywords: []string{}}
		asset2 := &domain.Asset{Name: "asset2", Description: "Test asset 2", Keywords: []string{"keyword1"}}
		assets := []*domain.Asset{asset1, asset2}

		mockRepo.On("FindAll").Return(assets, nil)

		input := ports.BulkEnrichmentInput{
			AssetNames:    []string{},
			FilterBy:      "missing-keywords",
			MaxConcurrent: 1,
			DryRun:        true,
		}

		ctx := context.Background()
		result, err := adapter.ProcessAssets(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalProcessed) // Only asset1 should be processed

		mockRepo.AssertExpectations(t)
	})

	t.Run("should apply empty-description filter", func(t *testing.T) {
		mockRepo := &MockAssetRepository{}
		mockKeywords := &MockKeywordsService{}
		mockFields := &MockFieldsService{}

		adapter := NewBulkEnrichmentAdapter(mockRepo, mockKeywords, mockFields)

		// Create assets with and without descriptions
		asset1 := &domain.Asset{Name: "asset1", Description: ""}
		asset2 := &domain.Asset{Name: "asset2", Description: "Test description"}
		assets := []*domain.Asset{asset1, asset2}

		mockRepo.On("FindAll").Return(assets, nil)

		input := ports.BulkEnrichmentInput{
			AssetNames:    []string{},
			FilterBy:      "empty-description",
			MaxConcurrent: 1,
			DryRun:        true,
		}

		ctx := context.Background()
		result, err := adapter.ProcessAssets(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalProcessed) // Only asset1 should be processed

		mockRepo.AssertExpectations(t)
	})

	t.Run("should handle no matching assets after filter", func(t *testing.T) {
		mockRepo := &MockAssetRepository{}
		mockKeywords := &MockKeywordsService{}
		mockFields := &MockFieldsService{}

		adapter := NewBulkEnrichmentAdapter(mockRepo, mockKeywords, mockFields)

		// Create assets that don't match the filter
		asset1 := &domain.Asset{Name: "asset1", Description: "Test description", Keywords: []string{"keyword1"}}
		assets := []*domain.Asset{asset1}

		mockRepo.On("FindAll").Return(assets, nil)

		input := ports.BulkEnrichmentInput{
			AssetNames:    []string{},
			FilterBy:      "missing-keywords",
			MaxConcurrent: 1,
			DryRun:        true,
		}

		ctx := context.Background()
		result, err := adapter.ProcessAssets(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, 0, result.TotalProcessed)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should handle unknown filter", func(t *testing.T) {
		mockRepo := &MockAssetRepository{}
		mockKeywords := &MockKeywordsService{}
		mockFields := &MockFieldsService{}

		adapter := NewBulkEnrichmentAdapter(mockRepo, mockKeywords, mockFields)

		assets := []*domain.Asset{
			{Name: "asset1", Description: "Test asset 1"},
		}

		mockRepo.On("FindAll").Return(assets, nil)

		input := ports.BulkEnrichmentInput{
			AssetNames:    []string{},
			FilterBy:      "unknown-filter",
			MaxConcurrent: 1,
			DryRun:        true,
		}

		ctx := context.Background()
		result, err := adapter.ProcessAssets(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalProcessed) // Should process all assets for unknown filter

		mockRepo.AssertExpectations(t)
	})
}

func TestBulkEnrichmentAdapter_ProcessAssets_WithMinimalAssets(t *testing.T) {
	t.Run("should process single asset with minimal configuration", func(t *testing.T) {
		// Create mock services
		mockRepo := &MockAssetRepository{}
		mockKeywords := &MockKeywordsService{}
		mockFields := &MockFieldsService{}

		// Create adapter with all required services
		adapter := NewBulkEnrichmentAdapter(mockRepo, mockKeywords, mockFields)

		// Create test asset
		asset := &domain.Asset{
			ID:          "test-id",
			Name:        "Test Asset",
			Description: "Test description",
		}

		// Setup mocks for specific asset name lookup
		mockRepo.On("FindByName", "Test Asset").Return(asset, nil)

		// Test ProcessAssets with single asset to ensure processAsset is called
		input := ports.BulkEnrichmentInput{
			AssetNames:    []string{"Test Asset"},
			FilterBy:      "all",
			MaxConcurrent: 1,
			DryRun:        false, // Don't use dry run to ensure processAsset is called
		}

		ctx := context.Background()
		start := time.Now()
		result, err := adapter.ProcessAssets(ctx, input)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalProcessed)

		// Verify it took at least some time due to the processAsset sleep
		// This indirectly tests that processAsset was called
		assert.GreaterOrEqual(t, duration, 50*time.Millisecond)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should handle processing with timeout context", func(t *testing.T) {
		// Create mock services
		mockRepo := &MockAssetRepository{}
		mockKeywords := &MockKeywordsService{}
		mockFields := &MockFieldsService{}

		// Create adapter
		adapter := NewBulkEnrichmentAdapter(mockRepo, mockKeywords, mockFields)

		// Create test asset
		asset := &domain.Asset{
			ID:          "test-id",
			Name:        "Test Asset",
			Description: "Test description",
		}

		// Setup mocks for specific asset name lookup
		mockRepo.On("FindByName", "Test Asset").Return(asset, nil)

		// Create context with short timeout to test timeout behavior
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		input := ports.BulkEnrichmentInput{
			AssetNames:    []string{"Test Asset"},
			FilterBy:      "all",
			MaxConcurrent: 1,
			DryRun:        false, // Don't use dry run to test timeout during actual processing
		}

		// This may timeout or complete depending on timing
		result, err := adapter.ProcessAssets(ctx, input)

		// Either should succeed or timeout - both are valid
		if err != nil {
			assert.Contains(t, err.Error(), "context")
		} else {
			assert.NotNil(t, result)
		}

		mockRepo.AssertExpectations(t)
	})

	t.Run("should process multiple assets concurrently", func(t *testing.T) {
		// Create mock services
		mockRepo := &MockAssetRepository{}
		mockKeywords := &MockKeywordsService{}
		mockFields := &MockFieldsService{}

		// Create adapter
		adapter := NewBulkEnrichmentAdapter(mockRepo, mockKeywords, mockFields)

		// Create multiple test assets
		assets := []*domain.Asset{
			{ID: "test-id-1", Name: "Asset 1", Description: "Description 1"},
			{ID: "test-id-2", Name: "Asset 2", Description: "Description 2"},
			{ID: "test-id-3", Name: "Asset 3", Description: "Description 3"},
		}

		// Setup mocks for specific asset name lookups
		mockRepo.On("FindByName", "Asset 1").Return(assets[0], nil)
		mockRepo.On("FindByName", "Asset 2").Return(assets[1], nil)
		mockRepo.On("FindByName", "Asset 3").Return(assets[2], nil)

		input := ports.BulkEnrichmentInput{
			AssetNames:    []string{"Asset 1", "Asset 2", "Asset 3"},
			FilterBy:      "all",
			MaxConcurrent: 2,     // Process 2 at a time
			DryRun:        false, // Don't use dry run to ensure processAsset timing is tested
		}

		ctx := context.Background()
		start := time.Now()
		result, err := adapter.ProcessAssets(ctx, input)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Equal(t, 3, result.TotalProcessed)

		// With 3 assets and concurrency of 2, should take roughly 200ms
		// (100ms for first batch of 2, then 100ms for the remaining 1)
		assert.GreaterOrEqual(t, duration, 150*time.Millisecond)
		assert.LessOrEqual(t, duration, 300*time.Millisecond)

		mockRepo.AssertExpectations(t)
	})
}
