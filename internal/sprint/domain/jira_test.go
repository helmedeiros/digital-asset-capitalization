package domain

import (
	"testing"
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
									ToString:   "In Progress",
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
							ToString:   "In Progress",
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
									ToString:   "In Progress",
								},
								{
									Field:      "status",
									FromString: "In Progress",
									ToString:   "Done",
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
							ToString:   "In Progress",
						},
						{
							Field:      "status",
							FromString: "In Progress",
							ToString:   "Done",
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
									ToString:   "In Progress",
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
									FromString: "In Progress",
									ToString:   "Done",
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
									FromString: "In Progress",
									ToString:   "Done",
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
									FromString: "In Progress",
									ToString:   "Won't Do",
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
									ToString:   "In Progress",
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
