package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/confluence"
)

// PublishMockAssetRepository is a mock for the asset repository
type PublishMockAssetRepository struct {
	mock.Mock
}

func (m *PublishMockAssetRepository) Save(asset *domain.Asset) error {
	args := m.Called(asset)
	return args.Error(0)
}

func (m *PublishMockAssetRepository) FindByName(name string) (*domain.Asset, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *PublishMockAssetRepository) FindByID(id string) (*domain.Asset, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *PublishMockAssetRepository) FindAll() ([]*domain.Asset, error) {
	args := m.Called()
	return args.Get(0).([]*domain.Asset), args.Error(1)
}

func (m *PublishMockAssetRepository) Delete(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// MockConfluencePublisher is a mock for the Confluence publisher
type MockConfluencePublisher struct {
	mock.Mock
}

func (m *MockConfluencePublisher) CreatePage(ctx context.Context, title, spaceKey, content, parentPageID string) (*confluence.PagePublishResult, error) {
	args := m.Called(ctx, title, spaceKey, content, parentPageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*confluence.PagePublishResult), args.Error(1)
}

func (m *MockConfluencePublisher) AddLabels(ctx context.Context, pageID string, labels []string) error {
	args := m.Called(ctx, pageID, labels)
	return args.Error(0)
}

func (m *MockConfluencePublisher) PageExistsByTitle(ctx context.Context, spaceKey, title string) (bool, string, error) {
	args := m.Called(ctx, spaceKey, title)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockConfluencePublisher) UpdatePage(ctx context.Context, pageID, title, spaceKey, content string) (*confluence.PagePublishResult, error) {
	args := m.Called(ctx, pageID, title, spaceKey, content)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*confluence.PagePublishResult), args.Error(1)
}

// MockIDGenerator is a mock for the ID generator
type MockIDGenerator struct {
	mock.Mock
}

func (m *MockIDGenerator) GenerateID(name string) string {
	args := m.Called(name)
	return args.String(0)
}

func TestPublishToConfluenceUseCase_Execute(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	launchDate := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	t.Run("successful publish", func(t *testing.T) {
		mockRepo := new(PublishMockAssetRepository)
		mockPublisher := new(MockConfluencePublisher)
		mockIDGen := new(MockIDGenerator)

		asset := &domain.Asset{
			ID:         "cap-asset-test-asset",
			Name:       "Test Asset",
			Why:        "Why content",
			Benefits:   "Benefits content",
			How:        "How content",
			Metrics:    "Metrics content",
			Status:     "live",
			LaunchDate: launchDate,
		}

		mockRepo.On("FindByName", "Test Asset").Return(asset, nil)
		mockPublisher.On("PageExistsByTitle", ctx, "SPACE", "Test Asset").Return(false, "", nil)
		mockPublisher.On("CreatePage", ctx, "Test Asset", "SPACE", mock.Anything, mock.Anything).Return(&confluence.PagePublishResult{
			PageID:   "12345",
			PageURL:  "https://confluence.example.com/wiki/spaces/SPACE/pages/12345/Test+Asset",
			SpaceKey: "SPACE",
			Title:    "Test Asset",
			Created:  true,
		}, nil)
		mockPublisher.On("AddLabels", ctx, "12345", []string{"cap-asset", "cap-asset-test-asset"}).Return(nil)
		mockRepo.On("Save", mock.Anything).Return(nil)

		useCase := NewPublishToConfluenceUseCase(mockRepo, mockPublisher, mockIDGen)

		result, err := useCase.Execute(ctx, PublishToConfluenceInput{
			AssetName: "Test Asset",
			SpaceKey:  "SPACE",
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Asset", result.AssetName)
		assert.Equal(t, "12345", result.PageID)
		assert.Equal(t, "SPACE", result.SpaceKey)
		assert.True(t, result.Created)
		assert.True(t, result.DocLinkSaved)
		assert.Contains(t, result.Labels, "cap-asset")
		assert.Contains(t, result.Labels, "cap-asset-test-asset")

		mockRepo.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
	})

	t.Run("dry run mode", func(t *testing.T) {
		mockRepo := new(PublishMockAssetRepository)
		mockPublisher := new(MockConfluencePublisher)
		mockIDGen := new(MockIDGenerator)

		asset := &domain.Asset{
			ID:   "cap-asset-dry-run",
			Name: "Dry Run Asset",
		}

		mockRepo.On("FindByName", "Dry Run Asset").Return(asset, nil)
		mockPublisher.On("PageExistsByTitle", ctx, "SPACE", "Dry Run Asset").Return(false, "", nil)

		useCase := NewPublishToConfluenceUseCase(mockRepo, mockPublisher, mockIDGen)

		result, err := useCase.Execute(ctx, PublishToConfluenceInput{
			AssetName: "Dry Run Asset",
			SpaceKey:  "SPACE",
			DryRun:    true,
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Dry Run Asset", result.AssetName)
		assert.False(t, result.Created)
		assert.NotEmpty(t, result.Preview)
		assert.Contains(t, result.Preview, "<h1>Asset Capitalisation</h1>")

		// CreatePage should not be called in dry-run mode
		mockPublisher.AssertNotCalled(t, "CreatePage")
		mockPublisher.AssertNotCalled(t, "AddLabels")
		mockRepo.AssertNotCalled(t, "Save")
	})

	t.Run("page already exists", func(t *testing.T) {
		mockRepo := new(PublishMockAssetRepository)
		mockPublisher := new(MockConfluencePublisher)
		mockIDGen := new(MockIDGenerator)

		asset := &domain.Asset{
			ID:   "cap-asset-existing",
			Name: "Existing Asset",
		}

		mockRepo.On("FindByName", "Existing Asset").Return(asset, nil)
		mockPublisher.On("PageExistsByTitle", ctx, "SPACE", "Existing Asset").Return(true, "99999", nil)

		useCase := NewPublishToConfluenceUseCase(mockRepo, mockPublisher, mockIDGen)

		result, err := useCase.Execute(ctx, PublishToConfluenceInput{
			AssetName: "Existing Asset",
			SpaceKey:  "SPACE",
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "already exists")
		assert.Contains(t, err.Error(), "99999")
	})

	t.Run("asset not found", func(t *testing.T) {
		mockRepo := new(PublishMockAssetRepository)
		mockPublisher := new(MockConfluencePublisher)
		mockIDGen := new(MockIDGenerator)

		mockRepo.On("FindByName", "Unknown Asset").Return(nil, errors.New("asset not found"))

		useCase := NewPublishToConfluenceUseCase(mockRepo, mockPublisher, mockIDGen)

		result, err := useCase.Execute(ctx, PublishToConfluenceInput{
			AssetName: "Unknown Asset",
			SpaceKey:  "SPACE",
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to find asset")
	})

	t.Run("empty asset name", func(t *testing.T) {
		mockRepo := new(PublishMockAssetRepository)
		mockPublisher := new(MockConfluencePublisher)
		mockIDGen := new(MockIDGenerator)

		useCase := NewPublishToConfluenceUseCase(mockRepo, mockPublisher, mockIDGen)

		result, err := useCase.Execute(ctx, PublishToConfluenceInput{
			AssetName: "",
			SpaceKey:  "SPACE",
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "asset name is required")
	})

	t.Run("empty space key", func(t *testing.T) {
		mockRepo := new(PublishMockAssetRepository)
		mockPublisher := new(MockConfluencePublisher)
		mockIDGen := new(MockIDGenerator)

		useCase := NewPublishToConfluenceUseCase(mockRepo, mockPublisher, mockIDGen)

		result, err := useCase.Execute(ctx, PublishToConfluenceInput{
			AssetName: "Test Asset",
			SpaceKey:  "",
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "space key is required")
	})

	t.Run("page creation fails", func(t *testing.T) {
		mockRepo := new(PublishMockAssetRepository)
		mockPublisher := new(MockConfluencePublisher)
		mockIDGen := new(MockIDGenerator)

		asset := &domain.Asset{
			ID:   "cap-asset-fail",
			Name: "Fail Asset",
		}

		mockRepo.On("FindByName", "Fail Asset").Return(asset, nil)
		mockPublisher.On("PageExistsByTitle", ctx, "SPACE", "Fail Asset").Return(false, "", nil)
		mockPublisher.On("CreatePage", ctx, "Fail Asset", "SPACE", mock.Anything, mock.Anything).Return(nil, errors.New("confluence API error"))

		useCase := NewPublishToConfluenceUseCase(mockRepo, mockPublisher, mockIDGen)

		result, err := useCase.Execute(ctx, PublishToConfluenceInput{
			AssetName: "Fail Asset",
			SpaceKey:  "SPACE",
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to create page")
	})

	t.Run("label addition fails but page is created", func(t *testing.T) {
		mockRepo := new(PublishMockAssetRepository)
		mockPublisher := new(MockConfluencePublisher)
		mockIDGen := new(MockIDGenerator)

		asset := &domain.Asset{
			ID:   "cap-asset-label-fail",
			Name: "Label Fail Asset",
		}

		mockRepo.On("FindByName", "Label Fail Asset").Return(asset, nil)
		mockPublisher.On("PageExistsByTitle", ctx, "SPACE", "Label Fail Asset").Return(false, "", nil)
		mockPublisher.On("CreatePage", ctx, "Label Fail Asset", "SPACE", mock.Anything, mock.Anything).Return(&confluence.PagePublishResult{
			PageID:   "12345",
			PageURL:  "https://confluence.example.com/wiki/spaces/SPACE/pages/12345",
			SpaceKey: "SPACE",
			Title:    "Label Fail Asset",
			Created:  true,
		}, nil)
		mockPublisher.On("AddLabels", ctx, "12345", mock.Anything).Return(errors.New("label API error"))
		mockRepo.On("Save", mock.Anything).Return(nil)

		useCase := NewPublishToConfluenceUseCase(mockRepo, mockPublisher, mockIDGen)

		result, err := useCase.Execute(ctx, PublishToConfluenceInput{
			AssetName: "Label Fail Asset",
			SpaceKey:  "SPACE",
		})

		// Should succeed despite label failure
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Created)
		assert.Equal(t, "12345", result.PageID)
	})

	t.Run("uses ID generator when asset ID is not in cap-asset format", func(t *testing.T) {
		mockRepo := new(PublishMockAssetRepository)
		mockPublisher := new(MockConfluencePublisher)
		mockIDGen := new(MockIDGenerator)

		asset := &domain.Asset{
			ID:   "old-format-id",
			Name: "New Format Asset",
		}

		mockRepo.On("FindByName", "New Format Asset").Return(asset, nil)
		mockPublisher.On("PageExistsByTitle", ctx, "SPACE", "New Format Asset").Return(false, "", nil)
		mockPublisher.On("CreatePage", ctx, "New Format Asset", "SPACE", mock.Anything, mock.Anything).Return(&confluence.PagePublishResult{
			PageID:   "12345",
			PageURL:  "https://confluence.example.com/wiki/spaces/SPACE/pages/12345",
			SpaceKey: "SPACE",
			Title:    "New Format Asset",
			Created:  true,
		}, nil)
		mockIDGen.On("GenerateID", "New Format Asset").Return("cap-asset-new-format-asset")
		mockPublisher.On("AddLabels", ctx, "12345", []string{"cap-asset", "cap-asset-new-format-asset"}).Return(nil)
		mockRepo.On("Save", mock.Anything).Return(nil)

		useCase := NewPublishToConfluenceUseCase(mockRepo, mockPublisher, mockIDGen)

		result, err := useCase.Execute(ctx, PublishToConfluenceInput{
			AssetName: "New Format Asset",
			SpaceKey:  "SPACE",
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.Labels, "cap-asset-new-format-asset")
		mockIDGen.AssertCalled(t, "GenerateID", "New Format Asset")
	})
}

func TestPublishToConfluenceUseCase_getAssetLabel(t *testing.T) {
	t.Parallel()
	mockIDGen := new(MockIDGenerator)
	useCase := &PublishToConfluenceUseCase{
		idGenerator: mockIDGen,
	}

	t.Run("uses existing ID if valid cap-asset format", func(t *testing.T) {
		asset := &domain.Asset{
			ID:   "cap-asset-valid-id",
			Name: "Valid Asset",
		}

		label := useCase.getAssetLabel(asset)
		assert.Equal(t, "cap-asset-valid-id", label)
		mockIDGen.AssertNotCalled(t, "GenerateID")
	})

	t.Run("generates new ID if not cap-asset format", func(t *testing.T) {
		mockIDGen := new(MockIDGenerator)
		useCase := &PublishToConfluenceUseCase{
			idGenerator: mockIDGen,
		}

		asset := &domain.Asset{
			ID:   "some-other-format",
			Name: "Other Asset",
		}

		mockIDGen.On("GenerateID", "Other Asset").Return("cap-asset-other-asset")

		label := useCase.getAssetLabel(asset)
		assert.Equal(t, "cap-asset-other-asset", label)
		mockIDGen.AssertCalled(t, "GenerateID", "Other Asset")
	})
}
