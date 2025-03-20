package main

import (
	"fmt"
	"log"
	"os"

	"github.com/helmedeiros/jira-time-allocator/assetcap/action"
	"github.com/urfave/cli/v2"
)

func main() {
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
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
