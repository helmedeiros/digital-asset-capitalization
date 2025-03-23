package infrastructure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEnv(t *testing.T) func() {
	// Create a temporary teams.json file
	teams := domain.TeamMap{
		"TEST": domain.Team{
			Team: []string{"Test User 1", "Test User 2"},
		},
	}

	data, err := json.Marshal(teams)
	require.NoError(t, err, "Failed to marshal teams data")

	err = os.WriteFile("teams.json", data, 0644)
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

func TestJiraAdapter_GetIssues(t *testing.T) {
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
						"status": map[string]interface{}{
							"name": "Done",
						},
						"customfield_13192": 5.0,
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

	adapter, err := NewJiraAdapter()
	require.NoError(t, err, "NewJiraAdapter should not return error")

	issues, err := adapter.GetIssuesForSprint("TEST", "TEST-1")
	require.NoError(t, err, "GetIssuesForSprint should not return error")
	require.Len(t, issues, 1, "Should return exactly one issue")

	issue := issues[0]
	assert.Equal(t, "TEST-123", issue.Key)
	assert.Equal(t, "Test Issue 1", issue.Summary)
	assert.Equal(t, "Test User 1", issue.Assignee)
	assert.Equal(t, "Done", issue.Status)
	assert.NotNil(t, issue.StoryPoints)
	assert.Equal(t, 5.0, *issue.StoryPoints)
}

func TestJiraAdapter_GetTeamIssues(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Parse the query to get the assignee
		query := r.URL.Query().Get("jql")
		if strings.Contains(query, "Test User 1") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"issues": []map[string]interface{}{
					{
						"key": "TEST-123",
						"fields": map[string]interface{}{
							"summary": "Test Issue 1",
							"assignee": map[string]interface{}{
								"displayName": "Test User 1",
							},
							"status": map[string]interface{}{
								"name": "Done",
							},
							"customfield_13192": 5.0,
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
		} else if strings.Contains(query, "Test User 2") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"issues": []map[string]interface{}{
					{
						"key": "TEST-124",
						"fields": map[string]interface{}{
							"summary": "Test Issue 2",
							"assignee": map[string]interface{}{
								"displayName": "Test User 2",
							},
							"status": map[string]interface{}{
								"name": "In Progress",
							},
							"customfield_13192": 3.0,
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
		}
	}))
	defer server.Close()

	// Set the base URL to our test server
	os.Setenv("JIRA_BASE_URL", server.URL)

	adapter, err := NewJiraAdapter()
	require.NoError(t, err, "NewJiraAdapter should not return error")

	// Create a test team
	team := &domain.Team{
		Team: []string{"Test User 1", "Test User 2"},
	}

	issues, err := adapter.GetTeamIssues(team)
	require.NoError(t, err, "GetTeamIssues should not return error")
	require.Len(t, issues, 2, "Should return exactly two issues")

	// Verify first issue
	issue1 := issues[0]
	assert.Equal(t, "TEST-123", issue1.Key)
	assert.Equal(t, "Test Issue 1", issue1.Summary)
	assert.Equal(t, "Test User 1", issue1.Assignee)
	assert.Equal(t, "Done", issue1.Status)
	assert.NotNil(t, issue1.StoryPoints)
	assert.Equal(t, 5.0, *issue1.StoryPoints)

	// Verify second issue
	issue2 := issues[1]
	assert.Equal(t, "TEST-124", issue2.Key)
	assert.Equal(t, "Test Issue 2", issue2.Summary)
	assert.Equal(t, "Test User 2", issue2.Assignee)
	assert.Equal(t, "In Progress", issue2.Status)
	assert.NotNil(t, issue2.StoryPoints)
	assert.Equal(t, 3.0, *issue2.StoryPoints)
}

func TestJiraAdapter_ServerError(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server that returns a server error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Set the base URL to our test server
	os.Setenv("JIRA_BASE_URL", server.URL)

	adapter, err := NewJiraAdapter()
	require.NoError(t, err, "NewJiraAdapter should not return error")

	_, err = adapter.GetIssuesForSprint("TEST", "TEST-1")
	assert.Error(t, err, "GetIssuesForSprint should return error")
}

func TestJiraAdapter_InvalidJSON(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	// Set the base URL to our test server
	os.Setenv("JIRA_BASE_URL", server.URL)

	adapter, err := NewJiraAdapter()
	require.NoError(t, err, "NewJiraAdapter should not return error")

	_, err = adapter.GetIssuesForSprint("TEST", "TEST-1")
	assert.Error(t, err, "GetIssuesForSprint should return error")
}

func TestJiraAdapter_GetSprintIssues(t *testing.T) {
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
						"status": map[string]interface{}{
							"name": "Done",
						},
						"customfield_13192": 5.0,
					},
				},
			},
		})
	}))
	defer server.Close()

	// Set the base URL to our test server
	os.Setenv("JIRA_BASE_URL", server.URL)

	adapter, err := NewJiraAdapter()
	require.NoError(t, err, "NewJiraAdapter should not return error")

	sprint := &domain.Sprint{
		ID:      "TEST-1",
		Name:    "Sprint 1",
		Project: "TEST",
		Team: domain.Team{
			Team: []string{"Test User 1", "Test User 2"},
		},
		Status:    domain.SprintStatusActive,
		StartDate: time.Now().Add(-7 * 24 * time.Hour).Format("2006-01-02"),
		EndDate:   time.Now().Add(7 * 24 * time.Hour).Format("2006-01-02"),
	}

	issues, err := adapter.GetSprintIssues(sprint)
	require.NoError(t, err, "GetSprintIssues should not return error")
	require.NotEmpty(t, issues, "Issues should not be empty")
}
