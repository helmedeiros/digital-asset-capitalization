package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusMapper(t *testing.T) {
	t.Parallel()
	// Test data
	teamConfigs := map[string]TeamConfig{
		"AD": {
			Team:      []string{"user1@example.com", "user2@example.com"},
			Nicknames: []string{"AAA", "Alpha"},
			Boards: map[string]BoardConfig{
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
			Team: []string{"dev1@example.com", "dev2@example.com"},
			Boards: map[string]BoardConfig{
				"123": {
					ID:   "123",
					Name: "Dev Board",
					StatusMappings: map[string][]string{
						"done":        {"Complete", "Finished"},
						"in_progress": {"Working", "Coding"},
						"todo":        {"Planned", "New"},
					},
				},
			},
		},
	}

	mapper := NewStatusMapper(teamConfigs)

	t.Run("NormalizeStatus with team-specific mappings", func(t *testing.T) {
		// Test exact matches
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("Done", "AD", "92"))
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("Done / Deployed to Live", "AD", "92"))
		assert.Equal(t, StatusTypeInProgress, mapper.NormalizeStatus("In Progress", "AD", "92"))
		assert.Equal(t, StatusTypeWontDo, mapper.NormalizeStatus("Won't Do", "AD", "92"))
		assert.Equal(t, StatusTypeTodo, mapper.NormalizeStatus("To Do", "AD", "92"))

		// Test case insensitive matching
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("done", "AD", "92"))
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("DONE", "AD", "92"))
		assert.Equal(t, StatusTypeInProgress, mapper.NormalizeStatus("in progress", "AD", "92"))

		// Test with extra whitespace
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("  Done  ", "AD", "92"))
		assert.Equal(t, StatusTypeInProgress, mapper.NormalizeStatus("\tIn Progress\n", "AD", "92"))

		// Test different team mappings
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("Complete", "DEV", "123"))
		assert.Equal(t, StatusTypeInProgress, mapper.NormalizeStatus("Working", "DEV", "123"))
		assert.Equal(t, StatusTypeTodo, mapper.NormalizeStatus("Planned", "DEV", "123"))
	})

	t.Run("NormalizeStatus falls back to default mappings", func(t *testing.T) {
		// Non-existent team should use default mappings
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("done", "NONEXISTENT", "92"))
		assert.Equal(t, StatusTypeInProgress, mapper.NormalizeStatus("in progress", "NONEXISTENT", "92"))
		assert.Equal(t, StatusTypeTodo, mapper.NormalizeStatus("todo", "NONEXISTENT", "92"))

		// Non-existent board should use default mappings
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("closed", "AD", "999"))
		assert.Equal(t, StatusTypeInProgress, mapper.NormalizeStatus("development", "AD", "999"))

		// Test default mapping patterns
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("resolved", "UNKNOWN", "999"))
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("deployed", "UNKNOWN", "999"))
		assert.Equal(t, StatusTypeInProgress, mapper.NormalizeStatus("review", "UNKNOWN", "999"))
		assert.Equal(t, StatusTypeWontDo, mapper.NormalizeStatus("cancelled", "UNKNOWN", "999"))
		assert.Equal(t, StatusTypeTodo, mapper.NormalizeStatus("backlog", "UNKNOWN", "999"))
		assert.Equal(t, StatusTypeUnknown, mapper.NormalizeStatus("random status", "UNKNOWN", "999"))
	})

	t.Run("NormalizeStatus with custom status containing default patterns", func(t *testing.T) {
		// Custom status that contains default patterns should use team mapping
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("Done / Deployed to Live", "AD", "92"))

		// If not in team mapping, should fall back to default pattern matching
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("Something Done", "AD", "999"))
		assert.Equal(t, StatusTypeInProgress, mapper.NormalizeStatus("In Development Mode", "AD", "999"))
	})

	t.Run("IsDone helper method", func(t *testing.T) {
		assert.True(t, mapper.IsDone("Done", "AD", "92"))
		assert.True(t, mapper.IsDone("Done / Deployed to Live", "AD", "92"))
		assert.False(t, mapper.IsDone("In Progress", "AD", "92"))
		assert.False(t, mapper.IsDone("To Do", "AD", "92"))
		assert.False(t, mapper.IsDone("Won't Do", "AD", "92"))
	})

	t.Run("IsWontDo helper method", func(t *testing.T) {
		assert.True(t, mapper.IsWontDo("Won't Do", "AD", "92"))
		assert.True(t, mapper.IsWontDo("Cancelled", "AD", "92"))
		assert.False(t, mapper.IsWontDo("Done", "AD", "92"))
		assert.False(t, mapper.IsWontDo("In Progress", "AD", "92"))
		assert.False(t, mapper.IsWontDo("To Do", "AD", "92"))
	})

	t.Run("IsInProgress helper method", func(t *testing.T) {
		assert.True(t, mapper.IsInProgress("In Progress", "AD", "92"))
		assert.True(t, mapper.IsInProgress("In Development", "AD", "92"))
		assert.False(t, mapper.IsInProgress("Done", "AD", "92"))
		assert.False(t, mapper.IsInProgress("To Do", "AD", "92"))
		assert.False(t, mapper.IsInProgress("Won't Do", "AD", "92"))
	})

	t.Run("findStatusType internal method behavior", func(t *testing.T) {
		testMappings := map[string][]string{
			"done":        {"Done", "Complete"},
			"in_progress": {"Working", "Active"},
			"todo":        {"New", "Planned"},
		}

		// Test exact matches
		assert.Equal(t, StatusTypeDone, mapper.findStatusType("Done", testMappings))
		assert.Equal(t, StatusTypeInProgress, mapper.findStatusType("Working", testMappings))
		assert.Equal(t, StatusTypeTodo, mapper.findStatusType("New", testMappings))

		// Test case insensitive
		assert.Equal(t, StatusTypeDone, mapper.findStatusType("done", testMappings))
		assert.Equal(t, StatusTypeDone, mapper.findStatusType("COMPLETE", testMappings))

		// Test with whitespace
		assert.Equal(t, StatusTypeInProgress, mapper.findStatusType("  Working  ", testMappings))

		// Test non-existent status
		assert.Equal(t, StatusTypeUnknown, mapper.findStatusType("Random", testMappings))
	})

	t.Run("getDefaultStatusType patterns", func(t *testing.T) {
		// Test done patterns
		assert.Equal(t, StatusTypeDone, mapper.getDefaultStatusType("done"))
		assert.Equal(t, StatusTypeDone, mapper.getDefaultStatusType("closed"))
		assert.Equal(t, StatusTypeDone, mapper.getDefaultStatusType("resolved"))
		assert.Equal(t, StatusTypeDone, mapper.getDefaultStatusType("deployed"))
		assert.Equal(t, StatusTypeDone, mapper.getDefaultStatusType("Something Done Here"))
		assert.Equal(t, StatusTypeDone, mapper.getDefaultStatusType("Task Resolved Successfully"))

		// Test in progress patterns
		assert.Equal(t, StatusTypeInProgress, mapper.getDefaultStatusType("in progress"))
		assert.Equal(t, StatusTypeInProgress, mapper.getDefaultStatusType("development"))
		assert.Equal(t, StatusTypeInProgress, mapper.getDefaultStatusType("review"))
		assert.Equal(t, StatusTypeInProgress, mapper.getDefaultStatusType("Code Review Phase"))
		assert.Equal(t, StatusTypeInProgress, mapper.getDefaultStatusType("Under Development"))

		// Test won't do patterns
		// Note: "won't" pattern is checked before "done" pattern in switch statement
		assert.Equal(t, StatusTypeWontDo, mapper.getDefaultStatusType("won't fix"))
		assert.Equal(t, StatusTypeWontDo, mapper.getDefaultStatusType("won't do"))
		assert.Equal(t, StatusTypeWontDo, mapper.getDefaultStatusType("cancelled"))
		assert.Equal(t, StatusTypeWontDo, mapper.getDefaultStatusType("duplicate"))
		assert.Equal(t, StatusTypeWontDo, mapper.getDefaultStatusType("Task Won't Fix"))

		// Test todo patterns
		assert.Equal(t, StatusTypeTodo, mapper.getDefaultStatusType("todo"))
		assert.Equal(t, StatusTypeTodo, mapper.getDefaultStatusType("open"))
		assert.Equal(t, StatusTypeTodo, mapper.getDefaultStatusType("backlog"))
		assert.Equal(t, StatusTypeTodo, mapper.getDefaultStatusType("Item in Backlog"))

		// Test unknown status
		assert.Equal(t, StatusTypeUnknown, mapper.getDefaultStatusType("random status"))
		assert.Equal(t, StatusTypeUnknown, mapper.getDefaultStatusType(""))
		assert.Equal(t, StatusTypeUnknown, mapper.getDefaultStatusType("   "))
	})

	t.Run("Edge cases and error conditions", func(t *testing.T) {
		// Empty strings
		assert.Equal(t, StatusTypeUnknown, mapper.NormalizeStatus("", "AD", "92"))
		assert.Equal(t, StatusTypeUnknown, mapper.NormalizeStatus("   ", "AD", "92"))

		// Empty team key
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("done", "", "92"))

		// Empty board ID
		assert.Equal(t, StatusTypeDone, mapper.NormalizeStatus("done", "AD", ""))

		// All empty
		assert.Equal(t, StatusTypeUnknown, mapper.NormalizeStatus("", "", ""))
	})
}

func TestNewStatusMapper(t *testing.T) {
	t.Parallel()
	t.Run("should create mapper with valid configs", func(t *testing.T) {
		configs := map[string]TeamConfig{
			"TEST": {
				Team: []string{"test@example.com"},
				Boards: map[string]BoardConfig{
					"1": {
						ID:             "1",
						Name:           "Test Board",
						StatusMappings: map[string][]string{"done": {"Complete"}},
					},
				},
			},
		}

		mapper := NewStatusMapper(configs)
		assert.NotNil(t, mapper)
		assert.Equal(t, configs, mapper.teamConfigs)
	})

	t.Run("should handle empty configs", func(t *testing.T) {
		mapper := NewStatusMapper(map[string]TeamConfig{})
		assert.NotNil(t, mapper)
		assert.Empty(t, mapper.teamConfigs)
	})

	t.Run("should handle nil configs", func(t *testing.T) {
		mapper := NewStatusMapper(nil)
		assert.NotNil(t, mapper)
		assert.Nil(t, mapper.teamConfigs)
	})
}

func TestStatusType(t *testing.T) {
	t.Parallel()
	t.Run("constants are properly defined", func(t *testing.T) {
		assert.Equal(t, StatusType("done"), StatusTypeDone)
		assert.Equal(t, StatusType("in_progress"), StatusTypeInProgress)
		assert.Equal(t, StatusType("wont_do"), StatusTypeWontDo)
		assert.Equal(t, StatusType("todo"), StatusTypeTodo)
		assert.Equal(t, StatusType("unknown"), StatusTypeUnknown)
	})
}

func TestBoardConfig(t *testing.T) {
	t.Parallel()
	t.Run("should create board config with all fields", func(t *testing.T) {
		config := BoardConfig{
			ID:   "123",
			Name: "Test Board",
			StatusMappings: map[string][]string{
				"done": {"Complete", "Finished"},
				"todo": {"New", "Open"},
			},
		}

		assert.Equal(t, "123", config.ID)
		assert.Equal(t, "Test Board", config.Name)
		assert.Len(t, config.StatusMappings, 2)
		assert.Equal(t, []string{"Complete", "Finished"}, config.StatusMappings["done"])
		assert.Equal(t, []string{"New", "Open"}, config.StatusMappings["todo"])
	})
}

func TestTeamConfig(t *testing.T) {
	t.Parallel()
	t.Run("should create team config with all fields", func(t *testing.T) {
		config := TeamConfig{
			Team:      []string{"user1@example.com", "user2@example.com"},
			Nicknames: []string{"Alpha", "Beta"},
			Boards: map[string]BoardConfig{
				"1": {
					ID:             "1",
					Name:           "Board 1",
					StatusMappings: map[string][]string{"done": {"Complete"}},
				},
			},
		}

		assert.Len(t, config.Team, 2)
		assert.Equal(t, []string{"user1@example.com", "user2@example.com"}, config.Team)
		assert.Len(t, config.Nicknames, 2)
		assert.Equal(t, []string{"Alpha", "Beta"}, config.Nicknames)
		assert.Len(t, config.Boards, 1)
		assert.Contains(t, config.Boards, "1")
	})

	t.Run("should handle optional fields", func(t *testing.T) {
		config := TeamConfig{
			Team: []string{"user1@example.com"},
		}

		assert.Len(t, config.Team, 1)
		assert.Nil(t, config.Nicknames)
		assert.Nil(t, config.Boards)
	})
}
