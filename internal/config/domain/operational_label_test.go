package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamConfig_OperationalLabel(t *testing.T) {
	t.Parallel()

	config, err := NewTeamConfig(map[string][]string{"PRI": {"Ann"}})
	require.NoError(t, err)

	assert.Equal(t, "", config.GetOperationalLabel("PRI"), "optional: unset by default")

	require.NoError(t, config.SetOperationalLabel("PRI", " operational "))
	assert.Equal(t, "operational", config.GetOperationalLabel("PRI"))
	assert.Equal(t, map[string]string{"PRI": "operational"}, config.GetAllOperationalLabels())

	require.NoError(t, config.SetOperationalLabel("PRI", ""))
	assert.Equal(t, "", config.GetOperationalLabel("PRI"), "empty label clears it")

	assert.Error(t, config.SetOperationalLabel("MISSING", "operational"))
	assert.Error(t, config.SetOperationalLabel("", "operational"))
}
