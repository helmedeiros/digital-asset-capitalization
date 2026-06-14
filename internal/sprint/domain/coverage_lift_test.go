package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharedjira "github.com/helmedeiros/digital-asset-capitalization/internal/shared/jira"
)

func TestJiraIssue_ParentHelpers(t *testing.T) {
	t.Run("issue without parent", func(t *testing.T) {
		i := &JiraIssue{Fields: JiraFields{Parent: nil}}
		assert.False(t, i.HasParent())
		assert.Equal(t, "", i.GetParentKey())
	})

	t.Run("issue with parent", func(t *testing.T) {
		i := &JiraIssue{Fields: JiraFields{Parent: &JiraParent{Key: "EPIC-1"}}}
		assert.True(t, i.HasParent())
		assert.Equal(t, "EPIC-1", i.GetParentKey())
	})
}

func TestJiraIssue_IsSubTask(t *testing.T) {
	cases := []struct {
		typeName string
		want     bool
	}{
		{"Sub-task", true},
		{"Story", false},
		{"Bug", false},
		{"", false},
	}
	for _, c := range cases {
		t.Run(c.typeName, func(t *testing.T) {
			i := &JiraIssue{Fields: JiraFields{IssueType: IssueType{Name: c.typeName}}}
			assert.Equal(t, c.want, i.IsSubTask())
		})
	}
}

func TestJiraFields_UnmarshalJSON(t *testing.T) {
	t.Run("populates standard and raw fields", func(t *testing.T) {
		payload := []byte(`{
			"summary": "Implement feature",
			"assignee": {"displayName": "Alice"},
			"status": {"name": "Done"},
			"issuetype": {"name": "Story"},
			"labels": ["cap-development"],
			"customfield_123": "raw value"
		}`)
		var f JiraFields
		require.NoError(t, json.Unmarshal(payload, &f))

		assert.Equal(t, "Implement feature", f.Summary)
		assert.Equal(t, "Alice", f.Assignee.DisplayName)
		assert.Equal(t, "Done", f.Status.Name)
		assert.Equal(t, "Story", f.IssueType.Name)
		assert.Equal(t, []string{"cap-development"}, f.Labels)
		require.NotNil(t, f.RawFields)
		assert.Equal(t, "raw value", f.RawFields["customfield_123"])
	})

	t.Run("invalid JSON propagates the error", func(t *testing.T) {
		var f JiraFields
		err := json.Unmarshal([]byte(`{not valid json`), &f)
		require.Error(t, err)
	})
}

func TestJiraIssue_EnrichCustomFields(t *testing.T) {
	t.Run("nil RawFields is a safe no-op", func(t *testing.T) {
		i := &JiraIssue{Fields: JiraFields{RawFields: nil}}
		i.EnrichCustomFields(sharedjira.CustomFieldIDs{
			TPDBusinessUnit:  "cf_tpd",
			EngineeringHours: "cf_hours",
			WorkStream:       "cf_ws",
		})
		assert.Nil(t, i.Fields.TPDBusinessUnits)
		assert.Nil(t, i.Fields.EngineeringHours)
		assert.Equal(t, "", i.Fields.WorkStream)
	})

	t.Run("populates all three custom-field shapes", func(t *testing.T) {
		i := &JiraIssue{Fields: JiraFields{RawFields: map[string]interface{}{
			"cf_tpd": []interface{}{
				map[string]interface{}{"value": "Payments"},
				map[string]interface{}{"value": "Search"},
			},
			"cf_hours": 7.5,
			"cf_ws":    map[string]interface{}{"value": "Pricing"},
		}}}
		i.EnrichCustomFields(sharedjira.CustomFieldIDs{
			TPDBusinessUnit:  "cf_tpd",
			EngineeringHours: "cf_hours",
			WorkStream:       "cf_ws",
		})
		assert.Equal(t, []string{"Payments", "Search"}, i.Fields.TPDBusinessUnits)
		require.NotNil(t, i.Fields.EngineeringHours)
		assert.Equal(t, 7.5, *i.Fields.EngineeringHours)
		assert.Equal(t, "Pricing", i.Fields.WorkStream)
	})

	t.Run("empty field IDs skip the corresponding lookups", func(t *testing.T) {
		i := &JiraIssue{Fields: JiraFields{RawFields: map[string]interface{}{
			"cf_tpd": []interface{}{map[string]interface{}{"value": "X"}},
		}}}
		i.EnrichCustomFields(sharedjira.CustomFieldIDs{}) // all field IDs empty
		assert.Nil(t, i.Fields.TPDBusinessUnits)
		assert.Nil(t, i.Fields.EngineeringHours)
		assert.Equal(t, "", i.Fields.WorkStream)
	})

	t.Run("nil raw value at known field ID is skipped", func(t *testing.T) {
		i := &JiraIssue{Fields: JiraFields{RawFields: map[string]interface{}{
			"cf_tpd": nil,
		}}}
		i.EnrichCustomFields(sharedjira.CustomFieldIDs{TPDBusinessUnit: "cf_tpd"})
		assert.Nil(t, i.Fields.TPDBusinessUnits)
	})
}

func TestParseMultiSelectField(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want []string
	}{
		{"non-array input returns nil", "not an array", nil},
		{"empty array returns nil", []interface{}{}, nil},
		{
			"extracts values from objects",
			[]interface{}{
				map[string]interface{}{"value": "A"},
				map[string]interface{}{"value": "B"},
			},
			[]string{"A", "B"},
		},
		{
			"skips non-objects and missing/empty values",
			[]interface{}{
				"scalar",
				map[string]interface{}{"label": "no value key"},
				map[string]interface{}{"value": ""},
				map[string]interface{}{"value": "C"},
			},
			[]string{"C"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, parseMultiSelectField(c.in))
		})
	}
}

func TestParseNumericField(t *testing.T) {
	t.Run("float64 returns pointer to value", func(t *testing.T) {
		got := parseNumericField(3.5)
		require.NotNil(t, got)
		assert.Equal(t, 3.5, *got)
	})
	t.Run("int returns pointer to float-converted value", func(t *testing.T) {
		got := parseNumericField(7)
		require.NotNil(t, got)
		assert.Equal(t, 7.0, *got)
	})
	t.Run("string returns nil", func(t *testing.T) {
		assert.Nil(t, parseNumericField("7"))
	})
}

func TestParseSingleSelectField(t *testing.T) {
	t.Run("object with value returns that string", func(t *testing.T) {
		assert.Equal(t, "Pricing", parseSingleSelectField(map[string]interface{}{"value": "Pricing"}))
	})
	t.Run("object without value key returns empty", func(t *testing.T) {
		assert.Equal(t, "", parseSingleSelectField(map[string]interface{}{"label": "x"}))
	})
	t.Run("non-object returns empty", func(t *testing.T) {
		assert.Equal(t, "", parseSingleSelectField([]interface{}{"x"}))
	})
}

func TestTeam_IsTeamMember(t *testing.T) {
	team := &Team{Team: []string{"Alice", "Bob"}}
	assert.True(t, team.IsTeamMember("Alice"))
	assert.True(t, team.IsTeamMember("Bob"))
	assert.False(t, team.IsTeamMember("Carol"))
	assert.False(t, team.IsTeamMember(""))
}

func TestTeamMap_GetTeam(t *testing.T) {
	tm := TeamMap{
		"PROJ": Team{Team: []string{"Alice"}, ExcludedIssueTypes: []string{"Experiment"}},
	}

	t.Run("known project returns team and true", func(t *testing.T) {
		team, ok := tm.GetTeam("PROJ")
		require.True(t, ok)
		require.NotNil(t, team)
		assert.Equal(t, []string{"Alice"}, team.Team)
		assert.Equal(t, []string{"Experiment"}, team.ExcludedIssueTypes)
	})

	t.Run("unknown project returns nil and false", func(t *testing.T) {
		team, ok := tm.GetTeam("UNKNOWN")
		assert.False(t, ok)
		assert.Nil(t, team)
	})
}

func TestStatusChangePeriod_Duration(t *testing.T) {
	start := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	t.Run("zero end time returns zero duration", func(t *testing.T) {
		scp := StatusChangePeriod{StartTime: start}
		assert.Equal(t, time.Duration(0), scp.Duration())
	})

	t.Run("non-zero end time returns the delta", func(t *testing.T) {
		scp := StatusChangePeriod{StartTime: start, EndTime: start.Add(2 * time.Hour)}
		assert.Equal(t, 2*time.Hour, scp.Duration())
	})
}

// fakeStatusChecker is a minimal in-test StatusChecker that classifies
// statuses against a small static map. Lets us drive the work-time
// branches in IsWorkTimeWithStatusChecker without dragging the
// infrastructure-side status maps into the domain test surface.
type fakeStatusChecker struct {
	inProgress map[string]bool
	done       map[string]bool
	wontDo     map[string]bool
}

func (f *fakeStatusChecker) IsInProgress(status, _, _ string) bool {
	return f.inProgress[status]
}
func (f *fakeStatusChecker) IsDone(status, _, _ string) bool {
	return f.done[status]
}
func (f *fakeStatusChecker) IsWontDo(status, _, _ string) bool {
	return f.wontDo[status]
}

func TestStatusChangePeriod_IsWorkTimeWithStatusChecker(t *testing.T) {
	scp := StatusChangePeriod{Status: "In Progress"}

	t.Run("nil checker falls back to the pattern-matching path", func(t *testing.T) {
		// "In Progress" matches the fallback's in-progress patterns.
		assert.True(t, scp.IsWorkTimeWithStatusChecker(nil, "TEAM", "1"))
	})

	t.Run("explicit in-progress returns true", func(t *testing.T) {
		c := &fakeStatusChecker{inProgress: map[string]bool{"In Progress": true}}
		assert.True(t, scp.IsWorkTimeWithStatusChecker(c, "TEAM", "1"))
	})

	t.Run("explicit done returns false", func(t *testing.T) {
		done := StatusChangePeriod{Status: "Done"}
		c := &fakeStatusChecker{done: map[string]bool{"Done": true}}
		assert.False(t, done.IsWorkTimeWithStatusChecker(c, "TEAM", "1"))
	})

	t.Run("explicit wont-do returns false", func(t *testing.T) {
		wontDo := StatusChangePeriod{Status: "Won't Do"}
		c := &fakeStatusChecker{wontDo: map[string]bool{"Won't Do": true}}
		assert.False(t, wontDo.IsWorkTimeWithStatusChecker(c, "TEAM", "1"))
	})

	t.Run("unclassified status falls back to pattern matching", func(t *testing.T) {
		// Empty checker maps -> nothing is explicitly classified.
		// "In Progress" still hits the fallback's regex/pattern path.
		c := &fakeStatusChecker{}
		assert.True(t, scp.IsWorkTimeWithStatusChecker(c, "TEAM", "1"))
	})
}

func TestNewSprintBoundedTimeCalculatorWithStatusChecker(t *testing.T) {
	checker := &fakeStatusChecker{}
	calc := NewSprintBoundedTimeCalculatorWithStatusChecker(checker, "TEAM", "BOARD-1")
	require.NotNil(t, calc)
	assert.Equal(t, "TEAM", calc.teamKey)
	assert.Equal(t, "BOARD-1", calc.boardID)
	assert.Equal(t, StatusChecker(checker), calc.statusChecker)
}
