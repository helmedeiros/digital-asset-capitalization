package application

import (
	"fmt"
	"strings"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
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
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify the mock was called correctly
			if !mockRepo.findCalled {
				t.Error("FindByName was not called")
			}
			if mockRepo.findName != tt.assetName {
				t.Errorf("FindByName called with %q, expected %q", mockRepo.findName, tt.assetName)
			}
			if !mockRepo.saveCalled {
				t.Error("Save was not called")
			}
			if mockRepo.saveAsset != nil && mockRepo.saveAsset.Description != tt.description {
				t.Errorf("Save called with description %q, expected %q", mockRepo.saveAsset.Description, tt.description)
			}
		})
	}
}
