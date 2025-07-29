package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// Mock implementations for testing
type MockAssetRepositoryForTeams struct {
	mock.Mock
}

func (m *MockAssetRepositoryForTeams) Save(asset *domain.Asset) error {
	args := m.Called(asset)
	return args.Error(0)
}

func (m *MockAssetRepositoryForTeams) FindByName(name string) (*domain.Asset, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *MockAssetRepositoryForTeams) FindAll() ([]*domain.Asset, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Asset), args.Error(1)
}

func (m *MockAssetRepositoryForTeams) FindByID(id string) (*domain.Asset, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *MockAssetRepositoryForTeams) Delete(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func TestAssignTeamUseCase_NewAssignTeamUseCase(t *testing.T) {
	mockRepo := &MockAssetRepositoryForTeams{}
	useCase := NewAssignTeamUseCase(mockRepo)
	assert.NotNil(t, useCase)
}

func TestAssignTeamUseCase_Execute(t *testing.T) {
	t.Run("should assign owning team to asset successfully", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewAssignTeamUseCase(mockRepo)

		// Create test asset
		asset, err := domain.NewAsset("test-asset", "Test description")
		require.NoError(t, err)

		// Set up mock expectations
		mockRepo.On("FindByName", "test-asset").Return(asset, nil)
		mockRepo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)

		// Execute
		input := AssignTeamInput{
			AssetName:  "test-asset",
			OwningTeam: "team-alpha",
		}

		err = useCase.Execute(input)
		require.NoError(t, err)

		// Verify team was assigned
		assert.Equal(t, "team-alpha", asset.GetOwningTeam())

		mockRepo.AssertExpectations(t)
	})

	t.Run("should assign contributing teams", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewAssignTeamUseCase(mockRepo)

		asset, err := domain.NewAsset("test-asset", "Test description")
		require.NoError(t, err)

		mockRepo.On("FindByName", "test-asset").Return(asset, nil)
		mockRepo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)

		input := AssignTeamInput{
			AssetName:         "test-asset",
			ContributingTeams: []string{"team-beta", "team-gamma"},
		}

		err = useCase.Execute(input)
		require.NoError(t, err)

		teams := asset.GetContributingTeams()
		assert.Contains(t, teams, "team-beta")
		assert.Contains(t, teams, "team-gamma")

		mockRepo.AssertExpectations(t)
	})

	t.Run("should assign both owning and contributing teams", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewAssignTeamUseCase(mockRepo)

		asset, err := domain.NewAsset("test-asset", "Test description")
		require.NoError(t, err)

		mockRepo.On("FindByName", "test-asset").Return(asset, nil)
		mockRepo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)

		input := AssignTeamInput{
			AssetName:         "test-asset",
			OwningTeam:        "team-alpha",
			ContributingTeams: []string{"team-beta"},
		}

		err = useCase.Execute(input)
		require.NoError(t, err)

		assert.Equal(t, "team-alpha", asset.GetOwningTeam())
		teams := asset.GetContributingTeams()
		assert.Contains(t, teams, "team-beta")

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error for empty asset name", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewAssignTeamUseCase(mockRepo)

		input := AssignTeamInput{
			AssetName:  "",
			OwningTeam: "team-alpha",
		}

		err := useCase.Execute(input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "asset name is required")
	})

	t.Run("should return error for asset not found", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewAssignTeamUseCase(mockRepo)

		mockRepo.On("FindByName", "nonexistent").Return(nil, assert.AnError)

		input := AssignTeamInput{
			AssetName:  "nonexistent",
			OwningTeam: "team-alpha",
		}

		err := useCase.Execute(input)
		assert.Error(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error for save failure", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewAssignTeamUseCase(mockRepo)

		asset, err := domain.NewAsset("test-asset", "Test description")
		require.NoError(t, err)

		mockRepo.On("FindByName", "test-asset").Return(asset, nil)
		mockRepo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(assert.AnError)

		input := AssignTeamInput{
			AssetName:  "test-asset",
			OwningTeam: "team-alpha",
		}

		err = useCase.Execute(input)
		assert.Error(t, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestGetAssetTeamsUseCase_NewGetAssetTeamsUseCase(t *testing.T) {
	mockRepo := &MockAssetRepositoryForTeams{}
	useCase := NewGetAssetTeamsUseCase(mockRepo)
	assert.NotNil(t, useCase)
}

func TestGetAssetTeamsUseCase_Execute(t *testing.T) {
	t.Run("should return all asset teams", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewGetAssetTeamsUseCase(mockRepo)

		// Create assets with different teams
		asset1, _ := domain.NewAsset("asset1", "Description 1")
		asset1.SetOwningTeam("team-alpha")
		asset1.AddContributingTeam("team-beta")

		asset2, _ := domain.NewAsset("asset2", "Description 2")
		asset2.SetOwningTeam("team-gamma")

		asset3, _ := domain.NewAsset("asset3", "Description 3")
		// No teams assigned to asset3

		assets := []*domain.Asset{asset1, asset2, asset3}

		mockRepo.On("FindAll").Return(assets, nil)

		result, err := useCase.Execute()
		require.NoError(t, err)
		assert.Len(t, result, 2) // Only assets with teams should be returned

		// Check asset1 teams
		assert.Equal(t, "asset1", result[0].AssetName)
		assert.Equal(t, "team-alpha", result[0].OwningTeam)
		assert.Contains(t, result[0].ContributingTeams, "team-beta")

		// Check asset2 teams
		assert.Equal(t, "asset2", result[1].AssetName)
		assert.Equal(t, "team-gamma", result[1].OwningTeam)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should handle repository error", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewGetAssetTeamsUseCase(mockRepo)

		mockRepo.On("FindAll").Return(nil, assert.AnError)

		_, err := useCase.Execute()
		assert.Error(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return empty list when no assets have teams", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewGetAssetTeamsUseCase(mockRepo)

		asset1, _ := domain.NewAsset("asset1", "Description 1")
		assets := []*domain.Asset{asset1}

		mockRepo.On("FindAll").Return(assets, nil)

		result, err := useCase.Execute()
		require.NoError(t, err)
		assert.Len(t, result, 0)

		mockRepo.AssertExpectations(t)
	})
}

func TestGetAssetTeamInfoUseCase_NewGetAssetTeamInfoUseCase(t *testing.T) {
	mockRepo := &MockAssetRepositoryForTeams{}
	useCase := NewGetAssetTeamInfoUseCase(mockRepo)
	assert.NotNil(t, useCase)
}

func TestGetAssetTeamInfoUseCase_Execute(t *testing.T) {
	t.Run("should return team info for specific asset", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewGetAssetTeamInfoUseCase(mockRepo)

		asset, _ := domain.NewAsset("asset1", "Description 1")
		asset.SetOwningTeam("team-alpha")
		asset.AddContributingTeam("team-beta")

		mockRepo.On("FindByName", "asset1").Return(asset, nil)

		result, err := useCase.Execute("asset1")
		require.NoError(t, err)
		assert.NotNil(t, result)

		assert.Equal(t, "asset1", result.AssetName)
		assert.Equal(t, "team-alpha", result.OwningTeam)
		assert.Contains(t, result.ContributingTeams, "team-beta")

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error for empty asset name", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewGetAssetTeamInfoUseCase(mockRepo)

		_, err := useCase.Execute("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "asset name is required")
	})

	t.Run("should return error for asset not found", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewGetAssetTeamInfoUseCase(mockRepo)

		mockRepo.On("FindByName", "nonexistent").Return(nil, assert.AnError)

		_, err := useCase.Execute("nonexistent")
		assert.Error(t, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestAddContributingTeamUseCase_NewAddContributingTeamUseCase(t *testing.T) {
	mockRepo := &MockAssetRepositoryForTeams{}
	useCase := NewAddContributingTeamUseCase(mockRepo)
	assert.NotNil(t, useCase)
}

func TestAddContributingTeamUseCase_Execute(t *testing.T) {
	t.Run("should add contributing team successfully", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewAddContributingTeamUseCase(mockRepo)

		asset, err := domain.NewAsset("test-asset", "Test description")
		require.NoError(t, err)

		mockRepo.On("FindByName", "test-asset").Return(asset, nil)
		mockRepo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)

		err = useCase.Execute("test-asset", "team-beta")
		require.NoError(t, err)

		teams := asset.GetContributingTeams()
		assert.Contains(t, teams, "team-beta")

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error for empty asset name", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewAddContributingTeamUseCase(mockRepo)

		err := useCase.Execute("", "team-beta")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "asset name is required")
	})

	t.Run("should return error for empty team name", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewAddContributingTeamUseCase(mockRepo)

		err := useCase.Execute("test-asset", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team name is required")
	})

	t.Run("should return error for asset not found", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewAddContributingTeamUseCase(mockRepo)

		mockRepo.On("FindByName", "nonexistent").Return(nil, assert.AnError)

		err := useCase.Execute("nonexistent", "team-beta")
		assert.Error(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error for save failure", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewAddContributingTeamUseCase(mockRepo)

		asset, err := domain.NewAsset("test-asset", "Test description")
		require.NoError(t, err)

		mockRepo.On("FindByName", "test-asset").Return(asset, nil)
		mockRepo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(assert.AnError)

		err = useCase.Execute("test-asset", "team-beta")
		assert.Error(t, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestRemoveContributingTeamUseCase_NewRemoveContributingTeamUseCase(t *testing.T) {
	mockRepo := &MockAssetRepositoryForTeams{}
	useCase := NewRemoveContributingTeamUseCase(mockRepo)
	assert.NotNil(t, useCase)
}

func TestRemoveContributingTeamUseCase_Execute(t *testing.T) {
	t.Run("should remove contributing team successfully", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewRemoveContributingTeamUseCase(mockRepo)

		asset, err := domain.NewAsset("test-asset", "Test description")
		require.NoError(t, err)
		asset.AddContributingTeam("team-beta")
		asset.AddContributingTeam("team-gamma")

		mockRepo.On("FindByName", "test-asset").Return(asset, nil)
		mockRepo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)

		err = useCase.Execute("test-asset", "team-beta")
		require.NoError(t, err)

		teams := asset.GetContributingTeams()
		assert.NotContains(t, teams, "team-beta")
		assert.Contains(t, teams, "team-gamma")

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error for empty asset name", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewRemoveContributingTeamUseCase(mockRepo)

		err := useCase.Execute("", "team-beta")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "asset name is required")
	})

	t.Run("should return error for empty team name", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewRemoveContributingTeamUseCase(mockRepo)

		err := useCase.Execute("test-asset", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team name is required")
	})

	t.Run("should return error for asset not found", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewRemoveContributingTeamUseCase(mockRepo)

		mockRepo.On("FindByName", "nonexistent").Return(nil, assert.AnError)

		err := useCase.Execute("nonexistent", "team-beta")
		assert.Error(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error for save failure", func(t *testing.T) {
		mockRepo := &MockAssetRepositoryForTeams{}
		useCase := NewRemoveContributingTeamUseCase(mockRepo)

		asset, err := domain.NewAsset("test-asset", "Test description")
		require.NoError(t, err)
		asset.AddContributingTeam("team-beta")

		mockRepo.On("FindByName", "test-asset").Return(asset, nil)
		mockRepo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(assert.AnError)

		err = useCase.Execute("test-asset", "team-beta")
		assert.Error(t, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestParseTeamsInput(t *testing.T) {
	t.Run("should parse comma-separated teams", func(t *testing.T) {
		input := "team-alpha,team-beta,team-gamma"
		teams := ParseTeamsInput(input)

		expected := []string{"team-alpha", "team-beta", "team-gamma"}
		assert.Equal(t, expected, teams)
	})

	t.Run("should handle single team", func(t *testing.T) {
		input := "team-alpha"
		teams := ParseTeamsInput(input)

		expected := []string{"team-alpha"}
		assert.Equal(t, expected, teams)
	})

	t.Run("should handle empty input", func(t *testing.T) {
		input := ""
		teams := ParseTeamsInput(input)

		assert.Nil(t, teams)
	})

	t.Run("should trim whitespace", func(t *testing.T) {
		input := " team-alpha , team-beta , team-gamma "
		teams := ParseTeamsInput(input)

		expected := []string{"team-alpha", "team-beta", "team-gamma"}
		assert.Equal(t, expected, teams)
	})

	t.Run("should filter empty teams", func(t *testing.T) {
		input := "team-alpha,,team-beta, ,team-gamma"
		teams := ParseTeamsInput(input)

		expected := []string{"team-alpha", "team-beta", "team-gamma"}
		assert.Equal(t, expected, teams)
	})
}
