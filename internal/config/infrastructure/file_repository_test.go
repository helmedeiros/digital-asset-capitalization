package infrastructure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

func TestNewFileRepository(t *testing.T) {
	repo := NewFileRepository("/test/config")

	assert.NotNil(t, repo)
	assert.Equal(t, "/test/config", repo.configDir)
}

func TestFileRepository_InitializeConfigDirectory(t *testing.T) {
	t.Run("should create directory successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, "test-config")
		repo := NewFileRepository(configDir)

		err := repo.InitializeConfigDirectory()

		require.NoError(t, err)

		// Verify directory was created
		stat, err := os.Stat(configDir)
		require.NoError(t, err)
		assert.True(t, stat.IsDir())
	})
}

func TestFileRepository_ConfigExists(t *testing.T) {
	t.Run("should return false when no config files exist", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		exists, err := repo.ConfigExists()

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("should return true when jira config exists", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create jira.json file
		jiraPath := filepath.Join(tempDir, "jira.json")
		err := os.WriteFile(jiraPath, []byte(`{"base_url":"test"}`), 0644)
		require.NoError(t, err)

		exists, err := repo.ConfigExists()

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should return true when teams config exists", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create teams.json file
		teamsPath := filepath.Join(tempDir, "teams.json")
		err := os.WriteFile(teamsPath, []byte(`{"FN":{"team":["test"]}}`), 0644)
		require.NoError(t, err)

		exists, err := repo.ConfigExists()

		require.NoError(t, err)
		assert.True(t, exists)
	})
}

func TestFileRepository_JiraConfig(t *testing.T) {
	t.Run("should save and load jira config successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create test config
		originalConfig, err := domain.NewJiraConfig(
			"https://test.atlassian.net",
			"test@example.com",
			"test-token",
		)
		require.NoError(t, err)

		// Save config
		err = repo.SaveJiraConfig(originalConfig)
		require.NoError(t, err)

		// Load config
		loadedConfig, err := repo.LoadJiraConfig()
		require.NoError(t, err)

		// Verify config matches
		assert.Equal(t, originalConfig.BaseURL(), loadedConfig.BaseURL())
		assert.Equal(t, originalConfig.Email(), loadedConfig.Email())
		assert.Equal(t, originalConfig.Token(), loadedConfig.Token())
	})

	t.Run("should return error when jira config file doesn't exist", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		_, err := repo.LoadJiraConfig()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "jira configuration not found")
	})

	t.Run("should return error when jira config file is invalid JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create invalid JSON file
		jiraPath := filepath.Join(tempDir, "jira.json")
		err := os.WriteFile(jiraPath, []byte(`{invalid json`), 0644)
		require.NoError(t, err)

		_, err = repo.LoadJiraConfig()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse jira config")
	})

	t.Run("should create directories when saving config", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, "nested", "config")
		repo := NewFileRepository(configDir)

		config, err := domain.NewJiraConfig(
			"https://test.atlassian.net",
			"test@example.com",
			"test-token",
		)
		require.NoError(t, err)

		err = repo.SaveJiraConfig(config)
		require.NoError(t, err)

		// Verify file was created
		jiraPath := filepath.Join(configDir, "jira.json")
		_, err = os.Stat(jiraPath)
		require.NoError(t, err)
	})
}

func TestFileRepository_TeamConfig(t *testing.T) {
	t.Run("should save and load team config with format transformation", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create test team config in domain format
		teams := map[string][]string{
			"FN": {"helio.medeiros", "alice.smith"},
			"BE": {"bob.jones"},
		}
		originalConfig, err := domain.NewTeamConfig(teams)
		require.NoError(t, err)

		// Save config
		err = repo.SaveTeamConfig(originalConfig)
		require.NoError(t, err)

		// Verify file format matches existing pattern
		teamsPath := filepath.Join(tempDir, "teams.json")
		data, err := os.ReadFile(teamsPath)
		require.NoError(t, err)

		var fileFormat map[string]struct {
			Team []string `json:"team"`
		}
		err = json.Unmarshal(data, &fileFormat)
		require.NoError(t, err)

		assert.Equal(t, []string{"helio.medeiros", "alice.smith"}, fileFormat["FN"].Team)
		assert.Equal(t, []string{"bob.jones"}, fileFormat["BE"].Team)

		// Load config
		loadedConfig, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		// Verify domain format matches
		fnTeam, exists := loadedConfig.GetTeam("FN")
		require.True(t, exists)
		assert.Equal(t, []string{"helio.medeiros", "alice.smith"}, fnTeam)

		beTeam, exists := loadedConfig.GetTeam("BE")
		require.True(t, exists)
		assert.Equal(t, []string{"bob.jones"}, beTeam)
	})

	t.Run("should load existing teams.json format correctly", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create file in existing format
		existingFormat := map[string]struct {
			Team []string `json:"team"`
		}{
			"FN": {Team: []string{"Ahmed Naser", "Georgii Maltsev", "Helio Medeiros"}},
			"MZ": {Team: []string{"Viktor Kovarik", "Pernelle Naidoo"}},
		}

		data, err := json.MarshalIndent(existingFormat, "", "  ")
		require.NoError(t, err)

		teamsPath := filepath.Join(tempDir, "teams.json")
		err = os.WriteFile(teamsPath, data, 0644)
		require.NoError(t, err)

		// Load config
		config, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		// Verify transformation worked
		fnTeam, exists := config.GetTeam("FN")
		require.True(t, exists)
		assert.Contains(t, fnTeam, "Ahmed Naser")
		assert.Contains(t, fnTeam, "Georgii Maltsev")
		assert.Contains(t, fnTeam, "Helio Medeiros")

		mzTeam, exists := config.GetTeam("MZ")
		require.True(t, exists)
		assert.Contains(t, mzTeam, "Viktor Kovarik")
		assert.Contains(t, mzTeam, "Pernelle Naidoo")

		// Verify team membership
		assert.True(t, config.IsTeamMember("FN", "Ahmed Naser"))
		assert.True(t, config.IsTeamMember("MZ", "Viktor Kovarik"))
		assert.False(t, config.IsTeamMember("FN", "Viktor Kovarik"))
	})

	t.Run("should return error when teams config file doesn't exist", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		_, err := repo.LoadTeamConfig()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "team configuration not found")
	})

	t.Run("should return error when teams config file is invalid JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create invalid JSON file
		teamsPath := filepath.Join(tempDir, "teams.json")
		err := os.WriteFile(teamsPath, []byte(`{invalid json`), 0644)
		require.NoError(t, err)

		_, err = repo.LoadTeamConfig()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse team config")
	})
}

func TestFileRepository_EdgeCases(t *testing.T) {
	t.Run("should handle empty team config", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create empty team config
		emptyConfig, err := domain.NewTeamConfig(map[string][]string{})
		require.NoError(t, err)

		// Save and load
		err = repo.SaveTeamConfig(emptyConfig)
		require.NoError(t, err)

		loadedConfig, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		assert.True(t, loadedConfig.IsEmpty())
		assert.Empty(t, loadedConfig.GetProjects())
	})

	t.Run("should handle atomic file operations", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		config, err := domain.NewJiraConfig(
			"https://test.atlassian.net",
			"test@example.com",
			"test-token",
		)
		require.NoError(t, err)

		// Save config
		err = repo.SaveJiraConfig(config)
		require.NoError(t, err)

		// Verify no temporary files remain
		entries, err := os.ReadDir(tempDir)
		require.NoError(t, err)

		for _, entry := range entries {
			assert.False(t, filepath.Ext(entry.Name()) == ".tmp", "Temporary file should not exist: %s", entry.Name())
		}
	})
}

func TestFileRepository_TeamConfigWithTribes(t *testing.T) {
	t.Run("should load team config with tribes", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create file with tribe field
		teamsJSON := `{
			"FN": {
				"team": ["Alice", "Bob"],
				"nicknames": ["pricing"],
				"tribe": "Engineering"
			},
			"COP": {
				"team": ["Charlie"],
				"tribe": "Platform"
			}
		}`

		teamsPath := filepath.Join(tempDir, "teams.json")
		err := os.WriteFile(teamsPath, []byte(teamsJSON), 0644)
		require.NoError(t, err)

		// Load config
		config, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		// Verify tribes loaded
		assert.Equal(t, "Engineering", config.GetTribe("FN"))
		assert.Equal(t, "Platform", config.GetTribe("COP"))

		// Verify teams and nicknames still work
		fnTeam, exists := config.GetTeam("FN")
		require.True(t, exists)
		assert.Equal(t, []string{"Alice", "Bob"}, fnTeam)
		assert.Equal(t, []string{"pricing"}, config.GetNicknames("FN"))
	})

	t.Run("should save and load team config with tribes", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create config with tribes
		teams := map[string][]string{
			"FN":  {"Alice", "Bob"},
			"COP": {"Charlie"},
		}
		nicknames := map[string][]string{
			"FN": {"pricing"},
		}
		tribes := map[string]string{
			"FN":  "Engineering",
			"COP": "Platform",
		}

		config, err := domain.NewTeamConfigWithTribes(teams, nicknames, tribes)
		require.NoError(t, err)

		// Save config
		err = repo.SaveTeamConfig(config)
		require.NoError(t, err)

		// Load config back
		loadedConfig, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		// Verify tribes
		assert.Equal(t, "Engineering", loadedConfig.GetTribe("FN"))
		assert.Equal(t, "Platform", loadedConfig.GetTribe("COP"))

		// Verify teams
		fnTeam, exists := loadedConfig.GetTeam("FN")
		require.True(t, exists)
		assert.Equal(t, []string{"Alice", "Bob"}, fnTeam)

		// Verify nicknames
		assert.Equal(t, []string{"pricing"}, loadedConfig.GetNicknames("FN"))
	})

	t.Run("should handle empty tribe field", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create file without tribe field
		teamsJSON := `{
			"FN": {
				"team": ["Alice"],
				"nicknames": ["pricing"]
			}
		}`

		teamsPath := filepath.Join(tempDir, "teams.json")
		err := os.WriteFile(teamsPath, []byte(teamsJSON), 0644)
		require.NoError(t, err)

		// Load config
		config, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		// Tribe should be empty
		assert.Equal(t, "", config.GetTribe("FN"))

		// Team should still work
		fnTeam, exists := config.GetTeam("FN")
		require.True(t, exists)
		assert.Equal(t, []string{"Alice"}, fnTeam)
	})
}

func TestFileRepository_TeamConfigWithCompanies(t *testing.T) {
	t.Run("should load team config with companies", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create file with company field
		teamsJSON := `{
			"FN": {
				"team": ["Alice", "Bob"],
				"nicknames": ["pricing"],
				"tribe": "Engineering",
				"company": "TechCorp"
			},
			"COP": {
				"team": ["Charlie"],
				"tribe": "Platform",
				"company": "PartnerInc"
			}
		}`

		teamsPath := filepath.Join(tempDir, "teams.json")
		err := os.WriteFile(teamsPath, []byte(teamsJSON), 0644)
		require.NoError(t, err)

		// Load config
		config, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		// Verify companies loaded
		assert.Equal(t, "TechCorp", config.GetCompany("FN"))
		assert.Equal(t, "PartnerInc", config.GetCompany("COP"))

		// Verify tribes still work
		assert.Equal(t, "Engineering", config.GetTribe("FN"))
		assert.Equal(t, "Platform", config.GetTribe("COP"))

		// Verify teams and nicknames still work
		fnTeam, exists := config.GetTeam("FN")
		require.True(t, exists)
		assert.Equal(t, []string{"Alice", "Bob"}, fnTeam)
		assert.Equal(t, []string{"pricing"}, config.GetNicknames("FN"))
	})

	t.Run("should save and load team config with companies", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create config with companies
		teams := map[string][]string{
			"FN":  {"Alice", "Bob"},
			"COP": {"Charlie"},
		}
		nicknames := map[string][]string{
			"FN": {"pricing"},
		}
		tribes := map[string]string{
			"FN":  "Engineering",
			"COP": "Platform",
		}
		companies := map[string]string{
			"FN":  "TechCorp",
			"COP": "PartnerInc",
		}

		config, err := domain.NewTeamConfigFull(teams, nicknames, tribes, companies)
		require.NoError(t, err)

		// Save config
		err = repo.SaveTeamConfig(config)
		require.NoError(t, err)

		// Load config back
		loadedConfig, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		// Verify companies
		assert.Equal(t, "TechCorp", loadedConfig.GetCompany("FN"))
		assert.Equal(t, "PartnerInc", loadedConfig.GetCompany("COP"))

		// Verify tribes
		assert.Equal(t, "Engineering", loadedConfig.GetTribe("FN"))
		assert.Equal(t, "Platform", loadedConfig.GetTribe("COP"))

		// Verify teams
		fnTeam, exists := loadedConfig.GetTeam("FN")
		require.True(t, exists)
		assert.Equal(t, []string{"Alice", "Bob"}, fnTeam)

		// Verify nicknames
		assert.Equal(t, []string{"pricing"}, loadedConfig.GetNicknames("FN"))
	})

	t.Run("should handle empty company field", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		// Create file without company field
		teamsJSON := `{
			"FN": {
				"team": ["Alice"],
				"nicknames": ["pricing"],
				"tribe": "Engineering"
			}
		}`

		teamsPath := filepath.Join(tempDir, "teams.json")
		err := os.WriteFile(teamsPath, []byte(teamsJSON), 0644)
		require.NoError(t, err)

		// Load config
		config, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		// Company should be empty
		assert.Equal(t, "", config.GetCompany("FN"))

		// Team should still work
		fnTeam, exists := config.GetTeam("FN")
		require.True(t, exists)
		assert.Equal(t, []string{"Alice"}, fnTeam)
	})
}

func TestFileRepository_TeamConfigWithExcludedIssueTypes(t *testing.T) {
	t.Run("should load excluded_issue_types from teams.json", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		teamsJSON := `{
			"COP": {
				"team": ["Alice", "Bob"],
				"excluded_issue_types": ["Experiment", "Spike"]
			},
			"FN": {
				"team": ["Charlie"]
			}
		}`

		teamsPath := filepath.Join(tempDir, "teams.json")
		err := os.WriteFile(teamsPath, []byte(teamsJSON), 0644)
		require.NoError(t, err)

		config, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		assert.Equal(t, []string{"Experiment", "Spike"}, config.GetExcludedIssueTypes("COP"))
		assert.Nil(t, config.GetExcludedIssueTypes("FN"))
	})

	t.Run("should save and load excluded_issue_types round-trip", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		teams := map[string][]string{
			"COP": {"Alice"},
		}
		excludedTypes := map[string][]string{
			"COP": {"Experiment"},
		}

		config, err := domain.NewTeamConfigWithExcludedTypes(teams, nil, nil, nil, nil, nil, excludedTypes)
		require.NoError(t, err)

		err = repo.SaveTeamConfig(config)
		require.NoError(t, err)

		loadedConfig, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		assert.Equal(t, []string{"Experiment"}, loadedConfig.GetExcludedIssueTypes("COP"))
	})

	t.Run("should handle missing excluded_issue_types field gracefully", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		teamsJSON := `{
			"COP": {
				"team": ["Alice"]
			}
		}`

		teamsPath := filepath.Join(tempDir, "teams.json")
		err := os.WriteFile(teamsPath, []byte(teamsJSON), 0644)
		require.NoError(t, err)

		config, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		assert.Nil(t, config.GetExcludedIssueTypes("COP"))
	})
}

func TestFileRepository_TeamConfigWithTimeline(t *testing.T) {
	t.Run("should load team_timeline from teams.json", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		teamsJSON := `{
			"PROJECT-A": {
				"team": ["Alice", "Charlie"],
				"team_timeline": [
					{"member": "Alice", "joined": "2024-01-01"},
					{"member": "Bob", "joined": "2024-01-01", "left": "2024-06-30"},
					{"member": "Charlie", "joined": "2024-03-01"}
				]
			}
		}`

		teamsPath := filepath.Join(tempDir, "teams.json")
		err := os.WriteFile(teamsPath, []byte(teamsJSON), 0644)
		require.NoError(t, err)

		config, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		assert.True(t, config.HasTeamTimeline("PROJECT-A"))

		timeline := config.GetTeamTimeline("PROJECT-A")
		require.Len(t, timeline, 3)

		assert.Equal(t, "Alice", timeline[0].Member)
		assert.Nil(t, timeline[0].Left)

		assert.Equal(t, "Bob", timeline[1].Member)
		require.NotNil(t, timeline[1].Left)
		assert.Equal(t, time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC), *timeline[1].Left)

		assert.Equal(t, "Charlie", timeline[2].Member)
		assert.Nil(t, timeline[2].Left)
	})

	t.Run("should save and load team_timeline round-trip", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		joined := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		left := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

		teams := map[string][]string{"PROJECT-A": {"Alice", "Bob"}}
		timelines := map[string][]domain.TeamMemberPeriod{
			"PROJECT-A": {
				{Member: "Alice", Joined: joined},
				{Member: "Bob", Joined: joined, Left: &left},
			},
		}

		config, err := domain.NewTeamConfigWithTimelines(teams, nil, nil, nil, nil, nil, nil, nil, timelines)
		require.NoError(t, err)

		err = repo.SaveTeamConfig(config)
		require.NoError(t, err)

		loadedConfig, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		assert.True(t, loadedConfig.HasTeamTimeline("PROJECT-A"))

		loadedTimeline := loadedConfig.GetTeamTimeline("PROJECT-A")
		require.Len(t, loadedTimeline, 2)
		assert.Equal(t, "Alice", loadedTimeline[0].Member)
		assert.Nil(t, loadedTimeline[0].Left)
		assert.Equal(t, "Bob", loadedTimeline[1].Member)
		require.NotNil(t, loadedTimeline[1].Left)
	})

	t.Run("save derives active team from timeline", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		joined := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		left := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

		teams := map[string][]string{"PROJECT-A": {"Alice", "Bob"}}
		timelines := map[string][]domain.TeamMemberPeriod{
			"PROJECT-A": {
				{Member: "Alice", Joined: joined},
				{Member: "Bob", Joined: joined, Left: &left},
			},
		}

		config, err := domain.NewTeamConfigWithTimelines(teams, nil, nil, nil, nil, nil, nil, nil, timelines)
		require.NoError(t, err)

		err = repo.SaveTeamConfig(config)
		require.NoError(t, err)

		// Read raw JSON to verify team array was derived
		data, err := os.ReadFile(filepath.Join(tempDir, "teams.json"))
		require.NoError(t, err)

		var raw map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(data, &raw))

		var projectData struct {
			Team []string `json:"team"`
		}
		require.NoError(t, json.Unmarshal(raw["PROJECT-A"], &projectData))

		// Only Alice should be in the active team (Bob has left)
		assert.Equal(t, []string{"Alice"}, projectData.Team)
	})

	t.Run("should handle missing team_timeline field gracefully", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileRepository(tempDir)

		teamsJSON := `{
			"PROJECT-A": {
				"team": ["Alice", "Bob"]
			}
		}`

		teamsPath := filepath.Join(tempDir, "teams.json")
		err := os.WriteFile(teamsPath, []byte(teamsJSON), 0644)
		require.NoError(t, err)

		config, err := repo.LoadTeamConfig()
		require.NoError(t, err)

		assert.False(t, config.HasTeamTimeline("PROJECT-A"))

		// Should fall back to flat team
		members, exists := config.GetTeam("PROJECT-A")
		assert.True(t, exists)
		assert.Equal(t, []string{"Alice", "Bob"}, members)
	})
}
