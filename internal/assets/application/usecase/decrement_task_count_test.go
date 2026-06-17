package usecase

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestDecrementTaskCountUseCase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		assetName string
		setupMock func(*testutil.MockAssetRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid decrement",
			assetName: "test-asset",
			setupMock: func(repo *testutil.MockAssetRepository) {
				testAsset := &domain.Asset{
					Name:                "test-asset",
					Description:         "Test description",
					AssociatedTaskCount: 2,
				}
				repo.On("FindByName", "test-asset").Return(testAsset, nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "non-existent asset",
			assetName: "non-existent",
			setupMock: func(repo *testutil.MockAssetRepository) {
				repo.On("FindByName", "non-existent").Return(nil, errors.New("asset not found"))
			},
			wantErr: true,
			errMsg:  "asset not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := testutil.NewMockAssetRepository()
			tt.setupMock(mockRepo)

			useCase := NewDecrementTaskCountUseCase(mockRepo)
			err := useCase.Execute(tt.assetName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				require.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}

	// Test decrementing below zero
	t.Run("decrement below zero", func(t *testing.T) {
		mockRepo := testutil.NewMockAssetRepository()

		// Asset with zero task count
		testAsset := &domain.Asset{
			Name:                "test-asset",
			Description:         "Test description",
			AssociatedTaskCount: 0,
		}
		mockRepo.On("FindByName", "test-asset").Return(testAsset, nil)

		useCase := NewDecrementTaskCountUseCase(mockRepo)
		err := useCase.Execute("test-asset")

		require.Error(t, err)
		assert.Equal(t, domain.ErrNegativeTaskCount.Error(), err.Error())

		mockRepo.AssertExpectations(t)
	})
}
