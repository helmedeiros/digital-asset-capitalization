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

func TestCreateAssetUseCase(t *testing.T) {
	tests := []struct {
		name        string
		assetName   string
		description string
		setupMock   func(*testutil.MockAssetRepository)
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid asset creation",
			assetName:   "test-asset",
			description: "Test description",
			setupMock: func(repo *testutil.MockAssetRepository) {
				repo.On("FindByName", "test-asset").Return(nil, errors.New("asset not found"))
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "duplicate asset",
			assetName:   "test-asset",
			description: "Duplicate description",
			setupMock: func(repo *testutil.MockAssetRepository) {
				existingAsset := &domain.Asset{Name: "test-asset"}
				repo.On("FindByName", "test-asset").Return(existingAsset, nil)
			},
			wantErr: true,
			errMsg:  "asset already exists",
		},
		{
			name:        "empty name",
			assetName:   "",
			description: "Test description",
			setupMock: func(repo *testutil.MockAssetRepository) {
				// FindByName is called before validation, so we need to mock it
				repo.On("FindByName", "").Return(nil, errors.New("asset not found"))
			},
			wantErr: true,
			errMsg:  domain.ErrEmptyName.Error(),
		},
		{
			name:        "empty description",
			assetName:   "new-asset",
			description: "",
			setupMock: func(repo *testutil.MockAssetRepository) {
				// FindByName is called before validation, so we need to mock it
				repo.On("FindByName", "new-asset").Return(nil, errors.New("asset not found"))
			},
			wantErr: true,
			errMsg:  domain.ErrEmptyDescription.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := testutil.NewMockAssetRepository()
			tt.setupMock(mockRepo)

			useCase := NewCreateAssetUseCase(mockRepo)
			err := useCase.Execute(tt.assetName, tt.description)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				require.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
