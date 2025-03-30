package usecase

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/config"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// setupTestEnv sets up the test environment and returns a cleanup function
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

	// Create test data
	team := &domain.Team{
		Team: []string{"Test User 1", "Test User 2"},
	}

	// Create test issues with changelog entries
	now := time.Now()
	issues := []domain.JiraIssue{
		{
			Key: "TEST-123",
			Fields: domain.JiraFields{
				Summary: "Test Issue 1",
				Assignee: domain.JiraAssignee{
					DisplayName: "Test User 1",
				},
				Status: domain.JiraStatus{
					Name: "Done",
				},
			},
			Changelog: domain.JiraChangelog{
				Histories: []domain.JiraChangeHistory{
					{
						Created: now.Add(-24 * time.Hour).Format("2006-01-02T15:04:05.000-0700"),
						Items: []domain.JiraChangeItem{
							{
								Field:      "status",
								FromString: "To Do",
								ToString:   "In Progress",
							},
						},
					},
					{
						Created: now.Format("2006-01-02T15:04:05.000-0700"),
						Items: []domain.JiraChangeItem{
							{
								Field:      "status",
								FromString: "In Progress",
								ToString:   "Done",
							},
						},
					},
				},
			},
		},
		{
			Key: "TEST-124",
			Fields: domain.JiraFields{
				Summary: "Test Issue 2",
				Assignee: domain.JiraAssignee{
					DisplayName: "Test User 2",
				},
				Status: domain.JiraStatus{
					Name: "In Progress",
				},
			},
			Changelog: domain.JiraChangelog{
				Histories: []domain.JiraChangeHistory{
					{
						Created: now.Add(-48 * time.Hour).Format("2006-01-02T15:04:05.000-0700"),
						Items: []domain.JiraChangeItem{
							{
								Field:      "status",
								FromString: "To Do",
								ToString:   "In Progress",
							},
						},
					},
				},
			},
		},
	}

	// Create a new processor
	processor := &SprintTimeAllocationUseCase{
		project:  "TEST",
		sprint:   "TEST-1",
		override: "",
		teams: domain.TeamMap{
			"TEST": domain.Team{
				Team: []string{"Test User 1", "Test User 2"},
			},
		},
	}

	// Calculate total hours
	totalHoursByPerson := processor.calculateTotalHours(*team, issues, nil)

	// Assert that both team members have non-zero hours
	assert.NotZero(t, totalHoursByPerson["Test User 1"], "Test User 1 should have non-zero hours")
	assert.NotZero(t, totalHoursByPerson["Test User 2"], "Test User 2 should have non-zero hours")
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
							{
								"created": "2024-03-02T15:00:00.000+0000",
								"items": []map[string]interface{}{
									{
										"field":      "status",
										"fromString": "In Progress",
										"toString":   "Done",
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

	processor, err := NewSprintTimeAllocationUseCase("TEST", "Sprint 1", "")
	require.NoError(t, err, "NewJiraProcessor should not return error")

	issues, err := processor.fetchIssues()
	require.NoError(t, err, "fetchIssues should not return error")
	require.NotEmpty(t, issues, "Issues should not be empty")

	startTime, endTime := processor.getIssueTimeRange(issues[0])
	assert.NotZero(t, startTime, "Start time should not be zero")
	assert.NotZero(t, endTime, "End time should not be zero")
	assert.Equal(t, time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC), startTime)
	assert.Equal(t, time.Date(2024, 3, 2, 15, 0, 0, 0, time.UTC), endTime)
}

func TestJiraProcessor_Process(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a mock Jira adapter
	mockJira := new(MockJiraAdapter)

	// Create a new processor with the mock adapter
	processor := &SprintTimeAllocationUseCase{
		project:  "TEST",
		sprint:   "TEST-1",
		override: "",
		teams: domain.TeamMap{
			"TEST": domain.Team{
				Team: []string{"Test User 1", "Test User 2"},
			},
		},
		jiraPort: mockJira,
		config:   &config.JiraConfig{},
	}

	// Set up mock expectations for GetIssuesForSprint
	mockJira.On("GetIssuesForSprint", "TEST", "TEST-1").Return([]ports.JiraIssue{
		{
			Key:      "TEST-123",
			Summary:  "Test Issue 1",
			Assignee: "Test User 1",
			Status:   "Done",
			Changelog: ports.JiraChangelog{
				Histories: []ports.JiraChangeHistory{
					{
						Created: "2024-03-20T10:00:00.000Z",
						Items: []ports.JiraChangeItem{
							{
								Field:      "status",
								FromString: "To Do",
								ToString:   "In Progress",
							},
						},
					},
					{
						Created: "2024-03-21T15:00:00.000Z",
						Items: []ports.JiraChangeItem{
							{
								Field:      "status",
								FromString: "In Progress",
								ToString:   "Done",
							},
						},
					},
				},
			},
		},
		{
			Key:      "TEST-124",
			Summary:  "Test Issue 2",
			Assignee: "Test User 2",
			Status:   "In Progress",
			Changelog: ports.JiraChangelog{
				Histories: []ports.JiraChangeHistory{
					{
						Created: "2024-03-20T11:00:00.000Z",
						Items: []ports.JiraChangeItem{
							{
								Field:      "status",
								FromString: "To Do",
								ToString:   "In Progress",
							},
						},
					},
				},
			},
		},
	}, nil)

	// Process the issues
	csvData, err := processor.Process()

	// Assert no error occurred
	assert.NoError(t, err)

	// Assert we got CSV data
	assert.NotEmpty(t, csvData)

	// Verify mock expectations were met
	mockJira.AssertExpectations(t)
}

// MockJiraAdapter is a mock implementation of the JiraAdapter interface
type MockJiraAdapter struct {
	mock.Mock
}

func (m *MockJiraAdapter) GetSprintIssues(sprint *domain.Sprint) ([]ports.JiraIssue, error) {
	args := m.Called(sprint)
	return args.Get(0).([]ports.JiraIssue), args.Error(1)
}

func (m *MockJiraAdapter) GetTeamIssues(team *domain.Team) ([]ports.JiraIssue, error) {
	args := m.Called(team)
	return args.Get(0).([]ports.JiraIssue), args.Error(1)
}

func (m *MockJiraAdapter) GetIssuesForSprint(project, sprintID string) ([]ports.JiraIssue, error) {
	args := m.Called(project, sprintID)
	return args.Get(0).([]ports.JiraIssue), args.Error(1)
}

func (m *MockJiraAdapter) GetIssuesForTeamMember(teamMember string) ([]ports.JiraIssue, error) {
	args := m.Called(teamMember)
	return args.Get(0).([]ports.JiraIssue), args.Error(1)
}

func TestGetIssueTimeRange(t *testing.T) {
	processor := &SprintTimeAllocationUseCase{}

	tests := []struct {
		name           string
		issue          domain.JiraIssue
		expectedStart  time.Time
		expectedEnd    time.Time
		expectedStatus string
	}{
		{
			name: "Direct to Done",
			issue: domain.JiraIssue{
				Key: "TEST-1",
				Changelog: domain.JiraChangelog{
					Histories: []domain.JiraChangeHistory{
						{
							Created: "2024-03-20T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "To Do",
									ToString:   "Done",
								},
							},
						},
					},
				},
			},
			expectedStart: time.Date(2024, 3, 20, 10, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2024, 3, 20, 10, 0, 0, 0, time.UTC),
		},
		{
			name: "Through In Progress to Done",
			issue: domain.JiraIssue{
				Key: "TEST-2",
				Changelog: domain.JiraChangelog{
					Histories: []domain.JiraChangeHistory{
						{
							Created: "2024-03-20T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "To Do",
									ToString:   "In Progress",
								},
							},
						},
						{
							Created: "2024-03-21T15:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "In Progress",
									ToString:   "Done",
								},
							},
						},
					},
				},
			},
			expectedStart: time.Date(2024, 3, 20, 10, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2024, 3, 21, 15, 0, 0, 0, time.UTC),
		},
		{
			name: "Multiple In Progress periods",
			issue: domain.JiraIssue{
				Key: "TEST-3",
				Changelog: domain.JiraChangelog{
					Histories: []domain.JiraChangeHistory{
						{
							Created: "2024-03-20T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "To Do",
									ToString:   "In Progress",
								},
							},
						},
						{
							Created: "2024-03-20T12:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "In Progress",
									ToString:   "Blocked",
								},
							},
						},
						{
							Created: "2024-03-21T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "Blocked",
									ToString:   "In Progress",
								},
							},
						},
						{
							Created: "2024-03-21T15:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "In Progress",
									ToString:   "Done",
								},
							},
						},
					},
				},
			},
			expectedStart: time.Date(2024, 3, 20, 10, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2024, 3, 21, 15, 0, 0, 0, time.UTC),
		},
		{
			name: "Still In Progress",
			issue: domain.JiraIssue{
				Key: "TEST-4",
				Changelog: domain.JiraChangelog{
					Histories: []domain.JiraChangeHistory{
						{
							Created: "2024-03-20T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "To Do",
									ToString:   "In Progress",
								},
							},
						},
					},
				},
			},
			expectedStart: time.Date(2024, 3, 20, 10, 0, 0, 0, time.UTC),
			expectedEnd:   time.Time{},
		},
		{
			name: "No status changes",
			issue: domain.JiraIssue{
				Key: "TEST-5",
				Changelog: domain.JiraChangelog{
					Histories: []domain.JiraChangeHistory{
						{
							Created: "2024-03-20T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "description",
									FromString: "",
									ToString:   "Updated description",
								},
							},
						},
					},
				},
			},
			expectedStart: time.Time{},
			expectedEnd:   time.Time{},
		},
		{
			name: "Direct to Won't Do",
			issue: domain.JiraIssue{
				Key: "TEST-6",
				Changelog: domain.JiraChangelog{
					Histories: []domain.JiraChangeHistory{
						{
							Created: "2024-03-20T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "To Do",
									ToString:   "Won't Do",
								},
							},
						},
					},
				},
			},
			expectedStart: time.Date(2024, 3, 20, 10, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2024, 3, 20, 10, 0, 0, 0, time.UTC),
		},
		{
			name: "Through In Progress to Won't Do",
			issue: domain.JiraIssue{
				Key: "TEST-7",
				Changelog: domain.JiraChangelog{
					Histories: []domain.JiraChangeHistory{
						{
							Created: "2024-03-20T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "To Do",
									ToString:   "In Progress",
								},
							},
						},
						{
							Created: "2024-03-21T15:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "In Progress",
									ToString:   "Won't Do",
								},
							},
						},
					},
				},
			},
			expectedStart: time.Date(2024, 3, 20, 10, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2024, 3, 21, 15, 0, 0, 0, time.UTC),
		},
		{
			name: "Multiple Non-Progress Status Changes",
			issue: domain.JiraIssue{
				Key: "TEST-8",
				Changelog: domain.JiraChangelog{
					Histories: []domain.JiraChangeHistory{
						{
							Created: "2024-03-20T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "To Do",
									ToString:   "Blocked",
								},
							},
						},
						{
							Created: "2024-03-21T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "Blocked",
									ToString:   "To Do",
								},
							},
						},
						{
							Created: "2024-03-22T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "To Do",
									ToString:   "Under Review",
								},
							},
						},
					},
				},
			},
			expectedStart: time.Time{},
			expectedEnd:   time.Time{},
		},
		{
			name: "Multiple Status Changes Before Won't Do",
			issue: domain.JiraIssue{
				Key: "TEST-9",
				Changelog: domain.JiraChangelog{
					Histories: []domain.JiraChangeHistory{
						{
							Created: "2024-03-20T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "To Do",
									ToString:   "Blocked",
								},
							},
						},
						{
							Created: "2024-03-21T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "Blocked",
									ToString:   "Under Review",
								},
							},
						},
						{
							Created: "2024-03-22T10:00:00.000+0000",
							Items: []domain.JiraChangeItem{
								{
									Field:      "status",
									FromString: "Under Review",
									ToString:   "Won't Do",
								},
							},
						},
					},
				},
			},
			expectedStart: time.Date(2024, 3, 22, 10, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2024, 3, 22, 10, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime, endTime := processor.getIssueTimeRange(tt.issue)
			assert.Equal(t, tt.expectedStart, startTime, "Start time mismatch for %s", tt.name)
			assert.Equal(t, tt.expectedEnd, endTime, "End time mismatch for %s", tt.name)
		})
	}
}

func TestCalculatePercentageLoad(t *testing.T) {
	processor := &SprintTimeAllocationUseCase{
		sprint: "Test Sprint",
	}

	team := domain.Team{
		Team: []string{"test.user"},
	}

	issues := []domain.JiraIssue{
		{
			Key: "TEST-1",
			Fields: domain.JiraFields{
				Assignee: domain.JiraAssignee{
					DisplayName: "test.user",
				},
				Summary: "Test Issue 1",
				IssueType: domain.IssueType{
					Name: "Task",
				},
			},
			Changelog: domain.JiraChangelog{
				Histories: []domain.JiraChangeHistory{
					{
						Created: "2024-03-20T10:00:00.000+0000",
						Items: []domain.JiraChangeItem{
							{
								Field:      "status",
								FromString: "To Do",
								ToString:   "Done",
							},
						},
					},
				},
			},
		},
	}

	totalHoursByPerson := map[string]float64{
		"test.user": 8.0,
	}

	results := processor.calculatePercentageLoad(team, issues, nil, totalHoursByPerson)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, "2024-03-20", result["dateCompleted"])
}
