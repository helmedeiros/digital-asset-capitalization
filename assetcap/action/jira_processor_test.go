package action

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/helmedeiros/jira-time-allocator/assetcap"
)

// mockGetJiraIssues mocks the GetJiraIssues function for testing
func mockGetJiraIssues(url, authHeader string) ([]assetcap.JiraIssue, error) {
	return []assetcap.JiraIssue{
		{
			Key: "TEST-123",
			Fields: assetcap.JiraFields{
				Summary: "Test Issue",
				Assignee: assetcap.JiraAssignee{
					DisplayName: "Test User",
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
				},
			},
		},
	}, nil
}

func TestJiraDoer(t *testing.T) {
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

	// Restore env vars after test
	defer func() {
		for k, v := range oldEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Create temporary teams.json for testing
	team := assetcap.Team{
		Members: []string{"Test User"},
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
	defer os.Remove(tmpTeamsFile)

	// Save original GetJiraIssues function and restore it after test
	originalGetJiraIssues := assetcap.GetJiraIssues
	assetcap.GetJiraIssues = mockGetJiraIssues
	defer func() {
		assetcap.GetJiraIssues = originalGetJiraIssues
	}()

	// Test cases
	tests := []struct {
		name     string
		project  string
		sprint   string
		override string
		wantErr  bool
	}{
		{
			name:     "valid project",
			project:  "TEST",
			sprint:   "Sprint 1",
			override: "",
			wantErr:  false,
		},
		{
			name:     "invalid project",
			project:  "INVALID",
			sprint:   "Sprint 1",
			override: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := JiraDoer(tt.project, tt.sprint, tt.override)
			if (err != nil) != tt.wantErr {
				t.Errorf("JiraDoer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
