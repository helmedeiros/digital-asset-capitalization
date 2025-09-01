package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharedDomain "github.com/helmedeiros/digital-asset-capitalization/internal/shared/domain"
)

func TestStatusService(t *testing.T) {
	// Test data for isolated testing
	testTeamConfigs := map[string]sharedDomain.TeamConfig{
		"AD": {
			Team:      []string{"user1@example.com", "user2@example.com"},
			Nicknames: []string{"AAA", "Alpha"},
			Boards: map[string]sharedDomain.BoardConfig{
				"92": {
					ID:   "92",
					Name: "AAA — Delivery Board",
					StatusMappings: map[string][]string{
						"done":        {"Done", "Done / Deployed to Live", "Closed", "Resolved"},
						"in_progress": {"In Progress", "In Development", "In Review", "In Testing"},
						"wont_do":     {"Won't Do", "Cancelled", "Duplicate", "Won't Fix"},
						"todo":        {"To Do", "Open", "Backlog", "Ready for Development"},
					},
				},
			},
		},
		"DEV": {
			Team: []string{"dev1@example.com"},
			Boards: map[string]sharedDomain.BoardConfig{
				"123": {
					ID:   "123",
					Name: "Dev Board",
					StatusMappings: map[string][]string{
						"done":        {"Complete"},
						"in_progress": {"Working"},
						"todo":        {"New"},
					},
				},
			},
		},
	}

	t.Run("NewStatusServiceWithConfigs", func(t *testing.T) {
		service := NewStatusServiceWithConfigs(testTeamConfigs)
		assert.NotNil(t, service)
		assert.NotNil(t, service.statusMapper)
		assert.NotNil(t, service.teamConfigs)
		assert.Len(t, service.teamConfigs, 2)
		assert.Contains(t, service.teamConfigs, "AD")
		assert.Contains(t, service.teamConfigs, "DEV")
	})

	t.Run("GetBoardIDForTeam", func(t *testing.T) {
		service := NewStatusServiceWithConfigs(testTeamConfigs)

		// Test existing teams
		boardID := service.GetBoardIDForTeam("AD")
		assert.Equal(t, "92", boardID)

		boardID = service.GetBoardIDForTeam("DEV")
		assert.Equal(t, "123", boardID)

		// Test non-existent team
		boardID = service.GetBoardIDForTeam("NONEXISTENT")
		assert.Equal(t, "", boardID)
	})

	t.Run("NormalizeStatus", func(t *testing.T) {
		service := NewStatusServiceWithConfigs(testTeamConfigs)

		// Test team-specific mapping
		statusType := service.NormalizeStatus("Done / Deployed to Live", "AD", "92")
		assert.Equal(t, sharedDomain.StatusTypeDone, statusType)

		statusType = service.NormalizeStatus("In Progress", "AD", "92")
		assert.Equal(t, sharedDomain.StatusTypeInProgress, statusType)

		// Test fallback to default mapping
		statusType = service.NormalizeStatus("in progress", "UNKNOWN", "999")
		assert.Equal(t, sharedDomain.StatusTypeInProgress, statusType)

		// Test unknown status
		statusType = service.NormalizeStatus("random status", "UNKNOWN", "999")
		assert.Equal(t, sharedDomain.StatusTypeUnknown, statusType)
	})

	t.Run("IsDone", func(t *testing.T) {
		service := NewStatusServiceWithConfigs(testTeamConfigs)

		assert.True(t, service.IsDone("Done / Deployed to Live", "AD", "92"))
		assert.True(t, service.IsDone("Closed", "AD", "92"))
		assert.False(t, service.IsDone("In Progress", "AD", "92"))
		assert.False(t, service.IsDone("To Do", "AD", "92"))
	})

	t.Run("IsWontDo", func(t *testing.T) {
		service := NewStatusServiceWithConfigs(testTeamConfigs)

		assert.True(t, service.IsWontDo("Won't Do", "AD", "92"))
		assert.True(t, service.IsWontDo("Cancelled", "AD", "92"))
		assert.False(t, service.IsWontDo("Done", "AD", "92"))
		assert.False(t, service.IsWontDo("In Progress", "AD", "92"))
	})

	t.Run("IsInProgress", func(t *testing.T) {
		service := NewStatusServiceWithConfigs(testTeamConfigs)

		assert.True(t, service.IsInProgress("In Progress", "AD", "92"))
		assert.True(t, service.IsInProgress("In Development", "AD", "92"))
		assert.False(t, service.IsInProgress("Done", "AD", "92"))
		assert.False(t, service.IsInProgress("To Do", "AD", "92"))
	})

	t.Run("Empty team configs", func(t *testing.T) {
		service := NewStatusServiceWithConfigs(map[string]sharedDomain.TeamConfig{})
		assert.NotNil(t, service)
		assert.Empty(t, service.teamConfigs)

		// Should fall back to default mappings
		statusType := service.NormalizeStatus("done", "ANY", "ANY")
		assert.Equal(t, sharedDomain.StatusTypeDone, statusType)

		boardID := service.GetBoardIDForTeam("ANY")
		assert.Equal(t, "", boardID)
	})
}

func TestNewStatusServiceWithPath(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("with valid config file", func(t *testing.T) {
		validJSON := `{
			"TEAM1": {
				"team": ["user1@example.com"],
				"boards": {
					"1": {
						"id": "1",
						"name": "Board 1",
						"statusMappings": {
							"done": ["Complete"]
						}
					}
				}
			}
		}`

		configPath := filepath.Join(tempDir, "teams.json")
		err := os.WriteFile(configPath, []byte(validJSON), 0644)
		require.NoError(t, err)

		service, err := NewStatusServiceWithPath(configPath)
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.Len(t, service.teamConfigs, 1)
		assert.Contains(t, service.teamConfigs, "TEAM1")
	})

	t.Run("with non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent.json")

		service, err := NewStatusServiceWithPath(nonExistentPath)
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.Empty(t, service.teamConfigs)
	})

	t.Run("with invalid JSON file", func(t *testing.T) {
		invalidPath := filepath.Join(tempDir, "invalid.json")
		err := os.WriteFile(invalidPath, []byte(`{"invalid": json}`), 0644)
		require.NoError(t, err)

		service, err := NewStatusServiceWithPath(invalidPath)
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "failed to load team configs")
	})
}

func TestLoadTeamConfigs(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("with valid file", func(t *testing.T) {
		validJSON := `{
			"TEAM1": {
				"team": ["user1@example.com"],
				"boards": {
					"1": {
						"id": "1",
						"name": "Board 1",
						"statusMappings": {
							"done": ["Complete"]
						}
					}
				}
			}
		}`

		configPath := filepath.Join(tempDir, "teams.json")
		err := os.WriteFile(configPath, []byte(validJSON), 0644)
		require.NoError(t, err)

		configs, err := loadTeamConfigs(configPath)
		require.NoError(t, err)
		assert.Len(t, configs, 1)
		assert.Contains(t, configs, "TEAM1")
		assert.Equal(t, []string{"user1@example.com"}, configs["TEAM1"].Team)
	})

	t.Run("with non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent.json")

		configs, err := loadTeamConfigs(nonExistentPath)
		require.NoError(t, err)
		assert.NotNil(t, configs)
		assert.Empty(t, configs)
	})

	t.Run("with invalid JSON", func(t *testing.T) {
		invalidPath := filepath.Join(tempDir, "invalid.json")
		err := os.WriteFile(invalidPath, []byte(`{"invalid": json}`), 0644)
		require.NoError(t, err)

		configs, err := loadTeamConfigs(invalidPath)
		assert.Error(t, err)
		assert.Nil(t, configs)
		assert.Contains(t, err.Error(), "character")
	})

	t.Run("with empty file", func(t *testing.T) {
		emptyPath := filepath.Join(tempDir, "empty.json")
		err := os.WriteFile(emptyPath, []byte(`{}`), 0644)
		require.NoError(t, err)

		configs, err := loadTeamConfigs(emptyPath)
		require.NoError(t, err)
		assert.Empty(t, configs)
	})

	t.Run("with malformed JSON", func(t *testing.T) {
		malformedPath := filepath.Join(tempDir, "malformed.json")
		err := os.WriteFile(malformedPath, []byte(`{`), 0644)
		require.NoError(t, err)

		configs, err := loadTeamConfigs(malformedPath)
		assert.Error(t, err)
		assert.Nil(t, configs)
	})

	t.Run("with unreadable file", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		unreadablePath := filepath.Join(tempDir, "unreadable.json")
		err := os.WriteFile(unreadablePath, []byte(`{}`), 0644)
		require.NoError(t, err)

		// Make file unreadable
		err = os.Chmod(unreadablePath, 0000)
		require.NoError(t, err)
		defer os.Chmod(unreadablePath, 0644) // Cleanup

		configs, err := loadTeamConfigs(unreadablePath)
		assert.Error(t, err)
		assert.Nil(t, configs)
	})
}
