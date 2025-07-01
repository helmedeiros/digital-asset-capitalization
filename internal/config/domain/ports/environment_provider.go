package ports

// EnvironmentProvider defines the contract for accessing environment variables
type EnvironmentProvider interface {
	// GetJiraBaseURL returns the Jira base URL from environment
	GetJiraBaseURL() string

	// GetJiraEmail returns the Jira email from environment
	GetJiraEmail() string

	// GetJiraToken returns the Jira token from environment
	GetJiraToken() string

	// SetJiraBaseURL sets the Jira base URL environment variable
	SetJiraBaseURL(value string) error

	// SetJiraEmail sets the Jira email environment variable
	SetJiraEmail(value string) error

	// SetJiraToken sets the Jira token environment variable
	SetJiraToken(value string) error

	// IsConfigured checks if all required environment variables are set
	IsConfigured() bool

	// GetMissingVars returns a list of missing environment variables
	GetMissingVars() []string
}
