package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mutatedSentinel is the value the deep-copy tests below write into
// returned snapshots; the assertions then check that the original
// internal state still holds its real value. Hoisted to a constant so
// `goconst` doesn't trip over the repetition.
const mutatedSentinel = "Mutated"

// baseTeams is the minimum TeamConfig seed every test below needs: two
// projects so we can exercise both the "project exists" and "project
// missing" branches of every setter without rebuilding the world per
// subtest.
func baseTeams() map[string][]string {
	return map[string][]string{
		"FN":  {"Alice", "Bob"},
		"COP": {"Carol"},
	}
}

func TestNewTeamConfigComplete(t *testing.T) {
	t.Run("threads confluence maps through to the underlying config", func(t *testing.T) {
		tc, err := NewTeamConfigComplete(
			baseTeams(),
			map[string][]string{"FN": {"fortuna"}},
			map[string]string{"FN": "Pricing"},
			map[string]string{"FN": "Omio"},
			map[string]string{"FN": "MZN"},
			map[string]string{"FN": "12345"},
		)
		require.NoError(t, err)
		require.NotNil(t, tc)
		assert.Equal(t, "MZN", tc.GetConfluenceSpace("FN"))
		assert.Equal(t, "12345", tc.GetConfluenceParentPage("FN"))
		assert.Equal(t, "Pricing", tc.GetTribe("FN"))
	})

	t.Run("entries for unknown projects are dropped", func(t *testing.T) {
		tc, err := NewTeamConfigComplete(
			baseTeams(),
			nil, nil, nil,
			map[string]string{"GHOST": "X"},
			map[string]string{"GHOST": "1"},
		)
		require.NoError(t, err)
		assert.Equal(t, "", tc.GetConfluenceSpace("GHOST"))
		assert.Equal(t, "", tc.GetConfluenceParentPage("GHOST"))
	})

	t.Run("invalid teams arg propagates the constructor error", func(t *testing.T) {
		// An empty project key inside the teams map makes the underlying
		// NewTeamConfig fail; NewTeamConfigComplete should propagate.
		_, err := NewTeamConfigComplete(
			map[string][]string{" ": {"Alice"}},
			nil, nil, nil, nil, nil,
		)
		require.Error(t, err)
	})
}

func TestNewTeamConfigWithExcludedTypes_ExcludedTypesBranch(t *testing.T) {
	tc, err := NewTeamConfigWithExcludedTypes(
		baseTeams(),
		nil, nil, nil, nil, nil,
		map[string][]string{
			"FN":    {"Experiment"},
			"GHOST": {"X"}, // unknown project -- should be dropped
			"":      {"Y"}, // empty project key -- should be dropped
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"Experiment"}, tc.GetExcludedIssueTypes("FN"))
	assert.Empty(t, tc.GetExcludedIssueTypes("GHOST"))
}

func TestTeamConfig_Confluence_GettersAndSetters(t *testing.T) {
	t.Run("setters reject empty project key", func(t *testing.T) {
		tc, err := NewTeamConfig(baseTeams())
		require.NoError(t, err)
		require.Error(t, tc.SetConfluenceSpace("", "MZN"))
		require.Error(t, tc.SetConfluenceParentPage("", "1"))
	})

	t.Run("setters reject unknown projects", func(t *testing.T) {
		tc, err := NewTeamConfig(baseTeams())
		require.NoError(t, err)
		require.Error(t, tc.SetConfluenceSpace("GHOST", "MZN"))
		require.Error(t, tc.SetConfluenceParentPage("GHOST", "1"))
	})

	t.Run("setters persist trimmed values readable by getters", func(t *testing.T) {
		tc, err := NewTeamConfig(baseTeams())
		require.NoError(t, err)
		require.NoError(t, tc.SetConfluenceSpace(" FN ", "  MZN  "))
		require.NoError(t, tc.SetConfluenceParentPage("FN", "9876"))
		assert.Equal(t, "MZN", tc.GetConfluenceSpace("FN"))
		assert.Equal(t, "9876", tc.GetConfluenceParentPage("FN"))
		assert.Equal(t, "", tc.GetConfluenceSpace("COP"), "untouched project stays empty")
	})
}

func TestNewTeamConfigWithBoardWorkStreams(t *testing.T) {
	tc, err := NewTeamConfigWithBoardWorkStreams(
		baseTeams(),
		nil, nil, nil, nil, nil, nil,
		map[string]map[int]string{
			"FN":    {1: "Pricing", 2: "Search"},
			"GHOST": {3: "X"}, // unknown project -> dropped
			"":      {4: "Y"}, // empty key     -> dropped
			"COP":   nil,      // empty mapping -> dropped
		},
	)
	require.NoError(t, err)

	assert.Equal(t, "Pricing", tc.GetBoardWorkStream("FN", 1))
	assert.Equal(t, "Search", tc.GetBoardWorkStream("FN", 2))
	assert.Equal(t, "", tc.GetBoardWorkStream("FN", 99), "unknown board returns empty")
	assert.Equal(t, "", tc.GetBoardWorkStream("GHOST", 3), "unknown project returns empty")
	assert.Nil(t, tc.GetBoardWorkStreams("COP"), "project with no mapping returns nil map")
}

func TestTeamConfig_BoardWorkStream_GettersAndSetters(t *testing.T) {
	tc, err := NewTeamConfig(baseTeams())
	require.NoError(t, err)

	require.NoError(t, tc.SetBoardWorkStreams("FN", map[int]string{
		1: "Pricing",
		2: "pricing", // case-insensitive lookup target
		3: "Search",
	}))

	t.Run("GetBoardsForWorkStream returns IDs in a case-insensitive match", func(t *testing.T) {
		ids := tc.GetBoardsForWorkStream("FN", "PRICING")
		assert.ElementsMatch(t, []int{1, 2}, ids)
	})

	t.Run("GetBoardsForWorkStream returns nil for unknown projects", func(t *testing.T) {
		assert.Nil(t, tc.GetBoardsForWorkStream("GHOST", "Pricing"))
	})

	t.Run("GetBoardWorkStreams returns a *copy* -- mutating it doesn't leak", func(t *testing.T) {
		got := tc.GetBoardWorkStreams("FN")
		require.NotNil(t, got)
		got[1] = mutatedSentinel
		assert.Equal(t, "Pricing", tc.GetBoardWorkStream("FN", 1), "internal state should be untouched")
	})

	t.Run("SetBoardWorkStreams rejects empty / unknown projects", func(t *testing.T) {
		require.Error(t, tc.SetBoardWorkStreams("", nil))
		require.Error(t, tc.SetBoardWorkStreams("GHOST", nil))
	})
}

func TestTeamConfig_TeamTimeline_SetAndQuery(t *testing.T) {
	tc, err := NewTeamConfig(baseTeams())
	require.NoError(t, err)

	t.Run("SetTeamTimeline rejects empty / unknown projects", func(t *testing.T) {
		require.Error(t, tc.SetTeamTimeline("", nil))
		require.Error(t, tc.SetTeamTimeline("GHOST", []TeamMemberPeriod{{Member: "x"}}))
	})

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	timeline := []TeamMemberPeriod{
		{Member: "Alice", Joined: start, Left: &end},
		{Member: "Bob", Joined: start.AddDate(0, 1, 0)}, // ongoing
	}
	require.NoError(t, tc.SetTeamTimeline("FN", timeline))

	t.Run("GetAllTeamTimelines returns a snapshot whose per-project slice is independently copied", func(t *testing.T) {
		all := tc.GetAllTeamTimelines()
		require.Len(t, all, 1)
		require.Len(t, all["FN"], 2)
		all["FN"][0].Member = mutatedSentinel
		got := tc.GetTeamTimeline("FN")
		assert.Equal(t, "Alice", got[0].Member, "snapshot mutation should not leak back to internal state")
	})
}

func TestTeamConfig_ToCompleteMapWithBoardWorkStreams_DeepCopy(t *testing.T) {
	tc, err := NewTeamConfigWithBoardWorkStreams(
		baseTeams(),
		nil, nil, nil, nil, nil, nil,
		map[string]map[int]string{"FN": {1: "Pricing"}},
	)
	require.NoError(t, err)

	_, _, _, _, _, _, _, boardWS := tc.ToCompleteMapWithBoardWorkStreams()
	require.NotNil(t, boardWS["FN"])
	boardWS["FN"][1] = mutatedSentinel
	assert.Equal(t, "Pricing", tc.GetBoardWorkStream("FN", 1), "snapshot mutation should not leak back to internal state")
}

func TestProjectTeamData_Validate(t *testing.T) {
	t.Run("missing project key", func(t *testing.T) {
		err := (&ProjectTeamData{Members: []TeamMember{{DisplayName: "Alice"}}}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project key is required")
	})

	t.Run("no members", func(t *testing.T) {
		err := (&ProjectTeamData{ProjectKey: "FN"}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no team members found")
	})

	t.Run("member missing both names", func(t *testing.T) {
		err := (&ProjectTeamData{
			ProjectKey: "FN",
			Members:    []TeamMember{{DisplayName: "Alice"}, {}},
		}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "member at index 1")
	})

	t.Run("happy path", func(t *testing.T) {
		err := (&ProjectTeamData{
			ProjectKey: "FN",
			Members:    []TeamMember{{Name: "alice@x"}, {DisplayName: "Bob"}},
		}).Validate()
		assert.NoError(t, err)
	})
}
