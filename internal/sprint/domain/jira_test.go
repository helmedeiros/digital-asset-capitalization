package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJiraChangeItem_IsStatusChange(t *testing.T) {
	tests := []struct {
		name string
		item JiraChangeItem
		want bool
	}{
		{
			name: "status change",
			item: JiraChangeItem{
				Field: "status",
			},
			want: true,
		},
		{
			name: "non-status change",
			item: JiraChangeItem{
				Field: "summary",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.item.IsStatusChange(); got != tt.want {
				t.Errorf("JiraChangeItem.IsStatusChange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJiraIssue_GetStatusChanges(t *testing.T) {
	tests := []struct {
		name  string
		issue JiraIssue
		want  []JiraChangeHistory
	}{
		{
			name: "single status change",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Created: "2024-03-23T10:00:00.000-0700",
							Items: []JiraChangeItem{
								{
									Field:      "status",
									FromString: "To Do",
									ToString:   StatusInProgress,
								},
							},
						},
					},
				},
			},
			want: []JiraChangeHistory{
				{
					Created: "2024-03-23T10:00:00.000-0700",
					Items: []JiraChangeItem{
						{
							Field:      "status",
							FromString: "To Do",
							ToString:   StatusInProgress,
						},
					},
				},
			},
		},
		{
			name: "multiple status changes",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Created: "2024-03-23T10:00:00.000-0700",
							Items: []JiraChangeItem{
								{
									Field:      "status",
									FromString: "To Do",
									ToString:   StatusInProgress,
								},
								{
									Field:      "status",
									FromString: StatusInProgress,
									ToString:   StatusDone,
								},
							},
						},
					},
				},
			},
			want: []JiraChangeHistory{
				{
					Created: "2024-03-23T10:00:00.000-0700",
					Items: []JiraChangeItem{
						{
							Field:      "status",
							FromString: "To Do",
							ToString:   StatusInProgress,
						},
						{
							Field:      "status",
							FromString: StatusInProgress,
							ToString:   StatusDone,
						},
					},
				},
			},
		},
		{
			name: "no status changes",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Created: "2024-03-23T10:00:00.000-0700",
							Items: []JiraChangeItem{
								{
									Field:      "summary",
									FromString: "Old Summary",
									ToString:   "New Summary",
								},
							},
						},
					},
				},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.issue.GetStatusChanges()
			if len(got) != len(tt.want) {
				t.Errorf("JiraIssue.GetStatusChanges() returned %d changes, want %d", len(got), len(tt.want))
			}
			for i, change := range got {
				if change.Created != tt.want[i].Created {
					t.Errorf("JiraIssue.GetStatusChanges()[%d].Created = %v, want %v", i, change.Created, tt.want[i].Created)
				}
				if len(change.Items) != len(tt.want[i].Items) {
					t.Errorf("JiraIssue.GetStatusChanges()[%d].Items length = %d, want %d", i, len(change.Items), len(tt.want[i].Items))
				}
				for j, item := range change.Items {
					if item.Field != tt.want[i].Items[j].Field {
						t.Errorf("JiraIssue.GetStatusChanges()[%d].Items[%d].Field = %v, want %v", i, j, item.Field, tt.want[i].Items[j].Field)
					}
					if item.FromString != tt.want[i].Items[j].FromString {
						t.Errorf("JiraIssue.GetStatusChanges()[%d].Items[%d].FromString = %v, want %v", i, j, item.FromString, tt.want[i].Items[j].FromString)
					}
					if item.ToString != tt.want[i].Items[j].ToString {
						t.Errorf("JiraIssue.GetStatusChanges()[%d].Items[%d].ToString = %v, want %v", i, j, item.ToString, tt.want[i].Items[j].ToString)
					}
				}
			}
		})
	}
}

func TestJiraIssue_IsInProgress(t *testing.T) {
	tests := []struct {
		name  string
		issue JiraIssue
		want  bool
	}{
		{
			name: "currently in progress",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Created: "2024-03-23T10:00:00.000-0700",
							Items: []JiraChangeItem{
								{
									Field:      "status",
									FromString: "To Do",
									ToString:   StatusInProgress,
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not in progress",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Created: "2024-03-23T10:00:00.000-0700",
							Items: []JiraChangeItem{
								{
									Field:      "status",
									FromString: StatusInProgress,
									ToString:   StatusDone,
								},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "no status changes",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.issue.IsInProgress(); got != tt.want {
				t.Errorf("JiraIssue.IsInProgress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJiraIssue_IsDone(t *testing.T) {
	tests := []struct {
		name  string
		issue JiraIssue
		want  bool
	}{
		{
			name: "done",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Created: "2024-03-23T10:00:00.000-0700",
							Items: []JiraChangeItem{
								{
									Field:      "status",
									FromString: StatusInProgress,
									ToString:   StatusDone,
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "wont do",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Created: "2024-03-23T10:00:00.000-0700",
							Items: []JiraChangeItem{
								{
									Field:      "status",
									FromString: StatusInProgress,
									ToString:   StatusWontDo,
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not done",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{
						{
							Created: "2024-03-23T10:00:00.000-0700",
							Items: []JiraChangeItem{
								{
									Field:      "status",
									FromString: "To Do",
									ToString:   StatusInProgress,
								},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "no status changes",
			issue: JiraIssue{
				Changelog: JiraChangelog{
					Histories: []JiraChangeHistory{},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.issue.IsDone(); got != tt.want {
				t.Errorf("JiraIssue.IsDone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJiraIssue_GetWorkType(t *testing.T) {
	issue := &JiraIssue{Fields: JiraFields{Labels: []string{"cap-maintenance", "other"}}}
	assert.Equal(t, "cap-maintenance", issue.GetWorkType())

	issue = &JiraIssue{Fields: JiraFields{Labels: []string{"cap-discovery"}}}
	assert.Equal(t, "cap-discovery", issue.GetWorkType())

	issue = &JiraIssue{Fields: JiraFields{Labels: []string{"cap-development"}}}
	assert.Equal(t, "cap-development", issue.GetWorkType())

	issue = &JiraIssue{Fields: JiraFields{Labels: []string{"other"}}}
	assert.Equal(t, "", issue.GetWorkType())
}

func TestJiraIssue_GetAssetName(t *testing.T) {
	issue := &JiraIssue{Fields: JiraFields{Labels: []string{"cap-asset-foo", "other"}}}
	assert.Equal(t, "cap-asset-foo", issue.GetAssetName())

	issue = &JiraIssue{Fields: JiraFields{Labels: []string{"other", "cap-asset-bar"}}}
	assert.Equal(t, "cap-asset-bar", issue.GetAssetName())

	issue = &JiraIssue{Fields: JiraFields{Labels: []string{"other"}}}
	assert.Equal(t, "", issue.GetAssetName())
}
