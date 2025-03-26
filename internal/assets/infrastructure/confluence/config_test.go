package confluence

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	// Save original env vars
	origEmail := os.Getenv("JIRA_EMAIL")
	origToken := os.Getenv("JIRA_TOKEN")
	origBaseURL := os.Getenv("JIRA_BASE_URL")

	// Set test env vars
	os.Setenv("JIRA_EMAIL", "test@example.com")
	os.Setenv("JIRA_TOKEN", "test-token")
	os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")

	// Restore env vars after test
	defer func() {
		os.Setenv("JIRA_EMAIL", origEmail)
		os.Setenv("JIRA_TOKEN", origToken)
		os.Setenv("JIRA_BASE_URL", origBaseURL)
	}()

	config := DefaultConfig()

	if config.Username != "test@example.com" {
		t.Errorf("Username = %v, want %v", config.Username, "test@example.com")
	}
	if config.Token != "test-token" {
		t.Errorf("Token = %v, want %v", config.Token, "test-token")
	}
	if config.BaseURL != "https://test.atlassian.net" {
		t.Errorf("BaseURL = %v, want %v", config.BaseURL, "https://test.atlassian.net")
	}
	if config.MaxResults != 25 {
		t.Errorf("MaxResults = %v, want %v", config.MaxResults, 25)
	}
}

func TestConfigWithEmptyEnvVars(t *testing.T) {
	// Save original env vars
	origEmail := os.Getenv("JIRA_EMAIL")
	origToken := os.Getenv("JIRA_TOKEN")
	origBaseURL := os.Getenv("JIRA_BASE_URL")

	// Clear env vars
	os.Unsetenv("JIRA_EMAIL")
	os.Unsetenv("JIRA_TOKEN")
	os.Unsetenv("JIRA_BASE_URL")

	// Restore env vars after test
	defer func() {
		os.Setenv("JIRA_EMAIL", origEmail)
		os.Setenv("JIRA_TOKEN", origToken)
		os.Setenv("JIRA_BASE_URL", origBaseURL)
	}()

	config := DefaultConfig()

	if config.Username != "" {
		t.Errorf("Username = %v, want empty string", config.Username)
	}
	if config.Token != "" {
		t.Errorf("Token = %v, want empty string", config.Token)
	}
	if config.BaseURL != "" {
		t.Errorf("BaseURL = %v, want empty string", config.BaseURL)
	}
	if config.MaxResults != 25 {
		t.Errorf("MaxResults = %v, want %v", config.MaxResults, 25)
	}
}
