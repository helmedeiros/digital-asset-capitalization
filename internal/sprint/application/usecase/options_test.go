package usecase

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithHours_SetsField(t *testing.T) {
	uc := &SprintTimeAllocationUseCase{}
	WithHours(true)(uc)
	assert.True(t, uc.withHours)

	WithHours(false)(uc)
	assert.False(t, uc.withHours)
}

func TestWithWorkStreams_SetsField(t *testing.T) {
	uc := &SprintTimeAllocationUseCase{}
	WithWorkStreams([]string{"Pricing", "Search"})(uc)
	assert.Equal(t, []string{"Pricing", "Search"}, uc.workStreams)

	// Empty slice overwrites whatever was there.
	WithWorkStreams([]string{})(uc)
	assert.Empty(t, uc.workStreams)
}

func TestGetJiraPort_ReturnsWiredPort(t *testing.T) {
	mock := new(MockJiraAdapter)
	uc := &SprintTimeAllocationUseCase{jiraPort: mock}
	assert.Same(t, mock, uc.GetJiraPort())
}

func TestExtractAllocationRecords(t *testing.T) {
	uc := &SprintTimeAllocationUseCase{}

	t.Run("returns empty slice for empty input", func(t *testing.T) {
		assert.Empty(t, uc.extractAllocationRecords(nil))
		assert.Empty(t, uc.extractAllocationRecords([]map[string]interface{}{}))
	})

	t.Run("skips rows with empty or missing issueKey", func(t *testing.T) {
		got := uc.extractAllocationRecords([]map[string]interface{}{
			{"issueKey": ""},
			{"workStream": "Pricing"}, // no key at all
			{"issueKey": "TEST-1"},
		})
		require.Len(t, got, 1)
		assert.Equal(t, "TEST-1", got[0].IssueKey)
	})

	t.Run("parses engineeringHours when present and numeric", func(t *testing.T) {
		got := uc.extractAllocationRecords([]map[string]interface{}{
			{"issueKey": "TEST-1", "engineeringHours": "7.5"},
			{"issueKey": "TEST-2", "engineeringHours": ""},    // empty string -> no value
			{"issueKey": "TEST-3", "engineeringHours": "ZZZ"}, // non-numeric -> no value
		})
		require.Len(t, got, 3)
		require.NotNil(t, got[0].EngineeringHours)
		assert.InDelta(t, 7.5, *got[0].EngineeringHours, 1e-9)
		assert.Nil(t, got[1].EngineeringHours)
		assert.Nil(t, got[2].EngineeringHours)
	})

	t.Run("populates workStream and tpdBusinessUnit when present", func(t *testing.T) {
		got := uc.extractAllocationRecords([]map[string]interface{}{
			{
				"issueKey":        "TEST-1",
				"workStream":      "Pricing",
				"tpdBusinessUnit": "Payments",
			},
		})
		require.Len(t, got, 1)
		assert.Equal(t, "Pricing", got[0].WorkStream)
		assert.Equal(t, "Payments", got[0].TPDBusinessUnit)
	})
}

func TestNewSprintTimeAllocationUseCaseWithOptions(t *testing.T) {
	t.Run("constructor failure short-circuits before any option applies", func(t *testing.T) {
		// Without setupTestEnv the underlying NewSprintTimeAllocationUseCaseWithStrategy
		// can't find a .assetcap directory and bubbles a config error.
		// Pin that the constructor returns nil + an error rather than a
		// half-built use case.
		_, err := NewSprintTimeAllocationUseCaseWithOptions("UNREAL", "Sprint 1", "", false, WithHours(true))
		require.Error(t, err)
	})

	t.Run("applies functional options in order on a built use case", func(t *testing.T) {
		cleanup := setupTestEnv(t)
		defer cleanup()

		// Stand up a stub JIRA server so the underlying constructor's
		// http client validation passes; we don't drive any data
		// through the use case in this test.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"issues": []map[string]interface{}{}})
		}))
		defer server.Close()
		os.Setenv("JIRA_BASE_URL", server.URL)

		uc, err := NewSprintTimeAllocationUseCaseWithOptions(
			"TEST", "Sprint 1", "", false,
			WithHours(true),
			WithWorkStreams([]string{"Pricing"}),
		)
		require.NoError(t, err)
		require.NotNil(t, uc)
		assert.True(t, uc.withHours, "WithHours option should have flipped the field")
		assert.Equal(t, []string{"Pricing"}, uc.workStreams, "WithWorkStreams option should have populated the slice")
	})
}
