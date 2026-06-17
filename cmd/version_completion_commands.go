package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/helmedeiros/digital-asset-capitalization/internal/shell/completion"
)

// createVersionCommand builds the `version` CLI command.
//
// It's a method on *App for symmetry with the other createXxxCommands
// helpers, even though version doesn't currently read App state — that
// way future fields (build flags, env-reported channels, etc.) have a
// natural home.
func (a *App) createVersionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Show version information",
		Action: func(_ *cli.Context) error {
			fmt.Printf("AssetCap %s\n", version)
			fmt.Printf("Commit: %s\n", commit)
			fmt.Printf("Built: %s\n", date)
			return nil
		},
	}
}

// createCompletionCommand builds the `completion` CLI command with its
// per-shell subcommands.
func (a *App) createCompletionCommand() *cli.Command {
	return &cli.Command{
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
	}
}
