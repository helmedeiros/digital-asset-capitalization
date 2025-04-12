package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestCreateAssetUseCase(t *testing.T) {
	// Create a mock repository
	mockRepo := testutil.NewMockAssetRepository()
	useCase := NewCreateAssetUseCase(mockRepo)

	tests := []struct {
		name        string
		assetName   string
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid asset creation",
			assetName:   "test-asset",
			description: "Test description",
			wantErr:     false,
		},
		{
			name:        "duplicate asset",
			assetName:   "test-asset",
			description: "Duplicate description",
			wantErr:     true,
			errMsg:      "asset already exists",
		},
		{
			name:        "empty name",
			assetName:   "",
			description: "Test description",
			wantErr:     true,
			errMsg:      domain.ErrEmptyName.Error(),
		},
		{
			name:        "empty description",
			assetName:   "new-asset",
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

			// Verify asset was created correctly
			asset, err := mockRepo.FindByName(tt.assetName)
			require.NoError(t, err)
			assert.Equal(t, tt.assetName, asset.Name)
			assert.Equal(t, tt.description, asset.Description)
			assert.NotEmpty(t, asset.ID)
			assert.Equal(t, 1, asset.Version)
		})
	}
}
