package assetcap

// Known Jira status values
const (
	StatusDone       = "Done"
	StatusWontDo     = "Won't Do"
	StatusInProgress = "In Progress"
)

// Team represents a group of team members
type Team struct {
	Members []string `json:"team"`
}

// IsTeamMember checks if a person is a member of the team
func (t *Team) IsTeamMember(person string) bool {
	for _, member := range t.Members {
		if member == person {
			return true
		}
	}
	return false
}

// TeamMap is a mapping of project keys to their respective teams
type TeamMap map[string]Team

// GetTeam returns a team for a given project key
func (tm TeamMap) GetTeam(projectKey string) (*Team, bool) {
	team, exists := tm[projectKey]
	if !exists {
		return nil, false
	}
	return &team, true
}

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

// JiraFields represents the fields of a Jira issue
type JiraFields struct {
	Summary     string       `json:"summary"`
	Assignee    JiraAssignee `json:"assignee"`
	StoryPoints *float64     `json:"customfield_13192"`
	Status      JiraStatus   `json:"status"`
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

// JiraResponse represents the response from a Jira API search query
type JiraResponse struct {
	Issues []JiraIssue `json:"issues"`
}

// GetIssuesForTeamMember returns all issues assigned to a specific team member
func (r *JiraResponse) GetIssuesForTeamMember(member string) []JiraIssue {
	var issues []JiraIssue
	for _, issue := range r.Issues {
		if issue.Fields.Assignee.DisplayName == member {
			issues = append(issues, issue)
		}
	}
	return issues
}
