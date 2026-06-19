package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// stubSyncTeamService is the hand-rolled in-package stub for
// SyncTeamFromJiraService. Records the projectKey passed to Execute
// and returns the injected result / err.
type stubSyncTeamService struct {
	projectKey string
	result     *configdomain.TeamSyncResult
	err        error
}

func (s *stubSyncTeamService) Execute(projectKey string) (*configdomain.TeamSyncResult, error) {
	s.projectKey = projectKey
	return s.result, s.err
}

func TestApp_configSyncTeamAction_RequiresProject(t *testing.T) {
	t.Parallel()
	a := &App{syncTeamService: &stubSyncTeamService{}}
	err := a.configSyncTeamAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project key is required")
}

func TestApp_configSyncTeamAction_FactoryErrorPropagates(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("adapter boom")
	a := &App{
		syncTeamServiceFactory: func() (SyncTeamFromJiraService, error) {
			return nil, wantErr
		},
	}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	err := a.configSyncTeamAction(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestApp_configSyncTeamAction_ExecuteErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{syncTeamService: &stubSyncTeamService{err: errors.New("jira down")}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	err := a.configSyncTeamAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync team")
}

func TestApp_configSyncTeamAction_SuccessMinimal(t *testing.T) {
	// no t.Parallel: prints to stdout
	stub := &stubSyncTeamService{result: &configdomain.TeamSyncResult{
		Source:       "jira",
		TotalMembers: 3,
	}}
	a := &App{syncTeamService: stub}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	out, err := captureStdout(t, func() error { return a.configSyncTeamAction(ctx) })
	require.NoError(t, err)
	assert.Equal(t, "FN", stub.projectKey)
	assert.Contains(t, out, "Team synchronization completed for project FN")
	assert.Contains(t, out, "Source: jira")
	assert.Contains(t, out, "Total members: 3")
	assert.NotContains(t, out, "Added members:")
	assert.NotContains(t, out, "Removed members:")
	assert.NotContains(t, out, "Warnings/Errors:")
}

func TestApp_configSyncTeamAction_SuccessWithDiffAndErrors(t *testing.T) {
	// no t.Parallel: prints to stdout
	stub := &stubSyncTeamService{result: &configdomain.TeamSyncResult{
		Source:         "jira",
		TotalMembers:   2,
		AddedMembers:   []string{"alice", "bob"},
		RemovedMembers: []string{"carol"},
		Errors: []configdomain.TeamSyncError{
			{Message: "rate limited", Type: "network"},
		},
	}}
	a := &App{syncTeamService: stub}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	out, err := captureStdout(t, func() error { return a.configSyncTeamAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Added members: alice, bob")
	assert.Contains(t, out, "Removed members: carol")
	assert.Contains(t, out, "Warnings/Errors:")
	assert.Contains(t, out, "rate limited (network)")
}

func TestApp_ensureSyncTeamService_IdempotentAndUsesFactory(t *testing.T) {
	t.Parallel()
	stub := &stubSyncTeamService{}
	calls := 0
	a := &App{
		syncTeamServiceFactory: func() (SyncTeamFromJiraService, error) {
			calls++
			return stub, nil
		},
	}
	require.NoError(t, a.ensureSyncTeamService())
	require.Same(t, stub, a.syncTeamService)
	require.NoError(t, a.ensureSyncTeamService())
	assert.Equal(t, 1, calls, "factory should only run on the first call")
}
