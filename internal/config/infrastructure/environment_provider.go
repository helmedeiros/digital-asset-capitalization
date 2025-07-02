package infrastructure

import (
	"os"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain/ports"
)

// EnvironmentProvider implements the ports.EnvironmentProvider interface
type EnvironmentProvider struct{}

// NewEnvironmentProvider creates a new EnvironmentProvider instance
func NewEnvironmentProvider() ports.EnvironmentProvider {
	return &EnvironmentProvider{}
}

// GetJiraBaseURL returns the Jira base URL from environment
func (e *EnvironmentProvider) GetJiraBaseURL() string {
	return os.Getenv("JIRA_BASE_URL")
}

// GetJiraEmail returns the Jira email from environment
func (e *EnvironmentProvider) GetJiraEmail() string {
	return os.Getenv("JIRA_EMAIL")
}

// GetJiraToken returns the Jira token from environment
func (e *EnvironmentProvider) GetJiraToken() string {
	return os.Getenv("JIRA_TOKEN")
}

// SetJiraBaseURL sets the Jira base URL environment variable
func (e *EnvironmentProvider) SetJiraBaseURL(value string) error {
	return os.Setenv("JIRA_BASE_URL", value)
}

// SetJiraEmail sets the Jira email environment variable
func (e *EnvironmentProvider) SetJiraEmail(value string) error {
	return os.Setenv("JIRA_EMAIL", value)
}

// SetJiraToken sets the Jira token environment variable
func (e *EnvironmentProvider) SetJiraToken(value string) error {
	return os.Setenv("JIRA_TOKEN", value)
}

// IsConfigured checks if all required environment variables are set
func (e *EnvironmentProvider) IsConfigured() bool {
	return e.GetJiraBaseURL() != "" &&
		e.GetJiraEmail() != "" &&
		e.GetJiraToken() != ""
}

// GetMissingVars returns a list of missing environment variables
func (e *EnvironmentProvider) GetMissingVars() []string {
	var missing []string

	if e.GetJiraBaseURL() == "" {
		missing = append(missing, "JIRA_BASE_URL")
	}
	if e.GetJiraEmail() == "" {
		missing = append(missing, "JIRA_EMAIL")
	}
	if e.GetJiraToken() == "" {
		missing = append(missing, "JIRA_TOKEN")
	}

	return missing
}
