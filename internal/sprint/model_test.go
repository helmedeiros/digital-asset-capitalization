package assetcap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTeam_IsTeamMember(t *testing.T) {
	tests := []struct {
		name     string
		team     Team
		person   string
		expected bool
	}{
		{
			name:     "person is team member",
			team:     Team{Members: []string{"John Doe", "Jane Smith"}},
			person:   "John Doe",
			expected: true,
		},
		{
			name:     "person is not team member",
			team:     Team{Members: []string{"John Doe", "Jane Smith"}},
			person:   "Bob Wilson",
			expected: false,
		},
		{
			name:     "empty team",
			team:     Team{Members: []string{}},
			person:   "John Doe",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.team.IsTeamMember(tt.person)
			assert.Equal(t, tt.expected, result, "IsTeamMember result mismatch")
		})
	}
}

func TestTeamMap_GetTeam(t *testing.T) {
	tests := []struct {
		name       string
		teamMap    TeamMap
		projectKey string
		wantTeam   *Team
		wantExists bool
	}{
		{
			name: "team exists",
			teamMap: TeamMap{
				"PROJ": Team{Members: []string{"John Doe"}},
			},
			projectKey: "PROJ",
			wantTeam:   &Team{Members: []string{"John Doe"}},
			wantExists: true,
		},
		{
			name: "team does not exist",
			teamMap: TeamMap{
				"PROJ": Team{Members: []string{"John Doe"}},
			},
			projectKey: "OTHER",
			wantTeam:   nil,
			wantExists: false,
		},
		{
			name:       "empty team map",
			teamMap:    TeamMap{},
			projectKey: "PROJ",
			wantTeam:   nil,
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTeam, gotExists := tt.teamMap.GetTeam(tt.projectKey)
			assert.Equal(t, tt.wantExists, gotExists, "GetTeam exists mismatch")
			if tt.wantTeam != nil && gotTeam != nil {
				assert.Equal(t, len(tt.wantTeam.Members), len(gotTeam.Members), "Team members count mismatch")
			} else {
				assert.Equal(t, tt.wantTeam, gotTeam, "Team pointer mismatch")
			}
		})
	}
}

func TestJiraChangeItem_IsStatusChange(t *testing.T) {
	tests := []struct {
		name     string
		item     JiraChangeItem
		expected bool
	}{
		{
			name:     "status change",
			item:     JiraChangeItem{Field: "status"},
			expected: true,
		},
		{
			name:     "not status change",
			item:     JiraChangeItem{Field: "summary"},
			expected: false,
		},
		{
			name:     "empty field",
			item:     JiraChangeItem{Field: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.IsStatusChange()
			assert.Equal(t, tt.expected, result, "IsStatusChange result mismatch")
		})
	}
}

func TestJiraIssue_GetStatusChanges(t *testing.T) {
	tests := []struct {
		name     string
		issue    JiraIssue
		expected int
	}{
		{
			name: "has status changes",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Items: []JiraChangeItem{
								{Field: "status", FromString: "To Do", ToString: StatusInProgress},
							},
						},
						{
							Items: []JiraChangeItem{
								{Field: "status", FromString: StatusInProgress, ToString: StatusDone},
							},
						},
					},
				},
			},
			expected: 2,
		},
		{
			name: "no status changes",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Items: []JiraChangeItem{
								{Field: "summary", FromString: "Old", ToString: "New"},
							},
						},
					},
				},
			},
			expected: 0,
		},
		{
			name:     "empty changelog",
			issue:    JiraIssue{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := tt.issue.GetStatusChanges()
			assert.Equal(t, tt.expected, len(changes), "GetStatusChanges count mismatch")
		})
	}
}

func TestJiraIssue_IsInProgress(t *testing.T) {
	tests := []struct {
		name     string
		issue    JiraIssue
		expected bool
	}{
		{
			name: "in progress",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Items: []JiraChangeItem{
								{Field: "status", FromString: "To Do", ToString: StatusInProgress},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "not in progress",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Items: []JiraChangeItem{
								{Field: "status", FromString: StatusInProgress, ToString: StatusDone},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name:     "empty changelog",
			issue:    JiraIssue{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.issue.IsInProgress()
			assert.Equal(t, tt.expected, result, "IsInProgress result mismatch")
		})
	}
}

func TestJiraIssue_IsDone(t *testing.T) {
	tests := []struct {
		name     string
		issue    JiraIssue
		expected bool
	}{
		{
			name: "done",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Items: []JiraChangeItem{
								{Field: "status", FromString: StatusInProgress, ToString: StatusDone},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "not done",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Items: []JiraChangeItem{
								{Field: "status", FromString: "To Do", ToString: StatusInProgress},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name:     "empty changelog",
			issue:    JiraIssue{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.issue.IsDone()
			assert.Equal(t, tt.expected, result, "IsDone result mismatch")
		})
	}
}

func TestJiraResponse_GetIssuesForTeamMember(t *testing.T) {
	tests := []struct {
		name     string
		response JiraResponse
		member   string
		expected int
	}{
		{
			name: "has issues for member",
			response: JiraResponse{
				Issues: []JiraIssue{
					{
						Fields: JiraFields{
							Assignee: JiraAssignee{DisplayName: "John Doe"},
						},
					},
					{
						Fields: JiraFields{
							Assignee: JiraAssignee{DisplayName: "John Doe"},
						},
					},
				},
			},
			member:   "John Doe",
			expected: 2,
		},
		{
			name: "no issues for member",
			response: JiraResponse{
				Issues: []JiraIssue{
					{
						Fields: JiraFields{
							Assignee: JiraAssignee{DisplayName: "Jane Smith"},
						},
					},
				},
			},
			member:   "John Doe",
			expected: 0,
		},
		{
			name:     "empty response",
			response: JiraResponse{},
			member:   "John Doe",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := tt.response.GetIssuesForTeamMember(tt.member)
			assert.Equal(t, tt.expected, len(issues), "GetIssuesForTeamMember count mismatch")
		})
	}
}
