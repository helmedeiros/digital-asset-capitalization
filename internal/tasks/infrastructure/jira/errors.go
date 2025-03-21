package jira

import "errors"

// Configuration errors related to Jira authentication and connection settings
var (
	// ErrMissingBaseURL indicates that the Jira base URL is not configured
	ErrMissingBaseURL = errors.New("Jira base URL is not configured. Please set the JIRA_BASE_URL environment variable")

	// ErrMissingEmail indicates that the Jira user email is not configured
	ErrMissingEmail = errors.New("Jira user email is not configured. Please set the JIRA_EMAIL environment variable")

	// ErrMissingToken indicates that the Jira API token is not configured
	ErrMissingToken = errors.New("Jira API token is not configured. Please set the JIRA_TOKEN environment variable")

	// ErrInvalidBaseURL indicates that the provided Jira base URL is not valid
	ErrInvalidBaseURL = errors.New("Invalid Jira base URL. Please provide a valid URL in the JIRA_BASE_URL environment variable")
)

// IsConfigurationError checks if the given error is a configuration-related error
func IsConfigurationError(err error) bool {
	return err == ErrMissingBaseURL ||
		err == ErrMissingEmail ||
		err == ErrMissingToken ||
		err == ErrInvalidBaseURL
}
