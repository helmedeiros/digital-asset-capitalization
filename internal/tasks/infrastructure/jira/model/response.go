package model

// SearchResponse represents the Jira API search response
type SearchResponse struct {
	Issues []Issue `json:"issues"`
}

// Issue represents a Jira issue
type Issue struct {
	Key    string `json:"key"`
	Fields Fields `json:"fields"`
}

// Fields represents the fields of a Jira issue
type Fields struct {
	Summary     string  `json:"summary"`
	Status      Status  `json:"status"`
	Project     Project `json:"project"`
	Sprint      Sprint  `json:"sprint"`
	Created     string  `json:"created"`
	Updated     string  `json:"updated"`
	Description string  `json:"description"`
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
	Name string `json:"name"`
}
