package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestDeleteAssetUseCase(t *testing.T) {
	// Create a mock repository
	mockRepo := testutil.NewMockAssetRepository()
	useCase := NewDeleteAssetUseCase(mockRepo)

	// Create a test asset
	testAsset := &domain.Asset{
		Name:        "test-asset",
		Description: "Test description",
	}
	err := mockRepo.Save(testAsset)
	require.NoError(t, err)

	tests := []struct {
		name      string
		assetName string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "existing asset",
			assetName: "test-asset",
			wantErr:   false,
		},
		{
			name:      "non-existent asset",
			assetName: "non-existent",
			wantErr:   true,
			errMsg:    "asset not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := useCase.Execute(tt.assetName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}

			require.NoError(t, err)

			// Verify asset was deleted
			_, err = mockRepo.FindByName(tt.assetName)
			require.Error(t, err)
			assert.Equal(t, "asset not found", err.Error())
		})
	}
}
