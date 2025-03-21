package action

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/assetcap"
)

// mockGetJiraIssues mocks the GetJiraIssues function for testing
func mockGetJiraIssues(url, authHeader string) ([]assetcap.JiraIssue, error) {
	return []assetcap.JiraIssue{
		{
			Key: "TEST-123",
			Fields: assetcap.JiraFields{
				Summary: "Test Issue 1",
				Assignee: assetcap.JiraAssignee{
					DisplayName: "Test User 1",
				},
			},
			Changelog: assetcap.JiraChangelog{
				Histories: []assetcap.JiraChangeHistory{
					{
						Created: "2024-03-01T10:00:00.000+0000",
						Items: []assetcap.JiraChangeItem{
							{
								Field:      "status",
								FromString: "To Do",
								ToString:   assetcap.StatusInProgress,
							},
						},
					},
					{
						Created: "2024-03-02T15:00:00.000+0000",
						Items: []assetcap.JiraChangeItem{
							{
								Field:      "status",
								FromString: assetcap.StatusInProgress,
								ToString:   assetcap.StatusDone,
							},
						},
					},
				},
			},
		},
		{
			Key: "TEST-124",
			Fields: assetcap.JiraFields{
				Summary: "Test Issue 2",
				Assignee: assetcap.JiraAssignee{
					DisplayName: "Test User 2",
				},
			},
			Changelog: assetcap.JiraChangelog{
				Histories: []assetcap.JiraChangeHistory{
					{
						Created: "2024-03-01T11:00:00.000+0000",
						Items: []assetcap.JiraChangeItem{
							{
								Field:      "status",
								FromString: "To Do",
								ToString:   assetcap.StatusInProgress,
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
	team := assetcap.Team{
		Members: []string{"Test User 1", "Test User 2"},
	}
	teams := assetcap.TeamMap{
		"TEST": team,
	}

	teamsData, err := json.Marshal(teams)
	if err != nil {
		t.Fatalf("Failed to marshal teams data: %v", err)
	}

	tmpTeamsFile := "teams.json"
	err = ioutil.WriteFile(tmpTeamsFile, teamsData, 0644)
	if err != nil {
		t.Fatalf("Failed to create temporary teams.json: %v", err)
	}

	// Save original GetJiraIssues function
	originalGetJiraIssues := assetcap.GetJiraIssues
	assetcap.GetJiraIssues = mockGetJiraIssues

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
		assetcap.GetJiraIssues = originalGetJiraIssues
	}
}

func TestJiraDoer_ValidProject(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	result, err := JiraDoer("TEST", "Sprint 1", "")
	if err != nil {
		t.Errorf("JiraDoer() error = %v, wantErr false", err)
	}
	if result == "" {
		t.Error("JiraDoer() returned empty result")
	}
}

func TestJiraDoer_InvalidProject(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_, err := JiraDoer("INVALID", "Sprint 1", "")
	if err == nil {
		t.Error("JiraDoer() error = nil, wantErr true")
	}
}

func TestJiraDoer_WithManualAdjustments(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	override := `{"TEST-123": 2.5}`
	result, err := JiraDoer("TEST", "Sprint 1", override)
	if err != nil {
		t.Errorf("JiraDoer() error = %v, wantErr false", err)
	}
	if result == "" {
		t.Error("JiraDoer() returned empty result")
	}
}

func TestJiraDoer_InvalidManualAdjustments(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	override := `invalid json`
	_, err := JiraDoer("TEST", "Sprint 1", override)
	if err == nil {
		t.Error("JiraDoer() error = nil, wantErr true")
	}
}

func TestJiraProcessor_CalculateTotalHours(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	processor, err := NewJiraProcessor("TEST", "Sprint 1", "")
	if err != nil {
		t.Fatalf("NewJiraProcessor() error = %v", err)
	}

	team, _ := processor.teams.GetTeam("TEST")
	issues, _ := processor.fetchIssues()
	manualAdjustments := map[string]float64{"TEST-123": 2.5}

	totalHours := processor.calculateTotalHours(*team, issues, manualAdjustments)

	if totalHours["Test User 1"] == 0 {
		t.Error("Expected non-zero hours for Test User 1")
	}
	if totalHours["Test User 2"] == 0 {
		t.Error("Expected non-zero hours for Test User 2")
	}
}

func TestJiraProcessor_GetIssueTimeRange(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	processor, err := NewJiraProcessor("TEST", "Sprint 1", "")
	if err != nil {
		t.Fatalf("NewJiraProcessor() error = %v", err)
	}

	issues, _ := processor.fetchIssues()
	startTime, endTime := processor.getIssueTimeRange(issues[0])

	expectedStart, _ := time.Parse("2006-01-02T15:04:05.000-0700", "2024-03-01T10:00:00.000+0000")
	expectedEnd, _ := time.Parse("2006-01-02T15:04:05.000-0700", "2024-03-02T15:00:00.000+0000")

	if !startTime.Equal(expectedStart) {
		t.Errorf("getIssueTimeRange() startTime = %v, want %v", startTime, expectedStart)
	}
	if !endTime.Equal(expectedEnd) {
		t.Errorf("getIssueTimeRange() endTime = %v, want %v", endTime, expectedEnd)
	}
}
