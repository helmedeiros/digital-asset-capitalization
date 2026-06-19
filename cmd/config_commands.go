package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/usecase"
	configinfra "github.com/helmedeiros/digital-asset-capitalization/internal/config/infrastructure"
)

// createConfigCommand builds the `config` CLI command with all its
// subcommands (init/show/validate/sync-team/team-nicknames/team-tribe/
// team-company/board-work-streams/excluded-issue-types/team-timeline).
// Extracted from cmd/main.go.
func (a *App) createConfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Manage configuration settings",
		Subcommands: []*cli.Command{
			{
				Name:   "init",
				Usage:  "Initialize configuration",
				Action: a.configInitAction,
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "non-interactive", Usage: "Run in non-interactive mode (requires environment variables)", Value: false},
					&cli.StringFlag{Name: "jira-url", Usage: "Jira base URL (e.g., https://company.atlassian.net)"},
					&cli.StringFlag{Name: "jira-email", Usage: "Jira email address"},
					&cli.StringFlag{Name: "jira-token", Usage: "Jira API token"},
				},
			},
			{
				Name:   "show",
				Usage:  "Show current configuration",
				Action: a.configShowAction,
			},
			{
				Name:   "validate",
				Usage:  "Validate current configuration",
				Action: a.configValidateAction,
			},
			{
				Name:  "sync-team",
				Usage: "Synchronize team members from JIRA for a project",
				Action: func(ctx *cli.Context) error {
					projectKey := ctx.String("project")
					if projectKey == "" {
						return fmt.Errorf("project key is required")
					}

					// Create JIRA team sync adapter
					teamSyncAdapter, err := configinfra.NewJiraTeamSyncAdapter(a.configService)
					if err != nil {
						return fmt.Errorf("failed to create team sync adapter: %v", err)
					}

					// Create team sync use case
					configRepo := configinfra.NewFileRepository(configDir)
					syncTeamUseCase := usecase.NewSyncTeamFromJira(teamSyncAdapter, configRepo)

					// Execute team synchronization
					result, err := syncTeamUseCase.Execute(projectKey)
					if err != nil {
						return fmt.Errorf("failed to sync team: %v", err)
					}

					// Display results
					fmt.Printf("✅ Team synchronization completed for project %s\n", projectKey)
					fmt.Printf("Source: %s\n", result.Source)
					fmt.Printf("Total members: %d\n", result.TotalMembers)

					if len(result.AddedMembers) > 0 {
						fmt.Printf("Added members: %s\n", strings.Join(result.AddedMembers, ", "))
					}

					if len(result.RemovedMembers) > 0 {
						fmt.Printf("Removed members: %s\n", strings.Join(result.RemovedMembers, ", "))
					}

					if result.HasErrors() {
						fmt.Printf("⚠️  Warnings/Errors:\n")
						for _, syncErr := range result.Errors {
							fmt.Printf("  - %s (%s)\n", syncErr.Message, syncErr.Type)
						}
					}

					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "project",
						Aliases:  []string{"p"},
						Usage:    "Project key (e.g., FN, TEST)",
						Required: true,
					},
				},
			},
			{
				Name:  "team-nicknames",
				Usage: "Manage team nicknames",
				Subcommands: []*cli.Command{
					{
						Name:   "add",
						Usage:  "Add nicknames for a project",
						Action: a.configTeamNicknamesAddAction,
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "project", Usage: "Project key (e.g., FN)", Required: true},
							&cli.StringFlag{Name: "nickname", Usage: "Comma-separated nicknames (e.g., 'Pricing,Fintech')", Required: true},
						},
					},
					{
						Name:   "list",
						Usage:  "List all team nicknames",
						Action: a.configTeamNicknamesListAction,
					},
					{
						Name:   "show",
						Usage:  "Show nicknames for a specific project",
						Action: a.configTeamNicknamesShowAction,
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "project", Usage: "Project key or nickname (e.g., FN or Pricing)", Required: true},
						},
					},
				},
			},
			{
				Name:  "team-tribe",
				Usage: "Manage team tribes (organizational groupings)",
				Subcommands: []*cli.Command{
					{
						Name:  "set",
						Usage: "Set the tribe for a project/team",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							tribe := ctx.String("tribe")

							if project == "" {
								return fmt.Errorf("project is required")
							}
							if tribe == "" {
								return fmt.Errorf("tribe is required")
							}

							// Create config service to save tribe
							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							if err := configSvc.SetTribeForProject(project, tribe); err != nil {
								return fmt.Errorf("failed to set tribe: %v", err)
							}

							fmt.Printf("✅ Set tribe '%s' for project '%s'\n", tribe, project)
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Aliases:  []string{"p"},
								Usage:    "Project key (e.g., FN, COP)",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "tribe",
								Aliases:  []string{"t"},
								Usage:    "Tribe name (e.g., 'Engineering', 'Platform')",
								Required: true,
							},
						},
					},
					{
						Name:  "list",
						Usage: "List all team tribes",
						Action: func(_ *cli.Context) error {
							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							teamConfig, err := configSvc.GetTeamConfig()
							if err != nil {
								return fmt.Errorf("failed to load team config: %v", err)
							}

							projects := teamConfig.GetProjects()
							if len(projects) == 0 {
								fmt.Println("No teams configured")
								return nil
							}

							fmt.Println("Team Tribes:")
							fmt.Println("=============")

							// Group by tribe
							tribeProjects := make(map[string][]string)
							noTribe := []string{}

							for _, project := range projects {
								tribe := teamConfig.GetTribe(project)
								if tribe != "" {
									tribeProjects[tribe] = append(tribeProjects[tribe], project)
								} else {
									noTribe = append(noTribe, project)
								}
							}

							for tribe, projs := range tribeProjects {
								fmt.Printf("\n%s:\n", tribe)
								for _, p := range projs {
									fmt.Printf("  - %s\n", p)
								}
							}

							if len(noTribe) > 0 {
								fmt.Printf("\n(No tribe assigned):\n")
								for _, p := range noTribe {
									fmt.Printf("  - %s\n", p)
								}
							}

							return nil
						},
					},
					{
						Name:  "show",
						Usage: "Show the tribe for a specific project",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							if project == "" {
								return fmt.Errorf("project is required")
							}

							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							tribe, err := configSvc.GetTribeForProject(project)
							if err != nil {
								return fmt.Errorf("failed to get tribe: %v", err)
							}

							if tribe == "" {
								fmt.Printf("Project '%s' has no tribe assigned\n", project)
							} else {
								fmt.Printf("Project '%s' belongs to tribe: %s\n", project, tribe)
							}

							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Aliases:  []string{"p"},
								Usage:    "Project key (e.g., FN)",
								Required: true,
							},
						},
					},
				},
			},
			{
				Name:  "team-company",
				Usage: "Manage team companies (organization ownership)",
				Subcommands: []*cli.Command{
					{
						Name:  "set",
						Usage: "Set the company for a project/team",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							company := ctx.String("company")

							if project == "" {
								return fmt.Errorf("project is required")
							}
							if company == "" {
								return fmt.Errorf("company is required")
							}

							// Create config service to save company
							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							if err := configSvc.SetCompanyForProject(project, company); err != nil {
								return fmt.Errorf("failed to set company: %v", err)
							}

							fmt.Printf("✅ Set company '%s' for project '%s'\n", company, project)
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Aliases:  []string{"p"},
								Usage:    "Project key (e.g., FN, COP)",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "company",
								Aliases:  []string{"c"},
								Usage:    "Company name (e.g., 'ACME Corp', 'Partner Co')",
								Required: true,
							},
						},
					},
					{
						Name:  "list",
						Usage: "List all team companies",
						Action: func(_ *cli.Context) error {
							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							teamConfig, err := configSvc.GetTeamConfig()
							if err != nil {
								return fmt.Errorf("failed to load team config: %v", err)
							}

							projects := teamConfig.GetProjects()
							if len(projects) == 0 {
								fmt.Println("No teams configured")
								return nil
							}

							fmt.Println("Team Companies:")
							fmt.Println("===============")

							// Group by company
							companyProjects := make(map[string][]string)
							noCompany := []string{}

							for _, project := range projects {
								company := teamConfig.GetCompany(project)
								if company != "" {
									companyProjects[company] = append(companyProjects[company], project)
								} else {
									noCompany = append(noCompany, project)
								}
							}

							for company, projs := range companyProjects {
								fmt.Printf("\n%s:\n", company)
								for _, p := range projs {
									fmt.Printf("  - %s\n", p)
								}
							}

							if len(noCompany) > 0 {
								fmt.Printf("\n(No company assigned):\n")
								for _, p := range noCompany {
									fmt.Printf("  - %s\n", p)
								}
							}

							return nil
						},
					},
					{
						Name:  "show",
						Usage: "Show the company for a specific project",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							if project == "" {
								return fmt.Errorf("project is required")
							}

							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							company, err := configSvc.GetCompanyForProject(project)
							if err != nil {
								return fmt.Errorf("failed to get company: %v", err)
							}

							if company == "" {
								fmt.Printf("Project '%s' has no company assigned\n", project)
							} else {
								fmt.Printf("Project '%s' belongs to company: %s\n", project, company)
							}

							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Aliases:  []string{"p"},
								Usage:    "Project key (e.g., FN)",
								Required: true,
							},
						},
					},
				},
			},
			{
				Name:  "board-work-streams",
				Usage: "Manage board-to-workstream mappings per project",
				Subcommands: []*cli.Command{
					{
						Name:  "set",
						Usage: "Set work stream for a board in a project",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							boardID := ctx.Int("board")
							workStream := ctx.String("work-stream")

							if project == "" {
								return fmt.Errorf("project is required")
							}
							if boardID == 0 {
								return fmt.Errorf("board ID is required")
							}
							if workStream == "" {
								return fmt.Errorf("work-stream is required")
							}

							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							if err := configSvc.SetBoardWorkStream(project, boardID, workStream); err != nil {
								return fmt.Errorf("failed to set board work stream: %v", err)
							}

							fmt.Printf("Set board %d -> '%s' for project '%s'\n", boardID, workStream, project)
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Aliases:  []string{"p"},
								Usage:    "Project key (e.g., COP)",
								Required: true,
							},
							&cli.IntFlag{
								Name:     "board",
								Aliases:  []string{"b"},
								Usage:    "Board ID (e.g., 5119)",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "work-stream",
								Aliases:  []string{"ws"},
								Usage:    "Work stream name (e.g., Product, Operational)",
								Required: true,
							},
						},
					},
					{
						Name:  "show",
						Usage: "Show board-to-workstream mappings for a specific project",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							if project == "" {
								return fmt.Errorf("project is required")
							}

							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							teamConfig, err := configSvc.GetTeamConfig()
							if err != nil {
								return fmt.Errorf("failed to load team config: %v", err)
							}

							mapping := teamConfig.GetBoardWorkStreams(project)
							if len(mapping) == 0 {
								fmt.Printf("Project '%s' has no board-to-workstream mappings\n", project)
							} else {
								fmt.Printf("Board Work Streams for '%s':\n", project)
								for boardID, ws := range mapping {
									fmt.Printf("  Board %d -> %s\n", boardID, ws)
								}
							}
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Aliases:  []string{"p"},
								Usage:    "Project key (e.g., COP)",
								Required: true,
							},
						},
					},
					{
						Name:  "list",
						Usage: "List board-to-workstream mappings for all projects",
						Action: func(_ *cli.Context) error {
							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							teamConfig, err := configSvc.GetTeamConfig()
							if err != nil {
								return fmt.Errorf("failed to load team config: %v", err)
							}

							projects := teamConfig.GetProjects()
							if len(projects) == 0 {
								fmt.Println("No teams configured")
								return nil
							}

							fmt.Println("Board Work Streams:")
							fmt.Println("===================")

							found := false
							for _, project := range projects {
								mapping := teamConfig.GetBoardWorkStreams(project)
								if len(mapping) > 0 {
									fmt.Printf("  %s:\n", project)
									for boardID, ws := range mapping {
										fmt.Printf("    Board %d -> %s\n", boardID, ws)
									}
									found = true
								}
							}

							if !found {
								fmt.Println("  No board-to-workstream mappings configured")
							}

							return nil
						},
					},
				},
			},
			{
				Name:  "excluded-issue-types",
				Usage: "Manage excluded issue types for sprint allocation per project",
				Subcommands: []*cli.Command{
					{
						Name:  "set",
						Usage: "Set excluded issue types for a project (comma-separated)",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							typesStr := ctx.String("types")

							if project == "" {
								return fmt.Errorf("project is required")
							}
							if typesStr == "" {
								return fmt.Errorf("types is required")
							}

							types := strings.Split(typesStr, ",")
							for i, t := range types {
								types[i] = strings.TrimSpace(t)
							}

							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							if err := configSvc.SetExcludedIssueTypesForProject(project, types); err != nil {
								return fmt.Errorf("failed to set excluded issue types: %v", err)
							}

							fmt.Printf("Set excluded issue types for project '%s': %s\n", project, strings.Join(types, ", "))
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Aliases:  []string{"p"},
								Usage:    "Project key (e.g., COP)",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "types",
								Aliases:  []string{"t"},
								Usage:    "Comma-separated issue types to exclude (e.g., 'Experiment,Spike')",
								Required: true,
							},
						},
					},
					{
						Name:  "clear",
						Usage: "Clear excluded issue types for a project",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							if project == "" {
								return fmt.Errorf("project is required")
							}

							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							if err := configSvc.SetExcludedIssueTypesForProject(project, nil); err != nil {
								return fmt.Errorf("failed to clear excluded issue types: %v", err)
							}

							fmt.Printf("Cleared excluded issue types for project '%s'\n", project)
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Aliases:  []string{"p"},
								Usage:    "Project key (e.g., COP)",
								Required: true,
							},
						},
					},
					{
						Name:  "list",
						Usage: "List excluded issue types for all projects",
						Action: func(_ *cli.Context) error {
							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							teamConfig, err := configSvc.GetTeamConfig()
							if err != nil {
								return fmt.Errorf("failed to load team config: %v", err)
							}

							projects := teamConfig.GetProjects()
							if len(projects) == 0 {
								fmt.Println("No teams configured")
								return nil
							}

							fmt.Println("Excluded Issue Types:")
							fmt.Println("=====================")

							found := false
							for _, project := range projects {
								types := teamConfig.GetExcludedIssueTypes(project)
								if len(types) > 0 {
									fmt.Printf("  %s: %s\n", project, strings.Join(types, ", "))
									found = true
								}
							}

							if !found {
								fmt.Println("  No excluded issue types configured for any project")
							}

							return nil
						},
					},
					{
						Name:  "show",
						Usage: "Show excluded issue types for a specific project",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							if project == "" {
								return fmt.Errorf("project is required")
							}

							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							types, err := configSvc.GetExcludedIssueTypesForProject(project)
							if err != nil {
								return fmt.Errorf("failed to get excluded issue types: %v", err)
							}

							if len(types) == 0 {
								fmt.Printf("Project '%s' has no excluded issue types\n", project)
							} else {
								fmt.Printf("Project '%s' excludes: %s\n", project, strings.Join(types, ", "))
							}

							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Aliases:  []string{"p"},
								Usage:    "Project key (e.g., COP)",
								Required: true,
							},
						},
					},
				},
			},
			{
				Name:  "team-timeline",
				Usage: "Manage team member timeline (join/leave dates) for time-aware sprint allocation",
				Subcommands: []*cli.Command{
					{
						Name:  "show",
						Usage: "Show team timeline for a project",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							if project == "" {
								return fmt.Errorf("project is required")
							}

							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							teamConfig, err := configSvc.GetTeamConfig()
							if err != nil {
								return fmt.Errorf("failed to load team config: %v", err)
							}

							timeline := teamConfig.GetTeamTimeline(project)
							if len(timeline) == 0 {
								fmt.Printf("Project '%s' has no team timeline configured\n", project)
								fmt.Println("Using flat team list for all sprints")
								members, exists := teamConfig.GetTeam(project)
								if exists {
									fmt.Printf("Current team: %s\n", strings.Join(members, ", "))
								}
								return nil
							}

							fmt.Printf("Team Timeline for '%s':\n", project)
							fmt.Println("===========================")
							for _, p := range timeline {
								status := "active"
								leftStr := ""
								if p.Left != nil {
									status = "departed"
									leftStr = p.Left.Format("2006-01-02")
								}
								fmt.Printf("  %-25s joined: %s  left: %-12s [%s]\n",
									p.Member, p.Joined.Format("2006-01-02"), leftStr, status)
							}

							active := teamConfig.DeriveActiveTeamFromTimeline(project)
							fmt.Printf("\nActive members (%d): %s\n", len(active), strings.Join(active, ", "))

							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Aliases:  []string{"p"},
								Usage:    "Project key",
								Required: true,
							},
						},
					},
					{
						Name:  "add",
						Usage: "Add a member to the team timeline with a join date",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							member := ctx.String("member")
							joinedStr := ctx.String("joined")

							if project == "" || member == "" || joinedStr == "" {
								return fmt.Errorf("project, member, and joined date are required")
							}

							joined, err := time.Parse("2006-01-02", joinedStr)
							if err != nil {
								return fmt.Errorf("invalid date format, use YYYY-MM-DD: %v", err)
							}

							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							teamConfig, err := configSvc.GetTeamConfig()
							if err != nil {
								return fmt.Errorf("failed to load team config: %v", err)
							}

							if err := teamConfig.AddMemberWithDates(project, member, joined); err != nil {
								return fmt.Errorf("failed to add member: %v", err)
							}

							if err := configSvc.SaveTeamConfig(teamConfig); err != nil {
								return fmt.Errorf("failed to save team config: %v", err)
							}

							fmt.Printf("Added '%s' to project '%s' timeline (joined: %s)\n", member, project, joinedStr)
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Aliases:  []string{"p"},
								Usage:    "Project key",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "member",
								Aliases:  []string{"m"},
								Usage:    "Team member name",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "joined",
								Aliases:  []string{"j"},
								Usage:    "Join date (YYYY-MM-DD)",
								Required: true,
							},
						},
					},
					{
						Name:  "remove",
						Usage: "Set a member's departure date in the team timeline",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							member := ctx.String("member")
							leftStr := ctx.String("left")

							if project == "" || member == "" || leftStr == "" {
								return fmt.Errorf("project, member, and left date are required")
							}

							left, err := time.Parse("2006-01-02", leftStr)
							if err != nil {
								return fmt.Errorf("invalid date format, use YYYY-MM-DD: %v", err)
							}

							configRepo := configinfra.NewFileRepository(configDir)
							configSvc := service.NewConfigService(configRepo)

							teamConfig, err := configSvc.GetTeamConfig()
							if err != nil {
								return fmt.Errorf("failed to load team config: %v", err)
							}

							if err := teamConfig.SetMemberLeft(project, member, left); err != nil {
								return fmt.Errorf("failed to set departure: %v", err)
							}

							if err := configSvc.SaveTeamConfig(teamConfig); err != nil {
								return fmt.Errorf("failed to save team config: %v", err)
							}

							fmt.Printf("Set '%s' departure from project '%s' (left: %s)\n", member, project, leftStr)
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Aliases:  []string{"p"},
								Usage:    "Project key",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "member",
								Aliases:  []string{"m"},
								Usage:    "Team member name",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "left",
								Aliases:  []string{"l"},
								Usage:    "Departure date (YYYY-MM-DD)",
								Required: true,
							},
						},
					},
				},
			},
		},
	}
}

// configInitAction backs `assetcap config init`. Pulls env-var
// overrides off the flags before delegating to the configuration
// service.
func (a *App) configInitAction(ctx *cli.Context) error {
	if a.configService == nil {
		return fmt.Errorf("configuration service not available")
	}

	interactive := !ctx.Bool("non-interactive")

	jiraURL := ctx.String("jira-url")
	jiraEmail := ctx.String("jira-email")
	jiraToken := ctx.String("jira-token")

	if jiraURL != "" {
		os.Setenv("JIRA_BASE_URL", jiraURL)
	}
	if jiraEmail != "" {
		os.Setenv("JIRA_EMAIL", jiraEmail)
	}
	if jiraToken != "" {
		os.Setenv("JIRA_TOKEN", jiraToken)
	}

	result, err := a.configService.InitializeConfig(interactive)
	if err != nil {
		return err
	}

	fmt.Println(result.Message)
	return nil
}

// configShowAction backs `assetcap config show`. Prints the current
// JIRA env vars (token masked) and reports whether the teams.json
// file is present.
func (a *App) configShowAction(_ *cli.Context) error {
	fmt.Println("Current Configuration:")
	fmt.Println("=====================")

	jiraURL := os.Getenv("JIRA_BASE_URL")
	jiraEmail := os.Getenv("JIRA_EMAIL")
	jiraToken := os.Getenv("JIRA_TOKEN")

	fmt.Printf("JIRA_BASE_URL: %s\n", jiraURL)
	fmt.Printf("JIRA_EMAIL: %s\n", jiraEmail)
	if jiraToken != "" {
		fmt.Printf("JIRA_TOKEN: %s\n", maskToken(jiraToken))
	} else {
		fmt.Printf("JIRA_TOKEN: <not set>\n")
	}

	teamsPath := filepath.Join(configDir, teamsFile)
	if _, err := os.Stat(teamsPath); err == nil {
		fmt.Printf("\nTeam Configuration: %s exists\n", teamsPath)
	} else {
		fmt.Printf("\nTeam Configuration: %s not found\n", teamsPath)
	}
	return nil
}

// configValidateAction backs `assetcap config validate`. Returns an
// error if any required env var is missing or teams.json is absent;
// the human-readable list of problems goes to stdout for visibility.
func (a *App) configValidateAction(_ *cli.Context) error {
	fmt.Println("Validating Configuration...")

	var problems []string
	if os.Getenv("JIRA_BASE_URL") == "" {
		problems = append(problems, "JIRA_BASE_URL is not set")
	}
	if os.Getenv("JIRA_EMAIL") == "" {
		problems = append(problems, "JIRA_EMAIL is not set")
	}
	if os.Getenv("JIRA_TOKEN") == "" {
		problems = append(problems, "JIRA_TOKEN is not set")
	}

	teamsPath := filepath.Join(configDir, teamsFile)
	if _, err := os.Stat(teamsPath); err != nil {
		problems = append(problems, fmt.Sprintf("%s file not found", teamsPath))
	}

	if len(problems) > 0 {
		fmt.Println("❌ Configuration validation failed:")
		for _, p := range problems {
			fmt.Printf("  - %s\n", p)
		}
		return fmt.Errorf("configuration validation failed")
	}

	fmt.Println("✅ Configuration is valid")
	return nil
}

// configTeamNicknamesAddAction backs `assetcap config team-nicknames add`.
// Currently informational only — full nickname-management requires a
// team-config update path that hasn't been wired through yet. The
// flag validation and the comma-split shape are real, though, so the
// action does light parsing and surfaces the input back to the user.
func (a *App) configTeamNicknamesAddAction(ctx *cli.Context) error {
	project := ctx.String("project")
	nickname := ctx.String("nickname")
	if project == "" {
		return fmt.Errorf("project is required")
	}
	if nickname == "" {
		return fmt.Errorf("nickname is required")
	}
	nicknames := parseCommaSeparated(nickname)
	fmt.Printf("⚠️  Note: Nickname management requires implementation of team config update functionality\n")
	fmt.Printf("Would add nickname(s) %s for project %s\n",
		strings.Join(nicknames, ", "), project)
	return nil
}

// configTeamNicknamesListAction backs `assetcap config team-nicknames list`.
// Reads the in-memory mapping the teamResolver loads at startup; an
// unconfigured resolver is a hard error since there's no recovery
// path without it.
func (a *App) configTeamNicknamesListAction(_ *cli.Context) error {
	if a.teamResolver == nil {
		return fmt.Errorf("team resolver not available")
	}

	mappings := a.teamResolver.GetAllMappings()
	if len(mappings) == 0 {
		fmt.Println("No team nicknames configured")
		return nil
	}

	fmt.Println("Team Nicknames:")
	fmt.Println("================")

	projectMap := make(map[string][]string)
	for nickname, project := range mappings {
		projectMap[project] = append(projectMap[project], nickname)
	}

	for project, nicks := range projectMap {
		fmt.Printf("%s: %s\n", project, strings.Join(nicks, ", "))
	}
	return nil
}

// configTeamNicknamesShowAction backs `assetcap config team-nicknames show`.
// Resolves the identifier so a nickname argument expands to the
// canonical project before display.
func (a *App) configTeamNicknamesShowAction(ctx *cli.Context) error {
	project := ctx.String("project")
	if project == "" {
		return fmt.Errorf("project is required")
	}
	if a.teamResolver == nil {
		return fmt.Errorf("team resolver not available")
	}

	resolvedProject, err := a.teamResolver.ResolveProjectIdentifier(project)
	if err != nil {
		return fmt.Errorf("unknown project: %s", project)
	}

	displayName := a.teamResolver.GetProjectWithNicknames(resolvedProject)
	fmt.Printf("Project: %s\n", displayName)
	return nil
}
