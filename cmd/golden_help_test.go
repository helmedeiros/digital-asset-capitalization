package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

// updateGolden flips the test from "assert equal" to "rewrite the
// fixture". Run with `go test ./cmd/ -update-golden` after an
// intentional CLI behaviour change.
var updateGolden = flag.Bool("update-golden", false, "rewrite golden help files")

// TestHelpOutputsAreGolden pins the --help output of every command
// group extracted from cmd/main.go. Together with the scientific
// validation we ran during the split, this turns the byte-level
// equivalence we manually verified into a permanent regression test.
//
// To update after an intentional change:
//
//	go test ./cmd/ -update-golden -run TestHelpOutputsAreGolden
func TestHelpOutputsAreGolden(t *testing.T) {
	// A bare App with the field defaults needed for each createXxx
	// helper. We don't wire any service so an Action-triggered call
	// would crash, but `--help` short-circuits Action entirely.
	a := &App{}

	cases := []struct {
		name    string
		command *cli.Command
	}{
		{"version", a.createVersionCommand()},
		{"completion", a.createCompletionCommand()},
		{"sprint", a.createSprintCommand()},
		{"tasks", a.createTasksCommand()},
		{"assets", a.createAssetsCommand()},
		{"investment", a.createInvestmentCommand()},
		{"config", a.createConfigCommand()},
		{"deployments", a.createDeploymentCommands()},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := captureHelp(t, c.command)
			path := filepath.Join("golden_fixtures", c.name+".help.txt")
			if *updateGolden {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("mkdir golden dir: %v", err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden %s: %v (run with -update-golden to create)", path, err)
			}
			if string(want) != got {
				t.Errorf("help output drift for %s.\nDiff (first 1000 chars of each):\nwant:\n%s\n---\ngot:\n%s",
					c.name, truncate(string(want), 1000), truncate(got, 1000))
			}
		})
	}
}

// captureHelp runs a one-command throwaway App against
// `<command> --help` and returns the stdout. urfave/cli's help
// writer is taken from App.Writer so a bytes.Buffer redirects it.
func captureHelp(t *testing.T, cmd *cli.Command) string {
	t.Helper()
	var buf bytes.Buffer
	app := &cli.App{
		Name:     "AssetCap",
		Writer:   &buf,
		Commands: []*cli.Command{cmd},
	}
	if err := app.Run([]string{"AssetCap", cmd.Name, "--help"}); err != nil {
		t.Fatalf("running %s --help: %v", cmd.Name, err)
	}
	return buf.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...<truncated>"
}

// init: keep gofmt happy that strings is used in non-truncated paths
// when -update-golden flips on. The reference also serves as a hook
// for future help-format normalisation (e.g. stripping ANSI codes).
var _ = strings.TrimSpace
