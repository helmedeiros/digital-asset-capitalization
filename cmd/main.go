package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	assetsusecase "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase"
	assetsinfra "github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure"
	assetid "github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/id"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/llama"
	configapp "github.com/helmedeiros/digital-asset-capitalization/internal/config/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
	configinfra "github.com/helmedeiros/digital-asset-capitalization/internal/config/infrastructure"
	investmentservice "github.com/helmedeiros/digital-asset-capitalization/internal/investment/application/service"
	investmentdomain "github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
	investmentinfra "github.com/helmedeiros/digital-asset-capitalization/internal/investment/infrastructure"
	sprintapp "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application"
	sprintusecase "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application/usecase"
	sprintinfra "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/infrastructure"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/infrastructure/formatting"
	tasksapp "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application"
	tasksusecase "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase"
	tasksdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
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
	assetService       assetsapp.AssetService
	taskService        tasksapp.TaskService
	sprintService      sprintapp.SprintService
	configService      ConfigService
	investmentService  *investmentservice.InvestmentService
	teamResolver       *configapp.TeamResolverService
	taskRepo           taskports.TaskRepository
	taskClassifier     taskports.TaskClassifier
	allocationLockRepo taskports.SprintLockRepository
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
   deployments        Manage deployment tracking
     record          Record a new deployment
     list            List deployments
     history         Show deployment history for an asset
     timeline        Show deployment timeline for a time range
     mock            Generate mock deployment data for testing

For more information about a command:
   assetcap [command] --help`,
		Commands: []*cli.Command{
			a.createVersionCommand(),
			a.createCompletionCommand(),
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

							// Resolve team identifier to actual project code
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
							workStreamsStr := ctx.String("work-streams")
							withHours := ctx.Bool("with-hours")
							apply := ctx.Bool("apply")
							dryRun := ctx.Bool("dry-run")
							force := ctx.Bool("force")

							// Resolve team identifier to actual project code
							if a.teamResolver != nil && project != "" {
								resolvedProject, err := a.teamResolver.ResolveProjectIdentifier(project)
								if err != nil {
									return fmt.Errorf("unknown project or team nickname: %s", project)
								}
								project = resolvedProject
							}

							// Parse work streams
							var workStreams []string
							if workStreamsStr != "" {
								for _, ws := range strings.Split(workStreamsStr, ",") {
									trimmed := strings.TrimSpace(ws)
									if trimmed != "" {
										workStreams = append(workStreams, trimmed)
									}
								}
							}

							// Build options
							var opts []sprintusecase.SprintAllocationOption
							if len(workStreams) > 0 {
								opts = append(opts, sprintusecase.WithWorkStreams(workStreams))
							}
							if withHours {
								opts = append(opts, sprintusecase.WithHours(true))
							}

							// If --apply is set, use the push-to-JIRA flow
							if apply {
								return a.handleAllocateApply(project, sprint, override, sprintBounded, dryRun, force, opts)
							}

							if len(opts) > 0 || sprintBounded {
								result, err := a.sprintService.ProcessJiraIssuesWithOptions(project, sprint, override, sprintBounded, opts...)
								if err != nil {
									return err
								}
								fmt.Print(result)
							} else {
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
						Name:  "delete",
						Usage: "Delete an existing asset",
						Action: func(ctx *cli.Context) error {
							name := ctx.String("name")
							deletePage := ctx.Bool("delete-page")
							if err := a.assetService.DeleteAsset(name, deletePage); err != nil {
								return err
							}
							fmt.Printf("Deleted asset: %s\n", name)
							if deletePage {
								fmt.Printf("Confluence page also deleted.\n")
							}
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Usage:    "Asset name",
								Required: true,
							},
							&cli.BoolFlag{
								Name:  "delete-page",
								Usage: "Also delete the associated Confluence page",
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
						Name:  "publish",
						Usage: "Publish an asset to Confluence as a new page",
						Action: func(ctx *cli.Context) error {
							name := ctx.String("name")
							space := ctx.String("space")
							parentPage := ctx.String("parent-page")
							dryRun := ctx.Bool("dry-run")
							debug := ctx.Bool("debug")

							result, err := a.assetService.PublishToConfluence(context.Background(), name, space, parentPage, dryRun, debug)
							if err != nil {
								return err
							}

							if dryRun {
								fmt.Printf("DRY RUN: Would create page for asset '%s' in space '%s'\n", result.AssetName, result.SpaceKey)
								fmt.Printf("Labels to add: %v\n", result.Labels)
								fmt.Printf("\nPreview of page content:\n")
								fmt.Println("────────────────────────────────────────")
								fmt.Println(result.Preview)
								fmt.Println("────────────────────────────────────────")
								return nil
							}

							fmt.Printf("Successfully published asset '%s' to Confluence\n", result.AssetName)
							fmt.Printf("  Page ID: %s\n", result.PageID)
							fmt.Printf("  Space: %s\n", result.SpaceKey)
							fmt.Printf("  URL: %s\n", result.PageURL)
							fmt.Printf("  Labels: %v\n", result.Labels)
							if result.DocLinkSaved {
								fmt.Printf("  DocLink updated in asset\n")
							}
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Usage:    "Asset name to publish",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "space",
								Usage:    "Confluence space key (e.g., Conversion)",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "parent-page",
								Usage: "Parent page ID to create under (overrides team config)",
							},
							&cli.BoolFlag{
								Name:  "dry-run",
								Usage: "Preview without creating the page",
								Value: false,
							},
							&cli.BoolFlag{
								Name:  "debug",
								Usage: "Enable debug output",
								Value: false,
							},
						},
					},
					{
						Name:  "update-confluence",
						Usage: "Update an existing Confluence page with the asset's current content",
						Action: func(ctx *cli.Context) error {
							name := ctx.String("name")
							dryRun := ctx.Bool("dry-run")
							debug := ctx.Bool("debug")

							result, err := a.assetService.UpdateConfluencePage(context.Background(), name, dryRun, debug)
							if err != nil {
								return err
							}

							if dryRun {
								fmt.Printf("DRY RUN: Would update page for asset '%s'\n", result.AssetName)
								fmt.Printf("  Page ID: %s\n", result.PageID)
								fmt.Printf("  Space: %s\n", result.SpaceKey)
								fmt.Printf("\nPreview of page content:\n")
								fmt.Println("────────────────────────────────────────")
								fmt.Println(result.Preview)
								fmt.Println("────────────────────────────────────────")
								return nil
							}

							fmt.Printf("Successfully updated Confluence page for asset '%s'\n", result.AssetName)
							fmt.Printf("  Page ID: %s\n", result.PageID)
							fmt.Printf("  Space: %s\n", result.SpaceKey)
							fmt.Printf("  URL: %s\n", result.PageURL)
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Usage:    "Asset name to update",
								Required: true,
							},
							&cli.BoolFlag{
								Name:  "dry-run",
								Usage: "Preview without updating the page",
								Value: false,
							},
							&cli.BoolFlag{
								Name:  "debug",
								Usage: "Enable debug output",
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
							maxConcurrent := ctx.Int("max-concurrent")
							if maxConcurrent < 1 {
								maxConcurrent = 1
							}
							// field-filter is reserved for the future bulk use case path;
							// the inline enrichment here always touches every synced asset.
							_ = ctx.String("field-filter")

							// Validate required parameters
							if label == "" {
								return fmt.Errorf("label is required")
							}

							fmt.Printf("Starting sync-and-enrich workflow...\n")
							fmt.Printf("Space: %s, Label: %s, Keywords: %v, Fields: %v, MaxConcurrent: %d\n", spaceKey, label, enrichKeywords, enrichFields, maxConcurrent)

							if dryRun {
								fmt.Printf("DRY RUN: Would sync assets and enrich with keywords=%v, fields=%v\n", enrichKeywords, enrichFields)
								return nil
							}

							// Step 1: Sync assets
							fmt.Printf("Step 1: Syncing assets from Confluence...\n")
							result, err := a.assetService.SyncFromConfluence(spaceKey, label, debug)
							if err != nil {
								return fmt.Errorf("failed to sync assets: %w", err)
							}

							fmt.Printf("Synced %d assets\n", len(result.SyncedAssets))

							// Step 2: Enrich keywords if requested
							if enrichKeywords && len(result.SyncedAssets) > 0 {
								fmt.Printf("Step 2: Generating keywords for synced assets (max %d in flight)...\n", maxConcurrent)
								eg := new(errgroup.Group)
								eg.SetLimit(maxConcurrent)
								for _, asset := range result.SyncedAssets {
									asset := asset
									eg.Go(func() error {
										if err := a.assetService.GenerateKeywords(asset.Name); err != nil {
											fmt.Printf("Warning: Failed to generate keywords for %s: %v\n", asset.Name, err)
										} else {
											fmt.Printf("Generated keywords for: %s\n", asset.Name)
										}
										return nil
									})
								}
								_ = eg.Wait()
							}

							// Step 3: Enrich fields if requested
							if len(enrichFields) > 0 && len(result.SyncedAssets) > 0 {
								fmt.Printf("Step 3: Enriching fields %v for synced assets (max %d in flight)...\n", enrichFields, maxConcurrent)
								eg := new(errgroup.Group)
								eg.SetLimit(maxConcurrent)
								for _, asset := range result.SyncedAssets {
									for _, field := range enrichFields {
										asset, field := asset, field
										eg.Go(func() error {
											if err := a.assetService.EnrichAsset(asset.Name, field); err != nil {
												fmt.Printf("Warning: Failed to enrich %s field for %s: %v\n", field, asset.Name, err)
											} else {
												fmt.Printf("Enriched %s field for: %s\n", field, asset.Name)
											}
											return nil
										})
									}
								}
								_ = eg.Wait()
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
			a.createTasksCommand(),
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
					{
						Name:  "team-nicknames",
						Usage: "Manage team nicknames",
						Subcommands: []*cli.Command{
							{
								Name:  "add",
								Usage: "Add nicknames for a project",
								Action: func(ctx *cli.Context) error {
									project := ctx.String("project")
									nickname := ctx.String("nickname")

									if project == "" {
										return fmt.Errorf("project is required")
									}
									if nickname == "" {
										return fmt.Errorf("nickname is required")
									}

									// Split comma-separated nicknames
									nicknames := strings.Split(nickname, ",")
									for i, nick := range nicknames {
										nicknames[i] = strings.TrimSpace(nick)
									}

									fmt.Printf("⚠️  Note: Nickname management requires implementation of team config update functionality\n")
									fmt.Printf("Would add nickname(s) %s for project %s\n",
										strings.Join(nicknames, ", "), project)

									return nil
								},
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "project",
										Usage:    "Project key (e.g., FN)",
										Required: true,
									},
									&cli.StringFlag{
										Name:     "nickname",
										Usage:    "Comma-separated nicknames (e.g., 'Pricing,Fintech')",
										Required: true,
									},
								},
							},
							{
								Name:  "list",
								Usage: "List all team nicknames",
								Action: func(_ *cli.Context) error {
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

									// Group by project
									projectMap := make(map[string][]string)
									for nickname, project := range mappings {
										projectMap[project] = append(projectMap[project], nickname)
									}

									for project, nicks := range projectMap {
										fmt.Printf("%s: %s\n", project, strings.Join(nicks, ", "))
									}

									return nil
								},
							},
							{
								Name:  "show",
								Usage: "Show nicknames for a specific project",
								Action: func(ctx *cli.Context) error {
									project := ctx.String("project")
									if project == "" {
										return fmt.Errorf("project is required")
									}

									if a.teamResolver == nil {
										return fmt.Errorf("team resolver not available")
									}

									// Resolve project to ensure it exists
									resolvedProject, err := a.teamResolver.ResolveProjectIdentifier(project)
									if err != nil {
										return fmt.Errorf("unknown project: %s", project)
									}

									displayName := a.teamResolver.GetProjectWithNicknames(resolvedProject)
									fmt.Printf("Project: %s\n", displayName)

									return nil
								},
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "project",
										Usage:    "Project key or nickname (e.g., FN or Pricing)",
										Required: true,
									},
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

							// Resolve team identifier to actual project code
							if a.teamResolver != nil && project != "" {
								resolvedProject, err := a.teamResolver.ResolveProjectIdentifier(project)
								if err != nil {
									return fmt.Errorf("unknown project or team nickname: %s", project)
								}
								project = resolvedProject
							}

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
			a.createDeploymentCommands(),
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

// handleAllocateApply handles the --apply flow for sprint allocate
func (a *App) handleAllocateApply(project, sprint, override string, sprintBounded, dryRun, force bool, opts []sprintusecase.SprintAllocationOption) error {
	bgCtx := context.Background()

	// Lock check (skip for dry-run)
	if !dryRun && a.allocationLockRepo != nil {
		lockKey := fmt.Sprintf("alloc::%s", sprint)
		existing, err := a.allocationLockRepo.FindLock(bgCtx, project, lockKey)
		if err != nil {
			return fmt.Errorf("failed to check allocation lock: %w", err)
		}
		if existing != nil {
			if !force {
				return fmt.Errorf("sprint %q in project %q was already pushed on %s (%d issues). Use --force to override",
					sprint, project, existing.LockedAt.Format("2006-01-02"), existing.TaskCount)
			}
			fmt.Printf("Warning: sprint %q in project %q was already pushed on %s (%d issues).\n",
				sprint, project, existing.LockedAt.Format("2006-01-02"), existing.TaskCount)
			fmt.Print("Are you sure you want to re-push? [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}
	}

	csvData, pushResult, err := a.sprintService.PushAllocationToJira(project, sprint, override, sprintBounded, dryRun, opts...)
	if err != nil {
		return err
	}

	// Print CSV output
	fmt.Print(csvData)

	// Print push summary table
	if pushResult != nil && len(pushResult.Details) > 0 {
		fmt.Println("\nAllocation Push Summary:")
		fmt.Printf("%-12s | %-20s | %-15s | %-15s | %-12s | %s\n",
			"Issue", "Field", "Current", "New", "Status", "Reason")
		fmt.Println(strings.Repeat("-", 95))
		for _, d := range pushResult.Details {
			oldVal := d.OldValue
			if oldVal == "" {
				oldVal = "(empty)"
			}
			newVal := d.NewValue
			if newVal == "" {
				newVal = ""
			}
			fmt.Printf("%-12s | %-20s | %-15s | %-15s | %-12s | %s\n",
				d.IssueKey, d.Field, oldVal, newVal, d.Status, d.Reason)
		}
		fmt.Printf("\nUpdated: %d  Skipped: %d  Errors: %d\n",
			pushResult.UpdatedCount, pushResult.SkippedCount, pushResult.ErrorCount)
	}

	if dryRun {
		fmt.Println("\nDry run complete. Remove --dry-run to push changes.")
		return nil
	}

	// Save allocation lock after successful push
	if a.allocationLockRepo != nil && pushResult != nil && pushResult.UpdatedCount > 0 {
		lockKey := fmt.Sprintf("alloc::%s", sprint)
		lock := tasksdomain.NewSprintLock(project, lockKey, pushResult.UpdatedCount)
		if err := a.allocationLockRepo.SaveLock(bgCtx, lock); err != nil {
			fmt.Printf("Warning: failed to save allocation lock: %v\n", err)
		}
	}

	return nil
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

	// Embedding-based asset classifier for comparison mode (optional, uses Ollama)
	llamaConfig := llama.DefaultConfig()
	var llmClassifier taskports.AssetClassifier
	if llamaConfig.BaseURL != "" {
		llamaClient, llamaErr := llama.NewClient(llamaConfig)
		if llamaErr == nil {
			embeddingService := classifier.NewOllamaEmbeddingAdapter(llamaClient, "nomic-embed-text")
			embeddingStorePath := filepath.Join(configDir, "embeddings.json")
			embeddingStore, storeErr := classifier.NewEmbeddingStore(embeddingStorePath)
			if storeErr == nil {
				llmClassifier = classifier.NewEmbeddingAssetClassifier(embeddingService, assetRepo, embeddingStore)
			}
		}
	}

	// Comprehensive classification chain with subtask inheritance and optional LLM support
	classificationChain := classifier.NewComprehensiveClassificationChainWithLLM(assetClassifier, workTypeClassifier, llmClassifier)

	// Create adapter to bridge comprehensive results with existing use case interface
	taskClassifier := classifier.NewComprehensiveClassifierAdapter(classificationChain)

	userInput := cliui.NewUserInput()

	// Initialize sprint service first (needed for SprintResolver)
	jiraAdapter, err := sprintinfra.NewJiraAdapter()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Jira adapter: %v", err)
	}
	sprintService := sprintapp.NewSprintService(jiraAdapter)

	// Create SprintResolver with interactive selection capability
	sprintResolver := tasksusecase.NewSprintResolver(jiraAdapter, userInput)

	// Initialize sprint lock storage
	sprintLockStorage := storage.NewSprintLockStorage(tasksDir, "sprint_locks.json")

	// Initialize task service with SprintResolver
	taskService := tasksapp.NewTasksService(jiraRepo, localRepo, taskClassifier, userInput, assetService, sprintResolver, sprintLockStorage)

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

	// Initialize team resolver
	teamConfig, err := sharedConfigService.GetTeamConfig()
	if err != nil {
		// Create empty team config if not exists
		teamConfig, _ = domain.NewTeamConfig(make(map[string][]string))
	}
	teamResolver := configapp.NewTeamResolverService(teamConfig)

	// Initialize allocation lock storage (separate from classification locks)
	allocationLockStorage := storage.NewSprintLockStorage(tasksDir, "sprint_allocation_locks.json")

	app := NewApp(assetService, taskService, sprintService)
	app.configService = configService
	app.investmentService = investmentService
	app.teamResolver = teamResolver
	app.taskRepo = jiraRepo
	app.taskClassifier = taskClassifier
	app.allocationLockRepo = allocationLockStorage
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
	fmt.Println("   investment  Calculate investment costs")
	fmt.Println("   deployments Manage deployment tracking")
	fmt.Println("   console     Start AI-powered interactive console")
	fmt.Println("   help        Show this help message")
	fmt.Println()
	fmt.Println("GLOBAL OPTIONS:")
	fmt.Println("   --help, -h  Show help")
	fmt.Println()
	fmt.Println("For detailed command help, use: assetcap [command] --help")
}
