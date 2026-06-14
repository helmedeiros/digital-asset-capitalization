package service

import (
	"errors"
	"testing"
	"time"

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

	t.Run("should propagate excluded issue types to sprint format", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"COP": {"alice", "bob"},
		}
		excludedTypes := map[string][]string{
			"COP": {"Experiment"},
		}
		teamConfig, err := domain.NewTeamConfigWithExcludedTypes(teams, nil, nil, nil, nil, nil, excludedTypes)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		teamMap, err := service.GetTeamMapForSprint()

		require.NoError(t, err)
		copTeam, exists := teamMap.GetTeam("COP")
		require.True(t, exists)
		assert.Equal(t, []string{"Experiment"}, copTeam.ExcludedIssueTypes)
		assert.True(t, copTeam.IsExcludedIssueType("Experiment"))
		assert.False(t, copTeam.IsExcludedIssueType("Story"))
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

func TestConfigService_SetTribeForProject(t *testing.T) {
	t.Run("should set tribe for existing project successfully", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)
		repo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(nil)

		err = service.SetTribeForProject("FN", "Engineering")

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("should return error when project does not exist", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		err = service.SetTribeForProject("NONEXISTENT", "Engineering")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set tribe")
		repo.AssertExpectations(t)
	})

	t.Run("should return error when load fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		repo.On("LoadTeamConfig").Return(nil, errors.New("repository error"))

		err := service.SetTribeForProject("FN", "Engineering")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load team configuration")
		repo.AssertExpectations(t)
	})

	t.Run("should return error when save fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)
		repo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(errors.New("save error"))

		err = service.SetTribeForProject("FN", "Engineering")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save team configuration")
		repo.AssertExpectations(t)
	})
}

func TestConfigService_GetTribeForProject(t *testing.T) {
	t.Run("should return tribe for project with tribe set", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		nicknames := map[string][]string{}
		tribes := map[string]string{
			"FN": "Engineering",
		}
		teamConfig, err := domain.NewTeamConfigWithTribes(teams, nicknames, tribes)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		tribe, err := service.GetTribeForProject("FN")

		require.NoError(t, err)
		assert.Equal(t, "Engineering", tribe)
		repo.AssertExpectations(t)
	})

	t.Run("should return empty string for project without tribe", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		tribe, err := service.GetTribeForProject("FN")

		require.NoError(t, err)
		assert.Equal(t, "", tribe)
		repo.AssertExpectations(t)
	})

	t.Run("should return error when load fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		repo.On("LoadTeamConfig").Return(nil, errors.New("repository error"))

		tribe, err := service.GetTribeForProject("FN")

		require.Error(t, err)
		assert.Equal(t, "", tribe)
		assert.Contains(t, err.Error(), "failed to load team configuration")
		repo.AssertExpectations(t)
	})
}

func TestConfigService_SetCompanyForProject(t *testing.T) {
	t.Run("should set company for existing project successfully", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)
		repo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(nil)

		err = service.SetCompanyForProject("FN", "TechCorp")

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("should return error when project does not exist", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		err = service.SetCompanyForProject("NONEXISTENT", "TechCorp")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set company")
		repo.AssertExpectations(t)
	})

	t.Run("should return error when load fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		repo.On("LoadTeamConfig").Return(nil, errors.New("repository error"))

		err := service.SetCompanyForProject("FN", "TechCorp")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load team configuration")
		repo.AssertExpectations(t)
	})

	t.Run("should return error when save fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)
		repo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(errors.New("save error"))

		err = service.SetCompanyForProject("FN", "TechCorp")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save team configuration")
		repo.AssertExpectations(t)
	})
}

func TestConfigService_GetCompanyForProject(t *testing.T) {
	t.Run("should return company for project with company set", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		nicknames := map[string][]string{}
		tribes := map[string]string{}
		companies := map[string]string{
			"FN": "TechCorp",
		}
		teamConfig, err := domain.NewTeamConfigFull(teams, nicknames, tribes, companies)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		company, err := service.GetCompanyForProject("FN")

		require.NoError(t, err)
		assert.Equal(t, "TechCorp", company)
		repo.AssertExpectations(t)
	})

	t.Run("should return empty string for project without company", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		company, err := service.GetCompanyForProject("FN")

		require.NoError(t, err)
		assert.Equal(t, "", company)
		repo.AssertExpectations(t)
	})

	t.Run("should return error when load fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)

		repo.On("LoadTeamConfig").Return(nil, errors.New("repository error"))

		company, err := service.GetCompanyForProject("FN")

		require.Error(t, err)
		assert.Equal(t, "", company)
		assert.Contains(t, err.Error(), "failed to load team configuration")
		repo.AssertExpectations(t)
	})
}

func TestConfigService_SetExcludedIssueTypesForProject(t *testing.T) {
	t.Run("should set excluded issue types successfully", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		svc := NewConfigService(repo)

		teams := map[string][]string{"COP": {"alice"}}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)
		repo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(nil)

		err = svc.SetExcludedIssueTypesForProject("COP", []string{"Experiment"})

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("should return error when project does not exist", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		svc := NewConfigService(repo)

		teams := map[string][]string{"COP": {"alice"}}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		err = svc.SetExcludedIssueTypesForProject("NONEXISTENT", []string{"Experiment"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set excluded issue types")
		repo.AssertExpectations(t)
	})

	t.Run("should return error when load fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		svc := NewConfigService(repo)

		repo.On("LoadTeamConfig").Return(nil, errors.New("repository error"))

		err := svc.SetExcludedIssueTypesForProject("COP", []string{"Experiment"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load team configuration")
		repo.AssertExpectations(t)
	})

	t.Run("should return error when save fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		svc := NewConfigService(repo)

		teams := map[string][]string{"COP": {"alice"}}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)
		repo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(errors.New("save error"))

		err = svc.SetExcludedIssueTypesForProject("COP", []string{"Experiment"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save team configuration")
		repo.AssertExpectations(t)
	})
}

func TestConfigService_GetExcludedIssueTypesForProject(t *testing.T) {
	t.Run("should return excluded types for project", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		svc := NewConfigService(repo)

		teams := map[string][]string{"COP": {"alice"}}
		excludedTypes := map[string][]string{"COP": {"Experiment"}}
		teamConfig, err := domain.NewTeamConfigWithExcludedTypes(teams, nil, nil, nil, nil, nil, excludedTypes)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		types, err := svc.GetExcludedIssueTypesForProject("COP")

		require.NoError(t, err)
		assert.Equal(t, []string{"Experiment"}, types)
		repo.AssertExpectations(t)
	})

	t.Run("should return nil for project without excluded types", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		svc := NewConfigService(repo)

		teams := map[string][]string{"COP": {"alice"}}
		teamConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		types, err := svc.GetExcludedIssueTypesForProject("COP")

		require.NoError(t, err)
		assert.Nil(t, types)
		repo.AssertExpectations(t)
	})

	t.Run("should return error when load fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		svc := NewConfigService(repo)

		repo.On("LoadTeamConfig").Return(nil, errors.New("repository error"))

		types, err := svc.GetExcludedIssueTypesForProject("COP")

		require.Error(t, err)
		assert.Nil(t, types)
		assert.Contains(t, err.Error(), "failed to load team configuration")
		repo.AssertExpectations(t)
	})
}

func TestConfigService_GetTeamMapForSprintWithDates(t *testing.T) {
	date := func(year int, month time.Month, day int) time.Time {
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}
	datePtr := func(year int, month time.Month, day int) *time.Time {
		t := date(year, month, day)
		return &t
	}

	t.Run("resolves team from timeline for given period", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		svc := NewConfigService(repo)

		teams := map[string][]string{"PROJECT-A": {"Alice", "Charlie"}}
		timelines := map[string][]domain.TeamMemberPeriod{
			"PROJECT-A": {
				{Member: "Alice", Joined: date(2024, 1, 1)},
				{Member: "Bob", Joined: date(2024, 1, 1), Left: datePtr(2024, 6, 30)},
				{Member: "Charlie", Joined: date(2024, 7, 1)},
			},
		}
		config, err := domain.NewTeamConfigWithTimelines(teams, nil, nil, nil, nil, nil, nil, nil, timelines)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(config, nil)

		// Q1: Alice + Bob active
		teamMap, err := svc.GetTeamMapForSprintWithDates(date(2024, 1, 1), date(2024, 3, 31))
		require.NoError(t, err)

		team, exists := teamMap.GetTeam("PROJECT-A")
		require.True(t, exists)
		assert.ElementsMatch(t, []string{"Alice", "Bob"}, team.Team)
	})

	t.Run("falls back to flat team without timeline", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		svc := NewConfigService(repo)

		teams := map[string][]string{"PROJECT-A": {"Alice", "Bob"}}
		config, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(config, nil)

		teamMap, err := svc.GetTeamMapForSprintWithDates(date(2024, 1, 1), date(2024, 3, 31))
		require.NoError(t, err)

		team, exists := teamMap.GetTeam("PROJECT-A")
		require.True(t, exists)
		assert.Equal(t, []string{"Alice", "Bob"}, team.Team)
	})

	t.Run("includes excluded issue types", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		svc := NewConfigService(repo)

		teams := map[string][]string{"PROJECT-A": {"Alice"}}
		timelines := map[string][]domain.TeamMemberPeriod{
			"PROJECT-A": {{Member: "Alice", Joined: date(2024, 1, 1)}},
		}
		excluded := map[string][]string{"PROJECT-A": {"Experiment"}}
		config, err := domain.NewTeamConfigWithTimelines(teams, nil, nil, nil, nil, nil, excluded, nil, timelines)
		require.NoError(t, err)

		repo.On("LoadTeamConfig").Return(config, nil)

		teamMap, err := svc.GetTeamMapForSprintWithDates(date(2024, 1, 1), date(2024, 3, 31))
		require.NoError(t, err)

		team, exists := teamMap.GetTeam("PROJECT-A")
		require.True(t, exists)
		assert.Equal(t, []string{"Experiment"}, team.ExcludedIssueTypes)
	})

	t.Run("returns error when config loading fails", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		svc := NewConfigService(repo)

		repo.On("LoadTeamConfig").Return(nil, errors.New("repository error"))

		teamMap, err := svc.GetTeamMapForSprintWithDates(date(2024, 1, 1), date(2024, 3, 31))
		require.Error(t, err)
		assert.Nil(t, teamMap)
		repo.AssertExpectations(t)
	})
}

func TestConfigService_GetBoardWorkStream(t *testing.T) {
	t.Run("returns workstream for known board", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		teamConfig, err := domain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
		require.NoError(t, err)
		require.NoError(t, teamConfig.SetBoardWorkStreams("FN", map[int]string{42: "Pricing"}))
		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		ws, err := service.GetBoardWorkStream("FN", 42)
		require.NoError(t, err)
		assert.Equal(t, "Pricing", ws)
		repo.AssertExpectations(t)
	})

	t.Run("returns empty for unknown board without erroring", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		teamConfig, err := domain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
		require.NoError(t, err)
		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		ws, err := service.GetBoardWorkStream("FN", 99)
		require.NoError(t, err)
		assert.Equal(t, "", ws)
		repo.AssertExpectations(t)
	})

	t.Run("wraps repository load errors", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		repo.On("LoadTeamConfig").Return(nil, errors.New("repo down"))

		_, err := service.GetBoardWorkStream("FN", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load team configuration")
		repo.AssertExpectations(t)
	})
}

func TestConfigService_GetBoardsForWorkStream(t *testing.T) {
	t.Run("returns matching boards (case-insensitive)", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		teamConfig, err := domain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
		require.NoError(t, err)
		require.NoError(t, teamConfig.SetBoardWorkStreams("FN", map[int]string{
			1: "Pricing",
			2: "pricing",
			3: "Search",
		}))
		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		ids, err := service.GetBoardsForWorkStream("FN", "PRICING")
		require.NoError(t, err)
		assert.ElementsMatch(t, []int{1, 2}, ids)
		repo.AssertExpectations(t)
	})

	t.Run("wraps repository load errors", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		repo.On("LoadTeamConfig").Return(nil, errors.New("repo down"))

		ids, err := service.GetBoardsForWorkStream("FN", "Pricing")
		require.Error(t, err)
		assert.Nil(t, ids)
		assert.Contains(t, err.Error(), "failed to load team configuration")
		repo.AssertExpectations(t)
	})
}

func TestConfigService_SetBoardWorkStreams(t *testing.T) {
	t.Run("happy path saves through to repository", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		teamConfig, err := domain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
		require.NoError(t, err)
		repo.On("LoadTeamConfig").Return(teamConfig, nil)
		repo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(nil)

		require.NoError(t, service.SetBoardWorkStreams("FN", map[int]string{1: "Pricing"}))
		repo.AssertExpectations(t)
	})

	t.Run("rejects unknown projects (domain-level validation)", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		teamConfig, err := domain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
		require.NoError(t, err)
		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		err = service.SetBoardWorkStreams("GHOST", map[int]string{1: "X"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set board work streams")
		repo.AssertExpectations(t)
	})

	t.Run("wraps repository load errors", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		repo.On("LoadTeamConfig").Return(nil, errors.New("repo down"))

		err := service.SetBoardWorkStreams("FN", map[int]string{1: "Pricing"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load team configuration")
		repo.AssertExpectations(t)
	})

	t.Run("wraps repository save errors", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		teamConfig, err := domain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
		require.NoError(t, err)
		repo.On("LoadTeamConfig").Return(teamConfig, nil)
		repo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(errors.New("disk full"))

		err = service.SetBoardWorkStreams("FN", map[int]string{1: "Pricing"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save team configuration")
		repo.AssertExpectations(t)
	})
}

func TestConfigService_SetBoardWorkStream(t *testing.T) {
	t.Run("creates a new mapping if none exists", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		teamConfig, err := domain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
		require.NoError(t, err)
		repo.On("LoadTeamConfig").Return(teamConfig, nil)
		repo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(nil)

		require.NoError(t, service.SetBoardWorkStream("FN", 7, "Pricing"))
		repo.AssertExpectations(t)
	})

	t.Run("merges into an existing mapping", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		teamConfig, err := domain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
		require.NoError(t, err)
		require.NoError(t, teamConfig.SetBoardWorkStreams("FN", map[int]string{1: "Existing"}))
		repo.On("LoadTeamConfig").Return(teamConfig, nil)
		repo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(nil)

		require.NoError(t, service.SetBoardWorkStream("FN", 2, "Added"))
		// The save call should carry both the original and the new mapping.
		final := teamConfig.GetBoardWorkStreams("FN")
		assert.Equal(t, "Existing", final[1])
		assert.Equal(t, "Added", final[2])
		repo.AssertExpectations(t)
	})

	t.Run("rejects unknown projects (domain-level validation)", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		teamConfig, err := domain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
		require.NoError(t, err)
		repo.On("LoadTeamConfig").Return(teamConfig, nil)

		err = service.SetBoardWorkStream("GHOST", 1, "X")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set board work stream")
		repo.AssertExpectations(t)
	})

	t.Run("wraps repository load errors", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		repo.On("LoadTeamConfig").Return(nil, errors.New("repo down"))

		err := service.SetBoardWorkStream("FN", 1, "X")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load team configuration")
		repo.AssertExpectations(t)
	})

	t.Run("wraps repository save errors", func(t *testing.T) {
		repo := &MockConfigurationRepository{}
		service := NewConfigService(repo)
		teamConfig, err := domain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
		require.NoError(t, err)
		repo.On("LoadTeamConfig").Return(teamConfig, nil)
		repo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(errors.New("disk full"))

		err = service.SetBoardWorkStream("FN", 1, "Pricing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save team configuration")
		repo.AssertExpectations(t)
	})
}
