package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetsdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// TestAssetServiceAdapter_GetAsset_OptionalFields fills in the optional
// branches in GetAsset that the bare 'successful get' subtest in
// console_test.go does not exercise: why/benefits/how/metrics,
// keywords, owning team, contributing teams. Without it the optional
// branches sit uncovered even when the happy path passes.
func TestAssetServiceAdapter_GetAsset_OptionalFields(t *testing.T) {
	mock := &MockAssetService{}
	adapter := &AssetServiceAdapter{service: mock}
	ctx := context.Background()

	asset, err := assetsdomain.NewAsset("Payment Gateway", "Processes payments")
	require.NoError(t, err)

	// Fill in the optional fields directly. NewAsset alone sets only
	// the required ones, so each optional branch below is otherwise
	// untestable through the constructor.
	asset.Why = "Why text"
	asset.Benefits = "Benefits text"
	asset.How = "How text"
	asset.Metrics = "Metrics text"
	asset.Keywords = []string{"payment", "gateway"}
	asset.LaunchDate = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	asset.LastDocUpdateAt = time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, asset.SetOwningTeam("payments-team"))
	require.NoError(t, asset.AddContributingTeam("infra-team"))

	mock.On("GetAsset", "Payment Gateway").Return(asset, nil)

	got, err := adapter.GetAsset(ctx, "Payment Gateway")
	require.NoError(t, err)
	m, ok := got.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "Why text", m["why"])
	assert.Equal(t, "Benefits text", m["benefits"])
	assert.Equal(t, "How text", m["how"])
	assert.Equal(t, "Metrics text", m["metrics"])
	assert.Equal(t, []string{"payment", "gateway"}, m["keywords"])
	assert.Equal(t, "payments-team", m["owning_team"])
	contribs, ok := m["contributing_teams"].([]string)
	require.True(t, ok)
	assert.ElementsMatch(t, []string{"infra-team"}, contribs)

	mock.AssertExpectations(t)
}
