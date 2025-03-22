package usecase

import (
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecrementTaskCountUseCase(t *testing.T) {
	// Create a mock repository
	mockRepo := testutil.NewMockAssetRepository()
	useCase := NewDecrementTaskCountUseCase(mockRepo)

	// Create a test asset with initial task count
	testAsset := &domain.Asset{
		Name:                "test-asset",
		Description:         "Test description",
		AssociatedTaskCount: 2,
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
			name:      "valid decrement",
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

			// Verify task count was decremented correctly
			asset, err := mockRepo.FindByName(tt.assetName)
			require.NoError(t, err)
			assert.Equal(t, 1, asset.AssociatedTaskCount)
		})
	}

	// Test decrementing below zero
	t.Run("decrement below zero", func(t *testing.T) {
		// First decrement to get to zero
		err := useCase.Execute("test-asset")
		require.NoError(t, err)

		// Try to decrement again
		err = useCase.Execute("test-asset")
		require.Error(t, err)
		assert.Equal(t, domain.ErrNegativeTaskCount.Error(), err.Error())
	})
}
