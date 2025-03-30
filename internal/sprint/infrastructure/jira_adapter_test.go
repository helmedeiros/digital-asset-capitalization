package infrastructure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEnv(t *testing.T) func() {
	// Create test directory
	testDir := filepath.Join("testdata", t.Name())
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err, "Failed to create test directory")

	// Create .assetcap directory
	assetcapDir := filepath.Join(testDir, ".assetcap")
	err = os.MkdirAll(assetcapDir, 0755)
	require.NoError(t, err, "Failed to create .assetcap directory")

	teamsFilePath := filepath.Join(assetcapDir, "teams.json")

	// Create a temporary teams.json file
	teams := domain.TeamMap{
		"TEST": domain.Team{
			Team: []string{"Test User 1", "Test User 2"},
		},
	}

	data, err := json.Marshal(teams)
	require.NoError(t, err, "Failed to marshal teams data")

	err = os.WriteFile(teamsFilePath, data, 0644)
	require.NoError(t, err, "Failed to write teams.json")

	// Get current working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")

	// Change working directory to test directory
	err = os.Chdir(testDir)
	require.NoError(t, err, "Failed to change working directory")

	// Set environment variables for testing
	os.Setenv("JIRA_BASE_URL", "http://test.jira.com")
	os.Setenv("JIRA_EMAIL", "test@example.com")
	os.Setenv("JIRA_TOKEN", "test-token")

	// Return cleanup function
	return func() {
		// Restore original working directory
		err := os.Chdir(originalWd)
		if err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}

		// Clean up test directory
		err = os.RemoveAll(filepath.Join(originalWd, "testdata", t.Name()))
		if err != nil {
			t.Errorf("Failed to clean up test directory: %v", err)
		}

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
		assert.Equal(t, "/rest/api/3/search", r.URL.Path)
		assert.Equal(t, "jql=project+%3D+TEST+AND+sprint+%3D+%27Test+Sprint%27&expand=changelog&fields=summary,assignee,status,changelog,issuetype,customfield_10014,customfield_10015", r.URL.RawQuery)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"issues": [
				{
					"key": "TEST-1",
					"fields": {
						"summary": "Test Issue 1",
						"assignee": {"displayName": "Test User 1"},
						"status": {"name": "In Progress"},
						"issuetype": {"name": "Task"},
						"customfield_10014": "Development",
						"customfield_10015": "Test Asset"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	// Create adapter with test server URL
	os.Setenv("JIRA_BASE_URL", server.URL)
	adapter, err := NewJiraAdapter(t.TempDir() + "/teams.json")
	require.NoError(t, err)
	require.NotNil(t, adapter)

	// Test getting issues
	issues, err := adapter.GetIssuesForSprint("TEST", "Test Sprint")
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "TEST-1", issues[0].Key)
	assert.Equal(t, "Test Issue 1", issues[0].Summary)
	assert.Equal(t, "Test User 1", issues[0].Assignee)
	assert.Equal(t, "In Progress", issues[0].Status)
}

func TestJiraAdapter_GetTeamIssues(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search", r.URL.Path)
		assert.Equal(t, "jql=assignee+%3D+%27Test+User+1%27&expand=changelog&fields=summary,assignee,status,changelog", r.URL.RawQuery)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"issues": [
				{
					"key": "TEST-1",
					"fields": {
						"summary": "Test Issue 1",
						"assignee": {"displayName": "Test User 1"},
						"status": {"name": "In Progress"}
					}
				}
			]
		}`))
	}))
	defer server.Close()

	// Create adapter with test server URL
	os.Setenv("JIRA_BASE_URL", server.URL)
	adapter, err := NewJiraAdapter(t.TempDir() + "/teams.json")
	require.NoError(t, err)
	require.NotNil(t, adapter)

	// Test getting team issues
	issues, err := adapter.GetIssuesForTeamMember("Test User 1")
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "TEST-1", issues[0].Key)
	assert.Equal(t, "Test Issue 1", issues[0].Summary)
	assert.Equal(t, "Test User 1", issues[0].Assignee)
	assert.Equal(t, "In Progress", issues[0].Status)
}

func TestJiraAdapter_ServerError(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal Server Error"}`))
	}))
	defer server.Close()

	// Create adapter with test server URL
	os.Setenv("JIRA_BASE_URL", server.URL)
	adapter, err := NewJiraAdapter(t.TempDir() + "/teams.json")
	require.NoError(t, err)
	require.NotNil(t, adapter)

	// Test getting issues with server error
	issues, err := adapter.GetIssuesForSprint("TEST", "Test Sprint")
	require.Error(t, err)
	assert.Nil(t, issues)
	assert.Contains(t, err.Error(), "failed to fetch sprint issues")
}

func TestJiraAdapter_InvalidJSON(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"invalid json`))
	}))
	defer server.Close()

	// Create adapter with test server URL
	os.Setenv("JIRA_BASE_URL", server.URL)
	adapter, err := NewJiraAdapter(t.TempDir() + "/teams.json")
	require.NoError(t, err)
	require.NotNil(t, adapter)

	// Test getting issues with invalid JSON
	issues, err := adapter.GetIssuesForSprint("TEST", "Test Sprint")
	require.Error(t, err)
	assert.Nil(t, issues)
	assert.Contains(t, err.Error(), "failed to fetch sprint issues")
}

func TestJiraAdapter_GetSprintIssues(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search", r.URL.Path)
		assert.Equal(t, "jql=project+%3D+TEST+AND+sprint+%3D+%27Test+Sprint%27&expand=changelog&fields=summary,assignee,status,changelog,issuetype,customfield_10014,customfield_10015", r.URL.RawQuery)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"issues": [
				{
					"key": "TEST-1",
					"fields": {
						"summary": "Test Issue 1",
						"assignee": {"displayName": "Test User 1"},
						"status": {"name": "In Progress"},
						"issuetype": {"name": "Task"},
						"customfield_10014": "Development",
						"customfield_10015": "Test Asset"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	// Create adapter with test server URL
	os.Setenv("JIRA_BASE_URL", server.URL)
	adapter, err := NewJiraAdapter(t.TempDir() + "/teams.json")
	require.NoError(t, err)
	require.NotNil(t, adapter)

	// Create a test sprint
	sprint := &domain.Sprint{
		ID:      "Test Sprint",
		Project: "TEST",
	}

	// Test getting sprint issues
	issues, err := adapter.GetSprintIssues(sprint)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "TEST-1", issues[0].Key)
	assert.Equal(t, "Test Issue 1", issues[0].Summary)
	assert.Equal(t, "Test User 1", issues[0].Assignee)
	assert.Equal(t, "In Progress", issues[0].Status)
}
