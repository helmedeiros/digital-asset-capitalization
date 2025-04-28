package ports

// SprintService defines the interface for sprint management operations
type SprintService interface {
	// ProcessJiraIssues processes Jira issues and returns CSV data
	ProcessJiraIssues(project, sprint, override string) (string, error)
}
