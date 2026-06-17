package usecase

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestUpdateAssetUseCase(t *testing.T) {
	t.Parallel()
	// Create a mock repository
	mockRepo := testutil.NewMockAssetRepository()
	useCase := NewUpdateAssetUseCase(mockRepo)

	// Create a test asset
	testAsset := &domain.Asset{
		Name:        "test-asset",
		Description: "Initial description",
	}

	// Set up mock expectation for setup
	mockRepo.On("Save", testAsset).Return(nil)
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
			// Reset mock expectations for each test
			mockRepo.ExpectedCalls = nil

			if tt.wantErr {
				if tt.assetName == "non-existent" {
					// Set up expectation for failed FindByName
					mockRepo.On("FindByName", tt.assetName).Return(nil, fmt.Errorf("asset not found"))
				} else {
					// For empty description, find returns the asset but update fails
					mockRepo.On("FindByName", tt.assetName).Return(testAsset, nil)
				}
			} else {
				// Set up expectations for successful execution
				mockRepo.On("FindByName", tt.assetName).Return(testAsset, nil)
				// The asset will be modified, so we need to expect Save with the modified asset
				mockRepo.On("Save", mock.MatchedBy(func(asset *domain.Asset) bool {
					return asset.Name == tt.assetName && asset.Description == tt.description
				})).Return(nil)
			}

			err := useCase.Execute(tt.assetName, tt.description)

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
