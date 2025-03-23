package action

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	sprint "github.com/helmedeiros/digital-asset-capitalization/internal/sprint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGetJiraIssues mocks the GetJiraIssues function for testing
func mockGetJiraIssues(url, authHeader string) ([]sprint.JiraIssue, error) {
	return []sprint.JiraIssue{
		{
			Key: "TEST-123",
			Fields: sprint.JiraFields{
				Summary: "Test Issue 1",
				Assignee: sprint.JiraAssignee{
					DisplayName: "Test User 1",
				},
			},
			Changelog: sprint.JiraChangelog{
				Histories: []sprint.JiraChangeHistory{
					{
						Created: "2024-03-01T10:00:00.000+0000",
						Items: []sprint.JiraChangeItem{
							{
								Field:      "status",
								FromString: "To Do",
								ToString:   sprint.StatusInProgress,
							},
						},
					},
					{
						Created: "2024-03-02T15:00:00.000+0000",
						Items: []sprint.JiraChangeItem{
							{
								Field:      "status",
								FromString: sprint.StatusInProgress,
								ToString:   sprint.StatusDone,
							},
						},
					},
				},
			},
		},
		{
			Key: "TEST-124",
			Fields: sprint.JiraFields{
				Summary: "Test Issue 2",
				Assignee: sprint.JiraAssignee{
					DisplayName: "Test User 2",
				},
			},
			Changelog: sprint.JiraChangelog{
				Histories: []sprint.JiraChangeHistory{
					{
						Created: "2024-03-01T11:00:00.000+0000",
						Items: []sprint.JiraChangeItem{
							{
								Field:      "status",
								FromString: "To Do",
								ToString:   sprint.StatusInProgress,
							},
						},
					},
				},
			},
		},
	}, nil
}

// setupTestEnv sets up the test environment and returns a cleanup function
func setupTestEnv(t *testing.T) func() {
	// Save current env vars
	oldEnv := make(map[string]string)
	envVars := []string{"JIRA_BASE_URL", "JIRA_EMAIL", "JIRA_TOKEN"}
	for _, v := range envVars {
		oldEnv[v] = os.Getenv(v)
	}

	// Set test env vars
	os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
	os.Setenv("JIRA_EMAIL", "test@example.com")
	os.Setenv("JIRA_TOKEN", "test-token")

	// Create temporary teams.json for testing
	team := sprint.Team{
		Members: []string{"Test User 1", "Test User 2"},
	}
	teams := sprint.TeamMap{
		"TEST": team,
	}

	teamsData, err := json.Marshal(teams)
	require.NoError(t, err, "Failed to marshal teams data")

	tmpTeamsFile := "teams.json"
	err = ioutil.WriteFile(tmpTeamsFile, teamsData, 0644)
	require.NoError(t, err, "Failed to create temporary teams.json")

	// Save original GetJiraIssues function
	originalGetJiraIssues := sprint.GetJiraIssues
	sprint.GetJiraIssues = mockGetJiraIssues

	// Return cleanup function
	return func() {
		// Restore env vars
		for k, v := range oldEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}

		// Remove temporary file
		os.Remove(tmpTeamsFile)

		// Restore original function
		sprint.GetJiraIssues = originalGetJiraIssues
	}
}

func TestJiraDoer_ValidProject(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	result, err := JiraDoer("TEST", "Sprint 1", "")
	require.NoError(t, err, "JiraDoer should not return error for valid project")
	assert.NotEmpty(t, result, "JiraDoer should return non-empty result")
}

func TestJiraDoer_InvalidProject(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_, err := JiraDoer("INVALID", "Sprint 1", "")
	assert.Error(t, err, "JiraDoer should return error for invalid project")
}

func TestJiraDoer_WithManualAdjustments(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	override := `{"TEST-123": 2.5}`
	result, err := JiraDoer("TEST", "Sprint 1", override)
	require.NoError(t, err, "JiraDoer should not return error with valid manual adjustments")
	assert.NotEmpty(t, result, "JiraDoer should return non-empty result with manual adjustments")
}

func TestJiraDoer_InvalidManualAdjustments(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	override := `invalid json`
	_, err := JiraDoer("TEST", "Sprint 1", override)
	assert.Error(t, err, "JiraDoer should return error with invalid manual adjustments")
}

func TestJiraProcessor_CalculateTotalHours(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	processor, err := NewJiraProcessor("TEST", "Sprint 1", "")
	require.NoError(t, err, "NewJiraProcessor should not return error")

	team, exists := processor.teams.GetTeam("TEST")
	require.True(t, exists, "Team should exist")
	require.NotNil(t, team, "Team should not be nil")

	issues, err := processor.fetchIssues()
	require.NoError(t, err, "fetchIssues should not return error")
	require.NotEmpty(t, issues, "Issues should not be empty")

	manualAdjustments := map[string]float64{"TEST-123": 2.5}
	totalHours := processor.calculateTotalHours(*team, issues, manualAdjustments)

	assert.NotZero(t, totalHours["Test User 1"], "Test User 1 should have non-zero hours")
	assert.NotZero(t, totalHours["Test User 2"], "Test User 2 should have non-zero hours")
}

func TestJiraProcessor_GetIssueTimeRange(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	processor, err := NewJiraProcessor("TEST", "Sprint 1", "")
	require.NoError(t, err, "NewJiraProcessor should not return error")

	issues, err := processor.fetchIssues()
	require.NoError(t, err, "fetchIssues should not return error")
	require.NotEmpty(t, issues, "Issues should not be empty")

	startTime, endTime := processor.getIssueTimeRange(issues[0])

	expectedStart, err := time.Parse("2006-01-02T15:04:05.000-0700", "2024-03-01T10:00:00.000+0000")
	require.NoError(t, err, "Failed to parse expected start time")
	expectedEnd, err := time.Parse("2006-01-02T15:04:05.000-0700", "2024-03-02T15:00:00.000+0000")
	require.NoError(t, err, "Failed to parse expected end time")

	assert.True(t, startTime.Equal(expectedStart), "Start time mismatch")
	assert.True(t, endTime.Equal(expectedEnd), "End time mismatch")
}
