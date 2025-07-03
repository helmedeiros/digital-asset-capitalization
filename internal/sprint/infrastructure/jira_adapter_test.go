package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
)

// createTestJiraAdapter creates a JiraAdapter for testing with a mock server
// This approach uses environment variable isolation to avoid connecting to real servers
func createTestJiraAdapter(t *testing.T, server *httptest.Server) *JiraAdapter {
	t.Helper()

	// Store original environment variables
	originalJiraBaseURL := os.Getenv("JIRA_BASE_URL")
	originalJiraEmail := os.Getenv("JIRA_EMAIL")
	originalJiraToken := os.Getenv("JIRA_TOKEN")

	// Set test environment variables to point to mock server
	os.Setenv("JIRA_BASE_URL", server.URL)
	os.Setenv("JIRA_EMAIL", "test@example.com")
	os.Setenv("JIRA_TOKEN", "test-token")

	// Create adapter using the standard constructor
	adapter, err := NewJiraAdapter()
	require.NoError(t, err)

	// Schedule cleanup to restore original environment variables
	t.Cleanup(func() {
		if originalJiraBaseURL != "" {
			os.Setenv("JIRA_BASE_URL", originalJiraBaseURL)
		} else {
			os.Unsetenv("JIRA_BASE_URL")
		}

		if originalJiraEmail != "" {
			os.Setenv("JIRA_EMAIL", originalJiraEmail)
		} else {
			os.Unsetenv("JIRA_EMAIL")
		}

		if originalJiraToken != "" {
			os.Setenv("JIRA_TOKEN", originalJiraToken)
		} else {
			os.Unsetenv("JIRA_TOKEN")
		}
	})

	return adapter
}

// setupTestFiles creates necessary test files without touching environment variables
func setupTestFiles(t *testing.T) func() {
	t.Helper()

	// Create test directory
	originalWd, _ := os.Getwd()
	testDir := filepath.Join(originalWd, "testdata", t.Name())
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test teams.json
	teamsJSON := `{
		"FN": {
			"team": ["Test User 1", "Test User 2"]
		}
	}`
	err = os.WriteFile(filepath.Join(testDir, "teams.json"), []byte(teamsJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create test teams.json: %v", err)
	}

	// Create .assetcap directory and teams.json file
	assetCapDir := ".assetcap"
	err = os.MkdirAll(assetCapDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .assetcap directory: %v", err)
	}

	teamsFile := filepath.Join(assetCapDir, "teams.json")
	err = os.WriteFile(teamsFile, []byte(teamsJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create .assetcap/teams.json: %v", err)
	}

	return func() {
		// Clean up test files
		err = os.RemoveAll(assetCapDir)
		if err != nil {
			t.Errorf("Failed to clean up .assetcap directory: %v", err)
		}

		err = os.RemoveAll(filepath.Join(originalWd, "testdata", t.Name()))
		if err != nil {
			t.Errorf("Failed to clean up test directory: %v", err)
		}
	}
}

// Legacy setupTestEnv function - keeping for backwards compatibility with existing tests
func setupTestEnv(t *testing.T) func() {
	// Store original environment variables
	originalJiraBaseURL := os.Getenv("JIRA_BASE_URL")
	originalJiraEmail := os.Getenv("JIRA_EMAIL")
	originalJiraToken := os.Getenv("JIRA_TOKEN")

	// Set up test environment variables
	os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
	os.Setenv("JIRA_EMAIL", "test@example.com")
	os.Setenv("JIRA_TOKEN", "test-token")

	// Set up test files
	cleanupFiles := setupTestFiles(t)

	return func() {
		// Clean up files first
		cleanupFiles()

		// Restore original environment variables
		if originalJiraBaseURL != "" {
			os.Setenv("JIRA_BASE_URL", originalJiraBaseURL)
		} else {
			os.Unsetenv("JIRA_BASE_URL")
		}

		if originalJiraEmail != "" {
			os.Setenv("JIRA_EMAIL", originalJiraEmail)
		} else {
			os.Unsetenv("JIRA_EMAIL")
		}

		if originalJiraToken != "" {
			os.Setenv("JIRA_TOKEN", originalJiraToken)
		} else {
			os.Unsetenv("JIRA_TOKEN")
		}
	}
}

func TestJiraAdapter_GetIssues(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search", r.URL.Path)
		assert.Equal(t, "jql=project+%3D+TEST+AND+sprint+%3D+%27Test+Sprint%27&expand=changelog&fields=summary,assignee,status,changelog,issuetype,customfield_10014,customfield_10015,labels", r.URL.RawQuery)
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
						"customfield_10015": "Test Asset",
						"labels": ["cap-development", "cap-asset-booking"]
					}
				}
			]
		}`))
	}))
	defer server.Close()

	// Create adapter with test server URL
	os.Setenv("JIRA_BASE_URL", server.URL)
	adapter, err := NewJiraAdapter()
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
	assert.Equal(t, []string{"cap-development", "cap-asset-booking"}, issues[0].Labels)
}

func TestJiraAdapter_GetTeamIssues(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search", r.URL.Path)
		assert.Equal(t, "jql=assignee+%3D+%27Test+User+1%27&expand=changelog&fields=summary,assignee,status,changelog,issuetype,customfield_10014,customfield_10015,labels", r.URL.RawQuery)
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
						"customfield_10015": "Test Asset",
						"labels": ["cap-development", "cap-asset-booking"]
					}
				}
			]
		}`))
	}))
	defer server.Close()

	// Create adapter with test server URL
	os.Setenv("JIRA_BASE_URL", server.URL)
	adapter, err := NewJiraAdapter()
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
	assert.Equal(t, []string{"cap-development", "cap-asset-booking"}, issues[0].Labels)
}

func TestJiraAdapter_ServerError(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal Server Error"}`))
	}))
	defer server.Close()

	// Create adapter with test server URL
	os.Setenv("JIRA_BASE_URL", server.URL)
	adapter, err := NewJiraAdapter()
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"invalid json`))
	}))
	defer server.Close()

	// Create adapter with test server URL
	os.Setenv("JIRA_BASE_URL", server.URL)
	adapter, err := NewJiraAdapter()
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
		assert.Equal(t, "jql=project+%3D+TEST+AND+sprint+%3D+%27Test+Sprint%27&expand=changelog&fields=summary,assignee,status,changelog,issuetype,customfield_10014,customfield_10015,labels", r.URL.RawQuery)
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
						"customfield_10015": "Test Asset",
						"labels": ["cap-development", "cap-asset-booking"]
					}
				}
			]
		}`))
	}))
	defer server.Close()

	// Create adapter with test server URL
	os.Setenv("JIRA_BASE_URL", server.URL)
	adapter, err := NewJiraAdapter()
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
	assert.Equal(t, []string{"cap-development", "cap-asset-booking"}, issues[0].Labels)
}

func TestJiraAdapter_GetSprintsForProjectWithBoardInfo(t *testing.T) {
	// Set up test files
	cleanupFiles := setupTestFiles(t)
	defer cleanupFiles()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/board") && !strings.Contains(r.URL.Path, "/sprint") {
			// Boards endpoint
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"values": [
					{"id": 1, "name": "Test Board 1", "type": "scrum"},
					{"id": 2, "name": "Test Board 2", "type": "kanban"}
				]
			}`))
		} else if strings.Contains(r.URL.Path, "/sprint") {
			// Sprints endpoint
			// Extract board ID from URL like /rest/agile/1.0/board/1/sprint
			pathParts := strings.Split(r.URL.Path, "/")
			var boardID string
			for i, part := range pathParts {
				if part == "board" && i+1 < len(pathParts) {
					boardID = pathParts[i+1]
					break
				}
			}

			if boardID == "1" {
				// Scrum board has sprints
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"values": [
						{"id": "123", "name": "Sprint 1", "state": "active", "startDate": "2024-01-01T00:00:00Z", "endDate": "2024-01-15T00:00:00Z", "goal": "Goal 1"}
					],
					"isLast": true,
					"startAt": 0,
					"maxResults": 50
				}`))
			} else {
				// Kanban board has no sprints
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Board does not support sprints"}`))
			}
		}
	}))
	defer server.Close()

	// Create adapter with isolated test environment
	adapter := createTestJiraAdapter(t, server)

	// Test getting sprints with board info
	sprints, boardInfo, err := adapter.GetSprintsForProjectWithBoardInfo("TEST", []string{})
	require.NoError(t, err)
	require.Len(t, sprints, 1)
	require.Len(t, boardInfo, 2)

	// Check sprint
	assert.Equal(t, "123", sprints[0].ID)
	assert.Equal(t, "Sprint 1", sprints[0].Name)
	assert.Equal(t, "active", sprints[0].State)

	// Check board info
	assert.Equal(t, 1, boardInfo[0].ID)
	assert.Equal(t, "Test Board 1", boardInfo[0].Name)
	assert.Equal(t, "scrum", boardInfo[0].Type)
	assert.True(t, boardInfo[0].HasSprints)

	assert.Equal(t, 2, boardInfo[1].ID)
	assert.Equal(t, "Test Board 2", boardInfo[1].Name)
	assert.Equal(t, "kanban", boardInfo[1].Type)
	assert.False(t, boardInfo[1].HasSprints)
}

func TestJiraAdapter_GetSprintsForProject(t *testing.T) {
	// Set up test files
	cleanupFiles := setupTestFiles(t)
	defer cleanupFiles()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/board") && !strings.Contains(r.URL.Path, "/sprint") {
			// Boards endpoint
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"values": [
					{"id": 1, "name": "Test Board", "type": "scrum"}
				]
			}`))
		} else if strings.Contains(r.URL.Path, "/sprint") {
			// Sprints endpoint
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"values": [
					{"id": "123", "name": "Sprint 1", "state": "active", "startDate": "2024-01-01T00:00:00Z", "endDate": "2024-01-15T00:00:00Z", "goal": "Goal 1"}
				],
				"isLast": true,
				"startAt": 0,
				"maxResults": 50
			}`))
		}
	}))
	defer server.Close()

	// Create adapter with isolated test environment
	adapter := createTestJiraAdapter(t, server)

	// Test getting sprints
	sprints, err := adapter.GetSprintsForProject("TEST", []string{"active"})
	require.NoError(t, err)
	require.Len(t, sprints, 1)

	assert.Equal(t, "123", sprints[0].ID)
	assert.Equal(t, "Sprint 1", sprints[0].Name)
	assert.Equal(t, "active", sprints[0].State)
}

func TestJiraAdapter_GetTeamIssuesComplete(t *testing.T) {
	// Set up test files
	cleanupFiles := setupTestFiles(t)
	defer cleanupFiles()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search", r.URL.Path)
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
						"customfield_10015": "Test Asset",
						"labels": ["cap-development"]
					}
				}
			]
		}`))
	}))
	defer server.Close()

	// Create adapter with isolated test environment
	adapter := createTestJiraAdapter(t, server)

	// Create a test team
	team := &domain.Team{
		Team: []string{"Test User 1", "Test User 2"},
	}

	// Test getting team issues
	issues, err := adapter.GetTeamIssues(team)
	require.NoError(t, err)
	require.Len(t, issues, 2) // 2 team members, each returns 1 issue

	assert.Equal(t, "TEST-1", issues[0].Key)
	assert.Equal(t, "Test Issue 1", issues[0].Summary)
}

func TestJiraAdapter_ErrorHandling(t *testing.T) {
	// Set up test files
	cleanupFiles := setupTestFiles(t)
	defer cleanupFiles()

	t.Run("boards error", func(t *testing.T) {
		// Create a test server that returns an error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Internal Server Error"}`))
		}))
		defer server.Close()

		// Create adapter with isolated test environment
		adapter := createTestJiraAdapter(t, server)

		// Test error handling
		sprints, boardInfo, err := adapter.GetSprintsForProjectWithBoardInfo("TEST", []string{})
		require.Error(t, err)
		assert.Nil(t, sprints)
		assert.Nil(t, boardInfo)
		assert.Contains(t, err.Error(), "failed to get boards for project")
	})

	t.Run("sprint board error with warning", func(t *testing.T) {
		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/board") && !strings.Contains(r.URL.Path, "/sprint") {
				// Boards endpoint
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"values": [
						{"id": 1, "name": "Test Board", "type": "scrum"}
					]
				}`))
			} else if strings.Contains(r.URL.Path, "/sprint") {
				// Sprints endpoint returns error
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Board does not support sprints"}`))
			}
		}))
		defer server.Close()

		// Create adapter with isolated test environment
		adapter := createTestJiraAdapter(t, server)

		// Test that it continues with other boards when one fails
		sprints, err := adapter.GetSprintsForProject("TEST", []string{"active"})
		require.NoError(t, err)
		assert.Len(t, sprints, 0) // No sprints due to error, but no failure
	})
}
