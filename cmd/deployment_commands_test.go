package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// newContextWithSliceFlags extends newContextWithFlags (helpers_test.go)
// with a StringSlice flag, used by the deployments record action for
// --tasks.
func newContextWithSliceFlags(t *testing.T, strFlags map[string]string, boolFlags map[string]bool, sliceFlags map[string][]string) *cli.Context {
	t.Helper()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	for k, v := range strFlags {
		set.String(k, v, "")
	}
	for k, v := range boolFlags {
		set.Bool(k, v, "")
	}
	for k, v := range sliceFlags {
		s := cli.NewStringSlice(v...)
		set.Var(s, k, "")
	}
	return cli.NewContext(nil, set, nil)
}

func TestApp_deploymentsRecordAction_InvalidEnvironment(t *testing.T) {
	t.Parallel()
	a := &App{}
	a.ensureDeploymentService() // wire so action can find service
	ctx := newContextWithSliceFlags(t,
		map[string]string{"env": "notarealenv", "version": "1.0", "deployed-by": "ci"},
		nil,
		map[string][]string{"tasks": {"FN-1"}},
	)
	err := a.deploymentsRecordAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid environment")
}

func TestApp_deploymentsTimelineAction_InvalidFromDate(t *testing.T) {
	t.Parallel()
	a := &App{}
	a.ensureDeploymentService()
	ctx := newContextWithFlags(t,
		map[string]string{"from": "not-a-date", "to": "2026-01-31"},
		nil,
	)
	err := a.deploymentsTimelineAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid from date")
}

func TestApp_deploymentsTimelineAction_InvalidToDate(t *testing.T) {
	t.Parallel()
	a := &App{}
	a.ensureDeploymentService()
	ctx := newContextWithFlags(t,
		map[string]string{"from": "2026-01-01", "to": "not-a-date"},
		nil,
	)
	err := a.deploymentsTimelineAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid to date")
}

func TestApp_deploymentsMockAction_InvalidFromDate(t *testing.T) {
	t.Parallel()
	a := &App{}
	a.ensureDeploymentService()
	ctx := newContextWithFlags(t,
		map[string]string{"from": "not-a-date"},
		map[string]bool{"sample-file": false},
	)
	err := a.deploymentsMockAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid from date")
}

func TestApp_deploymentsMockAction_InvalidToDate(t *testing.T) {
	t.Parallel()
	a := &App{}
	a.ensureDeploymentService()
	ctx := newContextWithFlags(t,
		map[string]string{"to": "not-a-date"},
		map[string]bool{"sample-file": false},
	)
	err := a.deploymentsMockAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid to date")
}

// TestApp_deploymentsMockAction_SampleFile drives the sample-file
// branch which doesn't depend on the deployment service. Uses
// t.Chdir to keep the generated file under TempDir so a parallel test
// run doesn't write into the repo root.
func TestApp_deploymentsMockAction_SampleFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	a := &App{}
	a.ensureDeploymentService()
	ctx := newContextWithFlags(t,
		nil,
		map[string]bool{"sample-file": true},
	)
	err := a.deploymentsMockAction(ctx)
	require.NoError(t, err)
	// The action writes "mock_deployments.json" relative to cwd.
	_, statErr := assertFileExists(t, filepath.Join(dir, "mock_deployments.json"))
	require.NoError(t, statErr)
}

func TestApp_ensureDeploymentService_LazyInit(t *testing.T) {
	t.Parallel()
	a := &App{}
	assert.Nil(t, a.deploymentService)
	assert.Nil(t, a.deploymentRepo)
	a.ensureDeploymentService()
	assert.NotNil(t, a.deploymentService, "first call should wire the service")
	assert.NotNil(t, a.deploymentRepo, "first call should wire the repo")

	svcBefore := a.deploymentService
	a.ensureDeploymentService()
	assert.Same(t, svcBefore, a.deploymentService, "subsequent calls must be a no-op")
}

func assertFileExists(t *testing.T, path string) (string, error) {
	t.Helper()
	_, err := os.Stat(path)
	return path, err
}
