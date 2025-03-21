package main

import (
	"fmt"
	"log"
	"os"

	"github.com/helmedeiros/digital-asset-capitalization/assetcap/action"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure"
	"github.com/helmedeiros/digital-asset-capitalization/internal/shell/completion"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/command"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/jira"
	"github.com/urfave/cli/v2"
)

const (
	assetsDir  = ".assetcap"
	assetsFile = "assets.json"
)

var assetService application.AssetService

func init() {
	// Initialize the asset service with JSON repository
	repo := infrastructure.NewJSONRepository(assetsDir, assetsFile)
	assetService = application.NewAssetService(repo)
}

func Run() error {
	app := &cli.App{
		Name:                 "AssetCap",
		Usage:                "Digital Asset Capitalization Management Tool",
		EnableBashCompletion: true,
		UsageText: `assetcap [global options] command [command options] [arguments...]

COMMANDS:
   timeallocation-calc  Calculate time allocation for JIRA issues
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
				Name:  "timeallocation-calc",
				Usage: "Process JIRA issues",
				Action: func(ctx *cli.Context) error {
					fmt.Print(action.JiraDoer(ctx.Value("project").(string), ctx.Value("sprint").(string), ctx.Value("override").(string)))
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

							// Create repository
							repo, err := jira.NewRepository()
							if err != nil {
								return fmt.Errorf("failed to create Jira repository: %w", err)
							}

							// Create handler
							handler := command.NewFetchTasksHandler(repo)

							// Execute command
							return handler.Handle(ctx.Context, command.FetchTasksCommand{
								Project:  project,
								Sprint:   sprint,
								Platform: platform,
							})
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "project",
								Aliases:  []string{"p"},
								Usage:    "Project key",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "sprint",
								Aliases: []string{"s"},
								Usage:   "Sprint name",
							},
							&cli.StringFlag{
								Name:     "platform",
								Aliases:  []string{"l"},
								Usage:    "Platform name (e.g., jira)",
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

func main() {
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}
