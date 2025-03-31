package api

import (
	"encoding/json"
	"strings"
)

// JiraIssue represents a task in our domain
type JiraIssue struct {
	Key      string
	Summary  string
	Status   string
	Assignee string
	Sprint   []string
}

// SearchResult represents the Jira API search response
type SearchResult struct {
	Issues []Issue `json:"issues"`
}

// Issue represents a Jira issue
type Issue struct {
	Key    string `json:"key"`
	Fields Fields `json:"fields"`
}

// Fields represents the fields of a Jira issue
type Fields struct {
	Summary     string                 `json:"summary"`
	Description Description            `json:"description"`
	Status      Status                 `json:"status"`
	Project     Project                `json:"project"`
	Sprint      []Sprint               `json:"sprint"`
	Changelog   Changelog              `json:"changelog"`
	Created     string                 `json:"created"`
	Updated     string                 `json:"updated"`
	Assignee    Assignee               `json:"assignee"`
	IssueType   IssueType              `json:"issuetype"`
	Parent      *Issue                 `json:"parent"`
	WorkType    string                 `json:"customfield_10014"`
	AssetName   string                 `json:"customfield_10015"`
	Labels      []string               `json:"labels"`
	RawFields   map[string]interface{} `json:"-"`
}

// UnmarshalJSON implements custom JSON unmarshaling for Fields
func (f *Fields) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to get all fields
	var rawFields map[string]interface{}
	if err := json.Unmarshal(data, &rawFields); err != nil {
		return err
	}

	// Store raw fields for later use
	f.RawFields = rawFields

	// Create a temporary struct for standard fields
	type tempFields Fields
	var temp tempFields
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Copy standard fields
	*f = Fields(temp)

	// Look for sprint field in all custom fields
	for key, value := range rawFields {
		if strings.HasPrefix(key, "customfield_") {
			// Try to unmarshal as sprint array
			if sprintData, ok := value.([]interface{}); ok && len(sprintData) > 0 {
				var sprints []Sprint
				for _, sprintItem := range sprintData {
					if sprintMap, ok := sprintItem.(map[string]interface{}); ok {
						var sprint Sprint
						if sprintJSON, err := json.Marshal(sprintMap); err == nil {
							if err := json.Unmarshal(sprintJSON, &sprint); err == nil && sprint.Name != "" {
								sprints = append(sprints, sprint)
							}
						}
					}
				}
				if len(sprints) > 0 {
					f.Sprint = sprints
					break
				}
			}
		}
	}

	return nil
}

// Status represents the status of a Jira issue
type Status struct {
	Name string `json:"name"`
}

// Project represents a Jira project
type Project struct {
	Key string `json:"key"`
}

// Sprint represents a Jira sprint
type Sprint struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	State        string `json:"state"`
	StartDate    string `json:"startDate,omitempty"`
	EndDate      string `json:"endDate,omitempty"`
	CompleteDate string `json:"completeDate,omitempty"`
	BoardID      int    `json:"boardId"`
	Goal         string `json:"goal,omitempty"`
}

// ChangelogItem represents a single change in a Jira issue's history
type ChangelogItem struct {
	Field      string `json:"field"`
	FromString string `json:"fromString"`
	ToString   string `json:"toString"`
}

// ChangelogHistory represents a historical change in a Jira issue
type ChangelogHistory struct {
	Created string          `json:"created"`
	Items   []ChangelogItem `json:"items"`
}

// Changelog represents the changelog of a Jira issue
type Changelog struct {
	Histories []ChangelogHistory `json:"histories"`
}

// Description represents the description content of a Jira issue
type Description struct {
	Type    string `json:"type"`
	Version int    `json:"version"`
	Content []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"content"`
}

// Assignee represents the assignee of a Jira issue
type Assignee struct {
	DisplayName string `json:"displayName"`
}

// Add IssueType struct
type IssueType struct {
	Name string `json:"name"`
}
