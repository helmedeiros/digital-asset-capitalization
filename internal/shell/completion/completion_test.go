package completion

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBashCompletion(t *testing.T) {
	script := GetBashCompletion()
	assert.NotEmpty(t, script, "Bash completion script should not be empty")

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
		assert.Contains(t, script, r, "Bash completion script missing required component: %q", r)
	}

	// Check script structure
	assert.True(t, strings.HasPrefix(script, "#! /bin/bash"), "Bash completion script should start with shebang")
	assert.Contains(t, script, "COMPREPLY=()", "Bash completion script should initialize COMPREPLY array")
}

func TestGetZshCompletion(t *testing.T) {
	script := GetZshCompletion()
	assert.NotEmpty(t, script, "Zsh completion script should not be empty")

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
		assert.Contains(t, script, r, "Zsh completion script missing required component: %q", r)
	}

	// Check script structure
	assert.True(t, strings.HasPrefix(script, "#compdef assetcap"), "Zsh completion script should start with compdef")
	assert.Contains(t, script, "_arguments -C", "Zsh completion script should use _arguments")
}

func TestGetFishCompletion(t *testing.T) {
	script := GetFishCompletion()
	assert.NotEmpty(t, script, "Fish completion script should not be empty")

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
		assert.Contains(t, script, r, "Fish completion script missing required component: %q", r)
	}

	// Check script structure
	assert.Contains(t, script, "function __fish_assetcap_no_subcommand", "Fish completion script should define helper function")
	assert.Contains(t, script, "complete -c assetcap", "Fish completion script should use complete command")
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
		assert.Contains(t, bash, cmd, "Bash completion missing command: %q", cmd)
		assert.Contains(t, zsh, cmd, "Zsh completion missing command: %q", cmd)
		assert.Contains(t, fish, cmd, "Fish completion missing command: %q", cmd)
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
		assert.Contains(t, bash, cmd, "Bash completion missing asset command: %q", cmd)
		assert.Contains(t, zsh, cmd, "Zsh completion missing asset command: %q", cmd)
		assert.Contains(t, fish, cmd, "Fish completion missing asset command: %q", cmd)
	}
}
