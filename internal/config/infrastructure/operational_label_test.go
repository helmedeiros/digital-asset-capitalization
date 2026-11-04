package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

func TestFileRepository_OperationalLabelRoundTrip(t *testing.T) {
	t.Parallel()

	repo := NewFileRepository(t.TempDir())
	config, err := domain.NewTeamConfig(map[string][]string{"PRI": {"Ann"}, "OTH": {"Bob"}})
	require.NoError(t, err)
	require.NoError(t, config.SetOperationalLabel("PRI", "operational"))

	require.NoError(t, repo.SaveTeamConfig(config))
	loaded, err := repo.LoadTeamConfig()

	require.NoError(t, err)
	assert.Equal(t, "operational", loaded.GetOperationalLabel("PRI"))
	assert.Equal(t, "", loaded.GetOperationalLabel("OTH"))
}
