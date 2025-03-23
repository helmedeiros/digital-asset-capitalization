package usecase

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestEnv sets up the test environment and returns a cleanup function
func setupTestEnv(t *testing.T) func() {
	// Create a temporary teams.json file
	teams := domain.TeamMap{
		"TEST": domain.Team{
			Members: []string{"Test User 1", "Test User 2"},
		},
	}

	data, err := json.Marshal(teams)
	require.NoError(t, err, "Failed to marshal teams data")

	err = ioutil.WriteFile("teams.json", data, 0644)
	require.NoError(t, err, "Failed to write teams.json")

	// Set environment variables for testing
	os.Setenv("JIRA_BASE_URL", "http://test.jira.com")
	os.Setenv("JIRA_EMAIL", "test@example.com")
	os.Setenv("JIRA_TOKEN", "test-token")

	// Return cleanup function
	return func() {
		os.Remove("teams.json")
		os.Unsetenv("JIRA_BASE_URL")
		os.Unsetenv("JIRA_EMAIL")
		os.Unsetenv("JIRA_TOKEN")
	}
}

func TestJiraDoer_WithManualAdjustments(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "TEST-123",
					"fields": map[string]interface{}{
						"summary": "Test Issue 1",
						"assignee": map[string]interface{}{
							"displayName": "Test User 1",
						},
					},
					"changelog": map[string]interface{}{
						"histories": []map[string]interface{}{
							{
								"created": "2024-03-01T10:00:00.000+0000",
								"items": []map[string]interface{}{
									{
										"field":      "status",
										"fromString": "To Do",
										"toString":   "In Progress",
									},
								},
							},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	// Set the base URL to our test server
	os.Setenv("JIRA_BASE_URL", server.URL)

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

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "TEST-123",
					"fields": map[string]interface{}{
						"summary": "Test Issue 1",
						"assignee": map[string]interface{}{
							"displayName": "Test User 1",
						},
					},
					"changelog": map[string]interface{}{
						"histories": []map[string]interface{}{
							{
								"created": "2024-03-01T10:00:00.000+0000",
								"items": []map[string]interface{}{
									{
										"field":      "status",
										"fromString": "To Do",
										"toString":   "In Progress",
									},
								},
							},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	// Set the base URL to our test server
	os.Setenv("JIRA_BASE_URL", server.URL)

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

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "TEST-123",
					"fields": map[string]interface{}{
						"summary": "Test Issue 1",
						"assignee": map[string]interface{}{
							"displayName": "Test User 1",
						},
					},
					"changelog": map[string]interface{}{
						"histories": []map[string]interface{}{
							{
								"created": "2024-03-01T10:00:00.000+0000",
								"items": []map[string]interface{}{
									{
										"field":      "status",
										"fromString": "To Do",
										"toString":   "In Progress",
									},
								},
							},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	// Set the base URL to our test server
	os.Setenv("JIRA_BASE_URL", server.URL)

	processor, err := NewJiraProcessor("TEST", "Sprint 1", "")
	require.NoError(t, err, "NewJiraProcessor should not return error")

	issues, err := processor.fetchIssues()
	require.NoError(t, err, "fetchIssues should not return error")
	require.NotEmpty(t, issues, "Issues should not be empty")

	startTime, endTime := processor.getIssueTimeRange(issues[0])
	assert.NotZero(t, startTime, "Start time should not be zero")
	assert.NotZero(t, endTime, "End time should not be zero")
}
