package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetsusecase "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase"
)

// stubSyncContributorsService records the input passed to Execute
// and returns the injected result / err.
type stubSyncContributorsService struct {
	gotInput assetsusecase.SyncContributorsInput
	result   *assetsusecase.SyncContributorsResult
	err      error
}

func (s *stubSyncContributorsService) Execute(_ context.Context, input assetsusecase.SyncContributorsInput) (*assetsusecase.SyncContributorsResult, error) {
	s.gotInput = input
	return s.result, s.err
}

func TestApp_assetsSyncContributorsAction_FactoryErrorPropagates(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("adapter boom")
	a := &App{
		syncContributorsServiceFactory: func() (SyncAssetContributorsService, error) {
			return nil, wantErr
		},
	}
	err := a.assetsSyncContributorsAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestApp_assetsSyncContributorsAction_ExecuteErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{syncContributorsService: &stubSyncContributorsService{err: errors.New("jira down")}}
	err := a.assetsSyncContributorsAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync contributors")
}

func TestApp_assetsSyncContributorsAction_DryRunUnfiltered(t *testing.T) {
	// no t.Parallel: prints to stdout
	stub := &stubSyncContributorsService{result: &assetsusecase.SyncContributorsResult{
		TotalTasks:    5,
		AssetsUpdated: 0,
		AssetsProcessed: []assetsusecase.AssetContributorSyncResult{{
			AssetName: "PaymentGateway", TasksAnalyzed: 5,
			TeamsFound:          []string{"FN"},
			CurrentContributors: []string{"alice"},
			NewContributors:     []string{"bob"},
			RemovedContributors: []string{"carol"},
		}},
	}}
	a := &App{syncContributorsService: stub}
	ctx := newContextWithFlags(t, nil, map[string]bool{"dry-run": true})
	out, err := captureStdout(t, func() error { return a.assetsSyncContributorsAction(ctx) })
	require.NoError(t, err)
	assert.True(t, stub.gotInput.DryRun)
	assert.NotContains(t, out, "Filtered sync")
	assert.Contains(t, out, "Analyzed 5 JIRA tasks")
	assert.Contains(t, out, "Processed 1 assets")
	assert.Contains(t, out, "DRY RUN - No changes were made")
	assert.Contains(t, out, "PaymentGateway")
	assert.Contains(t, out, "Teams found: FN")
	assert.Contains(t, out, "Current contributors: alice")
	assert.Contains(t, out, "New contributors: bob")
	assert.Contains(t, out, "Removed contributors: carol")
	// Updated is false + dryRun true → neither "Updated" nor "No changes needed".
	assert.NotContains(t, out, "Updated\n")
	assert.NotContains(t, out, "No changes needed")
}

func TestApp_assetsSyncContributorsAction_FilteredRealRunWithErrorsAndStatuses(t *testing.T) {
	// no t.Parallel: prints to stdout
	stub := &stubSyncContributorsService{result: &assetsusecase.SyncContributorsResult{
		TotalTasks:    10,
		AssetsUpdated: 1,
		AssetsProcessed: []assetsusecase.AssetContributorSyncResult{
			{AssetName: "PaymentGateway", TasksAnalyzed: 6, Updated: true},
			{AssetName: "Checkout", TasksAnalyzed: 4, Updated: false},
			{AssetName: "Failing", TasksAnalyzed: 0, Error: "permission denied"},
		},
		Errors: []string{"global rate limit hit"},
	}}
	a := &App{syncContributorsService: stub}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "sprint": "Alpha", "team": "Pricing", "asset": "PaymentGateway",
	}, nil)
	out, err := captureStdout(t, func() error { return a.assetsSyncContributorsAction(ctx) })
	require.NoError(t, err)
	assert.Equal(t, "FN", stub.gotInput.ProjectKey)
	assert.Equal(t, "Alpha", stub.gotInput.SprintName)
	assert.Equal(t, "Pricing", stub.gotInput.TeamName)
	assert.Equal(t, "PaymentGateway", stub.gotInput.AssetName)
	assert.Contains(t, out, "Filtered sync project:FN sprint:Alpha team:Pricing asset:PaymentGateway")
	assert.Contains(t, out, "Updated 1 assets")
	assert.Contains(t, out, "PaymentGateway")
	// Updated branch is hit.
	assert.Contains(t, out, "✅ Updated\n")
	// Not updated, real run → "No changes needed" branch.
	assert.Contains(t, out, "No changes needed")
	// Error-on-asset branch.
	assert.Contains(t, out, "❌ Error: permission denied")
	// Errors-encountered footer.
	assert.Contains(t, out, "Errors encountered:")
	assert.Contains(t, out, "global rate limit hit")
}

func TestApp_ensureSyncContributorsService_IdempotentAndUsesFactory(t *testing.T) {
	t.Parallel()
	stub := &stubSyncContributorsService{}
	calls := 0
	a := &App{
		syncContributorsServiceFactory: func() (SyncAssetContributorsService, error) {
			calls++
			return stub, nil
		},
	}
	require.NoError(t, a.ensureSyncContributorsService())
	require.Same(t, stub, a.syncContributorsService)
	require.NoError(t, a.ensureSyncContributorsService())
	assert.Equal(t, 1, calls, "factory should only run on the first call")
}
