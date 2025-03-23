package application

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/ports"
)

// SprintService handles sprint-related operations
type SprintService struct {
	jiraPort ports.JiraPort
}

// NewSprintService creates a new sprint service
func NewSprintService(jiraPort ports.JiraPort) *SprintService {
	return &SprintService{
		jiraPort: jiraPort,
	}
}

// ProcessSprint processes a sprint and its issues
func (s *SprintService) ProcessSprint(project string, sprint *domain.Sprint) error {
	// Set the project field
	sprint.Project = project

	// Get all issues for the sprint
	issues, err := s.jiraPort.GetSprintIssues(sprint)
	if err != nil {
		return err
	}

	// Process each issue
	for _, issue := range issues {
		// Log issue details for now
		fmt.Printf("Processing issue: %s - %s (Status: %s)\n",
			issue.Key, issue.Summary, issue.Status)
	}

	return nil
}

// ProcessTeamIssues processes issues for a team
func (s *SprintService) ProcessTeamIssues(team *domain.Team) error {
	// Get all issues for the team
	issues, err := s.jiraPort.GetTeamIssues(team)
	if err != nil {
		return err
	}

	// Process each issue
	for _, issue := range issues {
		// Log issue details for now
		fmt.Printf("Processing team issue: %s - %s (Status: %s)\n",
			issue.Key, issue.Summary, issue.Status)
	}

	return nil
}

// ProcessJiraIssues processes Jira issues and returns CSV data
func (s *SprintService) ProcessJiraIssues(project, sprint, override string) (string, error) {
	processor, err := usecase.NewJiraProcessor(project, sprint, override)
	if err != nil {
		return "", fmt.Errorf("failed to create Jira processor: %w", err)
	}

	return processor.Process()
}
