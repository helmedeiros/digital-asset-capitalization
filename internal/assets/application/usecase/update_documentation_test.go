package usecase

import (
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateDocumentationUseCase(t *testing.T) {
	// Create a mock repository
	mockRepo := testutil.NewMockAssetRepository()
	useCase := NewUpdateDocumentationUseCase(mockRepo)

	// Create a test asset
	initialTime := time.Now()
	testAsset := &domain.Asset{
		Name:            "test-asset",
		Description:     "Test description",
		LastDocUpdateAt: initialTime,
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
			name:      "valid update",
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

			// Verify documentation was updated correctly
			asset, err := mockRepo.FindByName(tt.assetName)
			require.NoError(t, err)
			assert.True(t, asset.LastDocUpdateAt.After(initialTime), "LastDocUpdateAt should be after initial time")
		})
	}
}
