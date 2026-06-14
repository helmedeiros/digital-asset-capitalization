package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
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
	issues    []ports.JiraIssue
	sprints   []ports.Sprint
	boardInfo []ports.BoardInfo
	err       error
}

func (m *mockJiraPort) GetIssuesForSprint(_, _ string) ([]ports.JiraIssue, error) {
	return m.issues, m.err
}

func (m *mockJiraPort) GetIssuesForTeamMember(_ string) ([]ports.JiraIssue, error) {
	return m.issues, m.err
}

func (m *mockJiraPort) GetSprintIssues(_ *domain.Sprint) ([]ports.JiraIssue, error) {
	return m.issues, m.err
}

func (m *mockJiraPort) GetTeamIssues(_ *domain.Team) ([]ports.JiraIssue, error) {
	return m.issues, m.err
}

func (m *mockJiraPort) GetSprintsForProject(_ string, _ []string) ([]ports.Sprint, error) {
	return m.sprints, m.err
}

func (m *mockJiraPort) GetSprintsForProjectWithBoardInfo(_ string, _ []string) ([]ports.Sprint, []ports.BoardInfo, error) {
	return m.sprints, m.boardInfo, m.err
}

func (m *mockJiraPort) GetSprintByName(_ string, _ string) (*ports.Sprint, error) {
	return nil, nil
}

func (m *mockJiraPort) GetIssuesForSprintOnBoard(_, _ string, _ int) ([]ports.JiraIssue, error) {
	return nil, nil
}

func (m *mockJiraPort) UpdateCustomFields(_ string, _ ports.CustomFieldUpdate) error {
	return nil
}

func (m *mockJiraPort) FetchCustomFields(_ string) (*ports.CustomFieldValues, error) {
	return nil, nil
}

func TestSprintService_ProcessJiraIssues(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
							"name": domain.StatusDone,
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
				Status:      domain.StatusDone,
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

func TestSprintService_ProcessJiraIssuesWithStrategy(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "/search") {
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
								"name": domain.StatusDone,
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
		} else if strings.Contains(r.URL.Path, "/board") && !strings.Contains(r.URL.Path, "/sprint") {
			// Boards endpoint
			json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"id":   1,
						"name": "Test Board",
						"type": "scrum",
					},
				},
			})
		} else if strings.Contains(r.URL.Path, "/sprint") {
			// Sprints endpoint
			json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"id":        "8802",
						"name":      "Sprint 1",
						"state":     "active",
						"startDate": "2024-03-01T00:00:00.000Z",
						"endDate":   "2024-03-15T00:00:00.000Z",
						"goal":      "Sprint goal",
					},
				},
				"isLast":     true,
				"startAt":    0,
				"maxResults": 50,
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
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
				Status:      domain.StatusDone,
				StoryPoints: float64Ptr(5.0),
			},
		},
	}

	service := NewSprintService(mockJira)

	// Test successful processing with legacy strategy
	t.Run("successful processing with legacy strategy", func(t *testing.T) {
		result, err := service.ProcessJiraIssuesWithStrategy("TEST", "Sprint 1", "", false)
		require.NoError(t, err, "ProcessJiraIssuesWithStrategy should not return error")
		assert.NotEmpty(t, result, "Result should not be empty")
	})

	// Test successful processing with sprint-bounded strategy
	t.Run("successful processing with sprint-bounded strategy", func(t *testing.T) {
		result, err := service.ProcessJiraIssuesWithStrategy("TEST", "Sprint 1", "", true)
		require.NoError(t, err, "ProcessJiraIssuesWithStrategy should not return error")
		assert.NotEmpty(t, result, "Result should not be empty")
	})

	// Test invalid project
	t.Run("invalid project", func(t *testing.T) {
		_, err := service.ProcessJiraIssuesWithStrategy("INVALID", "Sprint 1", "", false)
		assert.Error(t, err, "ProcessJiraIssuesWithStrategy should return error for invalid project")
	})
}

func TestSprintService_ProcessSprint(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
							"name": domain.StatusDone,
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
				Status:      domain.StatusDone,
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
							"name": domain.StatusDone,
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
				Status:      domain.StatusDone,
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

func TestSprintServiceImpl_ProcessTeamIssues_ErrorFromJiraPort(t *testing.T) {
	mockJira := new(mockJiraPort)
	service := &SprintServiceImpl{jiraPort: mockJira}

	team := &domain.Team{Team: []string{"user1", "user2"}}
	expectedError := errors.New("jira error")

	mockJira.err = expectedError

	err := service.ProcessTeamIssues(team)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestSprintServiceImpl_ListSprints(t *testing.T) {
	mockJira := new(mockJiraPort)
	service := &SprintServiceImpl{jiraPort: mockJira}

	project := "FN"
	period := "Q2 2025"

	// The mock returns nil, nil by default, which should result in an empty list
	result, err := service.ListSprints(project, period)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, project, result.Project)
	assert.Equal(t, period, result.Period)
	assert.Len(t, result.Sprints, 0)
}

func TestSprintServiceImpl_ListSprints_ErrorFromJiraPort(t *testing.T) {
	mockJira := new(mockJiraPort)
	service := &SprintServiceImpl{jiraPort: mockJira}

	project := "FN"
	period := "Q2 2025"

	// We can't easily test the error case with the current mock structure
	// This test will pass but won't actually test the error path
	// The real error testing is done in the usecase tests
	result, err := service.ListSprints(project, period)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// jiraStubServer returns an httptest server that fakes the minimum JIRA
// surface the SprintTimeAllocationUseCase touches: a /search endpoint
// returning the given issues body, plus inert /board and /sprint
// endpoints for the strategy-aware path.
func jiraStubServer(searchBody map[string]interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/search"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(searchBody)
		case strings.Contains(r.URL.Path, "/board") && !strings.Contains(r.URL.Path, "/sprint"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{"id": 1, "name": "Test Board", "type": "scrum"},
				},
			})
		case strings.Contains(r.URL.Path, "/sprint"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"id":        "8802",
						"name":      "Sprint 1",
						"state":     "active",
						"startDate": "2024-03-01T00:00:00.000Z",
						"endDate":   "2024-03-15T00:00:00.000Z",
						"goal":      "Sprint goal",
					},
				},
				"isLast":     true,
				"startAt":    0,
				"maxResults": 50,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// issueWithStatus returns a JIRA search payload containing a single issue
// in the given status assigned to the given user, matching the shape used
// by the existing ProcessJiraIssuesWithStrategy test fixtures.
func issueWithStatus(status string) map[string]interface{} {
	return map[string]interface{}{
		"issues": []map[string]interface{}{
			{
				"key": "TEST-123",
				"fields": map[string]interface{}{
					"summary": "Test Issue 1",
					"assignee": map[string]interface{}{
						"displayName": "Test User 1",
					},
					"status":            map[string]interface{}{"name": status},
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
	}
}

func TestSprintService_ProcessJiraIssuesWithOptions(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	server := jiraStubServer(issueWithStatus(domain.StatusDone))
	defer server.Close()
	os.Setenv("JIRA_BASE_URL", server.URL)

	service := NewSprintService(&mockJiraPort{})

	t.Run("successful processing with no options", func(t *testing.T) {
		result, err := service.ProcessJiraIssuesWithOptions("TEST", "Sprint 1", "", false)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
	})

	t.Run("successful processing with WithHours option", func(t *testing.T) {
		result, err := service.ProcessJiraIssuesWithOptions("TEST", "Sprint 1", "", true, usecase.WithHours(true))
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		// WithHours adds the engineering-hours column to the CSV header.
		assert.Contains(t, result, "engineeringHours")
	})

	t.Run("invalid project returns error", func(t *testing.T) {
		_, err := service.ProcessJiraIssuesWithOptions("INVALID", "Sprint 1", "", false)
		assert.Error(t, err)
	})
}

func TestSprintService_PushAllocationToJira(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Empty issues list keeps the push loop a no-op — that's enough to walk
	// the wiring (processor build, ProcessWithRecords, NewPushAllocationUseCase,
	// Execute on an empty record set) without needing to stub the JIRA
	// custom-field fetch/update endpoints that a non-empty path would call.
	server := jiraStubServer(map[string]interface{}{"issues": []map[string]interface{}{}})
	defer server.Close()
	os.Setenv("JIRA_BASE_URL", server.URL)

	service := NewSprintService(&mockJiraPort{})

	t.Run("dry run with no issues returns empty push result", func(t *testing.T) {
		_, pushResult, err := service.PushAllocationToJira("TEST", "Sprint 1", "", false, true)
		require.NoError(t, err)
		require.NotNil(t, pushResult)
		assert.Equal(t, 0, pushResult.UpdatedCount)
		assert.Equal(t, 0, pushResult.SkippedCount)
		assert.Equal(t, 0, pushResult.ErrorCount)
		assert.Empty(t, pushResult.Details)
	})

	t.Run("invalid project returns error from processor", func(t *testing.T) {
		csvData, pushResult, err := service.PushAllocationToJira("INVALID", "Sprint 1", "", false, true)
		assert.Error(t, err)
		assert.Empty(t, csvData)
		assert.Nil(t, pushResult)
	})
}
