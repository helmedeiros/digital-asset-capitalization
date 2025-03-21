package application

import (
	"fmt"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateAsset(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		description   string
		setupMock     func(*MockRepository)
		expectedError string
	}{
		{
			name:        "successful update",
			assetName:   "test-asset",
			description: "Updated description",
			setupMock: func(m *MockRepository) {
				m.findResult = &domain.Asset{
					Name:        "test-asset",
					Description: "Original description",
				}
			},
		},
		{
			name:        "asset not found",
			assetName:   "non-existent",
			description: "Updated description",
			setupMock: func(m *MockRepository) {
				m.findError = fmt.Errorf("not found")
			},
			expectedError: "asset not found",
		},
		{
			name:        "empty description",
			assetName:   "test-asset",
			description: "",
			setupMock: func(m *MockRepository) {
				m.findResult = &domain.Asset{
					Name:        "test-asset",
					Description: "Original description",
				}
			},
			expectedError: "asset description cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setupMock(mockRepo)
			service := NewAssetService(mockRepo)

			err := service.UpdateAsset(tt.assetName, tt.description)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)

			// Verify the mock was called correctly
			assert.True(t, mockRepo.findCalled, "FindByName was not called")
			assert.Equal(t, tt.assetName, mockRepo.findName, "FindByName called with wrong name")
			assert.True(t, mockRepo.saveCalled, "Save was not called")
			if mockRepo.saveAsset != nil {
				assert.Equal(t, tt.description, mockRepo.saveAsset.Description, "Save called with wrong description")
			}
		})
	}
}
