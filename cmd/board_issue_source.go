package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	sprintports "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

// boardIssueSourceAdapter implements taskports.BoardIssueSource by reusing
// the board-to-workstream configuration and the sprint allocation fetch, so
// `tasks fetch` and `sprint allocate` see the same issues per sprint.
type boardIssueSourceAdapter struct {
	configService *service.ConfigService
	sprintJira    sprintports.JiraPort
}

func (a boardIssueSourceAdapter) FindIssueKeysForSprint(_ context.Context, project, sprint string) ([]string, error) {
	teamConfig, err := a.configService.GetTeamConfig()
	if err != nil {
		return nil, nil // no team config: no boards configured
	}
	boards := teamConfig.GetBoardWorkStreams(project)
	ids := make([]int, 0, len(boards))
	for id := range boards {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	var keys []string
	for _, id := range ids {
		issues, err := a.sprintJira.GetIssuesForSprintOnBoard(project, sprint, id)
		if err != nil {
			return nil, fmt.Errorf("board %d: %w", id, err)
		}
		for _, issue := range issues {
			keys = append(keys, issue.Key)
		}
	}
	return keys, nil
}
