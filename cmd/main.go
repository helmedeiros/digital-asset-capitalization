package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	assetsinfra "github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
	configinfra "github.com/helmedeiros/digital-asset-capitalization/internal/config/infrastructure"
	"github.com/helmedeiros/digital-asset-capitalization/internal/shell/completion"
	sprintapp "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application"
	sprintinfra "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/infrastructure"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/infrastructure/formatting"
	tasksapp "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application"
	tasksdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	taskports "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/classifier"
	cliui "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/cli"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/jira"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/migration"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/storage"
)

const (
	configDir  = ".assetcap"
	assetsDir  = ".assetcap"
	assetsFile = "assets.json"
	tasksDir   = ".assetcap"
	tasksFile  = "tasks.json"
	teamsFile  = "teams.json"
)

// Version information - these will be overridden by GoReleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// App holds all the application dependencies
type App struct {
	assetService  assetsapp.AssetService
	taskService   tasksapp.TaskService
	sprintService sprintapp.SprintService
	configService ConfigService
}

// ConfigService interface for configuration operations
type ConfigService interface {
	InitializeConfig(interactive bool) (*usecase.InitializeConfigResult, error)
	GetJiraConfig() (*domain.JiraConfig, error)
}

// configServiceImpl implements ConfigService
type configServiceImpl struct {
	initializeConfig *usecase.InitializeConfig
	configService    *service.ConfigService
}

func (c *configServiceImpl) InitializeConfig(interactive bool) (*usecase.InitializeConfigResult, error) {
	return c.initializeConfig.Execute(interactive)
}

func (c *configServiceImpl) GetJiraConfig() (*domain.JiraConfig, error) {
	return c.configService.GetJiraConfig()
}

// NewApp creates a new App instance with the given dependencies
func NewApp(assetService assetsapp.AssetService, taskService tasksapp.TaskService, sprintService sprintapp.SprintService) *App {
	return &App{
		assetService:  assetService,
		taskService:   taskService,
		sprintService: sprintService,
		configService: nil, // Will be set in initializeApp
	}
}

// NewAppWithConfigService creates a new App instance with config service for testing
func NewAppWithConfigService(assetService assetsapp.AssetService, taskService tasksapp.TaskService, sprintService sprintapp.SprintService, configService ConfigService) *App {
	return &App{
		assetService:  assetService,
		taskService:   taskService,
		sprintService: sprintService,
		configService: configService,
	}
}

// Run executes the CLI application
func (a *App) Run() error {
	app := &cli.App{
		Name:                 "AssetCap",
		Usage:                "Digital Asset Capitalization Management Tool",
		Version:              version,
		EnableBashCompletion: true,
		UsageText: `assetcap [global options] command [command options] [arguments...]

COMMANDS:
   version            Show version information
   completion         Generate shell completion scripts
     bash            Generate bash completion script
     zsh             Generate zsh completion script
     fish            Generate fish completion script
   config             Manage configuration settings
     init            Initialize configuration interactively
     show            Show current configuration
     validate        Validate current configuration
   assets             Manage digital assets
     create          Create a new asset
     list            List all assets
     sync            Sync assets from Confluence
     update          Update an asset's description
     show            Show detailed information about an asset
     enrich          Enrich asset fields using LLaMA 3
     keywords        Generate keywords for an asset using LLaMA 3
     documentation   Manage asset documentation
       update        Mark asset documentation as updated
     tasks           Manage asset tasks
       increment     Increment task count for an asset
       decrement     Decrement task count for an asset
     teams           Manage asset team assignments
       assign        Assign teams to an asset
       list          List asset team assignments
       show          Show team assignments for a specific asset
       add-contributor    Add a contributing team to an asset
       remove-contributor Remove a contributing team from an asset
   tasks              Manage tasks from various platforms
     fetch           Fetch tasks from a platform (e.g., Jira)
     show            Show tasks for a project and sprint
     classify        Classify tasks for a specific project and sprint
     inspect         Inspect a specific task by its key
     migrate         Migrate sprint data from comma-separated strings to arrays
   sprint             Manage sprint-related operations
     list            List sprints for a project and time period
     allocate        Calculate time allocation for JIRA issues in a sprint

For more information about a command:
   assetcap [command] --help`,
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "Show version information",
				Action: func(_ *cli.Context) error {
					fmt.Printf("AssetCap %s\n", version)
					fmt.Printf("Commit: %s\n", commit)
					fmt.Printf("Built: %s\n", date)
					return nil
				},
			},
			{
				Name:  "completion",
				Usage: "Generate shell completion scripts",
				Subcommands: []*cli.Command{
					{
						Name:  "bash",
						Usage: "Generate bash completion script",
						Action: func(_ *cli.Context) error {
							fmt.Println(completion.GetBashCompletion())
							return nil
						},
					},
					{
						Name:  "zsh",
						Usage: "Generate zsh completion script",
						Action: func(_ *cli.Context) error {
							fmt.Println(completion.GetZshCompletion())
							return nil
						},
					},
					{
						Name:  "fish",
						Usage: "Generate fish completion script",
						Action: func(_ *cli.Context) error {
							fmt.Println(completion.GetFishCompletion())
							return nil
						},
					},
				},
			},
			{
				Name:  "sprint",
				Usage: "Manage sprint-related operations",
				Subcommands: []*cli.Command{
					{
						Name:  "list",
						Usage: "List sprints for a project and time period",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							period := ctx.String("period")
							result, err := a.sprintService.ListSprints(project, period)
							if err != nil {
								return err
							}

							// Use the new formatter for colorful output
							formatter := formatting.NewOutputFormatter()
							output := formatter.FormatSprintList(project, period, result.Sprints, result.BoardInfo)
							fmt.Print(output)

							return nil
						},
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
						Name:  "allocate",
						Usage: "Calculate time allocation for JIRA issues in a sprint",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							sprint := ctx.String("sprint")
							override := ctx.String("override")
							sprintBounded := ctx.Bool("sprint-bounded")

							if sprintBounded {
								// Use the new sprint-bounded calculation
								result, err := a.sprintService.ProcessJiraIssuesWithStrategy(project, sprint, override, true)
								if err != nil {
									return err
								}
								fmt.Print(result)
							} else {
								// Use legacy calculation (default)
								result, err := a.sprintService.ProcessJiraIssues(project, sprint, override)
								if err != nil {
									return err
								}
								fmt.Print(result)
							}
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
								Value:   false,
							},
						},
					},
				},
			},
			{
				Name:  "assets",
				Usage: "Manage digital assets",
				Subcommands: []*cli.Command{
					{
						Name:  "create",
						Usage: "Create a new asset",
						Action: func(ctx *cli.Context) error {
							name := ctx.String("name")
							description := ctx.String("description")
							if err := a.assetService.CreateAsset(name, description); err != nil {
								return err
							}
							fmt.Printf("Created asset: %s\n", name)
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Usage:    "Asset name",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "description",
								Usage:    "Asset description",
								Required: true,
							},
						},
					},
					{
						Name:  "list",
						Usage: "List all assets",
						Action: func(_ *cli.Context) error {
							assets, err := a.assetService.ListAssets()
							if err != nil {
								return err
							}
							if len(assets) == 0 {
								fmt.Println("No assets found")
								return nil
							}
							fmt.Println("Assets:")
							for _, asset := range assets {
								fmt.Printf("- %s:\n", asset.Name)
								fmt.Printf("  Description: %s\n", asset.Description)
								fmt.Printf("  Why: %s\n", asset.Why)
								fmt.Printf("  Benefits: %s\n", asset.Benefits)
								fmt.Printf("  How: %s\n", asset.How)
								fmt.Printf("  Metrics: %s\n", asset.Metrics)
								if asset.DocLink != "" {
									fmt.Printf("  DocLink: %s\n", asset.DocLink)
								}
								// Show team information
								if asset.OwningTeam != "" {
									fmt.Printf("  👤 Owner: %s\n", asset.OwningTeam)
								}
								if len(asset.ContributingTeams) > 0 {
									fmt.Printf("  🤝 Contributors: %s\n", strings.Join(asset.ContributingTeams, ", "))
								}
								fmt.Println()
							}
							return nil
						},
					},
					{
						Name:  "sync",
						Usage: "Sync assets from Confluence",
						Action: func(ctx *cli.Context) error {
							space := ctx.String("space")
							label := ctx.String("label")
							debug := ctx.Bool("debug")

							result, err := a.assetService.SyncFromConfluence(space, label, debug)
							if err != nil {
								if strings.Contains(err.Error(), "no assets found with label") {
									fmt.Println(err)
									return nil
								}
								return err
							}

							totalAssets := len(result.SyncedAssets) + len(result.NotSyncedAssets)
							fmt.Printf("Successfully synced %d/%d assets from Confluence\n", len(result.SyncedAssets), totalAssets)

							if len(result.NotSyncedAssets) > 0 {
								fmt.Printf("\nWarning: %d assets could not be synced due to missing information:\n", len(result.NotSyncedAssets))
								for _, asset := range result.NotSyncedAssets {
									fmt.Printf("\n- %s:\n", asset.Name)
									fmt.Printf("  Missing fields: %s\n", strings.Join(asset.MissingFields, ", "))
									fmt.Println("  Available fields:")
									for field, value := range asset.AvailableFields {
										if value != "" {
											fmt.Printf("    %s: %s\n", field, value)
										}
									}
								}
							}

							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "space",
								Usage:    "Confluence space key (e.g. MZN)",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "label",
								Usage:    "Filter pages by label (e.g. cap-asset)",
								Required: true,
							},
							&cli.BoolFlag{
								Name:  "debug",
								Usage: "Enable debug logging",
								Value: false,
							},
						},
					},
					{
						Name:  "update",
						Usage: "Update an asset's description",
						Action: func(ctx *cli.Context) error {
							name := ctx.String("name")
							description := ctx.String("description")
							why := ctx.String("why")
							benefits := ctx.String("benefits")
							how := ctx.String("how")
							metrics := ctx.String("metrics")
							if err := a.assetService.UpdateAsset(name, description, why, benefits, how, metrics); err != nil {
								return err
							}
							fmt.Printf("✓ Updated asset: %s\n", name)
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Usage:    "Asset name",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "description",
								Usage:    "New asset description",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "why",
								Usage:    "Why are we doing this?",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "benefits",
								Usage:    "Economic benefits",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "how",
								Usage:    "How it works?",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "metrics",
								Usage:    "How do we judge success?",
								Required: true,
							},
						},
					},
					{
						Name:  "show",
						Usage: "Show detailed information about an asset",
						Action: func(ctx *cli.Context) error {
							name := ctx.String("name")
							asset, err := a.assetService.GetAsset(name)
							if err != nil {
								return err
							}
							fmt.Printf("Asset: %s\n", asset.Name)
							fmt.Printf("Description: %s\n", asset.Description)
							fmt.Printf("Why: %s\n", asset.Why)
							fmt.Printf("Benefits: %s\n", asset.Benefits)
							fmt.Printf("How: %s\n", asset.How)
							fmt.Printf("Metrics: %s\n", asset.Metrics)
							fmt.Printf("Created: %s\n", asset.CreatedAt.Format("2006-01-02 15:04:05"))
							fmt.Printf("Updated: %s\n", asset.UpdatedAt.Format("2006-01-02 15:04:05"))
							fmt.Printf("Task Count: %d\n", asset.AssociatedTaskCount)
							if len(asset.Keywords) > 0 {
								fmt.Printf("Keywords: %s\n", strings.Join(asset.Keywords, ", "))
							}
							if asset.DocLink != "" {
								fmt.Printf("DocLink: %s\n", asset.DocLink)
							}
							// Show team information
							if asset.OwningTeam != "" {
								fmt.Printf("👤 Owner: %s\n", asset.OwningTeam)
							}
							if len(asset.ContributingTeams) > 0 {
								fmt.Printf("🤝 Contributors: %s\n", strings.Join(asset.ContributingTeams, ", "))
							}
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Usage:    "Asset name",
								Required: true,
							},
						},
					},
					{
						Name:  "documentation",
						Usage: "Manage asset documentation",
						Subcommands: []*cli.Command{
							{
								Name:  "update",
								Usage: "Mark asset documentation as updated",
								Action: func(ctx *cli.Context) error {
									assetName := ctx.String("asset")
									// First check if the asset exists
									_, err := a.assetService.GetAsset(assetName)
									if err != nil {
										return err
									}
									if err := a.assetService.UpdateDocumentation(assetName); err != nil {
										return err
									}
									fmt.Printf("Marked documentation as updated for asset %s\n", assetName)
									return nil
								},
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "asset",
										Usage:    "Asset name",
										Required: true,
									},
								},
							},
						},
					},
					{
						Name:  "tasks",
						Usage: "Manage asset tasks",
						Subcommands: []*cli.Command{
							{
								Name:  "increment",
								Usage: "Increment task count for an asset",
								Action: func(ctx *cli.Context) error {
									assetName := ctx.String("asset")
									// First check if the asset exists
									_, err := a.assetService.GetAsset(assetName)
									if err != nil {
										return err
									}
									if err := a.assetService.IncrementTaskCount(assetName); err != nil {
										return err
									}
									fmt.Printf("Incremented task count for asset %s\n", assetName)
									return nil
								},
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "asset",
										Usage:    "Asset name",
										Required: true,
									},
								},
							},
							{
								Name:  "decrement",
								Usage: "Decrement task count for an asset",
								Action: func(ctx *cli.Context) error {
									assetName := ctx.String("asset")
									// First check if the asset exists
									_, err := a.assetService.GetAsset(assetName)
									if err != nil {
										return err
									}
									if err := a.assetService.DecrementTaskCount(assetName); err != nil {
										return err
									}
									fmt.Printf("Decremented task count for asset %s\n", assetName)
									return nil
								},
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "asset",
										Usage:    "Asset name",
										Required: true,
									},
								},
							},
						},
					},
					{
						Name:  "enrich",
						Usage: "Enrich asset fields using LLaMA 3",
						Action: func(ctx *cli.Context) error {
							name := ctx.String("name")
							field := ctx.String("field")
							if err := a.assetService.EnrichAsset(name, field); err != nil {
								return err
							}
							fmt.Printf("Enriched %s field for asset: %s\n", field, name)
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Usage:    "Asset name or ID",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "field",
								Usage:    "Field to enrich (e.g., description)",
								Required: true,
							},
						},
					},
					{
						Name:  "keywords",
						Usage: "Generate keywords for an asset using LLaMA 3",
						Action: func(ctx *cli.Context) error {
							name := ctx.String("name")
							// Check if asset exists
							_, err := a.assetService.GetAsset(name)
							if err != nil {
								return fmt.Errorf("asset not found: %s", name)
							}
							if err := a.assetService.GenerateKeywords(name); err != nil {
								return err
							}
							fmt.Printf("Generated keywords for asset: %s\n", name)
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Usage:    "Asset name or ID",
								Required: true,
							},
						},
					},
					{
						Name:  "teams",
						Usage: "Manage asset team assignments",
						Subcommands: []*cli.Command{
							{
								Name:  "assign",
								Usage: "Assign teams to an asset",
								Action: func(ctx *cli.Context) error {
									assetName := ctx.String("asset")
									owningTeam := ctx.String("owner")
									contributingTeamsInput := ctx.String("contributors")
									
									// Parse contributing teams from comma-separated string
									var contributingTeams []string
									if contributingTeamsInput != "" {
										teams := strings.Split(contributingTeamsInput, ",")
										for _, team := range teams {
											if trimmed := strings.TrimSpace(team); trimmed != "" {
												contributingTeams = append(contributingTeams, trimmed)
											}
										}
									}
									
									if err := a.assetService.AssignTeam(assetName, owningTeam, contributingTeams); err != nil {
										return err
									}
									
									fmt.Printf("✓ Successfully assigned teams to asset '%s'\n", assetName)
									if owningTeam != "" {
										fmt.Printf("  Owner: %s\n", owningTeam)
									}
									if len(contributingTeams) > 0 {
										fmt.Printf("  Contributors: %s\n", strings.Join(contributingTeams, ", "))
									}
									return nil
								},
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "asset",
										Usage:    "Asset name",
										Required: true,
									},
									&cli.StringFlag{
										Name:  "owner",
										Usage: "Owning team",
									},
									&cli.StringFlag{
										Name:  "contributors",
										Usage: "Contributing teams (comma-separated)",
									},
								},
							},
							{
								Name:  "list",
								Usage: "List asset team assignments",
								Action: func(ctx *cli.Context) error {
									assetTeams, err := a.assetService.GetAssetTeams()
									if err != nil {
										return err
									}
									
									if len(assetTeams) == 0 {
										fmt.Println("No team assignments found")
										return nil
									}
									
									fmt.Println("Asset Team Assignments:")
									fmt.Println("═══════════════════════════════════════")
									for _, info := range assetTeams {
										fmt.Printf("📦 %s\n", info.AssetName)
										if info.OwningTeam != "" {
											fmt.Printf("  👤 Owner: %s\n", info.OwningTeam)
										}
										if len(info.ContributingTeams) > 0 {
											fmt.Printf("  🤝 Contributors: %s\n", strings.Join(info.ContributingTeams, ", "))
										}
										fmt.Println()
									}
									return nil
								},
							},
							{
								Name:  "show",
								Usage: "Show team assignments for a specific asset",
								Action: func(ctx *cli.Context) error {
									assetName := ctx.String("asset")
									
									info, err := a.assetService.GetAssetTeamInfo(assetName)
									if err != nil {
										return err
									}
									
									fmt.Printf("Team Assignments for '%s':\n", info.AssetName)
									fmt.Println("─────────────────────────────────")
									if info.OwningTeam != "" {
										fmt.Printf("👤 Owner: %s\n", info.OwningTeam)
									} else {
										fmt.Println("👤 Owner: Not assigned")
									}
									
									if len(info.ContributingTeams) > 0 {
										fmt.Printf("🤝 Contributors: %s\n", strings.Join(info.ContributingTeams, ", "))
									} else {
										fmt.Println("🤝 Contributors: None")
									}
									
									return nil
								},
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "asset",
										Usage:    "Asset name",
										Required: true,
									},
								},
							},
							{
								Name:  "add-contributor",
								Usage: "Add a contributing team to an asset",
								Action: func(ctx *cli.Context) error {
									assetName := ctx.String("asset")
									teamName := ctx.String("team")
									
									if err := a.assetService.AddContributingTeam(assetName, teamName); err != nil {
										return err
									}
									
									fmt.Printf("✓ Added '%s' as contributor to asset '%s'\n", teamName, assetName)
									return nil
								},
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "asset",
										Usage:    "Asset name",
										Required: true,
									},
									&cli.StringFlag{
										Name:     "team",
										Usage:    "Team name to add as contributor",
										Required: true,
									},
								},
							},
							{
								Name:  "remove-contributor",
								Usage: "Remove a contributing team from an asset",
								Action: func(ctx *cli.Context) error {
									assetName := ctx.String("asset")
									teamName := ctx.String("team")
									
									if err := a.assetService.RemoveContributingTeam(assetName, teamName); err != nil {
										return err
									}
									
									fmt.Printf("✓ Removed '%s' as contributor from asset '%s'\n", teamName, assetName)
									return nil
								},
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "asset",
										Usage:    "Asset name",
										Required: true,
									},
									&cli.StringFlag{
										Name:     "team",
										Usage:    "Team name to remove as contributor",
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			{
				Name:  "tasks",
				Usage: "Manage tasks from various platforms",
				Subcommands: []*cli.Command{
					{
						Name:  "fetch",
						Usage: "Fetch tasks from a platform (e.g., Jira)",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							sprint := ctx.String("sprint")
							key := ctx.String("key")
							platform := ctx.String("platform")

							// Validate that either (project + sprint) or key is provided
							if key != "" {
								// Individual task fetch
								if project != "" || sprint != "" {
									return fmt.Errorf("when using --key, do not specify --project or --sprint")
								}
								if err := a.taskService.FetchTaskByKey(context.Background(), key, platform); err != nil {
									return err
								}
								fmt.Printf("✓ Successfully fetched task %s from %s\n", key, platform)
								return nil
							}

							// Sprint-based fetch
							if project == "" || sprint == "" {
								return fmt.Errorf("either --key or both --project and --sprint must be provided")
							}
							if err := a.taskService.FetchTasks(context.Background(), project, sprint, platform); err != nil {
								return err
							}
							fmt.Printf("✓ Successfully fetched tasks for project %s, sprint %s from %s\n", project, sprint, platform)
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "project",
								Usage: "Project key (e.g., FN) - required when not using --key",
							},
							&cli.StringFlag{
								Name:  "sprint",
								Usage: "Sprint name (e.g., Penguins) - required when not using --key",
							},
							&cli.StringFlag{
								Name:  "key",
								Usage: "Task key (e.g., FN-1015) - fetch individual task",
							},
							&cli.StringFlag{
								Name:     "platform",
								Usage:    "Platform to fetch tasks from (e.g., jira)",
								Required: true,
							},
						},
					},
					{
						Name:  "show",
						Usage: "Show tasks for a project and sprint",
						Action: func(ctx *cli.Context) error {
							asset := ctx.String("asset")
							if asset != "" {
								// Check if asset exists
								_, err := a.assetService.GetAsset(asset)
								if err != nil {
									return fmt.Errorf("asset not found: %s", asset)
								}

								tasks, err := a.taskService.GetTasksByAsset(ctx.Context, asset)
								if err != nil {
									return fmt.Errorf("failed to get tasks for asset %s: %w", asset, err)
								}

								fmt.Printf("Tasks for asset %s:\n", asset)
								fmt.Println("----------------------------------------")
								if len(tasks) == 0 {
									fmt.Println("No tasks found")
									return nil
								}

								for _, task := range tasks {
									fmt.Printf("Key: %s\nType: %s\nSummary: %s\nStatus: %s\nEpic: %s\nWork Type: %s\nLabels: %v\n\n",
										task.Key, task.Type, task.Summary, task.Status, task.Epic, task.WorkType, task.Labels)
								}
								return nil
							}

							project := ctx.String("project")
							sprint := ctx.String("sprint")

							if project == "" || sprint == "" {
								return fmt.Errorf("both project and sprint flags are required")
							}

							tasks, err := a.taskService.GetTasks(ctx.Context, project, sprint)
							if err != nil {
								return fmt.Errorf("failed to get tasks: %w", err)
							}

							if len(tasks) == 0 {
								fmt.Println("No tasks found")
								return nil
							}

							fmt.Printf("\nTasks for project %s and sprint %s:\n", project, sprint)
							fmt.Println("----------------------------------------")
							for _, task := range tasks {
								fmt.Printf("Key: %s\nType: %s\nSummary: %s\nStatus: %s\nEpic: %s\nWork Type: %s\nLabels: %v\n\n",
									task.Key, task.Type, task.Summary, task.Status, task.Epic, task.WorkType, task.Labels)
							}
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "project",
								Usage: "Project name",
							},
							&cli.StringFlag{
								Name:  "sprint",
								Usage: "Sprint name",
							},
							&cli.StringFlag{
								Name:  "asset",
								Usage: "Asset name or ID to filter tasks",
							},
						},
					},
					{
						Name:  "classify",
						Usage: "Classify tasks for a specific project and sprint",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							sprint := ctx.String("sprint")
							platform := ctx.String("platform")
							dryRun := ctx.Bool("dry-run")
							apply := ctx.Bool("apply")
							input := tasksdomain.ClassifyTasksInput{
								Project: project,
								Sprint:  sprint,
								DryRun:  dryRun,
								Apply:   apply,
							}
							if err := a.taskService.ClassifyTasks(context.Background(), input); err != nil {
								return err
							}
							if dryRun {
								fmt.Printf("✓ Classification preview completed for project %s, sprint %s\n", project, sprint)
								fmt.Printf("  Use --apply to write classifications to %s\n", platform)
							} else if apply {
								fmt.Printf("✓ Successfully classified and applied labels to tasks for project %s, sprint %s\n", project, sprint)
								fmt.Printf("  Labels written to %s with work types and asset associations\n", platform)
							} else {
								fmt.Printf("✓ Successfully classified tasks for project %s, sprint %s\n", project, sprint)
								fmt.Printf("  Classifications saved locally. Use --apply to write to %s\n", platform)
							}
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Usage:    "Project key (e.g., FN)",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "sprint",
								Usage:    "Sprint name (e.g., Penguins)",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "platform",
								Usage:    "Platform to classify tasks from (e.g., jira)",
								Required: true,
							},
							&cli.BoolFlag{
								Name:  "dry-run",
								Usage: "Preview classification without making changes",
								Value: false,
							},
							&cli.BoolFlag{
								Name:  "apply",
								Usage: "Write classifications back to Jira",
								Value: false,
							},
						},
					},
					{
						Name:  "inspect",
						Usage: "Inspect a specific task by its key",
						Action: func(ctx *cli.Context) error {
							key := ctx.String("key")
							if key == "" {
								return fmt.Errorf("task key is required")
							}

							task, err := a.taskService.GetTaskByKey(ctx.Context, key)
							if err != nil {
								return fmt.Errorf("failed to get task %s: %w", key, err)
							}

							if task == nil {
								fmt.Printf("Task %s not found\n", key)
								return nil
							}

							fmt.Printf("Task Details for %s:\n", key)
							fmt.Println("========================================")
							fmt.Printf("Key:           %s\n", task.Key)
							fmt.Printf("Type:          %s\n", task.Type)
							fmt.Printf("Summary:       %s\n", task.Summary)
							fmt.Printf("Status:        %s\n", task.Status)
							fmt.Printf("Project:       %s\n", task.Project)
							fmt.Printf("Sprint:        %s\n", task.Sprint)
							fmt.Printf("Epic:          %s\n", task.Epic)
							fmt.Printf("Work Type:     %s\n", task.WorkType)
							fmt.Printf("Priority:      %s\n", task.Priority)
							fmt.Printf("Platform:      %s\n", task.Platform)
							fmt.Printf("Labels:        %v\n", task.Labels)
							fmt.Printf("Created:       %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
							fmt.Printf("Updated:       %s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))
							fmt.Printf("Version:       %d\n", task.Version)

							if task.Description != "" {
								fmt.Printf("Description:\n%s\n", task.Description)
							}

							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "key",
								Usage:    "Task key (e.g., FN-1015)",
								Required: true,
							},
						},
					},
					{
						Name:  "migrate",
						Usage: "Migrate sprint data from comma-separated strings to arrays",
						Action: func(ctx *cli.Context) error {
							filePath := ctx.String("file")
							if filePath == "" {
								filePath = migration.DefaultTasksFilePath()
							}

							dryRun := ctx.Bool("dry-run")
							stats := ctx.Bool("stats")
							rollback := ctx.Bool("rollback")

							// Create storage repository
							storageDir := filepath.Dir(filePath)
							storageFile := filepath.Base(filePath)
							localStorage := storage.NewJSONStorage(storageDir, storageFile)

							// Create backup directory
							backupDir := filepath.Join(storageDir, "backups")

							migrator := migration.NewSprintMigration(localStorage, backupDir)

							if rollback {
								if err := migrator.RollbackMigration(context.Background()); err != nil {
									return fmt.Errorf("failed to rollback migration: %w", err)
								}
								fmt.Printf("✓ Successfully rolled back migration for %s\n", filePath)
								return nil
							}

							if stats {
								result, err := migrator.GetMigrationStats(context.Background())
								if err != nil {
									return fmt.Errorf("failed to get migration stats: %w", err)
								}

								fmt.Printf("Migration Statistics for %s:\n", filePath)
								fmt.Printf("========================================\n")
								fmt.Printf("Total tasks:      %d\n", result.TasksProcessed)
								fmt.Printf("Tasks to migrate: %d\n", result.Statistics["needs_migration"])
								fmt.Printf("Already migrated: %d\n", result.Statistics["already_migrated"])
								fmt.Printf("Migration %%:      %.1f%%\n", result.Statistics["migration_percentage"])
								return nil
							}

							// Validate compatibility before running migration
							if err := migrator.ValidateCompatibility(context.Background()); err != nil {
								return fmt.Errorf("migration compatibility check failed: %w", err)
							}

							result, err := migrator.MigrateToArrayFormat(context.Background(), dryRun)
							if err != nil {
								return fmt.Errorf("failed to migrate sprint data: %w", err)
							}

							fmt.Printf("Migration Results for %s:\n", filePath)
							fmt.Printf("========================================\n")
							fmt.Printf("Total tasks:       %d\n", result.TasksProcessed)
							fmt.Printf("Migrated tasks:    %d\n", result.TasksMigrated)
							fmt.Printf("Skipped tasks:     %d\n", result.TasksSkipped)
							if len(result.Errors) > 0 {
								fmt.Printf("Errors:            %d\n", len(result.Errors))
								for _, err := range result.Errors {
									fmt.Printf("  - %s\n", err)
								}
							}

							if dryRun {
								fmt.Printf("\n🔍 DRY RUN: No changes were made\n")
								fmt.Printf("   Run without --dry-run to apply changes\n")
							} else {
								if result.BackupCreated {
									fmt.Printf("Backup created:    %s\n", result.BackupPath)
								}
								fmt.Printf("\n✓ Migration completed successfully!\n")
								fmt.Printf("   Use --rollback to revert changes if needed\n")
							}

							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "file",
								Usage: "Path to tasks.json file (default: .assetcap/tasks.json)",
							},
							&cli.BoolFlag{
								Name:  "dry-run",
								Usage: "Preview migration without making changes",
								Value: false,
							},
							&cli.BoolFlag{
								Name:  "stats",
								Usage: "Show migration statistics without running migration",
								Value: false,
							},
							&cli.BoolFlag{
								Name:  "rollback",
								Usage: "Rollback previous migration using backup file",
								Value: false,
							},
						},
					},
				},
			},
			{
				Name:  "config",
				Usage: "Manage configuration settings",
				Subcommands: []*cli.Command{
					{
						Name:  "init",
						Usage: "Initialize configuration",
						Action: func(ctx *cli.Context) error {
							if a.configService == nil {
								return fmt.Errorf("configuration service not available")
							}

							interactive := !ctx.Bool("non-interactive")

							// Set environment variables if provided via flags
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
						},
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:  "non-interactive",
								Usage: "Run in non-interactive mode (requires environment variables)",
								Value: false,
							},
							&cli.StringFlag{
								Name:  "jira-url",
								Usage: "Jira base URL (e.g., https://company.atlassian.net)",
							},
							&cli.StringFlag{
								Name:  "jira-email",
								Usage: "Jira email address",
							},
							&cli.StringFlag{
								Name:  "jira-token",
								Usage: "Jira API token",
							},
						},
					},
					{
						Name:  "show",
						Usage: "Show current configuration",
						Action: func(_ *cli.Context) error {
							fmt.Println("Current Configuration:")
							fmt.Println("=====================")

							// Show environment variables (masked)
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

							// Show teams.json if it exists
							teamsPath := filepath.Join(configDir, teamsFile)
							if _, err := os.Stat(teamsPath); err == nil {
								fmt.Printf("\nTeam Configuration: %s exists\n", teamsPath)
							} else {
								fmt.Printf("\nTeam Configuration: %s not found\n", teamsPath)
							}

							return nil
						},
					},
					{
						Name:  "validate",
						Usage: "Validate current configuration",
						Action: func(_ *cli.Context) error {
							fmt.Println("Validating Configuration...")

							var errors []string

							// Check environment variables
							jiraURL := os.Getenv("JIRA_BASE_URL")
							jiraEmail := os.Getenv("JIRA_EMAIL")
							jiraToken := os.Getenv("JIRA_TOKEN")

							if jiraURL == "" {
								errors = append(errors, "JIRA_BASE_URL is not set")
							}
							if jiraEmail == "" {
								errors = append(errors, "JIRA_EMAIL is not set")
							}
							if jiraToken == "" {
								errors = append(errors, "JIRA_TOKEN is not set")
							}

							// Check teams.json
							teamsPath := filepath.Join(configDir, teamsFile)
							if _, err := os.Stat(teamsPath); err != nil {
								errors = append(errors, fmt.Sprintf("%s file not found", teamsPath))
							}

							if len(errors) > 0 {
								fmt.Println("❌ Configuration validation failed:")
								for _, err := range errors {
									fmt.Printf("  - %s\n", err)
								}
								return fmt.Errorf("configuration validation failed")
							}

							fmt.Println("✅ Configuration is valid")
							return nil
						},
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
				},
			},
		},
	}

	return app.Run(os.Args)
}

// maskToken masks sensitive token information for display
func maskToken(token string) string {
	if len(token) <= 4 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// initializeApp creates a new App instance with all dependencies
func initializeApp() (*App, error) {
	// Initialize shared configuration service first
	configRepo := configinfra.NewFileRepository(configDir)
	envProvider := configinfra.NewEnvironmentProvider()
	userInteraction := configinfra.NewCLIUserInteraction()
	initializeConfigUseCase := usecase.NewInitializeConfig(configRepo, envProvider, userInteraction)
	sharedConfigService := service.NewConfigService(configRepo)

	// Initialize repositories
	repoConfig := assetsinfra.RepositoryConfig{
		Directory: assetsDir,
		Filename:  assetsFile,
		FileMode:  0644,
		DirMode:   0755,
	}
	assetRepo := assetsinfra.NewJSONRepository(repoConfig)

	// Try to use shared configuration, fallback to legacy if config doesn't exist
	var assetService assetsapp.AssetService
	if configExists, _ := sharedConfigService.ConfigExists(); configExists {
		assetService = assetsapp.NewAssetService(assetRepo, sharedConfigService)
	} else {
		assetService = assetsapp.NewAssetServiceLegacy(assetRepo)
	}

	// Initialize task repositories with graceful fallback
	var jiraRepo taskports.TaskRepository
	var err error

	if configExists, _ := sharedConfigService.ConfigExists(); configExists {
		jiraRepo, err = jira.NewRepository(sharedConfigService)
		if err != nil {
			// Fallback to legacy if shared config fails
			jiraRepo, err = jira.NewRepositoryLegacy()
			if err != nil {
				return nil, fmt.Errorf("failed to initialize Jira repository: %v", err)
			}
		}
	} else {
		jiraRepo, err = jira.NewRepositoryLegacy()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Jira repository: %v", err)
		}
	}

	localRepo := storage.NewJSONStorage(tasksDir, tasksFile)

	// Initialize comprehensive classification system
	// Asset classifier for determining which asset a task belongs to
	assetClassifier := classifier.NewContentBasedAssetClassifier(assetRepo)

	// Work type classifier for determining capitalization category
	workTypeClassifier := classifier.NewBusinessRulesClassifier(assetRepo)

	// Comprehensive classification chain with subtask inheritance support
	classificationChain := classifier.NewComprehensiveClassificationChainWithInheritance(assetClassifier, workTypeClassifier)

	// Create adapter to bridge comprehensive results with existing use case interface
	taskClassifier := classifier.NewComprehensiveClassifierAdapter(classificationChain)

	userInput := cliui.NewUserInput()
	taskService := tasksapp.NewTasksService(jiraRepo, localRepo, taskClassifier, userInput, assetService)

	// Initialize sprint service
	jiraAdapter, err := sprintinfra.NewJiraAdapter()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Jira adapter: %v", err)
	}
	sprintService := sprintapp.NewSprintService(jiraAdapter)

	// Initialize config service for CLI
	configService := &configServiceImpl{
		initializeConfig: initializeConfigUseCase,
		configService:    sharedConfigService,
	}

	app := NewApp(assetService, taskService, sprintService)
	app.configService = configService
	return app, nil
}

func main() {
	// Handle version command without full initialization
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("assetcap version %s\n", version)
		if commit != "" {
			fmt.Printf("commit: %s\n", commit)
		}
		if date != "" {
			fmt.Printf("built: %s\n", date)
		}
		return
	}

	// Handle help command without full initialization
	if len(os.Args) <= 1 || os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help" {
		showHelp()
		return
	}

	// Initialize app only for business commands
	app, err := initializeApp()
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func showHelp() {
	fmt.Println("AssetCap - Digital Asset Capitalization Tool")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("   assetcap [global options] command [command options] [arguments...]")
	fmt.Println()
	fmt.Println("VERSION:")
	fmt.Printf("   %s\n", version)
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("   version     Show version information")
	fmt.Println("   completion  Generate shell completion scripts")
	fmt.Println("   config      Configure application settings")
	fmt.Println("   assets      Manage digital assets")
	fmt.Println("   tasks       Manage tasks and classification")
	fmt.Println("   sprint      Manage sprint-related operations")
	fmt.Println("   help        Show this help message")
	fmt.Println()
	fmt.Println("GLOBAL OPTIONS:")
	fmt.Println("   --help, -h  Show help")
	fmt.Println()
	fmt.Println("For detailed command help, use: assetcap [command] --help")
}
