package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// MockConfigurationRepository for testing
type MockConfigurationRepository struct {
	mock.Mock
}

func (m *MockConfigurationRepository) LoadJiraConfig() (*domain.JiraConfig, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.JiraConfig), args.Error(1)
}

func (m *MockConfigurationRepository) SaveJiraConfig(config *domain.JiraConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockConfigurationRepository) LoadTeamConfig() (*domain.TeamConfig, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TeamConfig), args.Error(1)
}

func (m *MockConfigurationRepository) SaveTeamConfig(config *domain.TeamConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockConfigurationRepository) ConfigExists() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *MockConfigurationRepository) InitializeConfigDirectory() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewConfigService(t *testing.T) {
	repo := &MockConfigurationRepository{}
	service := NewConfigService(repo)

	assert.NotNil(t, service)
	assert.Equal(t, repo, service.repo)
}

func TestConfigService_GetJiraConfig(t *testing.T) {
	t.Run("should return jira config successfully", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		expectedConfig, err := domain.NewJiraConfig(
			"https://test.atlassian.net",
			"test@example.com",
			"test-token",
		)
		require.NoError(t, err)

		repo.On("LoadJiraConfig").Return(expectedConfig, nil)

		config, err := service.GetJiraConfig()

		require.NoError(t, err)
		assert.Equal(t, expectedConfig, config)
		repo.AssertExpectations(t)
	})

	t.Run("should return error when repository fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		repo.On("LoadJiraConfig").Return(nil, errors.New("repository error"))

		config, err := service.GetJiraConfig()

		require.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "failed to load Jira configuration")
		repo.AssertExpectations(t)
	})
}

func TestConfigService_GetTeamConfig(t *testing.T) {
	t.Run("should return team config successfully", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"helio.medeiros", "alice.smith"},
			"BE": {"bob.jones"},
		}
		expectedConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(expectedConfig, nil)

		config, err := service.GetTeamConfig()

		require.NoError(t, err)
		assert.Equal(t, expectedConfig, config)
		repo.AssertExpectations(t)
	})

	t.Run("should return error when repository fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		repo.On("LoadTeamConfig").Return(nil, errors.New("repository error"))

		config, err := service.GetTeamConfig()

		require.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "failed to load team configuration")
		repo.AssertExpectations(t)
	})
}

func TestConfigService_GetTeamMapForSprint(t *testing.T) {
	t.Run("should convert team config to sprint format successfully", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		// Create test team config in our clean domain format
		teams := map[string][]string{
			"FN": {"helio.medeiros", "alice.smith"},
			"BE": {"bob.jones"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		teamMap, err := service.GetTeamMapForSprint()

		require.NoError(t, err)
		require.NotNil(t, teamMap)

		// Verify conversion to sprint format
		fnTeam, exists := teamMap.GetTeam("FN")
		require.True(t, exists)
		assert.Equal(t, []string{"helio.medeiros", "alice.smith"}, fnTeam.Team)

		beTeam, exists := teamMap.GetTeam("BE")
		require.True(t, exists)
		assert.Equal(t, []string{"bob.jones"}, beTeam.Team)

		// Verify team membership methods work
		assert.True(t, fnTeam.IsTeamMember("helio.medeiros"))
		assert.True(t, beTeam.IsTeamMember("bob.jones"))
		assert.False(t, fnTeam.IsTeamMember("bob.jones"))

		repo.AssertExpectations(t)
	})

	t.Run("should return error when team config loading fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		repo.On("LoadTeamConfig").Return(nil, errors.New("repository error"))

		teamMap, err := service.GetTeamMapForSprint()

		require.Error(t, err)
		assert.Nil(t, teamMap)
		assert.Contains(t, err.Error(), "failed to get team configuration")
		repo.AssertExpectations(t)
	})

	t.Run("should handle empty team config", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		emptyTeamConfig, err := domain.NewTeamConfig(map[string][]string{})
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(emptyTeamConfig, nil)

		teamMap, err := service.GetTeamMapForSprint()

		require.NoError(t, err)
		assert.NotNil(t, teamMap)
		assert.Empty(t, teamMap)
		repo.AssertExpectations(t)
	})
}

func TestConfigService_GetTeamForProject(t *testing.T) {
	t.Run("should return team members for existing project", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"helio.medeiros", "alice.smith"},
			"BE": {"bob.jones"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		members, err := service.GetTeamForProject("FN")

		require.NoError(t, err)
		assert.Equal(t, []string{"helio.medeiros", "alice.smith"}, members)
		repo.AssertExpectations(t)
	})

	t.Run("should return error for non-existing project", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"helio.medeiros", "alice.smith"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		members, err := service.GetTeamForProject("XYZ")

		require.Error(t, err)
		assert.Nil(t, members)
		assert.Contains(t, err.Error(), "project XYZ not found in team configuration")
		repo.AssertExpectations(t)
	})
}

func TestConfigService_IsTeamMember(t *testing.T) {
	t.Run("should return true for existing team member", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"helio.medeiros", "alice.smith"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		isMember, err := service.IsTeamMember("FN", "helio.medeiros")

		require.NoError(t, err)
		assert.True(t, isMember)
		repo.AssertExpectations(t)
	})

	t.Run("should return false for non-existing team member", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"helio.medeiros", "alice.smith"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		isMember, err := service.IsTeamMember("FN", "bob.jones")

		require.NoError(t, err)
		assert.False(t, isMember)
		repo.AssertExpectations(t)
	})

	t.Run("should return false for non-existing project", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"helio.medeiros", "alice.smith"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		isMember, err := service.IsTeamMember("XYZ", "helio.medeiros")

		require.NoError(t, err)
		assert.False(t, isMember)
		repo.AssertExpectations(t)
	})
}

func TestConfigService_ConfigExists(t *testing.T) {
	t.Run("should return true when config exists", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		repo.On("ConfigExists").Return(true, nil)

		exists, err := service.ConfigExists()

		require.NoError(t, err)
		assert.True(t, exists)
		repo.AssertExpectations(t)
	})

	t.Run("should return false when config doesn't exist", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		repo.On("ConfigExists").Return(false, nil)

		exists, err := service.ConfigExists()

		require.NoError(t, err)
		assert.False(t, exists)
		repo.AssertExpectations(t)
	})
}

func TestConfigService_InitializeConfigDirectory(t *testing.T) {
	t.Run("should initialize directory successfully", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		repo.On("InitializeConfigDirectory").Return(nil)

		err := service.InitializeConfigDirectory()

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("should return error when initialization fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		repo.On("InitializeConfigDirectory").Return(errors.New("permission denied"))

		err := service.InitializeConfigDirectory()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
		repo.AssertExpectations(t)
	})
}
