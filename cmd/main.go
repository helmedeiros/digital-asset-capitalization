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
	configinfra "github.com/helmedeiros/digital-asset-capitalization/internal/config/infrastructure"
	"github.com/helmedeiros/digital-asset-capitalization/internal/shell/completion"
	sprintapp "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application"
	sprintinfra "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/infrastructure"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/infrastructure/formatting"
	tasksapp "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	taskports "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/classifier"
	cliui "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/cli"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/jira"
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
}

// configServiceImpl implements ConfigService
type configServiceImpl struct {
	initializeConfig *usecase.InitializeConfig
}

func (c *configServiceImpl) InitializeConfig(interactive bool) (*usecase.InitializeConfigResult, error) {
	return c.initializeConfig.Execute(interactive)
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
   config             Manage configuration settings
     init            Initialize configuration interactively
     show            Show current configuration
     validate        Validate current configuration
   assets              Manage digital assets
     create           Create a new asset
     list            List all assets
     documentation   Manage asset documentation
       update        Mark asset documentation as updated
     tasks           Manage asset tasks
       increment     Increment task count for an asset
       decrement     Decrement task count for an asset
   tasks              Manage tasks from various platforms
     fetch           Fetch tasks from a platform (e.g., Jira)
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
							result, err := a.sprintService.ProcessJiraIssues(project, sprint, override)
							if err != nil {
								return err
							}
							fmt.Print(result)
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
							platform := ctx.String("platform")
							if err := a.taskService.FetchTasks(context.Background(), project, sprint, platform); err != nil {
								return err
							}
							fmt.Printf("✓ Successfully fetched tasks for project %s, sprint %s from %s\n", project, sprint, platform)
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
							input := domain.ClassifyTasksInput{
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

	// Comprehensive classification chain that orchestrates both classifiers
	classificationChain := classifier.NewComprehensiveClassificationChain(assetClassifier, workTypeClassifier)

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
	fmt.Println("   assets      Manage digital assets")
	fmt.Println("   tasks       Manage tasks and classification")
	fmt.Println("   sprints     Manage sprints and time allocation")
	fmt.Println("   config      Configure application settings")
	fmt.Println("   version     Show version information")
	fmt.Println("   help        Show this help message")
	fmt.Println()
	fmt.Println("GLOBAL OPTIONS:")
	fmt.Println("   --help, -h  Show help")
	fmt.Println()
	fmt.Println("For detailed command help, use: assetcap [command] --help")
}
