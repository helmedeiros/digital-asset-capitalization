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
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

const fieldAPIPath = "/rest/api/3/field"

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
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)
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
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
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
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)
		assert.Equal(t, "jql=project+%3D+TEST+AND+sprint+%3D+%27Test+Sprint%27&expand=changelog&fields=summary,assignee,status,changelog,issuetype,parent,customfield_10014,customfield_10015,labels", r.URL.RawQuery)
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
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
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
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
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
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)
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

// fieldAPIResponse returns a JSON response that maps well-known field names to custom field IDs.
const fieldAPIResponse = `[
	{"id":"customfield_17961","name":"TPD Business Unit"},
	{"id":"customfield_18000","name":"Engineering time spent (hours)"},
	{"id":"customfield_18837","name":"Work Stream"}
]`

func TestJiraAdapter_UpdateCustomFields_Success(t *testing.T) {
	cleanupFiles := setupTestFiles(t)
	defer cleanupFiles()

	var putBodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fieldAPIResponse))
			return
		}
		if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/") {
			body := make([]byte, r.ContentLength)
			r.Body.Read(body)
			putBodies = append(putBodies, string(body))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := createTestJiraAdapter(t, server)

	hours := 12.5
	ws := "Product"
	err := adapter.UpdateCustomFields("COP-1", ports.CustomFieldUpdate{
		EngineeringHours: &hours,
		WorkStream:       &ws,
		TPDBusinessUnits: []string{"B2C"},
	})
	require.NoError(t, err)
	assert.Len(t, putBodies, 3)

	// Verify each PUT sent individually
	assert.Contains(t, putBodies[0], "customfield_18000")
	assert.Contains(t, putBodies[0], "12.5")
	assert.Contains(t, putBodies[1], "customfield_18837")
	assert.Contains(t, putBodies[1], "Product")
	assert.Contains(t, putBodies[2], "customfield_17961")
	assert.Contains(t, putBodies[2], "B2C")
}

func TestJiraAdapter_UpdateCustomFields_PartialFailure(t *testing.T) {
	cleanupFiles := setupTestFiles(t)
	defer cleanupFiles()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fieldAPIResponse))
			return
		}
		if r.Method == http.MethodPut {
			callCount++
			if callCount == 1 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"errorMessages":["Field not on screen"]}`))
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := createTestJiraAdapter(t, server)

	hours := 10.0
	ws := "Operational"
	err := adapter.UpdateCustomFields("COP-2", ports.CustomFieldUpdate{
		EngineeringHours: &hours,
		WorkStream:       &ws,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "partial update")
	assert.Contains(t, err.Error(), "Engineering Hours")
	assert.Equal(t, 2, callCount) // both fields attempted
}

func TestJiraAdapter_UpdateCustomFields_NilFieldIDs(t *testing.T) {
	cleanupFiles := setupTestFiles(t)
	defer cleanupFiles()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`)) // no fields → empty fieldIDs
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := createTestJiraAdapter(t, server)
	// fieldIDs will be non-nil but all empty strings → no entries → nil return
	hours := 5.0
	err := adapter.UpdateCustomFields("COP-3", ports.CustomFieldUpdate{EngineeringHours: &hours})
	require.NoError(t, err) // no entries to send → no error
}

func TestJiraAdapter_FetchCustomFields_Success(t *testing.T) {
	cleanupFiles := setupTestFiles(t)
	defer cleanupFiles()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fieldAPIResponse))
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/COP-1") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"key": "COP-1",
				"fields": {
					"customfield_18000": 8.5,
					"customfield_18837": {"value": "Operational"},
					"customfield_17961": [{"value": "B2B"}, {"value": "B2C"}]
				}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := createTestJiraAdapter(t, server)

	vals, err := adapter.FetchCustomFields("COP-1")
	require.NoError(t, err)
	require.NotNil(t, vals)
	require.NotNil(t, vals.EngineeringHours)
	assert.Equal(t, 8.5, *vals.EngineeringHours)
	assert.Equal(t, "Operational", vals.WorkStream)
	assert.Equal(t, []string{"B2B", "B2C"}, vals.TPDBusinessUnits)
}

func TestJiraAdapter_FetchCustomFields_Empty(t *testing.T) {
	cleanupFiles := setupTestFiles(t)
	defer cleanupFiles()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fieldAPIResponse))
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/COP-2") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"key": "COP-2",
				"fields": {}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := createTestJiraAdapter(t, server)

	vals, err := adapter.FetchCustomFields("COP-2")
	require.NoError(t, err)
	require.NotNil(t, vals)
	assert.Nil(t, vals.EngineeringHours)
	assert.Empty(t, vals.WorkStream)
	assert.Empty(t, vals.TPDBusinessUnits)
}

func TestJiraAdapter_FetchCustomFields_ServerError(t *testing.T) {
	cleanupFiles := setupTestFiles(t)
	defer cleanupFiles()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fieldAPIResponse))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "server error"}`))
	}))
	defer server.Close()

	adapter := createTestJiraAdapter(t, server)

	vals, err := adapter.FetchCustomFields("COP-3")
	require.Error(t, err)
	assert.Nil(t, vals)
	assert.Contains(t, err.Error(), "failed to fetch issue COP-3")
}

func TestJiraAdapter_BuildFieldsParam_IncludesParent(t *testing.T) {
	cleanupFiles := setupTestFiles(t)
	defer cleanupFiles()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"issues":[]}`))
	}))
	defer server.Close()

	adapter := createTestJiraAdapter(t, server)
	fields := adapter.buildFieldsParam()
	assert.Contains(t, fields, "parent")
}

func TestJiraAdapter_ConvertToPortIssues_PopulatesParentKey(t *testing.T) {
	cleanupFiles := setupTestFiles(t)
	defer cleanupFiles()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fieldAPIPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"issues": [
				{
					"key": "PRJ-102",
					"fields": {
						"summary": "Subtask of PRJ-67",
						"assignee": {"displayName": "Developer A"},
						"status": {"name": "Done"},
						"issuetype": {"name": "Sub-task"},
						"parent": {"key": "PRJ-67"},
						"labels": []
					}
				},
				{
					"key": "PRJ-67",
					"fields": {
						"summary": "Parent Story",
						"assignee": {"displayName": "Developer B"},
						"status": {"name": "Done"},
						"issuetype": {"name": "Story"},
						"labels": []
					}
				}
			]
		}`))
	}))
	defer server.Close()

	adapter := createTestJiraAdapter(t, server)
	issues, err := adapter.GetIssuesForSprint("PRJ", "Sprint 1")
	require.NoError(t, err)
	require.Len(t, issues, 2)

	// Subtask should have parent key populated
	assert.Equal(t, "PRJ-102", issues[0].Key)
	assert.Equal(t, "PRJ-67", issues[0].ParentKey)

	// Parent story should have no parent key
	assert.Equal(t, "PRJ-67", issues[1].Key)
	assert.Equal(t, "", issues[1].ParentKey)
}

func TestJiraAdapter_ErrorHandling(t *testing.T) {
	// Set up test files
	cleanupFiles := setupTestFiles(t)
	defer cleanupFiles()

	t.Run("boards error", func(t *testing.T) {
		// Create a test server that returns an error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == fieldAPIPath {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[]`))
				return
			}
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
			if r.URL.Path == fieldAPIPath {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[]`))
				return
			}
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
