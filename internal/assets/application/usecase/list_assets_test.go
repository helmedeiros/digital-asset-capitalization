package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestListAssetsUseCase(t *testing.T) {
	// Create a mock repository
	mockRepo := testutil.NewMockAssetRepository()
	useCase := NewListAssetsUseCase(mockRepo)

	// Create some test assets
	testAssets := []struct {
		name        string
		description string
	}{
		{"asset1", "First asset"},
		{"asset2", "Second asset"},
		{"asset3", "Third asset"},
	}

	for _, asset := range testAssets {
		_, err := domain.NewAsset(asset.name, asset.description)
		require.NoError(t, err)
		err = mockRepo.Save(&domain.Asset{
			Name:        asset.name,
			Description: asset.description,
		})
		require.NoError(t, err)
	}

	// Test listing assets
	assets, err := useCase.Execute()
	require.NoError(t, err)
	assert.Len(t, assets, len(testAssets))

	// Verify asset contents
	assetMap := make(map[string]*domain.Asset)
	for _, asset := range assets {
		assetMap[asset.Name] = asset
	}

	for _, testAsset := range testAssets {
		assert.Contains(t, assetMap, testAsset.name)
		assert.Equal(t, testAsset.description, assetMap[testAsset.name].Description)
	}
}
