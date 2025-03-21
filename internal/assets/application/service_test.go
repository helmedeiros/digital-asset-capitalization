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
		oldName       string
		newName       string
		description   string
		setupMock     func(*MockRepository)
		expectedError string
	}{
		{
			name:        "successful update without name change",
			oldName:     "test-asset",
			newName:     "test-asset",
			description: "Updated description",
			setupMock: func(m *MockRepository) {
				m.findResult = &domain.Asset{
					Name:        "test-asset",
					Description: "Original description",
				}
			},
		},
		{
			name:        "successful update with name change",
			oldName:     "old-name",
			newName:     "new-name",
			description: "Updated description",
			setupMock: func(m *MockRepository) {
				m.findResult = &domain.Asset{
					Name:        "old-name",
					Description: "Original description",
				}
			},
		},
		{
			name:        "asset not found",
			oldName:     "non-existent",
			newName:     "new-name",
			description: "Updated description",
			setupMock: func(m *MockRepository) {
				m.findError = fmt.Errorf("not found")
			},
			expectedError: "asset not found",
		},
		{
			name:        "empty description",
			oldName:     "test-asset",
			newName:     "test-asset",
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

			err := service.UpdateAsset(tt.oldName, tt.newName, tt.description)
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
			if mockRepo.findName != tt.oldName {
				t.Errorf("FindByName called with %q, expected %q", mockRepo.findName, tt.oldName)
			}
			if tt.oldName != tt.newName {
				if !mockRepo.deleteCalled {
					t.Error("Delete was not called when name changed")
				}
				if mockRepo.deleteName != tt.oldName {
					t.Errorf("Delete called with %q, expected %q", mockRepo.deleteName, tt.oldName)
				}
			}
			if !mockRepo.saveCalled {
				t.Error("Save was not called")
			}
			if mockRepo.saveAsset != nil && mockRepo.saveAsset.Name != tt.newName {
				t.Errorf("Save called with name %q, expected %q", mockRepo.saveAsset.Name, tt.newName)
			}
		})
	}
}
