package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The ConfigServiceAdapter methods are intentionally stubs today --
// they return fixed shapes that downstream console wiring depends on.
// These tests pin those shapes so a future change of return contract
// surfaces at the test boundary instead of silently breaking the
// console code that reads the maps. Each adapter ignores its
// receiver's service so passing nil is fine for the assertion.

func TestConfigServiceAdapter_InitConfig(t *testing.T) {
	out, err := (&ConfigServiceAdapter{}).InitConfig(context.Background())
	require.NoError(t, err)
	got, ok := out.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "initialized", got["status"])
}

func TestConfigServiceAdapter_ShowConfig(t *testing.T) {
	out, err := (&ConfigServiceAdapter{}).ShowConfig(context.Background())
	require.NoError(t, err)
	got, ok := out.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "configured", got["jira_url"])
	assert.Equal(t, "configured", got["ollama_url"])
}

func TestConfigServiceAdapter_ValidateConfig(t *testing.T) {
	out, err := (&ConfigServiceAdapter{}).ValidateConfig(context.Background())
	require.NoError(t, err)
	got, ok := out.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "valid", got["status"])
}

func TestConfigServiceAdapter_SyncTeam(t *testing.T) {
	out, err := (&ConfigServiceAdapter{}).SyncTeam(context.Background(), "FN")
	require.NoError(t, err)
	got, ok := out.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "FN", got["project"])
	assert.Equal(t, 5, got["synced_members"])
}
