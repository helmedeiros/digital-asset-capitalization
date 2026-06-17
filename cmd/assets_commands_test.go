package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// unimplementedAssetService satisfies the full AssetService interface
// by panicking on every method. Embed it to opt-in only specific
// methods. (Duplicated in tasks_commands_test.go on the tasks branch —
// the two will collapse to one definition once both PRs land.)
type unimplementedAssetService struct{}

func (unimplementedAssetService) CreateAsset(string, string) error            { panic("not used") }
func (unimplementedAssetService) ListAssets() ([]*assetdomain.Asset, error)   { panic("not used") }
func (unimplementedAssetService) GetAsset(string) (*assetdomain.Asset, error) { panic("not used") }
func (unimplementedAssetService) DeleteAsset(string, bool) error              { panic("not used") }
func (unimplementedAssetService) UpdateAsset(string, string, string, string, string, string) error {
	panic("not used")
}
func (unimplementedAssetService) UpdateDocumentation(string) error { panic("not used") }
func (unimplementedAssetService) IncrementTaskCount(string) error  { panic("not used") }
func (unimplementedAssetService) DecrementTaskCount(string) error  { panic("not used") }
func (unimplementedAssetService) SyncFromConfluence(string, string, bool) (*assetdomain.SyncResult, error) {
	panic("not used")
}
func (unimplementedAssetService) EnrichAsset(string, string) error          { panic("not used") }
func (unimplementedAssetService) GenerateKeywords(string) error             { panic("not used") }
func (unimplementedAssetService) AssignTeam(string, string, []string) error { panic("not used") }
func (unimplementedAssetService) GetAssetTeams() ([]assetsapp.AssetTeamInfo, error) {
	panic("not used")
}
func (unimplementedAssetService) GetAssetTeamInfo(string) (*assetsapp.AssetTeamInfo, error) {
	panic("not used")
}
func (unimplementedAssetService) AddContributingTeam(string, string) error    { panic("not used") }
func (unimplementedAssetService) RemoveContributingTeam(string, string) error { panic("not used") }
func (unimplementedAssetService) PublishToConfluence(context.Context, string, string, string, bool, bool) (*assetsapp.PublishToConfluenceResult, error) {
	panic("not used")
}
func (unimplementedAssetService) UpdateConfluencePage(context.Context, string, bool, bool) (*assetsapp.PublishToConfluenceResult, error) {
	panic("not used")
}

// stubAssetServiceForActions provides a small stub of the AssetService
// surface used by the simple Actions extracted in this PR. Embeds the
// unimplementedAssetService (defined in tasks_commands_test.go) so a
// future Action that calls a different method panics rather than
// silently no-op'ing.
type stubAssetServiceForActions struct {
	unimplementedAssetService
	createErr   error
	deleteErr   error
	listAssets  []*assetdomain.Asset
	listErr     error
	updateErr   error
	asset       *assetdomain.Asset
	assetErr    error
	enrichErr   error
	keywordsErr error
}

func (s *stubAssetServiceForActions) CreateAsset(string, string) error {
	return s.createErr
}
func (s *stubAssetServiceForActions) DeleteAsset(string, bool) error {
	return s.deleteErr
}
func (s *stubAssetServiceForActions) ListAssets() ([]*assetdomain.Asset, error) {
	return s.listAssets, s.listErr
}
func (s *stubAssetServiceForActions) GetAsset(string) (*assetdomain.Asset, error) {
	return s.asset, s.assetErr
}
func (s *stubAssetServiceForActions) UpdateAsset(string, string, string, string, string, string) error {
	return s.updateErr
}
func (s *stubAssetServiceForActions) EnrichAsset(string, string) error {
	return s.enrichErr
}
func (s *stubAssetServiceForActions) GenerateKeywords(string) error {
	return s.keywordsErr
}

func TestApp_assetsCreateAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForActions{createErr: errors.New("dup")}}
	ctx := newContextWithFlags(t, map[string]string{"name": "Search", "description": "x"}, nil)
	err := a.assetsCreateAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "dup", err.Error())
}

func TestApp_assetsCreateAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForActions{}}
	ctx := newContextWithFlags(t, map[string]string{"name": "Search", "description": "x"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsCreateAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Created asset: Search")
}

func TestApp_assetsDeleteAction_WithoutPage(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForActions{}}
	ctx := newContextWithFlags(t,
		map[string]string{"name": "Search"},
		map[string]bool{"delete-page": false},
	)
	out, err := captureStdout(t, func() error { return a.assetsDeleteAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Deleted asset: Search")
	assert.NotContains(t, out, "Confluence page also deleted")
}

func TestApp_assetsDeleteAction_WithPage(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForActions{}}
	ctx := newContextWithFlags(t,
		map[string]string{"name": "Search"},
		map[string]bool{"delete-page": true},
	)
	out, err := captureStdout(t, func() error { return a.assetsDeleteAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Confluence page also deleted")
}

func TestApp_assetsDeleteAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForActions{deleteErr: errors.New("not found")}}
	ctx := newContextWithFlags(t,
		map[string]string{"name": "Search"},
		map[string]bool{"delete-page": false},
	)
	err := a.assetsDeleteAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "not found", err.Error())
}

func TestApp_assetsListAction_Empty(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForActions{}}
	ctx := newContextWithFlags(t, nil, nil)
	out, err := captureStdout(t, func() error { return a.assetsListAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "No assets found")
}

func TestApp_assetsListAction_RendersAssets(t *testing.T) {
	// no t.Parallel: prints to stdout
	now := time.Now()
	stub := &stubAssetServiceForActions{
		listAssets: []*assetdomain.Asset{
			{
				Name:              "Search",
				Description:       "Search subsystem",
				DocLink:           "https://example/wiki/search",
				OwningTeam:        "Voyager",
				ContributingTeams: []string{"Catalog", "Indexer"},
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
	}
	a := &App{assetService: stub}
	out, err := captureStdout(t, func() error { return a.assetsListAction(newContextWithFlags(t, nil, nil)) })
	require.NoError(t, err)
	assert.Contains(t, out, "Search")
	assert.Contains(t, out, "Owner: Voyager")
	assert.Contains(t, out, "Catalog, Indexer")
	assert.Contains(t, out, "https://example/wiki/search")
}

func TestApp_assetsListAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForActions{listErr: errors.New("disk")}}
	ctx := newContextWithFlags(t, nil, nil)
	err := a.assetsListAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "disk", err.Error())
}

func TestApp_assetsUpdateAction_PassesAllFields(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForActions{}}
	ctx := newContextWithFlags(t, map[string]string{
		"name":        "Search",
		"description": "d",
		"why":         "w",
		"benefits":    "b",
		"how":         "h",
		"metrics":     "m",
	}, nil)
	out, err := captureStdout(t, func() error { return a.assetsUpdateAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Updated asset: Search")
}

func TestApp_assetsUpdateAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForActions{updateErr: errors.New("validation")}}
	ctx := newContextWithFlags(t, map[string]string{
		"name": "Search", "description": "d", "why": "w", "benefits": "b", "how": "h", "metrics": "m",
	}, nil)
	err := a.assetsUpdateAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "validation", err.Error())
}

func TestApp_assetsShowAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForActions{assetErr: errors.New("not found")}}
	ctx := newContextWithFlags(t, map[string]string{"name": "Search"}, nil)
	err := a.assetsShowAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "not found", err.Error())
}

func TestApp_assetsShowAction_RendersAllFields(t *testing.T) {
	// no t.Parallel: prints to stdout
	now := time.Now()
	a := &App{assetService: &stubAssetServiceForActions{
		asset: &assetdomain.Asset{
			Name: "Search", Description: "d", Why: "w", Benefits: "b", How: "h", Metrics: "m",
			DocLink: "https://example.invalid/x", OwningTeam: "Voyager",
			ContributingTeams: []string{"Catalog"},
			Keywords:          []string{"k1", "k2"},
			CreatedAt:         now, UpdatedAt: now,
		},
	}}
	ctx := newContextWithFlags(t, map[string]string{"name": "Search"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Asset: Search")
	assert.Contains(t, out, "Keywords: k1, k2")
	assert.Contains(t, out, "Owner: Voyager")
	assert.Contains(t, out, "Contributors: Catalog")
}

func TestApp_assetsEnrichAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForActions{enrichErr: errors.New("ollama down")}}
	ctx := newContextWithFlags(t, map[string]string{"name": "Search", "field": "description"}, nil)
	err := a.assetsEnrichAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "ollama down", err.Error())
}

func TestApp_assetsEnrichAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForActions{}}
	ctx := newContextWithFlags(t, map[string]string{"name": "Search", "field": "description"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsEnrichAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Enriched description field for asset: Search")
}

func TestApp_assetsKeywordsAction_AssetNotFound(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForActions{assetErr: errors.New("missing")}}
	ctx := newContextWithFlags(t, map[string]string{"name": "Search"}, nil)
	err := a.assetsKeywordsAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "asset not found: Search")
}

func TestApp_assetsKeywordsAction_GenerateErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForActions{
		asset:       &assetdomain.Asset{Name: "Search"},
		keywordsErr: errors.New("llm boom"),
	}}
	ctx := newContextWithFlags(t, map[string]string{"name": "Search"}, nil)
	err := a.assetsKeywordsAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "llm boom", err.Error())
}

func TestApp_assetsKeywordsAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForActions{asset: &assetdomain.Asset{Name: "Search"}}}
	ctx := newContextWithFlags(t, map[string]string{"name": "Search"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsKeywordsAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Generated keywords for asset: Search")
}
