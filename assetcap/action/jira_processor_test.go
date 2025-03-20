package action

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helmedeiros/jira-time-allocator/assetcap"
)

// Mock GetJiraIssues function
func mockGetJiraIssues(url, authHeader string) ([]assetcap.JiraIssue, error) {
	// Return test data
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	return []assetcap.JiraIssue{
		{
			Key: "FN-123",
			Fields: struct {
				Summary  string `json:"summary"`
				Assignee struct {
					DisplayName string `json:"displayName"`
				} `json:"assignee"`
				StoryPoints *float64 `json:"customfield_13192"`
			}{
				Summary: "Test Issue 1",
				Assignee: struct {
					DisplayName string `json:"displayName"`
				}{
					DisplayName: "Test User 1",
				},
			},
			Changelog: struct {
				Histories []struct {
					Created string `json:"created"`
					Items   []struct {
						Field      string `json:"field"`
						FromString string `json:"fromString"`
						ToString   string `json:"toString"`
					} `json:"items"`
				} `json:"histories"`
			}{
				Histories: []struct {
					Created string `json:"created"`
					Items   []struct {
						Field      string `json:"field"`
						FromString string `json:"fromString"`
						ToString   string `json:"toString"`
					} `json:"items"`
				}{
					{
						Created: startTime.Format("2006-01-02T15:04:05.000-0700"),
						Items: []struct {
							Field      string `json:"field"`
							FromString string `json:"fromString"`
							ToString   string `json:"toString"`
						}{
							{
								Field:    "status",
								ToString: "In Progress",
							},
						},
					},
					{
						Created: endTime.Format("2006-01-02T15:04:05.000-0700"),
						Items: []struct {
							Field      string `json:"field"`
							FromString string `json:"fromString"`
							ToString   string `json:"toString"`
						}{
							{
								Field:    "status",
								ToString: "Done",
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
	testDataDir := "testdata"
	teamsJSONPath := filepath.Join(testDataDir, "teams.json")
	teamsData, err := ioutil.ReadFile(teamsJSONPath)
	if err != nil {
		t.Fatalf("Failed to read teams.json fixture: %v", err)
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
		name        string
		project     string
		sprint      string
		override    string
		wantError   bool
		errorString string
	}{
		{
			name:      "valid project and sprint",
			project:   "FN",
			sprint:    "test-sprint",
			override:  "",
			wantError: false,
		},
		{
			name:        "invalid project",
			project:     "INVALID",
			sprint:      "test-sprint",
			override:    "",
			wantError:   true,
			errorString: "project INVALID not found in teams.json",
		},
		{
			name:      "valid project with manual adjustments",
			project:   "FN",
			sprint:    "test-sprint",
			override:  `{"FN-123": 2.5}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := JiraDoer(tt.project, tt.sprint, tt.override)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorString) {
					t.Errorf("expected error containing %q, got %q", tt.errorString, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == "" {
				t.Error("expected non-empty result but got empty string")
			}
		})
	}
}
