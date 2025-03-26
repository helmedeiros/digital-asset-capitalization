package application

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of AssetRepository
type MockRepository struct {
	saveCalled     bool
	findCalled     bool
	findAllCalled  bool
	deleteCalled   bool
	findByIDCalled bool
	findByIDCalls  int
	saveAsset      *domain.Asset
	findName       string
	findResult     *domain.Asset
	findError      error
	findAllResult  []*domain.Asset
	findAllError   error
	deleteName     string
	deleteError    error
	findByIDResult *domain.Asset
	findByIDError  error
	// For multiple FindByID calls
	findByIDResults []*domain.Asset
	findByIDErrors  []error
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

func (m *MockRepository) FindByID(id string) (*domain.Asset, error) {
	m.findByIDCalled = true
	m.findByIDCalls++

	// If using multiple results
	if len(m.findByIDResults) > 0 && m.findByIDCalls <= len(m.findByIDResults) {
		return m.findByIDResults[m.findByIDCalls-1], m.findByIDErrors[m.findByIDCalls-1]
	}

	return m.findByIDResult, m.findByIDError
}

func TestCreateAsset(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		description   string
		setupMock     func(*MockRepository)
		expectedError error
		checkError    func(error) bool
	}{
		{
			name:        "successful creation",
			assetName:   "test-asset",
			description: "Test description",
			setupMock: func(m *MockRepository) {
				m.findError = errors.New("not found")
				m.findByIDResults = []*domain.Asset{
					nil, // First call (name check)
					nil, // Second call (generated ID check)
				}
				m.findByIDErrors = []error{
					errors.New("not found"), // First call (name check)
					errors.New("not found"), // Second call (generated ID check)
				}
			},
			expectedError: nil,
		},
		{
			name:        "asset already exists by name",
			assetName:   "existing-asset",
			description: "Test description",
			setupMock: func(m *MockRepository) {
				m.findResult = &domain.Asset{
					Name:        "existing-asset",
					Description: "Existing description",
				}
			},
			expectedError: fmt.Errorf("asset with name 'existing-asset' already exists"),
		},
		{
			name:        "name matches existing asset ID",
			assetName:   "existing-id",
			description: "Test description",
			setupMock: func(m *MockRepository) {
				m.findError = errors.New("not found")
				m.findByIDResults = []*domain.Asset{
					{
						ID:          "existing-id",
						Name:        "some-asset",
						Description: "Some description",
					},
				}
				m.findByIDErrors = []error{nil}
			},
			expectedError: fmt.Errorf("cannot create asset with name 'existing-id' as it matches an existing asset's ID"),
		},
		{
			name:        "asset already exists by generated ID",
			assetName:   "test-asset",
			description: "Test description",
			setupMock: func(m *MockRepository) {
				m.findError = errors.New("not found")
				m.findByIDResults = []*domain.Asset{
					nil,
					{
						ID:          "test-id",
						Name:        "test-asset",
						Description: "Test description",
					},
				}
				m.findByIDErrors = []error{
					errors.New("not found"), // First call (name check)
					nil,                     // Second call (generated ID check)
				}
			},
			checkError: func(err error) bool {
				return err != nil && strings.Contains(err.Error(), "asset with ID '") && strings.Contains(err.Error(), "' already exists")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setupMock(mockRepo)
			service := NewAssetService(mockRepo)

			err := service.CreateAsset(tt.assetName, tt.description)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				return
			}

			if tt.checkError != nil {
				assert.True(t, tt.checkError(err), "Error message did not match expected pattern")
				return
			}

			require.NoError(t, err)
			assert.True(t, mockRepo.findCalled, "FindByName was not called")
			assert.True(t, mockRepo.saveCalled, "Save was not called")
			if mockRepo.saveAsset != nil {
				assert.Equal(t, tt.assetName, mockRepo.saveAsset.Name)
				assert.Equal(t, tt.description, mockRepo.saveAsset.Description)
				assert.NotEmpty(t, mockRepo.saveAsset.ID, "Asset ID should not be empty")
				assert.Len(t, mockRepo.saveAsset.ID, 16, "Asset ID should be 16 characters long")
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
				require.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				return
			}

			require.NoError(t, err)
			assert.True(t, mockRepo.findAllCalled, "FindAll was not called")
			assert.Len(t, assets, len(tt.expectedAssets), "unexpected number of assets")

			for i, asset := range assets {
				assert.Equal(t, tt.expectedAssets[i].Name, asset.Name, "unexpected asset name")
				assert.Equal(t, tt.expectedAssets[i].Description, asset.Description, "unexpected asset description")
			}
		})
	}
}

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
			assert.True(t, mockRepo.findCalled, "FindByName was not called")
			assert.Equal(t, tt.assetName, mockRepo.findName, "FindByName called with wrong name")
			assert.True(t, mockRepo.saveCalled, "Save was not called")
			if mockRepo.saveAsset != nil {
				assert.Equal(t, tt.description, mockRepo.saveAsset.Description, "Save called with wrong description")
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
			expectedError: fmt.Errorf("asset not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setupMock(mockRepo)
			service := NewAssetService(mockRepo)

			err := service.UpdateDocumentation(tt.assetName)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				return
			}

			require.NoError(t, err)
			assert.True(t, mockRepo.findCalled, "FindByName was not called")
			assert.True(t, mockRepo.saveCalled, "Save was not called")
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
			expectedError: fmt.Errorf("task count cannot be negative"),
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
			expectedError: fmt.Errorf("asset not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setupMock(mockRepo)

			err := tt.operation(mockRepo, tt.assetName)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				return
			}

			require.NoError(t, err)
			assert.True(t, mockRepo.findCalled, "FindByName was not called")
			assert.True(t, mockRepo.saveCalled, "Save was not called")
		})
	}
}

func TestGetAsset(t *testing.T) {
	tests := []struct {
		name          string
		identifier    string
		setupMock     func(*MockRepository)
		expectedAsset *domain.Asset
		expectedError error
	}{
		{
			name:       "find by name",
			identifier: "test-asset",
			setupMock: func(m *MockRepository) {
				m.findResult = &domain.Asset{
					ID:          "test-id",
					Name:        "test-asset",
					Description: "Test description",
				}
			},
			expectedAsset: &domain.Asset{
				ID:          "test-id",
				Name:        "test-asset",
				Description: "Test description",
			},
			expectedError: nil,
		},
		{
			name:       "find by ID",
			identifier: "test-id",
			setupMock: func(m *MockRepository) {
				m.findError = errors.New("not found")
				m.findByIDResult = &domain.Asset{
					ID:          "test-id",
					Name:        "test-asset",
					Description: "Test description",
				}
			},
			expectedAsset: &domain.Asset{
				ID:          "test-id",
				Name:        "test-asset",
				Description: "Test description",
			},
			expectedError: nil,
		},
		{
			name:       "not found",
			identifier: "non-existent",
			setupMock: func(m *MockRepository) {
				m.findError = errors.New("not found")
				m.findByIDError = errors.New("not found")
			},
			expectedAsset: nil,
			expectedError: fmt.Errorf("asset not found by name or ID: non-existent"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setupMock(mockRepo)
			service := NewAssetService(mockRepo)

			asset, err := service.GetAsset(tt.identifier)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedAsset, asset)
			assert.True(t, mockRepo.findCalled, "FindByName was not called")
			if tt.name == "find by ID" {
				assert.True(t, mockRepo.findByIDCalled, "FindByID should be called when looking up by ID")
			} else {
				assert.False(t, mockRepo.findByIDCalled, "FindByID should not be called when looking up by name")
			}
		})
	}
}
