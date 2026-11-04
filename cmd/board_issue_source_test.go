package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/infrastructure"
	sprintports "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

func configServiceWith(t *testing.T, config *domain.TeamConfig) *service.ConfigService {
	t.Helper()
	repo := infrastructure.NewFileRepository(t.TempDir())
	require.NoError(t, repo.SaveTeamConfig(config))
	return service.NewConfigService(repo)
}

type stubLabelled struct{ gotLabel string }

func (s *stubLabelled) GetLabelledIssuesForSprint(_, _, label string) ([]sprintports.JiraIssue, error) {
	s.gotLabel = label
	return []sprintports.JiraIssue{{Key: "PRI-9"}}, nil
}

func TestBoardIssueSourceAdapter_OperationalLabelOnly(t *testing.T) {
	t.Parallel()

	config, err := domain.NewTeamConfig(map[string][]string{"PRI": {"Ann"}})
	require.NoError(t, err)
	require.NoError(t, config.SetOperationalLabel("PRI", "operational"))
	configService := configServiceWith(t, config)
	labelled := &stubLabelled{}

	// no boards configured and a nil board port: only the label is used
	keys, err := boardIssueSourceAdapter{configService: configService, labelled: labelled}.
		FindIssueKeysForSprint(context.Background(), "PRI", "S1")

	require.NoError(t, err)
	assert.Equal(t, []string{"PRI-9"}, keys)
	assert.Equal(t, "operational", labelled.gotLabel)
}

func TestBoardIssueSourceAdapter_NothingConfigured(t *testing.T) {
	t.Parallel()

	config, err := domain.NewTeamConfig(map[string][]string{"PRI": {"Ann"}})
	require.NoError(t, err)
	labelled := &stubLabelled{}

	keys, err := boardIssueSourceAdapter{configService: configServiceWith(t, config), labelled: labelled}.
		FindIssueKeysForSprint(context.Background(), "PRI", "S1")

	require.NoError(t, err)
	assert.Empty(t, keys)
	assert.Empty(t, labelled.gotLabel)
}
