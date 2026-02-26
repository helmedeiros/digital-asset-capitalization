package application

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

// SprintServiceImpl handles sprint-related operations
type SprintServiceImpl struct {
	jiraPort ports.JiraPort
}

// NewSprintService creates a new sprint service
func NewSprintService(jiraPort ports.JiraPort) SprintService {
	return &SprintServiceImpl{
		jiraPort: jiraPort,
	}
}

// ProcessSprint processes a sprint and its issues
func (s *SprintServiceImpl) ProcessSprint(project string, sprint *domain.Sprint) error {
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
func (s *SprintServiceImpl) ProcessTeamIssues(team *domain.Team) error {
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
func (s *SprintServiceImpl) ProcessJiraIssues(project, sprint, override string) (string, error) {
	processor, err := usecase.NewSprintTimeAllocationUseCase(project, sprint, override)
	if err != nil {
		return "", fmt.Errorf("failed to create Jira processor: %w", err)
	}

	return processor.Process()
}

// ProcessJiraIssuesWithStrategy processes Jira issues with configurable time calculation strategy
func (s *SprintServiceImpl) ProcessJiraIssuesWithStrategy(project, sprint, override string, useSprintBoundedCalculation bool) (string, error) {
	processor, err := usecase.NewSprintTimeAllocationUseCaseWithStrategy(project, sprint, override, useSprintBoundedCalculation)
	if err != nil {
		return "", fmt.Errorf("failed to create Jira processor with strategy: %w", err)
	}

	return processor.Process()
}

// ProcessJiraIssuesWithOptions processes Jira issues with functional options
func (s *SprintServiceImpl) ProcessJiraIssuesWithOptions(project, sprint, override string, useSprintBoundedCalculation bool, opts ...usecase.SprintAllocationOption) (string, error) {
	processor, err := usecase.NewSprintTimeAllocationUseCaseWithOptions(project, sprint, override, useSprintBoundedCalculation, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to create Jira processor with options: %w", err)
	}

	return processor.Process()
}

// ListSprints lists sprints for a project and time period
func (s *SprintServiceImpl) ListSprints(project, period string) (*usecase.ListSprintsResult, error) {
	listSprintsUseCase := usecase.NewListSprintsUseCase(s.jiraPort)
	return listSprintsUseCase.Execute(project, period)
}

// PushAllocationToJira calculates allocation and pushes results to JIRA custom fields
func (s *SprintServiceImpl) PushAllocationToJira(project, sprint, override string, useSprintBounded, dryRun bool, opts ...usecase.SprintAllocationOption) (string, *usecase.PushResult, error) {
	// Ensure --with-hours is always enabled for push (we need engineering hours)
	opts = append(opts, usecase.WithHours(true))

	processor, err := usecase.NewSprintTimeAllocationUseCaseWithOptions(project, sprint, override, useSprintBounded, opts...)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create Jira processor: %w", err)
	}

	csvData, records, err := processor.ProcessWithRecords()
	if err != nil {
		return "", nil, fmt.Errorf("failed to process allocation: %w", err)
	}

	pushUC := usecase.NewPushAllocationUseCase(processor.GetJiraPort(), dryRun)
	pushResult, err := pushUC.Execute(records)
	if err != nil {
		return csvData, nil, fmt.Errorf("failed to push allocation: %w", err)
	}

	return csvData, pushResult, nil
}
