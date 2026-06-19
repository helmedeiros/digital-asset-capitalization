package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// stubAssetServiceForConfluence covers the 3 Confluence-side
// AssetService methods consumed by sync/publish/update-confluence.
// Embeds unimplementedAssetService so unexpected method calls panic.
type stubAssetServiceForConfluence struct {
	unimplementedAssetService

	syncResult    *assetdomain.SyncResult
	syncErr       error
	publishResult *assetsapp.PublishToConfluenceResult
	publishErr    error
	updateResult  *assetsapp.PublishToConfluenceResult
	updateErr     error
}

func (s *stubAssetServiceForConfluence) SyncFromConfluence(string, string, bool) (*assetdomain.SyncResult, error) {
	return s.syncResult, s.syncErr
}
func (s *stubAssetServiceForConfluence) PublishToConfluence(context.Context, string, string, string, bool, bool) (*assetsapp.PublishToConfluenceResult, error) {
	return s.publishResult, s.publishErr
}
func (s *stubAssetServiceForConfluence) UpdateConfluencePage(context.Context, string, bool, bool) (*assetsapp.PublishToConfluenceResult, error) {
	return s.updateResult, s.updateErr
}

// sync

func TestApp_assetsSyncAction_NoAssetsLabelIsInformationalSuccess(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForConfluence{
		syncErr: errors.New("no assets found with label cap-asset"),
	}}
	ctx := newContextWithFlags(t, map[string]string{"space": "MZN", "label": "cap-asset"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsSyncAction(ctx) })
	require.NoError(t, err, "this specific error swallows to a no-op success")
	assert.Contains(t, out, "no assets found with label")
}

func TestApp_assetsSyncAction_GenericErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForConfluence{syncErr: errors.New("auth failed")}}
	ctx := newContextWithFlags(t, map[string]string{"label": "cap-asset"}, nil)
	err := a.assetsSyncAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "auth failed", err.Error())
}

func TestApp_assetsSyncAction_AllAssetsSyncedRendersCount(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForConfluence{
		syncResult: &assetdomain.SyncResult{
			SyncedAssets: []*assetdomain.Asset{{Name: "Search"}, {Name: "Payments"}},
		},
	}}
	ctx := newContextWithFlags(t, map[string]string{"label": "cap-asset"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsSyncAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Successfully synced 2/2 assets")
	assert.NotContains(t, out, "could not be synced")
}

func TestApp_assetsSyncAction_PartialSyncRendersNotSyncedDetails(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForConfluence{
		syncResult: &assetdomain.SyncResult{
			SyncedAssets: []*assetdomain.Asset{{Name: "Search"}},
			NotSyncedAssets: []*assetdomain.NotSyncedAsset{
				{
					Name:            "Broken",
					MissingFields:   []string{"Description", "Status"},
					AvailableFields: map[string]string{"Name": "Broken", "Why": "experimental"},
				},
			},
		},
	}}
	ctx := newContextWithFlags(t, map[string]string{"label": "cap-asset"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsSyncAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Successfully synced 1/2 assets")
	assert.Contains(t, out, "1 assets could not be synced")
	assert.Contains(t, out, "Broken")
	assert.Contains(t, out, "Description, Status")
	assert.Contains(t, out, "Why: experimental")
}

// publish

func TestApp_assetsPublishAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForConfluence{publishErr: errors.New("403")}}
	ctx := newContextWithFlags(t,
		map[string]string{"name": "Search", "space": "MZN"},
		nil,
	)
	err := a.assetsPublishAction(ctx)
	require.Error(t, err)
}

func TestApp_assetsPublishAction_DryRunRendersPreview(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForConfluence{
		publishResult: &assetsapp.PublishToConfluenceResult{
			AssetName: "Search",
			SpaceKey:  "MZN",
			Labels:    []string{"cap-asset"},
			Preview:   "<p>preview body</p>",
		},
	}}
	ctx := newContextWithFlags(t,
		map[string]string{"name": "Search", "space": "MZN"},
		map[string]bool{"dry-run": true},
	)
	out, err := captureStdout(t, func() error { return a.assetsPublishAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "DRY RUN")
	assert.Contains(t, out, "<p>preview body</p>")
	assert.Contains(t, out, "cap-asset")
}

func TestApp_assetsPublishAction_LiveSuccessRendersURLAndDocLink(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForConfluence{
		publishResult: &assetsapp.PublishToConfluenceResult{
			AssetName:    "Search",
			SpaceKey:     "MZN",
			PageID:       "12345",
			PageURL:      "https://example/wiki/12345",
			Labels:       []string{"cap-asset"},
			DocLinkSaved: true,
		},
	}}
	ctx := newContextWithFlags(t,
		map[string]string{"name": "Search", "space": "MZN"},
		nil,
	)
	out, err := captureStdout(t, func() error { return a.assetsPublishAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Successfully published asset 'Search' to Confluence")
	assert.Contains(t, out, "Page ID: 12345")
	assert.Contains(t, out, "https://example/wiki/12345")
	assert.Contains(t, out, "DocLink updated in asset")
}

// update-confluence

func TestApp_assetsUpdateConfluenceAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForConfluence{updateErr: errors.New("conflict")}}
	ctx := newContextWithFlags(t, map[string]string{"name": "Search"}, nil)
	err := a.assetsUpdateConfluenceAction(ctx)
	require.Error(t, err)
}

func TestApp_assetsUpdateConfluenceAction_DryRunRendersPreview(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForConfluence{
		updateResult: &assetsapp.PublishToConfluenceResult{
			AssetName: "Search",
			SpaceKey:  "MZN",
			PageID:    "12345",
			Preview:   "<p>preview body</p>",
		},
	}}
	ctx := newContextWithFlags(t, map[string]string{"name": "Search"}, map[string]bool{"dry-run": true})
	out, err := captureStdout(t, func() error { return a.assetsUpdateConfluenceAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "DRY RUN: Would update page for asset 'Search'")
	assert.Contains(t, out, "<p>preview body</p>")
}

func TestApp_assetsUpdateConfluenceAction_LiveSuccessRendersURL(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForConfluence{
		updateResult: &assetsapp.PublishToConfluenceResult{
			AssetName: "Search",
			SpaceKey:  "MZN",
			PageID:    "12345",
			PageURL:   "https://example/wiki/12345",
		},
	}}
	ctx := newContextWithFlags(t, map[string]string{"name": "Search"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsUpdateConfluenceAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Successfully updated Confluence page for asset 'Search'")
	assert.Contains(t, out, "Page ID: 12345")
	assert.Contains(t, out, "https://example/wiki/12345")
}
