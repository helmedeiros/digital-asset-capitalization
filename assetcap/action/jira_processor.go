package action

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/assetcap"
	"github.com/helmedeiros/digital-asset-capitalization/assetcap/config"
)

// JiraProcessor handles the processing of Jira issues and time calculations
type JiraProcessor struct {
	config   *config.JiraConfig
	teams    assetcap.TeamMap
	project  string
	sprint   string
	override string
}

// NewJiraProcessor creates a new JiraProcessor instance
func NewJiraProcessor(project, sprint, override string) (*JiraProcessor, error) {
	// Load Jira configuration
	jiraConfig, err := config.NewJiraConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Jira configuration: %w", err)
	}

	// Load teams data
	data, err := ioutil.ReadFile("teams.json")
	if err != nil {
		return nil, fmt.Errorf("error reading teams.json: %w", err)
	}

	var teams assetcap.TeamMap
	if err := json.Unmarshal(data, &teams); err != nil {
		return nil, fmt.Errorf("error unmarshaling teams data: %w", err)
	}

	return &JiraProcessor{
		config:   jiraConfig,
		teams:    teams,
		project:  project,
		sprint:   sprint,
		override: override,
	}, nil
}

// Process calculates time allocation and returns CSV data
func (p *JiraProcessor) Process() (string, error) {
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

	return p.generateCSV(*team, results)
}

func (p *JiraProcessor) fetchIssues() ([]assetcap.JiraIssue, error) {
	query := fmt.Sprintf("project = %s AND sprint in openSprints()", p.project)
	encodedQuery := url.QueryEscape(query)
	jiraURL := fmt.Sprintf("%s/rest/api/3/search?jql=%s", p.config.GetBaseURL(), encodedQuery)

	return assetcap.GetJiraIssues(jiraURL, p.config.GetAuthHeader())
}

func (p *JiraProcessor) parseManualAdjustments() (map[string]float64, error) {
	if p.override == "" {
		return nil, nil
	}

	var adjustments map[string]float64
	if err := json.Unmarshal([]byte(p.override), &adjustments); err != nil {
		return nil, fmt.Errorf("error parsing manual adjustments JSON: %w", err)
	}
	return adjustments, nil
}

func (p *JiraProcessor) calculateTotalHours(team assetcap.Team, issues []assetcap.JiraIssue, manualAdjustments map[string]float64) map[string]float64 {
	totalHoursByPerson := make(map[string]float64)
	for _, person := range team.Members {
		totalHoursByPerson[person] = 0
	}

	for _, issue := range issues {
		assignee := issue.Fields.Assignee.DisplayName
		if !team.IsTeamMember(assignee) {
			continue
		}

		startTime, endTime := p.getIssueTimeRange(issue)
		if startTime.IsZero() {
			continue
		}

		workingHours := assetcap.CalculateWorkingHours(issue.Key, manualAdjustments, startTime, endTime)
		totalHoursByPerson[assignee] += workingHours
	}

	return totalHoursByPerson
}

func (p *JiraProcessor) getIssueTimeRange(issue assetcap.JiraIssue) (time.Time, time.Time) {
	var startTime, endTime time.Time
	var inProgress, done bool

	for i := len(issue.Changelog.Histories) - 1; i >= 0; i-- {
		history := issue.Changelog.Histories[i]
		for _, item := range history.Items {
			if !item.IsStatusChange() {
				continue
			}

			if (item.ToString == assetcap.StatusDone || item.ToString == assetcap.StatusWontDo) && !done {
				endTime, _ = time.Parse("2006-01-02T15:04:05.000-0700", history.Created)
				done = true
			} else if item.ToString == assetcap.StatusInProgress && !inProgress {
				startTime, _ = time.Parse("2006-01-02T15:04:05.000-0700", history.Created)
				inProgress = true
			}
		}
	}

	if inProgress && !done {
		endTime = time.Now()
	}

	return startTime, endTime
}

func (p *JiraProcessor) calculatePercentageLoad(team assetcap.Team, issues []assetcap.JiraIssue, manualAdjustments map[string]float64, totalHoursByPerson map[string]float64) []map[string]interface{} {
	var results []map[string]interface{}

	for _, issue := range issues {
		assignee := issue.Fields.Assignee.DisplayName
		if !team.IsTeamMember(assignee) {
			continue
		}

		startTime, endTime := p.getIssueTimeRange(issue)
		if startTime.IsZero() {
			continue
		}

		workingHours := assetcap.CalculateWorkingHours(issue.Key, manualAdjustments, startTime, endTime)
		totalHours := totalHoursByPerson[assignee]
		percentageLoad := 0.0
		if totalHours != 0 {
			percentageLoad = (workingHours / totalHours) * 100
		}

		result := make(map[string]interface{})
		result["sprint"] = p.sprint
		result["issueKey"] = issue.Key
		result["title"] = issue.Fields.Summary

		for _, person := range team.Members {
			result[person] = ""
		}

		result[assignee] = fmt.Sprintf("%.2f%%", percentageLoad)
		results = append(results, result)
	}

	return results
}

func (p *JiraProcessor) generateCSV(team assetcap.Team, results []map[string]interface{}) (string, error) {
	headers := []string{"sprint", "issueKey", "title"}
	headers = append(headers, team.Members...)

	csvData, err := assetcap.StructArrayToCSVOrdered(results, headers)
	if err != nil {
		return "", fmt.Errorf("failed to generate CSV: %w", err)
	}

	return csvData, nil
}

// JiraDoer is the main entry point for processing Jira issues
func JiraDoer(project string, sprint string, override string) (string, error) {
	processor, err := NewJiraProcessor(project, sprint, override)
	if err != nil {
		return "", err
	}
	return processor.Process()
}
