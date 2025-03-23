package application

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
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

type mockJiraPort struct {
	issues []ports.JiraIssue
	err    error
}

func (m *mockJiraPort) GetIssuesForSprint(project, sprintID string) ([]ports.JiraIssue, error) {
	return m.issues, m.err
}

func (m *mockJiraPort) GetIssuesForTeamMember(member string) ([]ports.JiraIssue, error) {
	return m.issues, m.err
}

func (m *mockJiraPort) GetSprintIssues(sprint *domain.Sprint) ([]ports.JiraIssue, error) {
	return m.issues, m.err
}

func (m *mockJiraPort) GetTeamIssues(team *domain.Team) ([]ports.JiraIssue, error) {
	return m.issues, m.err
}

func TestSprintService_ProcessJiraIssues(t *testing.T) {
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

	mockJira := &mockJiraPort{
		issues: []ports.JiraIssue{
			{
				Key:         "TEST-123",
				Summary:     "Test Issue 1",
				Assignee:    "Test User 1",
				Status:      "Done",
				StoryPoints: float64Ptr(5.0),
			},
		},
	}

	service := NewSprintService(mockJira)

	// Test successful processing
	t.Run("successful processing", func(t *testing.T) {
		result, err := service.ProcessJiraIssues("TEST", "Sprint 1", "")
		require.NoError(t, err, "ProcessJiraIssues should not return error")
		assert.NotEmpty(t, result, "Result should not be empty")
	})

	// Test invalid project
	t.Run("invalid project", func(t *testing.T) {
		_, err := service.ProcessJiraIssues("INVALID", "Sprint 1", "")
		assert.Error(t, err, "ProcessJiraIssues should return error for invalid project")
	})
}

func TestSprintService_ProcessSprint(t *testing.T) {
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

	mockJira := &mockJiraPort{
		issues: []ports.JiraIssue{
			{
				Key:         "TEST-123",
				Summary:     "Test Issue 1",
				Assignee:    "Test User 1",
				Status:      "Done",
				StoryPoints: float64Ptr(5.0),
			},
		},
	}

	service := NewSprintService(mockJira)

	// Test successful processing
	t.Run("successful processing", func(t *testing.T) {
		sprint := &domain.Sprint{
			ID:        "TEST-1",
			Name:      "Sprint 1",
			Project:   "TEST",
			Status:    domain.SprintStatusActive,
			StartDate: time.Now().Add(-7 * 24 * time.Hour).Format("2006-01-02"),
			EndDate:   time.Now().Add(7 * 24 * time.Hour).Format("2006-01-02"),
		}

		err := service.ProcessSprint("TEST", sprint)
		require.NoError(t, err, "ProcessSprint should not return error")
	})

	// Test error from Jira port
	t.Run("error from Jira port", func(t *testing.T) {
		sprint := &domain.Sprint{
			ID:        "TEST-1",
			Name:      "Sprint 1",
			Project:   "TEST",
			Status:    domain.SprintStatusActive,
			StartDate: time.Now().Add(-7 * 24 * time.Hour).Format("2006-01-02"),
			EndDate:   time.Now().Add(7 * 24 * time.Hour).Format("2006-01-02"),
		}

		mockJiraWithError := &mockJiraPort{
			err: fmt.Errorf("jira error"),
		}
		serviceWithError := NewSprintService(mockJiraWithError)

		err := serviceWithError.ProcessSprint("TEST", sprint)
		assert.Error(t, err, "ProcessSprint should return error")
	})
}

func TestSprintService_ProcessTeamIssues(t *testing.T) {
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

	mockJira := &mockJiraPort{
		issues: []ports.JiraIssue{
			{
				Key:         "TEST-123",
				Summary:     "Test Issue 1",
				Assignee:    "Test User 1",
				Status:      "Done",
				StoryPoints: float64Ptr(5.0),
			},
		},
	}

	service := NewSprintService(mockJira)

	// Test successful processing
	t.Run("successful processing", func(t *testing.T) {
		team := &domain.Team{
			Team: []string{"Test User 1", "Test User 2"},
		}

		err := service.ProcessTeamIssues(team)
		require.NoError(t, err, "ProcessTeamIssues should not return error")
	})

	// Test error from Jira port
	t.Run("error from Jira port", func(t *testing.T) {
		team := &domain.Team{
			Team: []string{"Test User 1", "Test User 2"},
		}

		mockJiraWithError := &mockJiraPort{
			err: fmt.Errorf("jira error"),
		}
		serviceWithError := NewSprintService(mockJiraWithError)

		err := serviceWithError.ProcessTeamIssues(team)
		assert.Error(t, err, "ProcessTeamIssues should return error")
	})
}

func float64Ptr(v float64) *float64 {
	return &v
}
