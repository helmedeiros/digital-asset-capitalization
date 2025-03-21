package completion

import (
	"strings"
	"testing"
)

func TestGetBashCompletion(t *testing.T) {
	script := GetBashCompletion()
	if script == "" {
		t.Error("Bash completion script should not be empty")
	}

	// Check for required components
	required := []string{
		"#! /bin/bash",
		"_assetcap_completion()",
		"timeallocation-calc",
		"assets",
		"completion",
		"help",
		"create",
		"list",
		"contribution-type",
		"documentation",
		"tasks",
		"increment",
		"decrement",
		"complete -F _assetcap_completion assetcap",
	}

	for _, r := range required {
		if !strings.Contains(script, r) {
			t.Errorf("Bash completion script missing required component: %q", r)
		}
	}

	// Check script structure
	if !strings.HasPrefix(script, "#! /bin/bash") {
		t.Error("Bash completion script should start with shebang")
	}
	if !strings.Contains(script, "COMPREPLY=()") {
		t.Error("Bash completion script should initialize COMPREPLY array")
	}
}

func TestGetZshCompletion(t *testing.T) {
	script := GetZshCompletion()
	if script == "" {
		t.Error("Zsh completion script should not be empty")
	}

	// Check for required components
	required := []string{
		"#compdef assetcap",
		"_assetcap()",
		"timeallocation-calc",
		"assets",
		"completion",
		"help",
		"create",
		"list",
		"contribution-type",
		"documentation",
		"tasks",
		"increment",
		"decrement",
		"compdef _assetcap assetcap",
	}

	for _, r := range required {
		if !strings.Contains(script, r) {
			t.Errorf("Zsh completion script missing required component: %q", r)
		}
	}

	// Check script structure
	if !strings.HasPrefix(script, "#compdef assetcap") {
		t.Error("Zsh completion script should start with compdef")
	}
	if !strings.Contains(script, "_arguments -C") {
		t.Error("Zsh completion script should use _arguments")
	}
}

func TestGetFishCompletion(t *testing.T) {
	script := GetFishCompletion()
	if script == "" {
		t.Error("Fish completion script should not be empty")
	}

	// Check for required components
	required := []string{
		"__fish_assetcap_no_subcommand",
		"timeallocation-calc",
		"assets",
		"completion",
		"help",
		"create",
		"list",
		"contribution-type",
		"documentation",
		"tasks",
		"increment",
		"decrement",
	}

	for _, r := range required {
		if !strings.Contains(script, r) {
			t.Errorf("Fish completion script missing required component: %q", r)
		}
	}

	// Check script structure
	if !strings.Contains(script, "function __fish_assetcap_no_subcommand") {
		t.Error("Fish completion script should define helper function")
	}
	if !strings.Contains(script, "complete -c assetcap") {
		t.Error("Fish completion script should use complete command")
	}
}

func TestCompletionScriptsConsistency(t *testing.T) {
	// Check that all three shells support the same commands
	bash := GetBashCompletion()
	zsh := GetZshCompletion()
	fish := GetFishCompletion()

	commands := []string{
		"timeallocation-calc",
		"assets",
		"completion",
		"help",
	}

	for _, cmd := range commands {
		if !strings.Contains(bash, cmd) {
			t.Errorf("Bash completion missing command: %q", cmd)
		}
		if !strings.Contains(zsh, cmd) {
			t.Errorf("Zsh completion missing command: %q", cmd)
		}
		if !strings.Contains(fish, cmd) {
			t.Errorf("Fish completion missing command: %q", cmd)
		}
	}

	// Check that all three shells support the same asset subcommands
	assetCommands := []string{
		"create",
		"list",
		"contribution-type",
		"documentation",
		"tasks",
	}

	for _, cmd := range assetCommands {
		if !strings.Contains(bash, cmd) {
			t.Errorf("Bash completion missing asset command: %q", cmd)
		}
		if !strings.Contains(zsh, cmd) {
			t.Errorf("Zsh completion missing asset command: %q", cmd)
		}
		if !strings.Contains(fish, cmd) {
			t.Errorf("Fish completion missing asset command: %q", cmd)
		}
	}
}
