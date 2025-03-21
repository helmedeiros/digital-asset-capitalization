package application

import (
	"errors"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// MockRepository is a mock implementation of AssetRepository
type MockRepository struct {
	saveCalled    bool
	findCalled    bool
	findAllCalled bool
	deleteCalled  bool
	saveAsset     *domain.Asset
	findName      string
	findResult    *domain.Asset
	findError     error
	findAllResult []*domain.Asset
	findAllError  error
	deleteName    string
	deleteError   error
}

func (m *MockRepository) Save(asset *domain.Asset) error {
	m.saveCalled = true
	m.saveAsset = asset
	return nil
}

func (m *MockRepository) FindByName(name string) (*domain.Asset, error) {
	m.findCalled = true
	m.findName = name
	return m.findResult, m.findError
}

func (m *MockRepository) FindAll() ([]*domain.Asset, error) {
	m.findAllCalled = true
	return m.findAllResult, m.findAllError
}

func (m *MockRepository) Delete(name string) error {
	m.deleteCalled = true
	m.deleteName = name
	return m.deleteError
}

func TestCreateAsset(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		description   string
		setupMock     func(*MockRepository)
		expectedError error
	}{
		{
			name:        "successful creation",
			assetName:   "test-asset",
			description: "Test description",
			setupMock: func(m *MockRepository) {
				m.findError = errors.New("not found")
			},
			expectedError: nil,
		},
		{
			name:        "asset already exists",
			assetName:   "existing-asset",
			description: "Test description",
			setupMock: func(m *MockRepository) {
				m.findResult = &domain.Asset{
					Name:        "existing-asset",
					Description: "Existing description",
				}
			},
			expectedError: errors.New("asset already exists"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setupMock(mockRepo)
			service := NewAssetService(mockRepo)

			err := service.CreateAsset(tt.assetName, tt.description)
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !mockRepo.findCalled {
				t.Error("FindByName was not called")
			}
			if tt.expectedError == nil && !mockRepo.saveCalled {
				t.Error("Save was not called")
			}
		})
	}
}

func TestListAssets(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockRepository)
		expectedError  error
		expectedAssets []*domain.Asset
	}{
		{
			name: "successful listing",
			setupMock: func(m *MockRepository) {
				m.findAllResult = []*domain.Asset{
					{Name: "asset1", Description: "Description 1"},
					{Name: "asset2", Description: "Description 2"},
				}
			},
			expectedError: nil,
			expectedAssets: []*domain.Asset{
				{Name: "asset1", Description: "Description 1"},
				{Name: "asset2", Description: "Description 2"},
			},
		},
		{
			name: "empty list",
			setupMock: func(m *MockRepository) {
				m.findAllResult = []*domain.Asset{}
			},
			expectedError:  nil,
			expectedAssets: []*domain.Asset{},
		},
		{
			name: "repository error",
			setupMock: func(m *MockRepository) {
				m.findAllError = errors.New("repository error")
			},
			expectedError:  errors.New("repository error"),
			expectedAssets: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setupMock(mockRepo)
			service := NewAssetService(mockRepo)

			assets, err := service.ListAssets()
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !mockRepo.findAllCalled {
				t.Error("FindAll was not called")
			}
			if len(assets) != len(tt.expectedAssets) {
				t.Errorf("expected %d assets, got %d", len(tt.expectedAssets), len(assets))
			}
			for i, asset := range assets {
				if asset.Name != tt.expectedAssets[i].Name {
					t.Errorf("expected asset name %s, got %s", tt.expectedAssets[i].Name, asset.Name)
				}
				if asset.Description != tt.expectedAssets[i].Description {
					t.Errorf("expected asset description %s, got %s", tt.expectedAssets[i].Description, asset.Description)
				}
			}
		})
	}
}

func TestUpdateDocumentation(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		setupMock     func(*MockRepository)
		expectedError error
	}{
		{
			name:      "successful update",
			assetName: "test-asset",
			setupMock: func(m *MockRepository) {
				m.findResult = &domain.Asset{
					Name:        "test-asset",
					Description: "Test description",
				}
			},
			expectedError: nil,
		},
		{
			name:      "asset not found",
			assetName: "non-existent",
			setupMock: func(m *MockRepository) {
				m.findError = errors.New("not found")
			},
			expectedError: errors.New("asset not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setupMock(mockRepo)
			service := NewAssetService(mockRepo)

			err := service.UpdateDocumentation(tt.assetName)
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !mockRepo.findCalled {
				t.Error("FindByName was not called")
			}
			if tt.expectedError == nil && !mockRepo.saveCalled {
				t.Error("Save was not called")
			}
		})
	}
}

func TestTaskCountOperations(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		operation     func(*MockRepository, string) error
		setupMock     func(*MockRepository)
		expectedError error
	}{
		{
			name:      "increment success",
			assetName: "test-asset",
			operation: func(mockRepo *MockRepository, name string) error {
				service := NewAssetService(mockRepo)
				return service.IncrementTaskCount(name)
			},
			setupMock: func(m *MockRepository) {
				m.findResult = &domain.Asset{
					Name:        "test-asset",
					Description: "Test description",
				}
			},
			expectedError: nil,
		},
		{
			name:      "decrement success",
			assetName: "test-asset",
			operation: func(mockRepo *MockRepository, name string) error {
				service := NewAssetService(mockRepo)
				return service.DecrementTaskCount(name)
			},
			setupMock: func(m *MockRepository) {
				m.findResult = &domain.Asset{
					Name:                "test-asset",
					Description:         "Test description",
					AssociatedTaskCount: 1,
				}
			},
			expectedError: nil,
		},
		{
			name:      "decrement below zero",
			assetName: "test-asset",
			operation: func(mockRepo *MockRepository, name string) error {
				service := NewAssetService(mockRepo)
				return service.DecrementTaskCount(name)
			},
			setupMock: func(m *MockRepository) {
				m.findResult = &domain.Asset{
					Name:                "test-asset",
					Description:         "Test description",
					AssociatedTaskCount: 0,
				}
			},
			expectedError: errors.New("task count cannot be negative"),
		},
		{
			name:      "asset not found",
			assetName: "non-existent",
			operation: func(mockRepo *MockRepository, name string) error {
				service := NewAssetService(mockRepo)
				return service.IncrementTaskCount(name)
			},
			setupMock: func(m *MockRepository) {
				m.findError = errors.New("not found")
			},
			expectedError: errors.New("asset not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setupMock(mockRepo)

			err := tt.operation(mockRepo, tt.assetName)
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !mockRepo.findCalled {
				t.Error("FindByName was not called")
			}
			if tt.expectedError == nil && !mockRepo.saveCalled {
				t.Error("Save was not called")
			}
		})
	}
}
