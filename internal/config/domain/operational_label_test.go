package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamConfig_OperationalLabels(t *testing.T) {
	t.Parallel()

	config, err := NewTeamConfig(map[string][]string{"PRI": {"Ann"}})
	require.NoError(t, err)

	assert.Empty(t, config.GetOperationalLabels("PRI"), "optional: unset by default")

	require.NoError(t, config.SetOperationalLabels("PRI", []string{" operational ", "", "support", "operational"}))
	assert.Equal(t, []string{"operational", "support"}, config.GetOperationalLabels("PRI"), "trimmed, deduplicated")
	assert.Equal(t, map[string][]string{"PRI": {"operational", "support"}}, config.GetAllOperationalLabels())

	got := config.GetOperationalLabels("PRI")
	got[0] = "mutated"
	assert.Equal(t, "operational", config.GetOperationalLabels("PRI")[0], "returns a copy")

	require.NoError(t, config.SetOperationalLabels("PRI", nil))
	assert.Empty(t, config.GetOperationalLabels("PRI"), "empty clears")

	assert.Error(t, config.SetOperationalLabels("MISSING", []string{"operational"}))
	assert.Error(t, config.SetOperationalLabels("", []string{"operational"}))
}
