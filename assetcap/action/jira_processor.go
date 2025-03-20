package action

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/helmedeiros/jira-time-allocator/assetcap"
	"github.com/helmedeiros/jira-time-allocator/assetcap/config"
)

func JiraDoer(project string, sprint string, override string) (string, error) {
	// Load Jira configuration
	jiraConfig, err := config.NewJiraConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load Jira configuration: %w", err)
	}

	// Load teams data
	data, err := ioutil.ReadFile("teams.json")
	if err != nil {
		return "", fmt.Errorf("error reading JSON file: %v", err)
	}

	var teams assetcap.T
	err = json.Unmarshal(data, &teams)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	people, ok := teams[project]
	if !ok {
		return "", fmt.Errorf("project %s not found in teams.json", project)
	}

	// Build Jira API URL
	query := fmt.Sprintf("project = %s AND sprint in openSprints()", project)
	encodedQuery := url.QueryEscape(query)
	jiraURL := fmt.Sprintf("%s/rest/api/3/search?jql=%s", jiraConfig.GetBaseURL(), encodedQuery)

	// Create HTTP request
	req, err := http.NewRequest("GET", jiraURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication header
	req.Header.Set("Authorization", jiraConfig.GetAuthHeader())
	req.Header.Set("Accept", "application/json")

	// Fetch issues from Jira
	issues, err := assetcap.GetJiraIssues(jiraURL, jiraConfig.GetAuthHeader())
	if err != nil {
		return "", fmt.Errorf("failed to fetch Jira issues: %v", err)
	}

	// Parse manual adjustments if provided
	var manualAdjustments map[string]float64
	if override != "" {
		err := json.Unmarshal([]byte(override), &manualAdjustments)
		if err != nil {
			return "", fmt.Errorf("error parsing manual adjustments JSON: %v", err)
		}
	}

	// Initialize hours tracking
	totalHoursByPerson := make(map[string]float64)
	for _, person := range people.Team {
		totalHoursByPerson[person] = 0
	}

	// Calculate the total hours for each assignee
	for _, issue := range issues {
		assignee := issue.Fields.Assignee.DisplayName
		if assetcap.Contains(people.Team, assignee) {
			var startTime, endTime time.Time
			var inProgress, done bool

			for i := len(issue.Changelog.Histories) - 1; i >= 0; i-- {
				history := issue.Changelog.Histories[i]
				for _, item := range history.Items {
					if item.Field == "status" {
						if (item.ToString == "Done" || item.ToString == "Won't Do") && !done {
							endTime, _ = time.Parse("2006-01-02T15:04:05.000-0700", history.Created)
							done = true
						} else if item.ToString == "In Progress" && !inProgress {
							startTime, _ = time.Parse("2006-01-02T15:04:05.000-0700", history.Created)
							inProgress = true
						}
					}
				}
			}

			if inProgress && !done {
				endTime = time.Now()
			}

			if inProgress {
				workingHours := assetcap.CalculateWorkingHours(issue.Key, manualAdjustments, startTime, endTime)
				totalHoursByPerson[assignee] += workingHours
			}
		}
	}

	// Calculate the percentage load for each task and person
	var results []map[string]interface{}
	for _, issue := range issues {
		assignee := issue.Fields.Assignee.DisplayName
		if assetcap.Contains(people.Team, assignee) {
			var startTime, endTime time.Time
			var inProgress, done bool

			for i := len(issue.Changelog.Histories) - 1; i >= 0; i-- {
				history := issue.Changelog.Histories[i]
				for _, item := range history.Items {
					if item.Field == "status" {
						if (item.ToString == "Done" || item.ToString == "Won't Do") && !done {
							endTime, _ = time.Parse("2006-01-02T15:04:05.000-0700", history.Created)
							done = true
						} else if item.ToString == "In Progress" && !inProgress {
							startTime, _ = time.Parse("2006-01-02T15:04:05.000-0700", history.Created)
							inProgress = true
						}
					}
				}
			}

			if inProgress && !done {
				endTime = time.Now()
			}

			if inProgress {
				workingHours := assetcap.CalculateWorkingHours(issue.Key, manualAdjustments, startTime, endTime)
				totalHours := totalHoursByPerson[assignee]
				percentageLoad := 0.0
				if totalHours != 0 {
					percentageLoad = (workingHours / totalHours) * 100
				}

				result := make(map[string]interface{})
				result["sprint"] = sprint
				result["issueKey"] = issue.Key
				result["title"] = issue.Fields.Summary

				for _, person := range people.Team {
					result[person] = ""
				}

				result[assignee] = fmt.Sprintf("%.2f%%", percentageLoad)

				results = append(results, result)
			}
		}
	}

	// Prepare CSV headers and data
	headers := []string{"sprint", "issueKey", "title"}
	headers = append(headers, people.Team...)

	csvData, err := assetcap.StructArrayToCSVOrdered(results, headers)
	if err != nil {
		return "", fmt.Errorf("failed to generate CSV: %v", err)
	}

	return csvData, nil
}
