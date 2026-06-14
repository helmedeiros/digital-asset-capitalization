package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharedjira "github.com/helmedeiros/digital-asset-capitalization/internal/shared/jira"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
)

func TestNewJiraAdapterLegacy_DelegatesToNewJiraAdapter(t *testing.T) {
	// NewJiraAdapterLegacy is a deprecated shim that ignores its
	// argument and calls NewJiraAdapter. Without JIRA env vars
	// NewJiraAdapter returns an error; pin that the shim propagates
	// that error rather than returning a half-built adapter. t.Setenv
	// auto-restores the previous value when the test ends.
	t.Setenv("JIRA_BASE_URL", "")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("JIRA_TOKEN", "")

	got, err := NewJiraAdapterLegacy("ignored")
	assert.Nil(t, got)
	assert.Error(t, err)
}

func TestBuildFieldsParam(t *testing.T) {
	t.Run("no custom field IDs returns the base field set", func(t *testing.T) {
		a := &JiraAdapter{}
		got := a.buildFieldsParam()
		assert.Contains(t, got, "summary")
		assert.Contains(t, got, "assignee")
		assert.Contains(t, got, "changelog")
		assert.Contains(t, got, "customfield_10014")
		assert.Contains(t, got, "customfield_10015")
	})

	t.Run("empty CustomFieldIDs struct leaves the base set unchanged", func(t *testing.T) {
		a := &JiraAdapter{fieldIDs: &sharedjira.CustomFieldIDs{}}
		got := a.buildFieldsParam()
		assert.Contains(t, got, "summary")
		// Each empty field-ID branch is skipped so the comma-delimited
		// list shouldn't end with a trailing comma.
		assert.NotContains(t, got, ",,")
	})

	t.Run("populated CustomFieldIDs appends each custom field", func(t *testing.T) {
		a := &JiraAdapter{fieldIDs: &sharedjira.CustomFieldIDs{
			TPDBusinessUnit:  "customfield_100",
			EngineeringHours: "customfield_200",
			WorkStream:       "customfield_300",
		}}
		got := a.buildFieldsParam()
		assert.Contains(t, got, "customfield_100")
		assert.Contains(t, got, "customfield_200")
		assert.Contains(t, got, "customfield_300")
	})
}

func TestConvertChangelog(t *testing.T) {
	t.Run("empty changelog produces an empty histories slice", func(t *testing.T) {
		got := convertChangelog(domain.JiraChangelog{})
		require.NotNil(t, got.Histories)
		assert.Empty(t, got.Histories)
	})

	t.Run("histories and items are copied through unchanged", func(t *testing.T) {
		in := domain.JiraChangelog{Histories: []domain.JiraChangeHistory{
			{
				Created: "2026-01-02T03:04:05.000+0000",
				Items: []domain.JiraChangeItem{
					{Field: "status", FromString: "To Do", ToString: "In Progress"},
					{Field: "assignee", FromString: "Alice", ToString: "Bob"},
				},
			},
			{
				Created: "2026-02-03T04:05:06.000+0000",
				Items:   []domain.JiraChangeItem{{Field: "status", FromString: "In Progress", ToString: "Done"}},
			},
		}}

		got := convertChangelog(in)
		require.Len(t, got.Histories, 2)

		require.Len(t, got.Histories[0].Items, 2)
		assert.Equal(t, "2026-01-02T03:04:05.000+0000", got.Histories[0].Created)
		assert.Equal(t, "status", got.Histories[0].Items[0].Field)
		assert.Equal(t, "To Do", got.Histories[0].Items[0].FromString)
		assert.Equal(t, "In Progress", got.Histories[0].Items[0].ToString)
		assert.Equal(t, "assignee", got.Histories[0].Items[1].Field)

		require.Len(t, got.Histories[1].Items, 1)
		assert.Equal(t, "Done", got.Histories[1].Items[0].ToString)
	})

	t.Run("history with empty items still returns a non-nil items slice", func(t *testing.T) {
		got := convertChangelog(domain.JiraChangelog{
			Histories: []domain.JiraChangeHistory{{Created: "x"}},
		})
		require.Len(t, got.Histories, 1)
		require.NotNil(t, got.Histories[0].Items)
		assert.Empty(t, got.Histories[0].Items)
	})
}
