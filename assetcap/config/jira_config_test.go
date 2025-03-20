package config

import (
	"os"
	"testing"
)

// JiraConfigTestCase represents a test case for Jira configuration
type JiraConfigTestCase struct {
	Name    string
	EnvVars map[string]string
	WantErr bool
	ErrType error
}

// setupEnvVars sets up environment variables for a test and returns a cleanup function
func setupEnvVars(vars map[string]string) func() {
	// Clear all relevant environment variables
	os.Unsetenv("JIRA_BASE_URL")
	os.Unsetenv("JIRA_EMAIL")
	os.Unsetenv("JIRA_TOKEN")

	// Set test env vars
	for k, v := range vars {
		os.Setenv(k, v)
	}

	// Return cleanup function
	return func() {
		os.Unsetenv("JIRA_BASE_URL")
		os.Unsetenv("JIRA_EMAIL")
		os.Unsetenv("JIRA_TOKEN")
	}
}

func TestNewJiraConfig(t *testing.T) {
	tests := []JiraConfigTestCase{
		{
			Name: "valid configuration",
			EnvVars: map[string]string{
				"JIRA_BASE_URL": "https://test.atlassian.net",
				"JIRA_EMAIL":    "test@example.com",
				"JIRA_TOKEN":    "test-token",
			},
			WantErr: false,
		},
		{
			Name: "missing base URL",
			EnvVars: map[string]string{
				"JIRA_EMAIL": "test@example.com",
				"JIRA_TOKEN": "test-token",
			},
			WantErr: true,
			ErrType: ErrMissingBaseURL,
		},
		{
			Name: "missing email",
			EnvVars: map[string]string{
				"JIRA_BASE_URL": "https://test.atlassian.net",
				"JIRA_TOKEN":    "test-token",
			},
			WantErr: true,
			ErrType: ErrMissingEmail,
		},
		{
			Name: "missing token",
			EnvVars: map[string]string{
				"JIRA_BASE_URL": "https://test.atlassian.net",
				"JIRA_EMAIL":    "test@example.com",
			},
			WantErr: true,
			ErrType: ErrMissingToken,
		},
		{
			Name:    "all values empty",
			EnvVars: map[string]string{},
			WantErr: true,
			ErrType: ErrMissingBaseURL,
		},
		{
			Name: "invalid base URL format",
			EnvVars: map[string]string{
				"JIRA_BASE_URL": "not-a-url",
				"JIRA_EMAIL":    "test@example.com",
				"JIRA_TOKEN":    "test-token",
			},
			WantErr: true,
			ErrType: ErrInvalidBaseURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			cleanup := setupEnvVars(tt.EnvVars)
			defer cleanup()

			config, err := NewJiraConfig()

			if tt.WantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				if err != tt.ErrType {
					t.Errorf("expected error %v but got %v", tt.ErrType, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if config.BaseURL != tt.EnvVars["JIRA_BASE_URL"] {
				t.Errorf("expected BaseURL %s but got %s", tt.EnvVars["JIRA_BASE_URL"], config.BaseURL)
			}
			if config.Email != tt.EnvVars["JIRA_EMAIL"] {
				t.Errorf("expected Email %s but got %s", tt.EnvVars["JIRA_EMAIL"], config.Email)
			}
			if config.Token != tt.EnvVars["JIRA_TOKEN"] {
				t.Errorf("expected Token %s but got %s", tt.EnvVars["JIRA_TOKEN"], config.Token)
			}
		})
	}
}
