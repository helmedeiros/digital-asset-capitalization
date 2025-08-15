package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	assetsusecase "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase"
	assetsinfra "github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure"
	assetid "github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/id"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
	configinfra "github.com/helmedeiros/digital-asset-capitalization/internal/config/infrastructure"
	investmentservice "github.com/helmedeiros/digital-asset-capitalization/internal/investment/application/service"
	investmentdomain "github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
	investmentinfra "github.com/helmedeiros/digital-asset-capitalization/internal/investment/infrastructure"
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
	assetService      assetsapp.AssetService
	taskService       tasksapp.TaskService
	sprintService     sprintapp.SprintService
	configService     ConfigService
	investmentService *investmentservice.InvestmentService
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
   investment         Calculate investment costs for digital assets
     init-cost-model Initialize cost model for a project
     set-engineer-rate Set hourly rate for a specific engineer
     show-rates      Show current engineer rates for a project
     calculate       Calculate investment for an asset across sprints
     sprint          Calculate investment for a specific sprint
     list            List all saved investment calculations

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
								Usage:    "Confluence space key(s). Single: 'MZN', Multiple: 'MZN,CAP,DOC', All: '*' or omit",
								Required: false,
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
						Name:  "sync-and-enrich",
						Usage: "Sync assets from Confluence and enrich them with keywords and fields using AI",
						Action: func(ctx *cli.Context) error {
							spaceKey := ctx.String("space")
							label := ctx.String("label")
							debug := ctx.Bool("debug")
							enrichKeywords := ctx.Bool("keywords")
							enrichFields := ctx.StringSlice("fields")
							dryRun := ctx.Bool("dry-run")

							// Note: These parameters will be used when we implement the full orchestration
							_ = ctx.String("field-filter") // fieldFilter for future use
							_ = ctx.Int("max-concurrent")  // maxConcurrent for future use

							// Validate required parameters
							if label == "" {
								return fmt.Errorf("label is required")
							}

							// Import the usecase package
							// Note: This will need proper dependency injection in a real implementation
							fmt.Printf("Starting sync-and-enrich workflow...\n")
							fmt.Printf("Space: %s, Label: %s, Keywords: %v, Fields: %v\n", spaceKey, label, enrichKeywords, enrichFields)

							if dryRun {
								fmt.Printf("DRY RUN: Would sync assets and enrich with keywords=%v, fields=%v\n", enrichKeywords, enrichFields)
								return nil
							}

							// For now, call the individual operations in sequence
							// TODO: Replace with proper orchestration use case

							// Step 1: Sync assets
							fmt.Printf("Step 1: Syncing assets from Confluence...\n")
							result, err := a.assetService.SyncFromConfluence(spaceKey, label, debug)
							if err != nil {
								return fmt.Errorf("failed to sync assets: %w", err)
							}

							fmt.Printf("Synced %d assets\n", len(result.SyncedAssets))

							// Step 2: Enrich keywords if requested
							if enrichKeywords && len(result.SyncedAssets) > 0 {
								fmt.Printf("Step 2: Generating keywords for synced assets...\n")
								for _, asset := range result.SyncedAssets {
									if err := a.assetService.GenerateKeywords(asset.Name); err != nil {
										fmt.Printf("Warning: Failed to generate keywords for %s: %v\n", asset.Name, err)
									} else {
										fmt.Printf("Generated keywords for: %s\n", asset.Name)
									}
								}
							}

							// Step 3: Enrich fields if requested
							if len(enrichFields) > 0 && len(result.SyncedAssets) > 0 {
								fmt.Printf("Step 3: Enriching fields %v for synced assets...\n", enrichFields)
								for _, asset := range result.SyncedAssets {
									for _, field := range enrichFields {
										if err := a.assetService.EnrichAsset(asset.Name, field); err != nil {
											fmt.Printf("Warning: Failed to enrich %s field for %s: %v\n", field, asset.Name, err)
										} else {
											fmt.Printf("Enriched %s field for: %s\n", field, asset.Name)
										}
									}
								}
							}

							fmt.Printf("Sync-and-enrich workflow completed successfully!\n")
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "space",
								Usage:    "Confluence space key(s). Single: 'MZN', Multiple: 'MZN,CAP,DOC', All: '*' or omit",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "label",
								Usage:    "Confluence label to filter by (e.g., 'cap-asset')",
								Required: true,
							},
							&cli.BoolFlag{
								Name:  "debug",
								Usage: "Enable debug output",
							},
							&cli.BoolFlag{
								Name:  "keywords",
								Usage: "Generate keywords for synced assets using AI",
							},
							&cli.StringSliceFlag{
								Name:  "fields",
								Usage: "Fields to enrich using AI (description, why, benefits, how, metrics). Can be specified multiple times",
							},
							&cli.StringFlag{
								Name:  "field-filter",
								Usage: "Filter for field enrichment: 'all', 'missing-fields', 'empty-fields'",
								Value: "missing-fields",
							},
							&cli.IntFlag{
								Name:  "max-concurrent",
								Usage: "Maximum concurrent AI operations",
								Value: 2,
							},
							&cli.BoolFlag{
								Name:  "dry-run",
								Usage: "Show what would be done without making changes",
							},
						},
					},
					{
						Name:  "sync-contributors",
						Usage: "Synchronize asset contributors from JIRA task assignments with optional filtering",
						Action: func(ctx *cli.Context) error {
							dryRun := ctx.Bool("dry-run")
							maxResults := ctx.Int("max-results")
							projectKey := ctx.String("project")
							sprintName := ctx.String("sprint")
							teamName := ctx.String("team")
							assetName := ctx.String("asset")

							// Create JIRA query adapter
							jiraQueryAdapter, err := assetsinfra.NewJiraQueryAdapter(a.configService)
							if err != nil {
								return fmt.Errorf("failed to create JIRA query adapter: %v", err)
							}

							// Create team config adapter
							configRepo := configinfra.NewFileRepository(configDir)
							teamConfigAdapter := assetsinfra.NewTeamConfigAdapter(configRepo)

							// Create asset repository
							repoConfig := assetsinfra.RepositoryConfig{
								Directory: assetsDir,
								Filename:  assetsFile,
								FileMode:  0644,
								DirMode:   0755,
							}
							assetRepo := assetsinfra.NewJSONRepository(repoConfig)

							// Create use case
							syncUseCase := assetsusecase.NewSyncAssetContributorsFromJiraUseCase(
								assetRepo,
								jiraQueryAdapter,
								teamConfigAdapter,
							)

							// Execute sync
							input := assetsusecase.SyncContributorsInput{
								DryRun:     dryRun,
								MaxResults: maxResults,
								ProjectKey: projectKey,
								SprintName: sprintName,
								TeamName:   teamName,
								AssetName:  assetName,
							}

							result, err := syncUseCase.Execute(context.Background(), input)
							if err != nil {
								return fmt.Errorf("failed to sync contributors: %v", err)
							}

							// Display results with context
							if projectKey != "" || sprintName != "" || teamName != "" || assetName != "" {
								fmt.Printf("🎯 Filtered sync")
								if projectKey != "" {
									fmt.Printf(" project:%s", projectKey)
								}
								if sprintName != "" {
									fmt.Printf(" sprint:%s", sprintName)
								}
								if teamName != "" {
									fmt.Printf(" team:%s", teamName)
								}
								if assetName != "" {
									fmt.Printf(" asset:%s", assetName)
								}
								fmt.Println()
							}

							fmt.Printf("🔍 Analyzed %d JIRA tasks with asset labels\n", result.TotalTasks)
							fmt.Printf("📦 Processed %d assets\n", len(result.AssetsProcessed))

							if dryRun {
								fmt.Println("\n🔍 DRY RUN - No changes were made")
							} else {
								fmt.Printf("✅ Updated %d assets\n", result.AssetsUpdated)
							}

							// Show details for each asset
							for _, assetResult := range result.AssetsProcessed {
								fmt.Printf("\n📦 %s (analyzed %d tasks)\n", assetResult.AssetName, assetResult.TasksAnalyzed)

								if assetResult.Error != "" {
									fmt.Printf("  ❌ Error: %s\n", assetResult.Error)
									continue
								}

								if len(assetResult.TeamsFound) > 0 {
									fmt.Printf("  🔍 Teams found: %s\n", strings.Join(assetResult.TeamsFound, ", "))
								}

								if len(assetResult.CurrentContributors) > 0 {
									fmt.Printf("  👥 Current contributors: %s\n", strings.Join(assetResult.CurrentContributors, ", "))
								}

								if len(assetResult.NewContributors) > 0 {
									fmt.Printf("  ➕ New contributors: %s\n", strings.Join(assetResult.NewContributors, ", "))
								}

								if len(assetResult.RemovedContributors) > 0 {
									fmt.Printf("  ➖ Removed contributors: %s\n", strings.Join(assetResult.RemovedContributors, ", "))
								}

								if assetResult.Updated {
									fmt.Printf("  ✅ Updated\n")
								} else if !dryRun {
									fmt.Printf("  ⏸️  No changes needed\n")
								}
							}

							if len(result.Errors) > 0 {
								fmt.Printf("\n⚠️  Errors encountered:\n")
								for _, error := range result.Errors {
									fmt.Printf("  - %s\n", error)
								}
							}

							return nil
						},
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:  "dry-run",
								Usage: "Preview changes without making modifications",
								Value: false,
							},
							&cli.IntFlag{
								Name:  "max-results",
								Usage: "Maximum number of JIRA tasks to analyze",
								Value: 1000,
							},
							&cli.StringFlag{
								Name:  "project",
								Usage: "Filter by JIRA project key (e.g., FN)",
							},
							&cli.StringFlag{
								Name:  "sprint",
								Usage: "Filter by sprint name (e.g., Penguins)",
							},
							&cli.StringFlag{
								Name:  "team",
								Usage: "Only sync assets that this team works on",
							},
							&cli.StringFlag{
								Name:  "asset",
								Usage: "Only sync contributors for this specific asset",
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
								Action: func(_ *cli.Context) error {
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
			{
				Name:  "investment",
				Usage: "Calculate investment costs for digital assets",
				Subcommands: []*cli.Command{
					{
						Name:  "calculate",
						Usage: "Calculate investment for an asset across sprints",
						Action: func(ctx *cli.Context) error {
							if a.investmentService == nil {
								return fmt.Errorf("investment service not available")
							}

							assetName := ctx.String("asset")
							project := ctx.String("project")
							sprintsStr := ctx.String("sprints")

							var sprints []string
							if sprintsStr != "" {
								sprints = strings.Split(sprintsStr, ",")
								for i, sprint := range sprints {
									sprints[i] = strings.TrimSpace(sprint)
								}
							}

							investment, err := a.investmentService.CalculateAssetInvestment(ctx.Context, assetName, project, sprints)
							if err != nil {
								return fmt.Errorf("failed to calculate investment: %w", err)
							}

							// Display results
							fmt.Printf("💰 Investment Calculation for '%s'\n", investment.AssetName)
							fmt.Printf("════════════════════════════════════════════════\n")
							fmt.Printf("Project: %s\n", investment.Project)
							if len(investment.Sprints) > 0 {
								fmt.Printf("Sprints: %s\n", strings.Join(investment.Sprints, ", "))
							}
							fmt.Printf("Period: %s → %s (%d days)\n",
								investment.StartDate.Format("2006-01-02"),
								investment.EndDate.Format("2006-01-02"),
								investment.GetDurationInDays())
							fmt.Printf("\n💵 Cost Breakdown:\n")
							fmt.Printf("  Engineer Costs:      %.2f %s\n", investment.EngineerCosts.Amount, investment.EngineerCosts.Currency)
							fmt.Printf("  Overhead Costs:      %.2f %s\n", investment.OverheadCosts.Amount, investment.OverheadCosts.Currency)
							fmt.Printf("  Infrastructure:      %.2f %s\n", investment.InfrastructureCosts.Amount, investment.InfrastructureCosts.Currency)
							fmt.Printf("  ─────────────────────────────────\n")
							fmt.Printf("  TOTAL INVESTMENT:    %.2f %s\n", investment.TotalCost.Amount, investment.TotalCost.Currency)

							fmt.Printf("\n👥 Engineers (%d):\n", len(investment.EngineersInvolved))
							for _, eng := range investment.EngineersInvolved {
								fmt.Printf("  %s (%s): %.1fh @ %.2f/h = %.2f %s\n",
									eng.Name, eng.Level, eng.TotalHours, eng.HourlyRate, eng.TotalCost.Amount, eng.TotalCost.Currency)
							}

							if len(investment.WorkTypeBreakdown) > 0 {
								fmt.Printf("\n🏗️  Work Type Breakdown:\n")
								for workType, cost := range investment.WorkTypeBreakdown {
									fmt.Printf("  %s: %.2f %s\n", workType, cost.Amount, cost.Currency)
								}
							}

							fmt.Printf("\n📊 Summary:\n")
							fmt.Printf("  Tasks: %d\n", investment.GetTaskCount())
							fmt.Printf("  Engineers: %d\n", investment.GetEngineerCount())
							fmt.Printf("  Duration: %d days\n", investment.GetDurationInDays())
							fmt.Printf("  Calculated: %s\n", investment.CalculatedAt.Format("2006-01-02 15:04:05"))

							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "asset",
								Usage:    "Asset name",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "project",
								Usage:    "Project key (e.g., FN)",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "sprints",
								Usage: "Comma-separated list of sprint names (e.g., 'Spiderman,Onça')",
							},
						},
					},
					{
						Name:  "sprint",
						Usage: "Calculate investment for a specific sprint",
						Action: func(ctx *cli.Context) error {
							if a.investmentService == nil {
								return fmt.Errorf("investment service not available")
							}

							project := ctx.String("project")
							sprint := ctx.String("sprint")

							// Parse dates
							startDateStr := ctx.String("start-date")
							endDateStr := ctx.String("end-date")

							var startDate, endDate time.Time
							var err error

							if startDateStr != "" {
								startDate, err = time.Parse("2006-01-02", startDateStr)
								if err != nil {
									return fmt.Errorf("invalid start date format (use YYYY-MM-DD): %w", err)
								}
							}

							if endDateStr != "" {
								endDate, err = time.Parse("2006-01-02", endDateStr)
								if err != nil {
									return fmt.Errorf("invalid end date format (use YYYY-MM-DD): %w", err)
								}
							}

							investment, err := a.investmentService.CalculateSprintInvestment(ctx.Context, project, sprint, startDate, endDate)
							if err != nil {
								return fmt.Errorf("failed to calculate sprint investment: %w", err)
							}

							// Display results (similar to asset calculation)
							fmt.Printf("💰 Sprint Investment Calculation\n")
							fmt.Printf("════════════════════════════════════════════════\n")
							fmt.Printf("Project: %s\n", investment.Project)
							fmt.Printf("Sprint: %s\n", sprint)
							fmt.Printf("Total Investment: %.2f %s\n", investment.TotalCost.Amount, investment.TotalCost.Currency)
							fmt.Printf("Engineers Involved: %d\n", investment.GetEngineerCount())
							fmt.Printf("Tasks: %d\n", investment.GetTaskCount())

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
								Usage:    "Sprint name (e.g., Spiderman)",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "start-date",
								Usage: "Sprint start date (YYYY-MM-DD)",
							},
							&cli.StringFlag{
								Name:  "end-date",
								Usage: "Sprint end date (YYYY-MM-DD)",
							},
						},
					},
					{
						Name:  "list",
						Usage: "List all saved investment calculations",
						Action: func(ctx *cli.Context) error {
							if a.investmentService == nil {
								return fmt.Errorf("investment service not available")
							}

							project := ctx.String("project")

							investments, err := a.investmentService.ListInvestments(ctx.Context, project)
							if err != nil {
								return fmt.Errorf("failed to list investments: %w", err)
							}

							if len(investments) == 0 {
								fmt.Printf("No investment calculations found")
								if project != "" {
									fmt.Printf(" for project %s", project)
								}
								fmt.Println()
								return nil
							}

							fmt.Printf("💰 Investment Calculations")
							if project != "" {
								fmt.Printf(" for %s", project)
							}
							fmt.Printf(" (%d found):\n", len(investments))
							fmt.Printf("════════════════════════════════════════════════\n")

							for _, inv := range investments {
								fmt.Printf("📦 %s (%s)\n", inv.AssetName, inv.Project)
								fmt.Printf("   Total: %.2f %s | Engineers: %d | Tasks: %d\n",
									inv.TotalCost.Amount, inv.TotalCost.Currency,
									inv.GetEngineerCount(), inv.GetTaskCount())
								if len(inv.Sprints) > 0 {
									fmt.Printf("   Sprints: %s\n", strings.Join(inv.Sprints, ", "))
								}
								fmt.Printf("   Calculated: %s\n\n", inv.CalculatedAt.Format("2006-01-02 15:04:05"))
							}

							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "project",
								Usage: "Filter by project key (e.g., FN)",
							},
						},
					},
					{
						Name:  "init-cost-model",
						Usage: "Initialize cost model for a project",
						Action: func(ctx *cli.Context) error {
							if a.investmentService == nil {
								return fmt.Errorf("investment service not available")
							}

							project := ctx.String("project")

							costModel, err := a.investmentService.InitializeCostModel(ctx.Context, project)
							if err != nil {
								return fmt.Errorf("failed to initialize cost model: %w", err)
							}

							fmt.Printf("✅ Cost model initialized for project %s\n", project)
							fmt.Printf("Currency: %s\n", costModel.Currency)
							fmt.Printf("Working hours per day: %.1f\n", costModel.WorkingHoursPerDay)
							fmt.Printf("Overhead multiplier: %.1fx\n", costModel.OverheadMultiplier)
							fmt.Printf("Default engineer rates:\n")
							for level, rate := range costModel.DefaultRatesByLevel {
								fmt.Printf("  %s: %.2f %s/hour\n", level, rate, costModel.Currency)
							}
							fmt.Printf("Monthly infrastructure costs: %.2f %s\n", costModel.GetTotalMonthlyCost(), costModel.Currency)

							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Usage:    "Project key (e.g., FN)",
								Required: true,
							},
						},
					},
					{
						Name:  "set-engineer-rate",
						Usage: "Set hourly rate for a specific engineer",
						Action: func(ctx *cli.Context) error {
							if a.investmentService == nil {
								return fmt.Errorf("investment service not available")
							}

							project := ctx.String("project")
							engineerName := ctx.String("engineer")
							rateStr := ctx.String("rate")
							levelStr := ctx.String("level")

							rate, err := strconv.ParseFloat(rateStr, 64)
							if err != nil {
								return fmt.Errorf("invalid rate: %w", err)
							}

							// Get current cost model
							costModel, err := a.investmentService.GetCostModel(ctx.Context, project)
							if err != nil {
								return fmt.Errorf("failed to get cost model: %w", err)
							}

							// Parse engineer level
							var level investmentdomain.EngineerLevel = investmentdomain.Mid // Default
							switch strings.ToLower(levelStr) {
							case "junior":
								level = investmentdomain.Junior
							case "mid":
								level = investmentdomain.Mid
							case "senior":
								level = investmentdomain.Senior
							case "staff":
								level = investmentdomain.Staff
							case "principal":
								level = investmentdomain.Principal
							}

							// Add engineer rate
							engineerRate := investmentdomain.EngineerRate{
								Name:       engineerName,
								HourlyRate: rate,
								Level:      level,
								Team:       project,
							}

							if err := costModel.AddEngineerRate(engineerRate); err != nil {
								return fmt.Errorf("failed to add engineer rate: %w", err)
							}

							// Save updated cost model
							if err := a.investmentService.UpdateCostModel(ctx.Context, project, costModel); err != nil {
								return fmt.Errorf("failed to save cost model: %w", err)
							}

							fmt.Printf("✅ Set rate for %s: %.2f %s/hour (%s level)\n",
								engineerName, rate, costModel.Currency, level)

							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Usage:    "Project key (e.g., FN)",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "engineer",
								Usage:    "Engineer name (e.g., 'Santhosh Balakrishnan')",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "rate",
								Usage:    "Hourly rate (e.g., 75.50)",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "level",
								Usage: "Engineer level (junior, mid, senior, staff, principal)",
								Value: "mid",
							},
						},
					},
					{
						Name:  "show-rates",
						Usage: "Show current engineer rates for a project",
						Action: func(ctx *cli.Context) error {
							if a.investmentService == nil {
								return fmt.Errorf("investment service not available")
							}

							project := ctx.String("project")

							costModel, err := a.investmentService.GetCostModel(ctx.Context, project)
							if err != nil {
								return fmt.Errorf("failed to get cost model: %w", err)
							}

							fmt.Printf("💰 Engineer Rates for %s\n", project)
							fmt.Printf("════════════════════════════════════════════════\n")
							fmt.Printf("Currency: %s | Working Hours/Day: %.1f | Overhead: %.1fx\n\n",
								costModel.Currency, costModel.WorkingHoursPerDay, costModel.OverheadMultiplier)

							// Show individual engineer rates
							if len(costModel.EngineerRates) > 0 {
								fmt.Printf("👥 Individual Engineer Rates:\n")
								for name, rate := range costModel.EngineerRates {
									fmt.Printf("  %s (%s): %.2f %s/hour\n",
										name, rate.Level, rate.HourlyRate, costModel.Currency)
								}
								fmt.Println()
							}

							// Show default rates by level
							fmt.Printf("📊 Default Rates by Level:\n")
							for level, rate := range costModel.DefaultRatesByLevel {
								fmt.Printf("  %s: %.2f %s/hour\n", level, rate, costModel.Currency)
							}

							fmt.Printf("\n🏢 Infrastructure Costs (Monthly):\n")
							fmt.Printf("  Cloud: %.2f %s\n", costModel.InfrastructureCosts.CloudCostsPerMonth, costModel.Currency)
							fmt.Printf("  Tooling: %.2f %s\n", costModel.InfrastructureCosts.ToolingCostsPerMonth, costModel.Currency)
							fmt.Printf("  Licenses: %.2f %s\n", costModel.InfrastructureCosts.LicenseCostsPerMonth, costModel.Currency)
							fmt.Printf("  Total: %.2f %s/month\n", costModel.GetTotalMonthlyCost(), costModel.Currency)

							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Usage:    "Project key (e.g., FN)",
								Required: true,
							},
						},
					},
				},
			},
			a.createConsoleCommand(),
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

	// Create ID generator
	idGenerator := assetid.NewHashIDGenerator()

	// Try to use shared configuration, fallback to legacy if config doesn't exist
	var assetService assetsapp.AssetService
	if configExists, _ := sharedConfigService.ConfigExists(); configExists {
		assetService = assetsapp.NewAssetService(assetRepo, sharedConfigService, idGenerator)
	} else {
		assetService = assetsapp.NewAssetServiceLegacy(assetRepo, idGenerator)
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

	// Initialize investment service
	costModelRepo := investmentinfra.NewCostModelJSONRepository(configDir)
	allocationProvider := investmentinfra.NewTimeAllocationAdapter()
	investmentRepo := investmentinfra.NewInvestmentJSONRepository(configDir)
	investmentService := investmentservice.NewInvestmentService(costModelRepo, allocationProvider, investmentRepo)

	app := NewApp(assetService, taskService, sprintService)
	app.configService = configService
	app.investmentService = investmentService
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
	fmt.Println("   console     Start AI-powered interactive console")
	fmt.Println("   help        Show this help message")
	fmt.Println()
	fmt.Println("GLOBAL OPTIONS:")
	fmt.Println("   --help, -h  Show help")
	fmt.Println()
	fmt.Println("For detailed command help, use: assetcap [command] --help")
}
