package domain

import (
	"encoding/json"
	"strings"

	sharedjira "github.com/helmedeiros/digital-asset-capitalization/internal/shared/jira"
)

const (
	StatusToDo           = "To Do"
	StatusInProgress     = "In Progress"
	StatusUnderReview    = "Under Review"
	StatusCodeReview     = "Code Review"
	StatusTesting        = "Testing"
	StatusQA             = "QA"
	StatusReadyForReview = "Ready for Review"
	StatusBlocked        = "Blocked"
	StatusDone           = "Done"
	StatusWontDo         = "Won't Do"
	StatusCancelled      = "Cancelled"
	StatusOnHold         = "On Hold"
)

// JiraAssignee represents a Jira issue assignee
type JiraAssignee struct {
	DisplayName string `json:"displayName"`
}

// JiraChangeItem represents a single change in a Jira issue's history
type JiraChangeItem struct {
	Field      string `json:"field"`
	FromString string `json:"fromString"`
	ToString   string `json:"toString"`
}

// IsStatusChange checks if this change item represents a status change
func (i *JiraChangeItem) IsStatusChange() bool {
	return i.Field == "status"
}

// JiraChangeHistory represents a historical change in a Jira issue
type JiraChangeHistory struct {
	Created string           `json:"created"`
	Items   []JiraChangeItem `json:"items"`
}

// JiraParent represents a parent issue reference
type JiraParent struct {
	Key    string     `json:"key"`
	Fields JiraFields `json:"fields"`
}

// JiraFields represents the fields of a Jira issue
type JiraFields struct {
	Summary          string                 `json:"summary"`
	Assignee         JiraAssignee           `json:"assignee"`
	StoryPoints      *float64               `json:"customfield_13192"`
	Status           JiraStatus             `json:"status"`
	IssueType        IssueType              `json:"issuetype"`
	WorkType         string                 `json:"customfield_10014"`
	AssetName        string                 `json:"customfield_10015"`
	Labels           []string               `json:"labels"`
	Parent           *JiraParent            `json:"parent"`
	TPDBusinessUnits []string               `json:"-"`
	EngineeringHours *float64               `json:"-"`
	WorkStream       string                 `json:"-"`
	BoardWorkStream  string                 `json:"-"` // fallback from board-to-workstream config
	RawFields        map[string]interface{} `json:"-"`
}

// UnmarshalJSON implements custom JSON unmarshaling for JiraFields
func (f *JiraFields) UnmarshalJSON(data []byte) error {
	// Unmarshal into raw map to capture all fields
	var rawFields map[string]interface{}
	if err := json.Unmarshal(data, &rawFields); err != nil {
		return err
	}
	f.RawFields = rawFields

	// Unmarshal tagged fields via alias type
	type Alias JiraFields
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	// Copy standard fields
	f.Summary = alias.Summary
	f.Assignee = alias.Assignee
	f.StoryPoints = alias.StoryPoints
	f.Status = alias.Status
	f.IssueType = alias.IssueType
	f.WorkType = alias.WorkType
	f.AssetName = alias.AssetName
	f.Labels = alias.Labels
	f.Parent = alias.Parent

	return nil
}

// EnrichCustomFields populates the custom TPD fields from RawFields using discovered field IDs
func (issue *JiraIssue) EnrichCustomFields(fieldIDs sharedjira.CustomFieldIDs) {
	if issue.Fields.RawFields == nil {
		return
	}

	// TPD Business Unit (multi-select)
	if fieldIDs.TPDBusinessUnit != "" {
		if raw, ok := issue.Fields.RawFields[fieldIDs.TPDBusinessUnit]; ok && raw != nil {
			issue.Fields.TPDBusinessUnits = parseMultiSelectField(raw)
		}
	}

	// Engineering time spent (hours) (numeric)
	if fieldIDs.EngineeringHours != "" {
		if raw, ok := issue.Fields.RawFields[fieldIDs.EngineeringHours]; ok && raw != nil {
			issue.Fields.EngineeringHours = parseNumericField(raw)
		}
	}

	// Work Stream (single-select)
	if fieldIDs.WorkStream != "" {
		if raw, ok := issue.Fields.RawFields[fieldIDs.WorkStream]; ok && raw != nil {
			issue.Fields.WorkStream = parseSingleSelectField(raw)
		}
	}
}

// parseMultiSelectField parses a JIRA multi-select field (array of {value: "X"} objects)
func parseMultiSelectField(raw interface{}) []string {
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	var values []string
	for _, item := range arr {
		if obj, ok := item.(map[string]interface{}); ok {
			if val, ok := obj["value"].(string); ok && val != "" {
				values = append(values, val)
			}
		}
	}
	return values
}

// parseNumericField parses a JIRA numeric field
func parseNumericField(raw interface{}) *float64 {
	switch v := raw.(type) {
	case float64:
		return &v
	case int:
		f := float64(v)
		return &f
	default:
		return nil
	}
}

// parseSingleSelectField parses a JIRA single-select field ({value: "X"} object)
func parseSingleSelectField(raw interface{}) string {
	if obj, ok := raw.(map[string]interface{}); ok {
		if val, ok := obj["value"].(string); ok {
			return val
		}
	}
	return ""
}

// JiraStatus represents the status of a Jira issue
type JiraStatus struct {
	Name string `json:"name"`
}

// JiraChangelog represents the changelog of a Jira issue
type JiraChangelog struct {
	Histories []JiraChangeHistory `json:"histories"`
}

// JiraIssue represents a single Jira issue with its fields and changelog
type JiraIssue struct {
	Key       string        `json:"key"`
	Fields    JiraFields    `json:"fields"`
	Changelog JiraChangelog `json:"changelog"`
}

// GetStatusChanges returns all status changes in chronological order
func (i *JiraIssue) GetStatusChanges() []JiraChangeHistory {
	var statusChanges []JiraChangeHistory
	for _, history := range i.Changelog.Histories {
		for _, item := range history.Items {
			if item.IsStatusChange() {
				statusChanges = append(statusChanges, history)
				break
			}
		}
	}
	return statusChanges
}

// IsInProgress checks if the issue is currently in progress
func (i *JiraIssue) IsInProgress() bool {
	changes := i.GetStatusChanges()
	if len(changes) == 0 {
		return false
	}
	lastChange := changes[len(changes)-1]
	for _, item := range lastChange.Items {
		if item.IsStatusChange() && item.ToString == StatusInProgress {
			return true
		}
	}
	return false
}

// IsDone checks if the issue is completed
func (i *JiraIssue) IsDone() bool {
	changes := i.GetStatusChanges()
	if len(changes) == 0 {
		return false
	}
	lastChange := changes[len(changes)-1]
	for _, item := range lastChange.Items {
		if item.IsStatusChange() && (item.ToString == StatusDone || item.ToString == StatusWontDo) {
			return true
		}
	}
	return false
}

// IssueType represents the type of a Jira issue
type IssueType struct {
	Name string `json:"name"`
}

// GetWorkType returns the work type based on the issue's labels
func (i *JiraIssue) GetWorkType() string {
	for _, label := range i.Fields.Labels {
		switch label {
		case "cap-maintenance":
			return "cap-maintenance"
		case "cap-discovery":
			return "cap-discovery"
		case "cap-development":
			return "cap-development"
		}
	}
	return ""
}

// GetAssetName returns the asset name based on the issue's labels
func (i *JiraIssue) GetAssetName() string {
	for _, label := range i.Fields.Labels {
		if strings.HasPrefix(label, "cap-asset-") {
			return label
		}
	}
	return ""
}

// IsSubTask checks if this issue is a sub-task
func (i *JiraIssue) IsSubTask() bool {
	return i.Fields.IssueType.Name == "Sub-task"
}

// HasParent checks if this issue has a parent
func (i *JiraIssue) HasParent() bool {
	return i.Fields.Parent != nil
}

// GetParentKey returns the parent issue key if it exists
func (i *JiraIssue) GetParentKey() string {
	if i.HasParent() {
		return i.Fields.Parent.Key
	}
	return ""
}
