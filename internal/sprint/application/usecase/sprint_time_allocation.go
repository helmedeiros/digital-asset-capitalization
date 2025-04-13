package usecase

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/config"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/infrastructure"
)

const (
	issueTypeSubTask = "Sub-task"
	statusDone       = "Done"
	statusWontDo     = "Won't Do"
)

// SprintTimeAllocationUseCase handles the processing of Jira issues and time calculations
type SprintTimeAllocationUseCase struct {
	config   *config.JiraConfig
	teams    domain.TeamMap
	project  string
	sprint   string
	override string
	jiraPort ports.JiraPort
}

// NewSprintTimeAllocationUseCase creates a new JiraProcessor instance
func NewSprintTimeAllocationUseCase(project, sprint, override string) (*SprintTimeAllocationUseCase, error) {
	// Load Jira configuration
	jiraConfig, err := config.NewJiraConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Jira configuration: %w", err)
	}

	// Load teams data
	var teamsData []byte
	var teamsErr error

	// Get the assetcap home directory
	assetcapHome := os.Getenv("ASSETCAP_HOME")
	if assetcapHome == "" {
		assetcapHome = "."
	}

	// Try different paths for teams.json
	paths := []string{
		filepath.Join(assetcapHome, ".assetcap", "teams.json"), // .assetcap directory
	}

	for _, path := range paths {
		teamsData, teamsErr = os.ReadFile(path)
		if teamsErr == nil {
			break
		}
	}

	if teamsErr != nil {
		// If no file is found, create a default teams.json in the .assetcap directory
		teamsDir := filepath.Join(assetcapHome, ".assetcap")
		if mkdirErr := os.MkdirAll(teamsDir, 0755); mkdirErr != nil {
			return nil, fmt.Errorf("failed to create .assetcap directory: %w", mkdirErr)
		}
		teamsData = []byte(`{
			"FN": {
				"team": ["helio.medeiros", "julio.medeiros"]
			}
		}`)
		teamsPath := filepath.Join(teamsDir, "teams.json")
		if writeErr := os.WriteFile(teamsPath, teamsData, 0644); writeErr != nil {
			return nil, fmt.Errorf("failed to write teams file: %w", writeErr)
		}
	}

	var teams domain.TeamMap
	if unmarshalErr := json.Unmarshal(teamsData, &teams); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal teams data: %w", unmarshalErr)
	}

	// Create Jira adapter
	teamsPath := filepath.Join(assetcapHome, ".assetcap", "teams.json")
	jiraAdapter, err := infrastructure.NewJiraAdapter(teamsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira adapter: %w", err)
	}

	return &SprintTimeAllocationUseCase{
		config:   jiraConfig,
		teams:    teams,
		project:  project,
		sprint:   sprint,
		override: override,
		jiraPort: jiraAdapter,
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

	for _, issue := range issues {
		assignee := issue.Fields.Assignee.DisplayName

		if !team.IsTeamMember(assignee) {
			continue
		}

		// Skip Sub-tasks
		if issue.Fields.IssueType.Name == issueTypeSubTask {
			continue
		}

		startTime, endTime := p.getIssueTimeRange(issue)
		if startTime.IsZero() {
			continue
		}

		workingHours := p.calculateWorkingHours(issue.Key, manualAdjustments, startTime, endTime)

		totalHoursByPerson[assignee] += workingHours
	}

	return totalHoursByPerson
}

func (p *SprintTimeAllocationUseCase) getIssueTimeRange(issue domain.JiraIssue) (time.Time, time.Time) {
	var startTime, endTime time.Time
	var inProgress bool
	var firstInProgressTime time.Time

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

			// Look for transition into "In Progress" state
			if item.ToString == "In Progress" {
				if firstInProgressTime.IsZero() {
					firstInProgressTime = historyTime
				}
				startTime = firstInProgressTime // Always use the first In Progress time
				inProgress = true
			}

			// Look for transition to "Done" or "Won't Do" state
			if item.ToString == statusDone || item.ToString == statusWontDo {
				endTime = historyTime
				// If we weren't in progress, use the completion time as start time
				if !inProgress && startTime.IsZero() {
					startTime = historyTime
				}
			}

			// If moving out of "In Progress" to a non-Done state, consider this a pause
			if inProgress && item.FromString == "In Progress" &&
				item.ToString != statusDone && item.ToString != statusWontDo {
				// Calculate working hours up to this point and add to total
				p.calculateWorkingHours(issue.Key, nil, startTime, historyTime)
				inProgress = false
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
	personHours := make(map[string]float64) // Track total hours per person

	// First pass: calculate raw hours and percentages
	for _, issue := range issues {
		assignee := issue.Fields.Assignee.DisplayName

		if !team.IsTeamMember(assignee) {
			continue
		}

		// Skip Sub-tasks
		if issue.Fields.IssueType.Name == issueTypeSubTask {
			continue
		}

		startTime, endTime := p.getIssueTimeRange(issue)
		if startTime.IsZero() && len(issue.Changelog.Histories) > 0 {
			// If there's no start time but we have changelog entries,
			// use the first changelog entry as the start time
			startTime, _ = time.Parse(time.RFC3339, issue.Changelog.Histories[0].Created)
		}
		if startTime.IsZero() {
			// If we still don't have a start time, use a default duration of 8 hours
			endTime = time.Now()
			startTime = endTime.Add(-8 * time.Hour)
		}

		workingHours := p.calculateWorkingHours(issue.Key, manualAdjustments, startTime, endTime)

		// For percentage calculations, ensure a minimum of 1 hour for completed issues in the same day
		if workingHours < 1 && startTime.Year() == endTime.Year() && startTime.Month() == endTime.Month() && startTime.Day() == endTime.Day() &&
			(issue.Fields.Status.Name == statusDone || issue.Fields.Status.Name == statusWontDo) {
			workingHours = 1
		}

		personHours[assignee] += workingHours
	}

	// Second pass: calculate normalized percentages
	for _, issue := range issues {
		assignee := issue.Fields.Assignee.DisplayName

		if !team.IsTeamMember(assignee) {
			continue
		}

		// Skip Sub-tasks
		if issue.Fields.IssueType.Name == issueTypeSubTask {
			continue
		}

		startTime, endTime := p.getIssueTimeRange(issue)
		if startTime.IsZero() && len(issue.Changelog.Histories) > 0 {
			startTime, _ = time.Parse(time.RFC3339, issue.Changelog.Histories[0].Created)
		}
		if startTime.IsZero() {
			endTime = time.Now()
			startTime = endTime.Add(-8 * time.Hour)
		}

		workingHours := p.calculateWorkingHours(issue.Key, manualAdjustments, startTime, endTime)

		// For percentage calculations, ensure a minimum of 1 hour for completed issues in the same day
		if workingHours < 1 && startTime.Year() == endTime.Year() && startTime.Month() == endTime.Month() && startTime.Day() == endTime.Day() &&
			(issue.Fields.Status.Name == statusDone || issue.Fields.Status.Name == statusWontDo) {
			workingHours = 1
		}

		totalHours := totalHoursByPerson[assignee]
		percentageLoad := 0.0
		if totalHours != 0 {
			// Calculate percentage based on the proportion of hours this issue represents
			// of the person's total hours across all issues
			percentageLoad = (workingHours / personHours[assignee]) * 100
		}

		result := make(map[string]interface{})
		result["sprint"] = p.sprint
		result["issueKey"] = issue.Key
		result["issueType"] = issue.Fields.IssueType.Name
		result["issueTitle"] = issue.Fields.Summary
		result["workType"] = issue.GetWorkType()
		result["assetName"] = issue.GetAssetName()
		result["status"] = issue.Fields.Status.Name
		result["dateStarted"] = startTime.Format("2006-01-02")
		result["workingHours"] = workingHours

		// Only set completion date if the issue is actually completed
		if issue.Fields.Status.Name == statusDone || issue.Fields.Status.Name == statusWontDo {
			result["dateCompleted"] = endTime.Format("2006-01-02")
		} else {
			result["dateCompleted"] = ""
		}

		for _, person := range team.Team {
			result[person] = ""
		}

		result[assignee] = fmt.Sprintf("%.2f%%", percentageLoad)
		results = append(results, result)
	}

	return results
}

func (p *SprintTimeAllocationUseCase) generateCSV(team domain.Team, results []map[string]interface{}) (string, error) {
	headers := []string{"sprint", "issueKey", "issueType", "issueTitle", "workType", "assetName", "status", "dateStarted", "dateCompleted"}
	headers = append(headers, team.Team...)

	csvData, err := p.structArrayToCSVOrdered(results, headers)
	if err != nil {
		return "", fmt.Errorf("failed to generate CSV: %w", err)
	}

	return csvData, nil
}

// calculateWorkingHours calculates the working hours for an issue
func (p *SprintTimeAllocationUseCase) calculateWorkingHours(issueKey string, manualAdjustments map[string]float64, startTime, endTime time.Time) float64 {
	// Check for manual adjustments first
	if manualAdjustments != nil {
		if hours, ok := manualAdjustments[issueKey]; ok {
			return hours
		}
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
				record[i] = fmt.Sprintf("%v", val)
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

	// Add quotes to all fields
	lines := strings.Split(buffer.String(), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		for j, field := range fields {
			fields[j] = fmt.Sprintf("%q", field)
		}
		lines[i] = strings.Join(fields, ",")
	}
	return strings.Join(lines, "\n"), nil
}

// JiraDoer is the main entry point for processing Jira issues
func JiraDoer(project string, sprint string, override string) (string, error) {
	processor, err := NewSprintTimeAllocationUseCase(project, sprint, override)
	if err != nil {
		return "", err
	}
	return processor.Process()
}
