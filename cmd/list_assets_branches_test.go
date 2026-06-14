package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetsdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// TestAssetServiceAdapter_ListAssets_OptionalFields fills in the
// team-fields branches that the bare 'successful list with assets'
// subtest in console_test.go doesn't reach: owning_team and
// contributing_teams. Without it those map[string]interface{}
// assignments stay uncovered.
func TestAssetServiceAdapter_ListAssets_OptionalFields(t *testing.T) {
	mock := &MockAssetService{}
	adapter := &AssetServiceAdapter{service: mock}
	ctx := context.Background()

	withTeams, err := assetsdomain.NewAsset("Payment Gateway", "Processes payments")
	require.NoError(t, err)
	require.NoError(t, withTeams.SetOwningTeam("payments-team"))
	require.NoError(t, withTeams.AddContributingTeam("infra-team"))

	withoutTeams, err := assetsdomain.NewAsset("Search Service", "Search backend")
	require.NoError(t, err)

	mock.On("ListAssets").Return([]*assetsdomain.Asset{withTeams, withoutTeams}, nil)

	got, err := adapter.ListAssets(ctx)
	require.NoError(t, err)
	result, ok := got.([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, result, 2)

	// First entry carries both team keys.
	assert.Equal(t, "payments-team", result[0]["owning_team"])
	contribs, ok := result[0]["contributing_teams"].([]string)
	require.True(t, ok)
	assert.ElementsMatch(t, []string{"infra-team"}, contribs)

	// Second entry's team keys are omitted entirely.
	_, hasOwning := result[1]["owning_team"]
	_, hasContrib := result[1]["contributing_teams"]
	assert.False(t, hasOwning, "asset without an owning team should not carry an owning_team key")
	assert.False(t, hasContrib, "asset without contributing teams should not carry a contributing_teams key")

	mock.AssertExpectations(t)
}
