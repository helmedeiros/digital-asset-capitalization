package usecase

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetsdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
)

// loadAssetBusinessUnits reads from the .assetcap directory under
// the current working directory. setupTestEnv (defined in
// sprint_time_allocation_test.go) already does the t.Chdir +
// teams.json scaffolding; these tests build on top of it.

func TestSprintTimeAllocationUseCase_loadAssetBusinessUnits_EmptyRepoReturnsEmpty(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	p := &SprintTimeAllocationUseCase{project: "TEST", sprint: "S1"}
	got := p.loadAssetBusinessUnits()
	assert.Empty(t, got)
}

func TestSprintTimeAllocationUseCase_loadAssetBusinessUnits_IndexesByNameAndID(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// JSONRepository expects a map[string]*domain.Asset shape, keyed by ID.
	assetWithID, err := assetsdomain.NewAsset("Payment Gateway", "desc")
	require.NoError(t, err)
	assetWithID.BusinessUnit = "BU-A"

	assetNoID, err := assetsdomain.NewAsset("Checkout", "desc")
	require.NoError(t, err)
	assetNoID.BusinessUnit = "BU-B"
	assetNoID.ID = "" // simulate ID-less entry — only Name should be indexed

	assetNoBU, err := assetsdomain.NewAsset("Profile", "desc")
	require.NoError(t, err)
	// BusinessUnit zero-value → entry should be skipped entirely

	store := map[string]*assetsdomain.Asset{
		assetWithID.ID: assetWithID,
		"no-id":        assetNoID,
		assetNoBU.ID:   assetNoBU,
	}

	data, err := json.Marshal(store)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(".assetcap", "assets.json"), data, 0644))

	p := &SprintTimeAllocationUseCase{project: "TEST", sprint: "S1"}
	got := p.loadAssetBusinessUnits()

	// assetWithID indexed by both name and ID.
	assert.Equal(t, "BU-A", got["Payment Gateway"])
	assert.Equal(t, "BU-A", got[assetWithID.ID])
	// assetNoID indexed by name only.
	assert.Equal(t, "BU-B", got["Checkout"])
	// assetNoBU has no BusinessUnit → not indexed at all.
	_, hasNoBU := got["Profile"]
	assert.False(t, hasNoBU, "asset without BusinessUnit should be skipped")
}

// parseManualAdjustments

func TestSprintTimeAllocationUseCase_parseManualAdjustments_EmptyReturnsNil(t *testing.T) {
	t.Parallel()
	p := &SprintTimeAllocationUseCase{override: ""}
	adj, err := p.parseManualAdjustments()
	require.NoError(t, err)
	assert.Nil(t, adj)
}

func TestSprintTimeAllocationUseCase_parseManualAdjustments_InvalidJSONWraps(t *testing.T) {
	t.Parallel()
	p := &SprintTimeAllocationUseCase{override: "not-json"}
	_, err := p.parseManualAdjustments()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing manual adjustments JSON")
}

func TestSprintTimeAllocationUseCase_parseManualAdjustments_HappyPath(t *testing.T) {
	t.Parallel()
	p := &SprintTimeAllocationUseCase{override: `{"TEST-1": 8.5, "TEST-2": 4}`}
	adj, err := p.parseManualAdjustments()
	require.NoError(t, err)
	assert.InDelta(t, 8.5, adj["TEST-1"], 1e-9)
	assert.InDelta(t, 4.0, adj["TEST-2"], 1e-9)
}

// JiraDoer is the public entry point. The success path requires the
// full env-var + JIRA wiring, but the failure path can be triggered
// by leaving the env unset — NewSprintTimeAllocationUseCase will
// fail to construct the config and JiraDoer wraps the err.

func TestJiraDoer_NewUseCaseErrorPropagates(t *testing.T) {
	// no t.Parallel: mutates env via t.Setenv
	t.Setenv("JIRA_BASE_URL", "")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("JIRA_TOKEN", "")
	// Chdir somewhere without a .assetcap so loading falls through to
	// a missing-config error.
	t.Chdir(t.TempDir())

	_, err := JiraDoer("TEST", "Sprint 1", "")
	require.Error(t, err)
}

// getTeamKeyForAssignee

func TestSprintTimeAllocationUseCase_getTeamKeyForAssignee_FallbackToProject(t *testing.T) {
	t.Parallel()
	p := &SprintTimeAllocationUseCase{
		project: "TEST",
		teams:   domain.TeamMap{"OTHER": domain.Team{Team: []string{"someone"}}},
	}
	assert.Equal(t, "TEST", p.getTeamKeyForAssignee("not-a-team-member"))
}

func TestSprintTimeAllocationUseCase_getTeamKeyForAssignee_MatchedTeamReturns(t *testing.T) {
	t.Parallel()
	p := &SprintTimeAllocationUseCase{
		project: "TEST",
		teams: domain.TeamMap{
			"TEAM-A": domain.Team{Team: []string{"alice"}},
			"TEAM-B": domain.Team{Team: []string{"bob"}},
		},
	}
	assert.Equal(t, "TEAM-A", p.getTeamKeyForAssignee("alice"))
	assert.Equal(t, "TEAM-B", p.getTeamKeyForAssignee("bob"))
}
