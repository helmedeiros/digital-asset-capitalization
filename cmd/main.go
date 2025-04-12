package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	assetsinfra "github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure"
	"github.com/helmedeiros/digital-asset-capitalization/internal/shell/completion"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application"
	sprintinfra "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/infrastructure"
	tasksapp "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/classifier"
	cliui "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/cli"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/jira"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/storage"
)

const (
	assetsDir  = ".assetcap"
	assetsFile = "assets.json"
	tasksDir   = ".assetcap"
	tasksFile  = "tasks.json"
	teamsFile  = "teams.json"
)

var assetService ports.AssetService
var taskService *tasksapp.TaskService
var sprintService *application.SprintService

func init() {
	// Initialize repositories
	config := assetsinfra.RepositoryConfig{
		Directory: assetsDir,
		Filename:  assetsFile,
		FileMode:  0644,
		DirMode:   0755,
	}
	assetRepo := assetsinfra.NewJSONRepository(config)
	assetService = assetsapp.NewAssetService(assetRepo)

	// Initialize task repositories
	jiraRepo, err := jira.NewRepository()
	if err != nil {
		log.Fatalf("Failed to initialize Jira repository: %v", err)
	}

	localRepo := storage.NewJSONStorage(tasksDir, tasksFile)
	taskClassifier := classifier.NewRandomClassifier()
	userInput := cliui.NewCLIUserInput()

	taskService = tasksapp.NewTasksService(jiraRepo, localRepo, taskClassifier, userInput)
}

func initJiraAdapter() error {
	// Initialize sprint service
	jiraAdapter, err := sprintinfra.NewJiraAdapter(teamsFile)
	if err != nil {
		return fmt.Errorf("failed to initialize Jira adapter: %v", err)
	}
	sprintService = application.NewSprintService(jiraAdapter)
	return nil
}

func Run() error {
	app := &cli.App{
		Name:                 "AssetCap",
		Usage:                "Digital Asset Capitalization Management Tool",
		EnableBashCompletion: true,
		UsageText: `assetcap [global options] command [command options] [arguments...]

COMMANDS:
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
     allocate        Calculate time allocation for JIRA issues in a sprint

For more information about a command:
   assetcap [command] --help`,
		Commands: []*cli.Command{
			{
				Name:  "completion",
				Usage: "Generate shell completion scripts",
				Subcommands: []*cli.Command{
					{
						Name:  "bash",
						Usage: "Generate bash completion script",
						Action: func(c *cli.Context) error {
							fmt.Println(completion.GetBashCompletion())
							return nil
						},
					},
					{
						Name:  "zsh",
						Usage: "Generate zsh completion script",
						Action: func(c *cli.Context) error {
							fmt.Println(completion.GetZshCompletion())
							return nil
						},
					},
					{
						Name:  "fish",
						Usage: "Generate fish completion script",
						Action: func(c *cli.Context) error {
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
						Name:  "allocate",
						Usage: "Calculate time allocation for JIRA issues in a sprint",
						Action: func(ctx *cli.Context) error {
							project := ctx.String("project")
							sprint := ctx.String("sprint")
							override := ctx.String("override")
							if err := initJiraAdapter(); err != nil {
								return err
							}
							result, err := sprintService.ProcessJiraIssues(project, sprint, override)
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
							name := ctx.Value("name").(string)
							description := ctx.Value("description").(string)
							if err := assetService.CreateAsset(name, description); err != nil {
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
						Action: func(ctx *cli.Context) error {
							assets, err := assetService.ListAssets()
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

							result, err := assetService.SyncFromConfluence(space, label, debug)
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
							name := ctx.Value("name").(string)
							description := ctx.Value("description").(string)
							why := ctx.Value("why").(string)
							benefits := ctx.Value("benefits").(string)
							how := ctx.Value("how").(string)
							metrics := ctx.Value("metrics").(string)
							if err := assetService.UpdateAsset(name, description, why, benefits, how, metrics); err != nil {
								return err
							}
							fmt.Printf("Updated asset: %s\n", name)
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
							asset, err := assetService.GetAsset(name)
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
									assetName := ctx.Value("asset").(string)
									if err := assetService.UpdateDocumentation(assetName); err != nil {
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
									assetName := ctx.Value("asset").(string)
									if err := assetService.IncrementTaskCount(assetName); err != nil {
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
									assetName := ctx.Value("asset").(string)
									if err := assetService.DecrementTaskCount(assetName); err != nil {
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
							if err := assetService.EnrichAsset(name, field); err != nil {
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
							project := ctx.Value("project").(string)
							sprint := ctx.Value("sprint").(string)
							platform := ctx.Value("platform").(string)
							if err := taskService.FetchTasks(context.Background(), project, sprint, platform); err != nil {
								return err
							}
							fmt.Printf("Successfully fetched tasks for project %s, sprint %s from %s\n", project, sprint, platform)
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
								_, err := assetService.GetAsset(asset)
								if err != nil {
									return fmt.Errorf("asset not found: %s", asset)
								}

								tasks, err := taskService.GetTasksByAsset(ctx.Context, asset)
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

							tasks, err := taskService.GetTasks(ctx.Context, project, sprint)
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
							project := ctx.Value("project").(string)
							sprint := ctx.Value("sprint").(string)
							platform := ctx.Value("platform").(string)
							dryRun := ctx.Value("dry-run").(bool)
							apply := ctx.Value("apply").(bool)
							input := usecase.ClassifyTasksInput{
								Project: project,
								Sprint:  sprint,
								DryRun:  dryRun,
								Apply:   apply,
							}
							if err := taskService.ClassifyTasks(context.Background(), input); err != nil {
								return err
							}
							if dryRun {
								fmt.Printf("Preview: Would classify tasks for project %s, sprint %s from %s\n", project, sprint, platform)
							} else if apply {
								fmt.Printf("Successfully classified and applied labels to tasks for project %s, sprint %s from %s\n", project, sprint, platform)
							} else {
								fmt.Printf("Successfully classified tasks for project %s, sprint %s from %s\n", project, sprint, platform)
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
		},
	}

	return app.Run(os.Args)
}

func main() {
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}
