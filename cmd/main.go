package main

import (
	"fmt"
	"log"
	"os"

	"github.com/helmedeiros/jira-time-allocator/assetcap/action"
	"github.com/urfave/cli/v2"
)

func main() {
	assetManager := action.NewAssetManager()

	app := &cli.App{
		Name:  "AssetCap TimeAllocation calculator",
		Usage: "Process JIRA issues for a specific project and sprint",
		Commands: []*cli.Command{
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
							if err := assetManager.CreateAsset(name, description); err != nil {
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
							assets := assetManager.ListAssets()
							fmt.Print(action.FormatAssetList(assets))
							return nil
						},
					},
					{
						Name:  "contribution-type",
						Usage: "Manage contribution types",
						Subcommands: []*cli.Command{
							{
								Name:  "add",
								Usage: "Add a contribution type to an asset",
								Action: func(ctx *cli.Context) error {
									assetName := ctx.Value("asset").(string)
									contributionType := ctx.Value("type").(string)
									if err := assetManager.AddContributionType(assetName, contributionType); err != nil {
										return err
									}
									fmt.Printf("Added contribution type %s to asset %s\n", contributionType, assetName)
									return nil
								},
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "asset",
										Usage:    "Asset name",
										Required: true,
									},
									&cli.StringFlag{
										Name:     "type",
										Usage:    "Contribution type (discovery, development, or maintenance)",
										Required: true,
									},
								},
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
									if err := assetManager.UpdateDocumentation(assetName); err != nil {
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
									if err := assetManager.IncrementTaskCount(assetName); err != nil {
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
									if err := assetManager.DecrementTaskCount(assetName); err != nil {
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
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
