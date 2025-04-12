package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestUpdateAssetUseCase(t *testing.T) {
	// Create a mock repository
	mockRepo := testutil.NewMockAssetRepository()
	useCase := NewUpdateAssetUseCase(mockRepo)

	// Create a test asset
	testAsset := &domain.Asset{
		Name:        "test-asset",
		Description: "Initial description",
	}
	err := mockRepo.Save(testAsset)
	require.NoError(t, err)

	tests := []struct {
		name        string
		assetName   string
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid update",
			assetName:   "test-asset",
			description: "Updated description",
			wantErr:     false,
		},
		{
			name:        "non-existent asset",
			assetName:   "non-existent",
			description: "New description",
			wantErr:     true,
			errMsg:      "asset not found",
		},
		{
			name:        "empty description",
			assetName:   "test-asset",
			description: "",
			wantErr:     true,
			errMsg:      domain.ErrEmptyDescription.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := useCase.Execute(tt.assetName, tt.description)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}

			require.NoError(t, err)

			// Verify asset was updated correctly
			asset, err := mockRepo.FindByName(tt.assetName)
			require.NoError(t, err)
			assert.Equal(t, tt.description, asset.Description)
		})
	}
}
