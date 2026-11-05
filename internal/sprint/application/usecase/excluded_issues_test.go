package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configService "github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
	configinfra "github.com/helmedeiros/digital-asset-capitalization/internal/config/infrastructure"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
)

func excludedConfigService(t *testing.T, labels []string) *configService.ConfigService {
	t.Helper()
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"TEST": {"Test User 1"}})
	require.NoError(t, err)
	require.NoError(t, cfg.SetExcludedLabels("TEST", labels))
	repo := configinfra.NewFileRepository(t.TempDir())
	require.NoError(t, repo.SaveTeamConfig(cfg))
	return configService.NewConfigService(repo)
}

func issueWithLabels(key string, labels ...string) domain.JiraIssue {
	return domain.JiraIssue{Key: key, Fields: domain.JiraFields{Labels: labels}}
}

func TestDropExcludedIssues(t *testing.T) {
	t.Parallel()

	issues := []domain.JiraIssue{
		issueWithLabels("TEST-1", "cap-asset-pricing"),
		issueWithLabels("TEST-2", "operational", "assetcap-exclude"),
		issueWithLabels("TEST-3"),
	}

	t.Run("drops issues carrying an excluded label", func(t *testing.T) {
		p := &SprintTimeAllocationUseCase{project: "TEST", configSvc: excludedConfigService(t, []string{"assetcap-exclude"})}

		got := p.dropExcludedIssues(issues)

		require.Len(t, got, 2)
		assert.Equal(t, "TEST-1", got[0].Key)
		assert.Equal(t, "TEST-3", got[1].Key)
	})

	t.Run("keeps everything when no excluded labels are configured", func(t *testing.T) {
		p := &SprintTimeAllocationUseCase{project: "TEST", configSvc: excludedConfigService(t, nil)}

		assert.Len(t, p.dropExcludedIssues(issues), 3)
	})

	t.Run("drops explicitly excluded keys without any label config", func(t *testing.T) {
		p := &SprintTimeAllocationUseCase{project: "TEST", excludedKeys: []string{"TEST-3"}}

		got := p.dropExcludedIssues(issues)

		require.Len(t, got, 2)
		assert.Equal(t, "TEST-2", got[1].Key)
	})

	t.Run("keeps everything without a config service", func(t *testing.T) {
		p := &SprintTimeAllocationUseCase{project: "TEST"}

		assert.Len(t, p.dropExcludedIssues(issues), 3)
	})
}
