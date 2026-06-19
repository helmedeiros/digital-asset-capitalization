package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// stubAssetServiceForNested extends stubAssetServiceForActions with
// the methods called by the documentation/tasks/teams subcommand
// trees. We keep it separate so the existing PR 4 stub stays scoped
// to the simple Actions.
type stubAssetServiceForNested struct {
	unimplementedAssetService

	asset    *assetdomain.Asset
	assetErr error

	updateDocErr error
	incErr       error
	decErr       error

	assignErr error

	listTeams    []assetsapp.AssetTeamInfo
	listTeamsErr error

	teamInfo    *assetsapp.AssetTeamInfo
	teamInfoErr error

	addErr error
	rmErr  error
}

func (s *stubAssetServiceForNested) GetAsset(string) (*assetdomain.Asset, error) {
	return s.asset, s.assetErr
}
func (s *stubAssetServiceForNested) UpdateDocumentation(string) error { return s.updateDocErr }
func (s *stubAssetServiceForNested) IncrementTaskCount(string) error  { return s.incErr }
func (s *stubAssetServiceForNested) DecrementTaskCount(string) error  { return s.decErr }
func (s *stubAssetServiceForNested) AssignTeam(string, string, []string) error {
	return s.assignErr
}
func (s *stubAssetServiceForNested) GetAssetTeams() ([]assetsapp.AssetTeamInfo, error) {
	return s.listTeams, s.listTeamsErr
}
func (s *stubAssetServiceForNested) GetAssetTeamInfo(string) (*assetsapp.AssetTeamInfo, error) {
	return s.teamInfo, s.teamInfoErr
}
func (s *stubAssetServiceForNested) AddContributingTeam(string, string) error {
	return s.addErr
}
func (s *stubAssetServiceForNested) RemoveContributingTeam(string, string) error {
	return s.rmErr
}

// documentation/update

func TestApp_assetsDocUpdateAction_AssetNotFound(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForNested{assetErr: errors.New("nope")}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search"}, nil)
	err := a.assetsDocUpdateAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "nope", err.Error())
}

func TestApp_assetsDocUpdateAction_UpdateErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForNested{
		asset:        &assetdomain.Asset{Name: "Search"},
		updateDocErr: errors.New("disk"),
	}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search"}, nil)
	err := a.assetsDocUpdateAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "disk", err.Error())
}

func TestApp_assetsDocUpdateAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForNested{asset: &assetdomain.Asset{Name: "Search"}}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsDocUpdateAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Marked documentation as updated for asset Search")
}

// tasks/increment

func TestApp_assetsTasksIncrementAction_AssetNotFound(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForNested{assetErr: errors.New("nope")}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search"}, nil)
	err := a.assetsTasksIncrementAction(ctx)
	require.Error(t, err)
}

func TestApp_assetsTasksIncrementAction_IncrementErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForNested{
		asset:  &assetdomain.Asset{Name: "Search"},
		incErr: errors.New("overflow"),
	}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search"}, nil)
	err := a.assetsTasksIncrementAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "overflow", err.Error())
}

func TestApp_assetsTasksIncrementAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForNested{asset: &assetdomain.Asset{Name: "Search"}}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsTasksIncrementAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Incremented task count for asset Search")
}

// tasks/decrement

func TestApp_assetsTasksDecrementAction_AssetNotFound(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForNested{assetErr: errors.New("nope")}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search"}, nil)
	err := a.assetsTasksDecrementAction(ctx)
	require.Error(t, err)
}

func TestApp_assetsTasksDecrementAction_DecrementErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForNested{
		asset:  &assetdomain.Asset{Name: "Search"},
		decErr: errors.New("underflow"),
	}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search"}, nil)
	err := a.assetsTasksDecrementAction(ctx)
	require.Error(t, err)
}

func TestApp_assetsTasksDecrementAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForNested{asset: &assetdomain.Asset{Name: "Search"}}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsTasksDecrementAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Decremented task count for asset Search")
}

// teams/assign

func TestApp_assetsTeamsAssignAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForNested{assignErr: errors.New("conflict")}}
	ctx := newContextWithFlags(t,
		map[string]string{"asset": "Search", "owner": "Voyager", "contributors": "Catalog,Indexer"},
		nil,
	)
	err := a.assetsTeamsAssignAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "conflict", err.Error())
}

func TestApp_assetsTeamsAssignAction_PrintsContributors(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForNested{}}
	ctx := newContextWithFlags(t,
		map[string]string{"asset": "Search", "owner": "Voyager", "contributors": "Catalog,Indexer"},
		nil,
	)
	out, err := captureStdout(t, func() error { return a.assetsTeamsAssignAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "assigned teams to asset 'Search'")
	assert.Contains(t, out, "Owner: Voyager")
	assert.Contains(t, out, "Catalog, Indexer")
}

func TestApp_assetsTeamsAssignAction_OwnerOnlyNoContributors(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForNested{}}
	ctx := newContextWithFlags(t,
		map[string]string{"asset": "Search", "owner": "Voyager"},
		nil,
	)
	out, err := captureStdout(t, func() error { return a.assetsTeamsAssignAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Owner: Voyager")
	assert.NotContains(t, out, "Contributors:")
}

// teams/list

func TestApp_assetsTeamsListAction_EmptyList(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForNested{}}
	out, err := captureStdout(t, func() error { return a.assetsTeamsListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "No team assignments found")
}

func TestApp_assetsTeamsListAction_RendersAssignments(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForNested{
		listTeams: []assetsapp.AssetTeamInfo{
			{AssetName: "Search", OwningTeam: "Voyager", ContributingTeams: []string{"Catalog"}},
			{AssetName: "Payments", OwningTeam: "Pay"},
		},
	}}
	out, err := captureStdout(t, func() error { return a.assetsTeamsListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "Search")
	assert.Contains(t, out, "Payments")
	assert.Contains(t, out, "Voyager")
}

func TestApp_assetsTeamsListAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForNested{listTeamsErr: errors.New("disk")}}
	err := a.assetsTeamsListAction(nil)
	require.Error(t, err)
}

// teams/show

func TestApp_assetsTeamsShowAction_NotAssigned(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForNested{
		teamInfo: &assetsapp.AssetTeamInfo{AssetName: "Search"},
	}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsTeamsShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Owner: Not assigned")
	assert.Contains(t, out, "Contributors: None")
}

func TestApp_assetsTeamsShowAction_RendersBoth(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForNested{
		teamInfo: &assetsapp.AssetTeamInfo{
			AssetName:         "Search",
			OwningTeam:        "Voyager",
			ContributingTeams: []string{"Catalog", "Indexer"},
		},
	}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsTeamsShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Owner: Voyager")
	assert.Contains(t, out, "Catalog, Indexer")
}

func TestApp_assetsTeamsShowAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForNested{teamInfoErr: errors.New("missing")}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search"}, nil)
	err := a.assetsTeamsShowAction(ctx)
	require.Error(t, err)
}

// teams/add-contributor

func TestApp_assetsTeamsAddContributorAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForNested{addErr: errors.New("dup")}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search", "team": "Catalog"}, nil)
	err := a.assetsTeamsAddContributorAction(ctx)
	require.Error(t, err)
}

func TestApp_assetsTeamsAddContributorAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForNested{}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search", "team": "Catalog"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsTeamsAddContributorAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Added 'Catalog' as contributor to asset 'Search'")
}

// teams/remove-contributor

func TestApp_assetsTeamsRemoveContributorAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForNested{rmErr: errors.New("not present")}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search", "team": "Catalog"}, nil)
	err := a.assetsTeamsRemoveContributorAction(ctx)
	require.Error(t, err)
}

func TestApp_assetsTeamsRemoveContributorAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{assetService: &stubAssetServiceForNested{}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Search", "team": "Catalog"}, nil)
	out, err := captureStdout(t, func() error { return a.assetsTeamsRemoveContributorAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Removed 'Catalog' as contributor from asset 'Search'")
}
