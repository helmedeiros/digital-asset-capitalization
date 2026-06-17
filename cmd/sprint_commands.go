package main

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	sprintusecase "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/infrastructure/formatting"
)

// createSprintCommand builds the `sprint` CLI command with its
// list/allocate subcommands. The Action closures delegate to named
// methods on *App so each handler is unit-testable against stub
// services without going through the cli.App boot machinery.
func (a *App) createSprintCommand() *cli.Command {
	return &cli.Command{
		Name:  "sprint",
		Usage: "Manage sprint-related operations",
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List sprints for a project and time period",
				Action: a.sprintListAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "project",
						Aliases:  []string{"p"},
						Usage:    "Project key (e.g., FN)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "period",
						Aliases:  []string{"t"},
						Usage:    "Time period (e.g., Q2 2025, 2025, 2025-01-01:2025-03-31)",
						Required: true,
					},
				},
			},
			{
				Name:   "allocate",
				Usage:  "Calculate time allocation for JIRA issues in a sprint",
				Action: a.sprintAllocateAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "project",
						Aliases:  []string{"p"},
						Usage:    "Project key",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "sprint",
						Aliases:  []string{"s"},
						Usage:    "Sprint name or ID",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "override",
						Aliases: []string{"o"},
						Usage:   "Manual percentage adjustments as JSON where key is IssueID and value is amount of working hours being spent (e.g. '{\"ISSUE-1\": 6, \"ISSUE-2\": 36}')",
					},
					&cli.BoolFlag{
						Name:    "sprint-bounded",
						Aliases: []string{"sb"},
						Usage:   "Use sprint-bounded time calculation (respects sprint date boundaries)",
						Value:   true,
					},
					&cli.StringFlag{
						Name:    "work-streams",
						Aliases: []string{"ws"},
						Usage:   "Comma-separated work streams to include (e.g., 'product,operational')",
					},
					&cli.BoolFlag{
						Name:  "with-hours",
						Usage: "Include engineering hours column in CSV output",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "apply",
						Usage: "Push calculated allocation data back to JIRA custom fields",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Show what would be pushed without making changes (requires --apply)",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "force",
						Usage: "Override allocation lock (requires --apply)",
						Value: false,
					},
				},
			},
		},
	}
}

// sprintListAction backs `assetcap sprint list`. Resolves a team
// nickname through teamResolver (if configured), calls ListSprints on
// the sprint service, and writes the formatted result to stdout.
func (a *App) sprintListAction(ctx *cli.Context) error {
	project := ctx.String("project")
	period := ctx.String("period")

	if a.teamResolver != nil && project != "" {
		resolvedProject, err := a.teamResolver.ResolveProjectIdentifier(project)
		if err != nil {
			return fmt.Errorf("unknown project or team nickname: %s", project)
		}
		project = resolvedProject
	}

	result, err := a.sprintService.ListSprints(project, period)
	if err != nil {
		return err
	}

	formatter := formatting.NewOutputFormatter()
	output := formatter.FormatSprintList(project, period, result.Sprints, result.BoardInfo)
	fmt.Print(output)
	return nil
}

// sprintAllocateAction backs `assetcap sprint allocate`. With --apply
// it delegates to handleAllocateApply for the push-to-JIRA flow;
// otherwise it returns the CSV-formatted allocation for the sprint.
func (a *App) sprintAllocateAction(ctx *cli.Context) error {
	project := ctx.String("project")
	sprint := ctx.String("sprint")
	override := ctx.String("override")
	sprintBounded := ctx.Bool("sprint-bounded")
	workStreamsStr := ctx.String("work-streams")
	withHours := ctx.Bool("with-hours")
	apply := ctx.Bool("apply")
	dryRun := ctx.Bool("dry-run")
	force := ctx.Bool("force")

	if a.teamResolver != nil && project != "" {
		resolvedProject, err := a.teamResolver.ResolveProjectIdentifier(project)
		if err != nil {
			return fmt.Errorf("unknown project or team nickname: %s", project)
		}
		project = resolvedProject
	}

	workStreams := parseCommaSeparated(workStreamsStr)

	var opts []sprintusecase.SprintAllocationOption
	if len(workStreams) > 0 {
		opts = append(opts, sprintusecase.WithWorkStreams(workStreams))
	}
	if withHours {
		opts = append(opts, sprintusecase.WithHours(true))
	}

	if apply {
		return a.handleAllocateApply(project, sprint, override, sprintBounded, dryRun, force, opts)
	}

	if len(opts) > 0 || sprintBounded {
		result, err := a.sprintService.ProcessJiraIssuesWithOptions(project, sprint, override, sprintBounded, opts...)
		if err != nil {
			return err
		}
		fmt.Print(result)
		return nil
	}

	result, err := a.sprintService.ProcessJiraIssues(project, sprint, override)
	if err != nil {
		return err
	}
	fmt.Print(result)
	return nil
}

// parseCommaSeparated splits a comma-separated string, trims each
// element, and drops empties. Shared by allocate (work-streams) and
// any future multi-value flag.
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
