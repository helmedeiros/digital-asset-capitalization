package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

func TestFileRepository_ExcludedLabelRoundTrip(t *testing.T) {
	t.Parallel()

	repo := NewFileRepository(t.TempDir())
	config, err := domain.NewTeamConfig(map[string][]string{"PRI": {"Ann"}, "OTH": {"Bob"}})
	require.NoError(t, err)
	require.NoError(t, config.SetExcludedLabels("PRI", []string{"assetcap-exclude", "skip-me"}))

	require.NoError(t, repo.SaveTeamConfig(config))
	loaded, err := repo.LoadTeamConfig()

	require.NoError(t, err)
	assert.Equal(t, []string{"assetcap-exclude", "skip-me"}, loaded.GetExcludedLabels("PRI"))
	assert.Empty(t, loaded.GetExcludedLabels("OTH"))
}
