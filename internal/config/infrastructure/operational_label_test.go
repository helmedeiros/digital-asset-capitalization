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
	require.NoError(t, config.SetOperationalLabels("PRI", []string{"operational", "support"}))

	require.NoError(t, repo.SaveTeamConfig(config))
	loaded, err := repo.LoadTeamConfig()

	require.NoError(t, err)
	assert.Equal(t, []string{"operational", "support"}, loaded.GetOperationalLabels("PRI"))
	assert.Empty(t, loaded.GetOperationalLabels("OTH"))
}
