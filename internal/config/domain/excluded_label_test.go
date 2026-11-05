package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamConfig_ExcludedLabels(t *testing.T) {
	t.Parallel()

	config, err := NewTeamConfig(map[string][]string{"PRI": {"Ann"}})
	require.NoError(t, err)

	assert.Empty(t, config.GetExcludedLabels("PRI"), "optional: unset by default")

	require.NoError(t, config.SetExcludedLabels("PRI", []string{" assetcap-exclude ", "", "skip-me", "assetcap-exclude"}))
	assert.Equal(t, []string{"assetcap-exclude", "skip-me"}, config.GetExcludedLabels("PRI"), "trimmed, deduplicated")
	assert.Equal(t, map[string][]string{"PRI": {"assetcap-exclude", "skip-me"}}, config.GetAllExcludedLabels())

	got := config.GetExcludedLabels("PRI")
	got[0] = "mutated"
	assert.Equal(t, "assetcap-exclude", config.GetExcludedLabels("PRI")[0], "returns a copy")

	require.NoError(t, config.SetExcludedLabels("PRI", nil))
	assert.Empty(t, config.GetExcludedLabels("PRI"), "empty clears")

	assert.Error(t, config.SetExcludedLabels("MISSING", []string{"assetcap-exclude"}))
	assert.Error(t, config.SetExcludedLabels("", []string{"assetcap-exclude"}))
}
