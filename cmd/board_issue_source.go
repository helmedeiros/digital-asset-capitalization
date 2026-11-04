package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	sprintports "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

// labelledIssueLister is implemented by the sprint JIRA adapter.
type labelledIssueLister interface {
	GetLabelledIssuesForSprint(project, sprintName, label string) ([]sprintports.JiraIssue, error)
}

// boardIssueSourceAdapter implements taskports.BoardIssueSource by reusing
// the sprint allocation fetch, so `tasks fetch` and `sprint allocate` see the
// same issues per sprint. Both sources are optional and may be combined:
// configured boards (`config board-work-streams`) and the project's
// operational label (`config operational-label`).
type boardIssueSourceAdapter struct {
	configService *service.ConfigService
	sprintJira    sprintports.JiraPort
	labelled      labelledIssueLister
}

func (a boardIssueSourceAdapter) FindIssueKeysForSprint(_ context.Context, project, sprint string) ([]string, error) {
	teamConfig, err := a.configService.GetTeamConfig()
	if err != nil {
		return nil, nil // no team config: nothing configured
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

	if label := teamConfig.GetOperationalLabel(project); label != "" && a.labelled != nil {
		issues, err := a.labelled.GetLabelledIssuesForSprint(project, sprint, label)
		if err != nil {
			return nil, fmt.Errorf("label %q: %w", label, err)
		}
		for _, issue := range issues {
			keys = append(keys, issue.Key)
		}
	}
	return keys, nil
}
