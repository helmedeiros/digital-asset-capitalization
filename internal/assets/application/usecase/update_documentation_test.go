package usecase

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestUpdateDocumentationUseCase(t *testing.T) {
	t.Parallel()
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

	// Set up mock expectation for setup
	mockRepo.On("Save", testAsset).Return(nil)
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
			// Reset mock expectations for each test
			mockRepo.ExpectedCalls = nil

			if tt.wantErr {
				// Set up expectation for failed FindByName
				mockRepo.On("FindByName", tt.assetName).Return(nil, fmt.Errorf("asset not found"))
			} else {
				// Set up expectations for successful execution
				mockRepo.On("FindByName", tt.assetName).Return(testAsset, nil)
				// The asset will be modified, so we need to expect Save with the modified asset
				mockRepo.On("Save", mock.MatchedBy(func(asset *domain.Asset) bool {
					return asset.Name == tt.assetName && asset.LastDocUpdateAt.After(initialTime)
				})).Return(nil)
			}

			err := useCase.Execute(tt.assetName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}

			require.NoError(t, err)

			// Verify mock expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}
