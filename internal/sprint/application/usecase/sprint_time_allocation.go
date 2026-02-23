package usecase

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	configService "github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	configInfrastructure "github.com/helmedeiros/digital-asset-capitalization/internal/config/infrastructure"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application/service"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/config"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/infrastructure"
)

// SprintTimeAllocationUseCase handles the processing of Jira issues and time calculations
type SprintTimeAllocationUseCase struct {
	config         *config.JiraConfig
	teams          domain.TeamMap
	project        string
	sprint         string
	override       string
	jiraPort       ports.JiraPort
	statusPort     ports.StatusPort
	timeCalculator *domain.WorkTimeCalculator
	sprintBoundary *domain.SprintBoundary
}

// NewSprintTimeAllocationUseCase creates a new JiraProcessor instance
func NewSprintTimeAllocationUseCase(project, sprint, override string) (*SprintTimeAllocationUseCase, error) {
	return NewSprintTimeAllocationUseCaseWithStrategy(project, sprint, override, false)
}

// NewSprintTimeAllocationUseCaseWithStrategy creates a new use case with configurable time calculation strategy
func NewSprintTimeAllocationUseCaseWithStrategy(project, sprint, override string, useSprintBoundedCalculation bool) (*SprintTimeAllocationUseCase, error) {
	// Create Jira adapter with shared configuration
	jiraAdapter, err := infrastructure.NewJiraAdapter()
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira adapter: %w", err)
	}

	// Load legacy configuration for backward compatibility
	jiraConfig, err := config.NewJiraConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Jira configuration: %w", err)
	}

	// Initialize configuration service to get teams data
	configRepo := configInfrastructure.NewFileRepository(".assetcap")
	configSvc := configService.NewConfigService(configRepo)

	// Load teams data using shared configuration service
	teams, err := configSvc.GetTeamMapForSprint()
	if err != nil {
		return nil, fmt.Errorf("failed to load team configuration: %w", err)
	}

	// Create StatusService for team-specific status mapping
	statusService, err := service.NewStatusService()
	if err != nil {
		return nil, fmt.Errorf("failed to create status service: %w", err)
	}

	// Set up time calculation strategy
	var timeCalculator *domain.WorkTimeCalculator
	var sprintBoundary *domain.SprintBoundary

	if useSprintBoundedCalculation {
		// Get sprint details to create sprint boundary
		sprintDetails, err := jiraAdapter.GetSprintByName(project, sprint)
		if err != nil {
			return nil, fmt.Errorf("failed to get sprint details for %s: %w", sprint, err)
		}

		// Parse sprint dates
		startDate, err := time.Parse(time.RFC3339, sprintDetails.StartDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sprint start date %s: %w", sprintDetails.StartDate, err)
		}

		endDate, err := time.Parse(time.RFC3339, sprintDetails.EndDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sprint end date %s: %w", sprintDetails.EndDate, err)
		}

		// Create sprint boundary
		boundary, err := domain.NewSprintBoundary(startDate, endDate)
		if err != nil {
			return nil, fmt.Errorf("failed to create sprint boundary: %w", err)
		}
		sprintBoundary = &boundary

		// Use sprint-bounded time calculator with status checker
		// For sprint-bounded calculation, we need team info which we'll get from the first issue processed
		// For now, create without status checker and update it dynamically per issue
		strategy := domain.NewSprintBoundedTimeCalculator()
		timeCalculator = domain.NewWorkTimeCalculator(strategy)
	} else {
		// Use legacy time calculator for backward compatibility
		strategy := domain.NewLegacyTimeCalculator()
		timeCalculator = domain.NewWorkTimeCalculator(strategy)
	}

	return &SprintTimeAllocationUseCase{
		config:         jiraConfig,
		teams:          teams,
		project:        project,
		sprint:         sprint,
		override:       override,
		jiraPort:       jiraAdapter,
		statusPort:     statusService,
		timeCalculator: timeCalculator,
		sprintBoundary: sprintBoundary,
	}, nil
}

// Process calculates time allocation and returns CSV data
func (p *SprintTimeAllocationUseCase) Process() (string, error) {
	team, exists := p.teams.GetTeam(p.project)
	if !exists {
		return "", fmt.Errorf("project %s not found in teams.json", p.project)
	}

	issues, err := p.fetchIssues()
	if err != nil {
		return "", fmt.Errorf("failed to fetch issues: %w", err)
	}

	manualAdjustments, err := p.parseManualAdjustments()
	if err != nil {
		return "", err
	}

	totalHoursByPerson := p.calculateTotalHours(*team, issues, manualAdjustments)

	results := p.calculatePercentageLoad(*team, issues, manualAdjustments, totalHoursByPerson)

	csvData, err := p.generateCSV(*team, results)
	if err != nil {
		return "", fmt.Errorf("failed to generate CSV: %w", err)
	}

	return csvData, nil
}

func (p *SprintTimeAllocationUseCase) fetchIssues() ([]domain.JiraIssue, error) {
	issues, err := p.jiraPort.GetIssuesForSprint(p.project, p.sprint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sprint issues: %w", err)
	}

	var domainIssues = make([]domain.JiraIssue, 0, len(issues))
	for _, issue := range issues {
		domainIssue := domain.JiraIssue{
			Key: issue.Key,
			Fields: domain.JiraFields{
				Summary: issue.Summary,
				Assignee: domain.JiraAssignee{
					DisplayName: issue.Assignee,
				},
				Status: domain.JiraStatus{
					Name: issue.Status,
				},
				StoryPoints: issue.StoryPoints,
				IssueType: domain.IssueType{
					Name: issue.IssueType,
				},
				Labels: issue.Labels,
			},
			Changelog: domain.JiraChangelog{
				Histories: make([]domain.JiraChangeHistory, len(issue.Changelog.Histories)),
			},
		}

		// Convert changelog histories
		for i, history := range issue.Changelog.Histories {
			domainHistory := domain.JiraChangeHistory{
				Created: history.Created,
				Items:   make([]domain.JiraChangeItem, len(history.Items)),
			}

			// Convert changelog items
			for j, item := range history.Items {
				domainHistory.Items[j] = domain.JiraChangeItem{
					Field:      item.Field,
					FromString: item.FromString,
					ToString:   item.ToString,
				}
			}

			domainIssue.Changelog.Histories[i] = domainHistory
		}

		domainIssues = append(domainIssues, domainIssue)
	}

	return domainIssues, nil
}

func (p *SprintTimeAllocationUseCase) parseManualAdjustments() (map[string]float64, error) {
	if p.override == "" {
		return nil, nil
	}

	var adjustments map[string]float64
	if err := json.Unmarshal([]byte(p.override), &adjustments); err != nil {
		return nil, fmt.Errorf("error parsing manual adjustments JSON: %w", err)
	}
	return adjustments, nil
}

func (p *SprintTimeAllocationUseCase) calculateTotalHours(team domain.Team, issues []domain.JiraIssue, manualAdjustments map[string]float64) map[string]float64 {
	totalHoursByPerson := make(map[string]float64)
	for _, person := range team.Team {
		totalHoursByPerson[person] = 0
	}

	// Process issues with sub-task aggregation
	processedStories := make(map[string]bool)

	for _, issue := range issues {
		// Skip if already processed as part of parent-child aggregation
		if processedStories[issue.Key] {
			continue
		}

		assignee := issue.Fields.Assignee.DisplayName
		if !team.IsTeamMember(assignee) {
			continue
		}

		// If this is a sub-task, skip it here - it will be processed with its parent
		if issue.IsSubTask() {
			continue
		}

		// Check if this story has sub-tasks
		subTasks := p.findSubTasksForParent(issue.Key, issues)

		var totalWorkingHours float64
		if len(subTasks) > 0 {
			// Aggregate sub-task hours and distribute to actual assignees
			subTaskHours := p.aggregateSubTaskHours(subTasks, manualAdjustments, team)
			for assigneeName, hours := range subTaskHours {
				totalHoursByPerson[assigneeName] += hours
			}
			// Mark all sub-tasks as processed
			for _, subTask := range subTasks {
				processedStories[subTask.Key] = true
			}
		} else {
			// No sub-tasks, process parent story normally
			totalWorkingHours = p.calculateWorkingHours(issue.Key, manualAdjustments, issue)

			// Apply minimum hours logic for same-day completion (preserve original behavior)
			teamKey := p.getTeamKeyForAssignee(assignee)
			boardID := p.statusPort.GetBoardIDForTeam(teamKey)
			isCompleted := p.statusPort.IsDone(issue.Fields.Status.Name, teamKey, boardID) ||
				p.statusPort.IsWontDo(issue.Fields.Status.Name, teamKey, boardID)

			if totalWorkingHours < 1 && isCompleted {
				startTime, endTime := p.getIssueTimeRange(issue)
				if !startTime.IsZero() && !endTime.IsZero() &&
					startTime.Year() == endTime.Year() &&
					startTime.Month() == endTime.Month() &&
					startTime.Day() == endTime.Day() {
					totalWorkingHours = 1 // Minimum 1 hour for same-day completion
				}
			}

			totalHoursByPerson[assignee] += totalWorkingHours
		}

		// Mark parent as processed
		processedStories[issue.Key] = true
	}

	return totalHoursByPerson
}

// findSubTasksForParent finds all sub-tasks that belong to a given parent story
func (p *SprintTimeAllocationUseCase) findSubTasksForParent(parentKey string, issues []domain.JiraIssue) []domain.JiraIssue {
	var subTasks []domain.JiraIssue
	for _, issue := range issues {
		if issue.IsSubTask() && issue.GetParentKey() == parentKey {
			subTasks = append(subTasks, issue)
		}
	}
	return subTasks
}

// aggregateSubTaskHours calculates total working hours from sub-tasks, grouped by assignee
func (p *SprintTimeAllocationUseCase) aggregateSubTaskHours(subTasks []domain.JiraIssue, manualAdjustments map[string]float64, team domain.Team) map[string]float64 {
	hoursByAssignee := make(map[string]float64)

	for _, subTask := range subTasks {
		assignee := subTask.Fields.Assignee.DisplayName

		// Only count hours for team members
		if !team.IsTeamMember(assignee) {
			continue
		}

		// Calculate working hours for this sub-task
		workingHours := p.calculateWorkingHours(subTask.Key, manualAdjustments, subTask)
		hoursByAssignee[assignee] += workingHours
	}

	return hoursByAssignee
}

// getTeamKeyForAssignee determines which team an assignee belongs to
func (p *SprintTimeAllocationUseCase) getTeamKeyForAssignee(assignee string) string {
	for teamKey, team := range p.teams {
		if team.IsTeamMember(assignee) {
			return teamKey
		}
	}

	// Return project as fallback for unmapped assignees
	// This allows fallback to default status mapping
	return p.project
}

func (p *SprintTimeAllocationUseCase) getIssueTimeRange(issue domain.JiraIssue) (time.Time, time.Time) {
	var startTime, endTime time.Time
	var inProgress bool
	var firstInProgressTime time.Time

	// Get team information for status mapping
	assignee := issue.Fields.Assignee.DisplayName
	teamKey := p.getTeamKeyForAssignee(assignee)
	boardID := p.statusPort.GetBoardIDForTeam(teamKey)

	// Process histories in chronological order
	for i := 0; i < len(issue.Changelog.Histories); i++ {
		history := issue.Changelog.Histories[i]

		for _, item := range history.Items {
			if !item.IsStatusChange() {
				continue
			}

			// Parse the history timestamp and ensure UTC timezone
			historyTime, err := time.Parse("2006-01-02T15:04:05.000-0700", history.Created)
			if err != nil {
				// If parsing fails, try RFC3339 format
				historyTime, err = time.Parse(time.RFC3339, history.Created)
				if err != nil {
					continue
				}
			}
			historyTime = historyTime.UTC()

			// Look for transition into in-progress state using team-specific status mapping
			isInProgressStatus := p.statusPort.IsInProgress(item.ToString, teamKey, boardID)
			if isInProgressStatus {
				if firstInProgressTime.IsZero() {
					firstInProgressTime = historyTime
				}
				startTime = firstInProgressTime // Always use the first in-progress time
				inProgress = true
			}

			// Look for transition to completion state using team-specific status mapping
			isDoneStatus := p.statusPort.IsDone(item.ToString, teamKey, boardID)
			isWontDoStatus := p.statusPort.IsWontDo(item.ToString, teamKey, boardID)
			if isDoneStatus || isWontDoStatus {
				endTime = historyTime
				// If we weren't in progress, use the completion time as start time
				if !inProgress && startTime.IsZero() {
					startTime = historyTime
				}
			}

			// If moving out of in-progress state to a non-completion state, consider this a pause
			if inProgress && p.statusPort.IsInProgress(item.FromString, teamKey, boardID) {
				// Check if this is NOT a completion transition
				if !(p.statusPort.IsDone(item.ToString, teamKey, boardID) || p.statusPort.IsWontDo(item.ToString, teamKey, boardID)) {
					// This was an interruption in progress
					inProgress = false
				}
			}
		}
	}

	// Ensure endTime is not before startTime
	if !endTime.IsZero() && !startTime.IsZero() && endTime.Before(startTime) {
		// If endTime is before startTime, swap them
		startTime, endTime = endTime, startTime
	}

	return startTime, endTime
}

func (p *SprintTimeAllocationUseCase) calculatePercentageLoad(team domain.Team, issues []domain.JiraIssue, manualAdjustments map[string]float64, totalHoursByPerson map[string]float64) []map[string]interface{} {
	var results = make([]map[string]interface{}, 0, len(issues))
	processedStories := make(map[string]bool)

	for _, issue := range issues {
		// Skip if already processed or if this is a sub-task
		if processedStories[issue.Key] || issue.IsSubTask() {
			continue
		}

		assignee := issue.Fields.Assignee.DisplayName
		if !team.IsTeamMember(assignee) {
			continue
		}

		// Check if this story has sub-tasks
		subTasks := p.findSubTasksForParent(issue.Key, issues)

		var storyHoursByAssignee map[string]float64
		var storyTotalHours float64
		var storyStartTime, storyEndTime time.Time

		if len(subTasks) > 0 {
			// Process story with sub-tasks
			storyHoursByAssignee = p.aggregateSubTaskHours(subTasks, manualAdjustments, team)
			storyStartTime, storyEndTime = p.getSubTaskTimeRange(subTasks)

			// Calculate total hours for this story
			for _, hours := range storyHoursByAssignee {
				storyTotalHours += hours
			}

			// Mark all sub-tasks as processed
			for _, subTask := range subTasks {
				processedStories[subTask.Key] = true
			}
		} else {
			// Process story without sub-tasks (original logic)
			storyStartTime, storyEndTime = p.getIssueTimeRange(issue)
			workingHours := p.calculateWorkingHours(issue.Key, manualAdjustments, issue)

			// Apply minimum hours logic for same-day completion (preserve original behavior)
			teamKey := p.getTeamKeyForAssignee(assignee)
			boardID := p.statusPort.GetBoardIDForTeam(teamKey)
			isCompleted := p.statusPort.IsDone(issue.Fields.Status.Name, teamKey, boardID) ||
				p.statusPort.IsWontDo(issue.Fields.Status.Name, teamKey, boardID)

			if workingHours < 1 && !storyStartTime.IsZero() && !storyEndTime.IsZero() &&
				storyStartTime.Year() == storyEndTime.Year() &&
				storyStartTime.Month() == storyEndTime.Month() &&
				storyStartTime.Day() == storyEndTime.Day() && isCompleted {
				workingHours = 1 // Minimum 1 hour for same-day completion
			}

			storyHoursByAssignee = map[string]float64{assignee: workingHours}
			storyTotalHours = workingHours
		}

		// Handle time range fallbacks for stories without sub-tasks
		if storyStartTime.IsZero() && len(issue.Changelog.Histories) > 0 {
			storyStartTime, _ = time.Parse(time.RFC3339, issue.Changelog.Histories[0].Created)
		}
		if storyStartTime.IsZero() {
			// If we still don't have a start time, use a default duration of 8 hours
			storyEndTime = time.Now()
			storyStartTime = storyEndTime.Add(-8 * time.Hour)
		}

		// Create CSV result for this parent story
		result := make(map[string]interface{})
		result["sprint"] = p.sprint
		result["issueKey"] = issue.Key
		result["issueType"] = issue.Fields.IssueType.Name
		result["issueTitle"] = issue.Fields.Summary
		result["workType"] = issue.GetWorkType()
		result["assetName"] = issue.GetAssetName()
		result["status"] = issue.Fields.Status.Name
		result["dateStarted"] = storyStartTime.Format("2006-01-02")
		result["workingHours"] = storyTotalHours

		// Handle completion date
		teamKey := p.getTeamKeyForAssignee(assignee)
		boardID := p.statusPort.GetBoardIDForTeam(teamKey)
		isCompleted := p.statusPort.IsDone(issue.Fields.Status.Name, teamKey, boardID) ||
			p.statusPort.IsWontDo(issue.Fields.Status.Name, teamKey, boardID)

		if isCompleted && !storyEndTime.IsZero() {
			result["dateCompleted"] = storyEndTime.Format("2006-01-02")
		} else {
			result["dateCompleted"] = ""
		}

		// Initialize all team member columns to empty
		for _, person := range team.Team {
			result[person] = ""
		}

		// Distribute percentages based on actual work done by each assignee
		for assigneeName, hours := range storyHoursByAssignee {
			if totalHoursByPerson[assigneeName] > 0 {
				percentageLoad := (hours / totalHoursByPerson[assigneeName]) * 100
				result[assigneeName] = fmt.Sprintf("%.2f%%", percentageLoad)
			}
		}

		results = append(results, result)
		processedStories[issue.Key] = true
	}

	return results
}

// getSubTaskTimeRange calculates the overall time range for a set of sub-tasks
func (p *SprintTimeAllocationUseCase) getSubTaskTimeRange(subTasks []domain.JiraIssue) (time.Time, time.Time) {
	var earliestStart, latestEnd time.Time

	for _, subTask := range subTasks {
		startTime, endTime := p.getIssueTimeRange(subTask)

		if !startTime.IsZero() && (earliestStart.IsZero() || startTime.Before(earliestStart)) {
			earliestStart = startTime
		}

		if !endTime.IsZero() && (latestEnd.IsZero() || endTime.After(latestEnd)) {
			latestEnd = endTime
		}
	}

	return earliestStart, latestEnd
}

func (p *SprintTimeAllocationUseCase) generateCSV(team domain.Team, results []map[string]interface{}) (string, error) {
	// Sort engineer names alphabetically before adding to headers
	sortedTeamMembers := make([]string, len(team.Team))
	copy(sortedTeamMembers, team.Team)
	sort.Strings(sortedTeamMembers)

	headers := make([]string, 0, 9+len(sortedTeamMembers))
	headers = append(headers, "sprint", "issueKey", "issueType", "issueTitle", "workType", "assetName", "status", "dateStarted", "dateCompleted")

	headers = append(headers, sortedTeamMembers...)

	csvData, err := p.structArrayToCSVOrdered(results, headers)
	if err != nil {
		return "", fmt.Errorf("failed to generate CSV: %w", err)
	}

	return csvData, nil
}

// calculateWorkingHours calculates the working hours for an issue
func (p *SprintTimeAllocationUseCase) calculateWorkingHours(issueKey string, manualAdjustments map[string]float64, issue domain.JiraIssue) float64 {
	// Check for manual adjustments first
	if manualAdjustments != nil {
		if hours, ok := manualAdjustments[issueKey]; ok {
			return hours
		}
	}

	// Use the new time calculation strategy
	ctx := context.Background()

	if p.sprintBoundary != nil {
		// For sprint-bounded calculation, use team-specific status mapping
		assignee := issue.Fields.Assignee.DisplayName
		teamKey := p.getTeamKeyForAssignee(assignee)
		boardID := p.statusPort.GetBoardIDForTeam(teamKey)

		// Create calculator with proper status checking for this specific issue
		strategy := domain.NewSprintBoundedTimeCalculatorWithStatusChecker(p.statusPort, teamKey, boardID)
		calculator := domain.NewWorkTimeCalculator(strategy)

		hours, err := calculator.CalculateWorkingHours(ctx, issue, *p.sprintBoundary)
		if err != nil {
			// Fall back to legacy calculation on error
			return p.calculateWorkingHoursLegacy(issue)
		}
		return hours
	}
	// Use legacy calculation (sprint boundary not set)
	return p.calculateWorkingHoursLegacy(issue)
}

// calculateWorkingHoursLegacy provides the legacy time calculation as fallback
func (p *SprintTimeAllocationUseCase) calculateWorkingHoursLegacy(issue domain.JiraIssue) float64 {
	startTime, endTime := p.getIssueTimeRange(issue)
	if startTime.IsZero() {
		return 0
	}

	// Calculate hours between start and end time
	duration := endTime.Sub(startTime)
	hours := duration.Hours()

	// Ensure hours is not negative
	if hours < 0 {
		hours = 0
	}

	// Round to 2 decimal places
	roundedHours := float64(int(hours*100)) / 100

	return roundedHours
}

// structArrayToCSVOrdered converts a slice of maps to CSV format
func (p *SprintTimeAllocationUseCase) structArrayToCSVOrdered(data []map[string]interface{}, headers []string) (string, error) {
	if len(data) == 0 {
		return "", nil
	}

	buffer := &strings.Builder{}
	writer := csv.NewWriter(buffer)

	// Configure writer
	writer.UseCRLF = false
	writer.Comma = ','

	// Write headers
	if err := writer.Write(headers); err != nil {
		return "", err
	}

	// Write data
	for _, row := range data {
		record := make([]string, len(headers))
		for i, header := range headers {
			if val, ok := row[header]; ok {
				strVal := fmt.Sprintf("%v", val)
				// Prefix issueKey with = and quotes to prevent Excel formula interpretation
				// Excel interprets "COP-38" as a formula (COP minus 38)
				if header == "issueKey" && strVal != "" {
					strVal = "=\"" + strVal + "\""
				}
				record[i] = strVal
			}
		}
		if err := writer.Write(record); err != nil {
			return "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

// JiraDoer is the main entry point for processing Jira issues
func JiraDoer(project string, sprint string, override string) (string, error) {
	processor, err := NewSprintTimeAllocationUseCase(project, sprint, override)
	if err != nil {
		return "", err
	}
	return processor.Process()
}
