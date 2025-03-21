package model

import (
	"encoding/json"
	"strings"
)

// Task represents a task in our domain
type Task struct {
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
	Summary   string      `json:"summary"`
	Status    Status      `json:"status"`
	Project   Project     `json:"project"`
	Sprint    []Sprint    `json:"-"`
	RawFields interface{} `json:"-"`
	Created   string      `json:"created,omitempty"`
	Updated   string      `json:"updated,omitempty"`
	Assignee  struct {
		DisplayName string `json:"displayName,omitempty"`
	} `json:"assignee"`
	Description struct {
		Type    string `json:"type,omitempty"`
		Version int    `json:"version,omitempty"`
		Content []struct {
			Type    string `json:"type,omitempty"`
			Content []struct {
				Type string `json:"type,omitempty"`
				Text string `json:"text,omitempty"`
			} `json:"content"`
		} `json:"content"`
	} `json:"description"`
	Changelog struct {
		Histories []struct {
			Created string `json:"created,omitempty"`
			Items   []struct {
				Field    string `json:"field,omitempty"`
				ToString string `json:"toString,omitempty"`
			} `json:"items"`
		} `json:"histories"`
	} `json:"changelog"`
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
