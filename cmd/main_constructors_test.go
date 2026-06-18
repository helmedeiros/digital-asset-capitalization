package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp_WiresAllProvidedServices(t *testing.T) {
	t.Parallel()
	asset := &stubAssetServiceForActions{}
	task := &stubTaskService{}
	sprint := &stubSprintService{}

	a := NewApp(asset, task, sprint)
	require.NotNil(t, a)
	assert.Same(t, asset, a.assetService.(*stubAssetServiceForActions))
	assert.Same(t, task, a.taskService.(*stubTaskService))
	assert.Same(t, sprint, a.sprintService.(*stubSprintService))
	assert.Nil(t, a.configService, "configService is wired in initializeApp, not NewApp")
}

func TestNewAppWithConfigService_WiresConfigService(t *testing.T) {
	t.Parallel()
	asset := &stubAssetServiceForActions{}
	task := &stubTaskService{}
	sprint := &stubSprintService{}
	cfg := &stubConfigService{}

	a := NewAppWithConfigService(asset, task, sprint, cfg)
	require.NotNil(t, a)
	assert.Same(t, asset, a.assetService.(*stubAssetServiceForActions))
	assert.Same(t, task, a.taskService.(*stubTaskService))
	assert.Same(t, sprint, a.sprintService.(*stubSprintService))
	assert.Same(t, cfg, a.configService.(*stubConfigService))
}

// configServiceImpl wraps concrete *usecase.InitializeConfig and
// *service.ConfigService fields, both of which require live JIRA
// configuration to construct. They are therefore covered
// transitively via the integration paths in configInitAction tests
// rather than directly here.
