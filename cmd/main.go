package main

import (
	"context"
	"fmt"
	"log"
	"os"

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
	"github.com/urfave/cli/v2"
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
								fmt.Printf("- %s: %s\n", asset.Name, asset.Description)
							}
							return nil
						},
					},
					{
						Name:  "update",
						Usage: "Update an asset's description",
						Action: func(ctx *cli.Context) error {
							name := ctx.Value("name").(string)
							description := ctx.Value("description").(string)
							if err := assetService.UpdateAsset(name, description); err != nil {
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
						},
					},
					{
						Name:  "show",
						Usage: "Show detailed information about an asset",
						Action: func(ctx *cli.Context) error {
							name := ctx.Value("name").(string)
							asset, err := assetService.GetAsset(name)
							if err != nil {
								return err
							}
							fmt.Printf("Asset: %s\n", asset.Name)
							fmt.Printf("Description: %s\n", asset.Description)
							fmt.Printf("Created: %s\n", asset.CreatedAt.Format("2006-01-02 15:04:05"))
							fmt.Printf("Updated: %s\n", asset.UpdatedAt.Format("2006-01-02 15:04:05"))
							fmt.Printf("Task Count: %d\n", asset.AssociatedTaskCount)
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
