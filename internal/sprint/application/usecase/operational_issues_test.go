package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configService "github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
	configinfra "github.com/helmedeiros/digital-asset-capitalization/internal/config/infrastructure"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

type labelledJira struct {
	ports.JiraPort
	issues    []ports.JiraIssue
	gotLabels []string
}

func (l *labelledJira) GetLabelledIssuesForSprint(_, _ string, labels []string) ([]ports.JiraIssue, error) {
	l.gotLabels = labels
	return l.issues, nil
}

func operationalConfigService(t *testing.T, labels []string) *configService.ConfigService {
	t.Helper()
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"TEST": {"Test User 1"}})
	require.NoError(t, err)
	require.NoError(t, cfg.SetOperationalLabels("TEST", labels))
	repo := configinfra.NewFileRepository(t.TempDir())
	require.NoError(t, repo.SaveTeamConfig(cfg))
	return configService.NewConfigService(repo)
}

func TestAddOperationalIssues(t *testing.T) {
	t.Parallel()

	sprintIssue := ports.JiraIssue{Key: "TEST-1"}
	opIssue := ports.JiraIssue{Key: "TEST-2"}
	opWithStream := ports.JiraIssue{Key: "TEST-3", BoardWorkStream: "custom"}

	newUseCase := func(t *testing.T, labels []string, streams []string) (*SprintTimeAllocationUseCase, *labelledJira) {
		jira := &labelledJira{issues: []ports.JiraIssue{sprintIssue, opIssue, opWithStream}}
		return &SprintTimeAllocationUseCase{
			project:     "TEST",
			sprint:      "S1",
			jiraPort:    jira,
			configSvc:   operationalConfigService(t, labels),
			workStreams: streams,
		}, jira
	}

	t.Run("adds labelled issues once and tags them operational", func(t *testing.T) {
		p, jira := newUseCase(t, []string{"operational", "support"}, nil)

		got := p.addOperationalIssues([]ports.JiraIssue{sprintIssue})

		require.Len(t, got, 3)
		assert.Equal(t, []string{"operational", "support"}, jira.gotLabels)
		assert.Equal(t, "operational", got[1].BoardWorkStream)
		assert.Equal(t, "custom", got[2].BoardWorkStream, "an existing work stream is kept")
	})

	t.Run("included when operational is among the requested work streams", func(t *testing.T) {
		p, _ := newUseCase(t, []string{"operational"}, []string{"product", " Operational "})
		assert.Len(t, p.addOperationalIssues(nil), 3)
	})

	t.Run("left out when only other work streams are requested", func(t *testing.T) {
		p, jira := newUseCase(t, []string{"operational"}, []string{"product"})
		assert.Empty(t, p.addOperationalIssues(nil))
		assert.Nil(t, jira.gotLabels, "no query is made")
	})

	t.Run("nothing happens without configured labels", func(t *testing.T) {
		p, jira := newUseCase(t, nil, nil)
		assert.Empty(t, p.addOperationalIssues(nil))
		assert.Nil(t, jira.gotLabels)
	})
}

func TestOnlyOperationalWithoutBoards(t *testing.T) {
	t.Parallel()

	newUseCase := func(streams ...string) *SprintTimeAllocationUseCase {
		return &SprintTimeAllocationUseCase{project: "TEST", configSvc: operationalConfigService(t, nil), workStreams: streams}
	}

	assert.True(t, newUseCase("operational").onlyOperationalWithoutBoards())
	assert.True(t, newUseCase(" Operational ").onlyOperationalWithoutBoards())
	assert.False(t, newUseCase().onlyOperationalWithoutBoards(), "no filter")
	assert.False(t, newUseCase("product").onlyOperationalWithoutBoards())
	assert.False(t, newUseCase("product", "operational").onlyOperationalWithoutBoards())
}
