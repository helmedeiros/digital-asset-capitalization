package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetsdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// TestAssetServiceAdapter_EnrichAsset_FieldSwitch fills in the
// post-enrichment branches of EnrichAsset that the existing tests
// don't reach: each of the five supported field values needs to land
// in enriched_content from the corresponding asset field. Without this
// the switch arms (why/benefits/how/metrics) sit uncovered even when
// the description happy path passes.
func TestAssetServiceAdapter_EnrichAsset_FieldSwitch(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		field   string
		applyTo func(a *assetsdomain.Asset)
		want    string
	}{
		{"description", func(a *assetsdomain.Asset) { a.Description = "Updated desc" }, "Updated desc"},
		{"why", func(a *assetsdomain.Asset) { a.Why = "Updated why" }, "Updated why"},
		{"benefits", func(a *assetsdomain.Asset) { a.Benefits = "Updated benefits" }, "Updated benefits"},
		{"how", func(a *assetsdomain.Asset) { a.How = "Updated how" }, "Updated how"},
		{"metrics", func(a *assetsdomain.Asset) { a.Metrics = "Updated metrics" }, "Updated metrics"},
	}

	for _, c := range cases {
		t.Run(c.field, func(t *testing.T) {
			mock := &MockAssetService{}
			adapter := &AssetServiceAdapter{service: mock}

			asset, err := assetsdomain.NewAsset("Payment Gateway", "original")
			require.NoError(t, err)
			c.applyTo(asset)

			mock.On("EnrichAsset", "Payment Gateway", c.field).Return(nil)
			mock.On("GetAsset", "Payment Gateway").Return(asset, nil)

			got, err := adapter.EnrichAsset(ctx, "Payment Gateway", c.field)
			require.NoError(t, err)
			m, ok := got.(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "enriched", m["status"])
			assert.Equal(t, c.want, m["enriched_content"], "switch arm for field %q should pull from the matching asset field", c.field)
		})
	}
}

// TestAssetServiceAdapter_EnrichAsset_GetAssetFallback exercises the
// path where the enrichment itself succeeds but the follow-up
// GetAsset errors -- the adapter is documented to fall back to a
// short map[string]string without an enriched_content key.
func TestAssetServiceAdapter_EnrichAsset_GetAssetFallback(t *testing.T) {
	mock := &MockAssetService{}
	mock.On("EnrichAsset", "Payment Gateway", "description").Return(nil)
	mock.On("GetAsset", "Payment Gateway").Return(nil, errors.New("read failed"))

	adapter := &AssetServiceAdapter{service: mock}
	got, err := adapter.EnrichAsset(context.Background(), "Payment Gateway", "description")
	require.NoError(t, err)
	m, ok := got.(map[string]string)
	require.True(t, ok, "fallback branch should return map[string]string")
	assert.Equal(t, "enriched", m["status"])
	_, hasContent := m["enriched_content"]
	assert.False(t, hasContent, "fallback branch should not carry enriched_content")
}

func TestAssetServiceAdapter_EnrichAsset_UnsupportedField(t *testing.T) {
	adapter := &AssetServiceAdapter{service: &MockAssetService{}}
	_, err := adapter.EnrichAsset(context.Background(), "Payment Gateway", "totally-not-a-field")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported field")
}
