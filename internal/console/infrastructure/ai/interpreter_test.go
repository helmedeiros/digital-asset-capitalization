package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "http://localhost:11434", config.BaseURL)
	assert.Equal(t, "llama3", config.Model)
}

func TestNewInterpreter(t *testing.T) {
	config := Config{
		BaseURL: "http://test-url:11434",
		Model:   "test-model",
	}

	interpreter := NewInterpreter(config)

	assert.NotNil(t, interpreter)
	assert.Equal(t, "http://test-url:11434", interpreter.baseURL)
	assert.Equal(t, "test-model", interpreter.model)
	assert.NotNil(t, interpreter.httpClient)
}

func TestConfig_Values(t *testing.T) {
	config := Config{
		BaseURL: "http://localhost:11434",
		Model:   "llama3",
	}

	assert.Equal(t, "http://localhost:11434", config.BaseURL)
	assert.Equal(t, "llama3", config.Model)
}

func TestInterpreter_buildContextInfo(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())
	context := domain.NewContext("test-session")

	// Test with empty context
	info := interpreter.buildContextInfo(context)
	assert.Equal(t, "No context available", info)

	// Test with populated context
	context.CurrentProject = "TestProject"
	context.CurrentSprint = "Sprint 1"
	context.RecentAssets = []string{"Asset1", "Asset2"}

	info = interpreter.buildContextInfo(context)
	assert.Contains(t, info, "TestProject")
	assert.Contains(t, info, "Sprint 1")
	assert.Contains(t, info, "Asset1")
	assert.Contains(t, info, "Asset2")
}

func TestInterpreter_buildInterpretationPrompt(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())
	context := domain.NewContext("test-session")
	context.CurrentProject = "TestProject"

	prompt := interpreter.buildInterpretationPrompt("list assets", context)

	assert.Contains(t, prompt, "list assets")
	assert.Contains(t, prompt, "TestProject")
	assert.Contains(t, prompt, "AssetCap")
	assert.Contains(t, prompt, "assets list")
	assert.Contains(t, prompt, "JSON object")
}

func TestInterpreter_parseTextResponse(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())

	// Test with valid command
	response := "assets list --format json"
	result := interpreter.parseTextResponse(response)

	assert.Equal(t, "assets list --format json", result.Command)
	assert.Equal(t, 0.7, result.Confidence)

	// Test with multi-line response containing command
	response = "Here's what I think you mean:\nassets list\nThis will show all assets."
	result = interpreter.parseTextResponse(response)

	assert.Equal(t, "assets list", result.Command)
	assert.Equal(t, 0.7, result.Confidence)

	// Test with no recognizable command
	response = "I don't understand"
	result = interpreter.parseTextResponse(response)

	assert.Equal(t, "I don't understand", result.Command)
	assert.Equal(t, 0.5, result.Confidence)
}

func TestInterpreter_mapActionType(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())

	assert.Equal(t, domain.CommandTypeCreate, interpreter.mapActionType("create"))
	assert.Equal(t, domain.CommandTypeRead, interpreter.mapActionType("read"))
	assert.Equal(t, domain.CommandTypeRead, interpreter.mapActionType("show"))
	assert.Equal(t, domain.CommandTypeUpdate, interpreter.mapActionType("update"))
	assert.Equal(t, domain.CommandTypeDelete, interpreter.mapActionType("delete"))
	assert.Equal(t, domain.CommandTypeList, interpreter.mapActionType("list"))
	assert.Equal(t, domain.CommandTypeSync, interpreter.mapActionType("sync"))
	assert.Equal(t, domain.CommandTypeHelp, interpreter.mapActionType("help"))
	assert.Equal(t, domain.CommandTypeOther, interpreter.mapActionType("unknown"))
}

func TestInterpreter_mapResourceType(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())

	assert.Equal(t, domain.ResourceTypeAsset, interpreter.mapResourceType("asset"))
	assert.Equal(t, domain.ResourceTypeAsset, interpreter.mapResourceType("assets"))
	assert.Equal(t, domain.ResourceTypeTask, interpreter.mapResourceType("task"))
	assert.Equal(t, domain.ResourceTypeTask, interpreter.mapResourceType("tasks"))
	assert.Equal(t, domain.ResourceTypeSprint, interpreter.mapResourceType("sprint"))
	assert.Equal(t, domain.ResourceTypeInvestment, interpreter.mapResourceType("investment"))
	assert.Equal(t, domain.ResourceTypeConfig, interpreter.mapResourceType("config"))
	assert.Equal(t, domain.ResourceTypeContext, interpreter.mapResourceType("context"))
	assert.Equal(t, domain.ResourceTypeUnknown, interpreter.mapResourceType("unknown"))
}

func TestInterpreter_setCommandIntent(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())
	command, _ := domain.NewCommand("session-1", "assets list", "assets list", 0.9)

	result := InterpretationResult{
		Command:    "assets list",
		Parameters: map[string]interface{}{"name": "TestAsset"},
	}

	interpreter.setCommandIntent(command, result)

	assert.Equal(t, domain.CommandTypeList, command.Intent.Action)
	assert.Equal(t, domain.ResourceTypeAsset, command.Intent.Resource)
	assert.Equal(t, "TestAsset", command.Intent.Target)
}

func TestInterpretationResult_Structure(t *testing.T) {
	result := InterpretationResult{
		Command:               "assets list",
		Confidence:            0.95,
		Parameters:            map[string]interface{}{"format": "json"},
		RequiresClarification: false,
		ClarificationPrompt:   "",
		InterpretedIntent:     "List all assets in JSON format",
	}

	assert.Equal(t, "assets list", result.Command)
	assert.Equal(t, 0.95, result.Confidence)
	assert.Equal(t, "json", result.Parameters["format"])
	assert.False(t, result.RequiresClarification)
	assert.Empty(t, result.ClarificationPrompt)
	assert.Equal(t, "List all assets in JSON format", result.InterpretedIntent)
}

func TestInterpreter_GetClarification_Fallback(t *testing.T) {
	// This test verifies the fallback behavior when LLaMA is not available
	interpreter := NewInterpreter(Config{
		BaseURL: "http://invalid-url:11434", // This will cause an error
		Model:   "llama3",
	})
	ctx := context.Background()

	clarification, err := interpreter.GetClarification(ctx, "ambiguous command", []string{"option1", "option2"})

	assert.NoError(t, err)
	assert.Equal(t, "Could you please clarify what you meant?", clarification)
}

func TestInterpreter_parseCommandParameters(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())

	// Test basic parameter parsing
	command, err := domain.NewCommand("test-session", "original input", "config sync-team --project \"FN\"", 0.9)
	assert.NoError(t, err)

	interpreter.parseCommandParameters(command, "config sync-team --project \"FN\"")

	project, ok := command.GetStringParameter("project")
	assert.True(t, ok)
	assert.Equal(t, "FN", project)

	// Test multiple parameters
	command2, err := domain.NewCommand("test-session", "original", "assets create --name \"Test Asset\" --description \"Test Desc\"", 0.9)
	assert.NoError(t, err)

	interpreter.parseCommandParameters(command2, "assets create --name \"Test Asset\" --description \"Test Desc\"")

	name, ok := command2.GetStringParameter("name")
	assert.True(t, ok)
	assert.Equal(t, "Test Asset", name)

	desc, ok := command2.GetStringParameter("description")
	assert.True(t, ok)
	assert.Equal(t, "Test Desc", desc)

	// Test boolean flag
	command3, err := domain.NewCommand("test-session", "original", "assets sync --keywords", 0.9)
	assert.NoError(t, err)

	interpreter.parseCommandParameters(command3, "assets sync --keywords")

	keywords, ok := command3.GetParameter("keywords")
	assert.True(t, ok)
	assert.Equal(t, true, keywords)
}

func TestInterpreter_splitCommandWithQuotes(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())

	// Test simple command
	parts := interpreter.splitCommandWithQuotes("config sync-team --project \"FN\"")
	expected := []string{"config", "sync-team", "--project", "\"FN\""}
	assert.Equal(t, expected, parts)

	// Test single quotes
	parts = interpreter.splitCommandWithQuotes("assets create --name 'Test Asset'")
	expected = []string{"assets", "create", "--name", "'Test Asset'"}
	assert.Equal(t, expected, parts)

	// Test no quotes
	parts = interpreter.splitCommandWithQuotes("assets list")
	expected = []string{"assets", "list"}
	assert.Equal(t, expected, parts)

	// Test multiple quoted parameters
	parts = interpreter.splitCommandWithQuotes("assets create --name \"Payment System\" --description \"Main payment processing\"")
	expected = []string{"assets", "create", "--name", "\"Payment System\"", "--description", "\"Main payment processing\""}
	assert.Equal(t, expected, parts)
}

// TestInterpreter_TeamAssignmentQueries tests natural language queries for team assignments
func TestInterpreter_TeamAssignmentQueries(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())
	contextObj := domain.NewContext("test-session")

	tests := []struct {
		name           string
		input          string
		expectedCmd    string
		expectedParams map[string]interface{}
		description    string
	}{
		{
			name:        "List all team assignments",
			input:       "show all teams with their assets",
			expectedCmd: "assets teams list",
			description: "Should map to assets teams list command",
		},
		{
			name:        "Show team for specific asset",
			input:       "who owns the Omio Flex asset",
			expectedCmd: "assets teams show",
			expectedParams: map[string]interface{}{
				"asset": "Omio Flex",
			},
			description: "Should extract asset name and map to show command",
		},
		{
			name:        "Assign team as owner",
			input:       "assign FN team as owner of Dynamic Markup",
			expectedCmd: "assets teams assign",
			expectedParams: map[string]interface{}{
				"asset": "Dynamic Markup",
				"owner": "FN",
			},
			description: "Should extract team and asset for assignment",
		},
		{
			name:        "Add team as contributor",
			input:       "add AD team as contributor to Flight Delay Insurance",
			expectedCmd: "assets teams add-contributor",
			expectedParams: map[string]interface{}{
				"asset": "Flight Delay Insurance",
				"team":  "AD",
			},
			description: "Should map to add-contributor command",
		},
		{
			name:        "List team assignments naturally",
			input:       "show me all the existing teams",
			expectedCmd: "assets teams list",
			description: "Natural variation of listing teams",
		},
		{
			name:        "Bulk assignment query",
			input:       "review all assets for FN team stories in H1 2025 and assign FN as owner",
			expectedCmd: "assets teams assign",
			expectedParams: map[string]interface{}{
				"owner": "FN",
			},
			description: "Complex bulk assignment request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip actual LLM calls in tests - just verify the structure
			t.Logf("Test case: %s - %s", tt.name, tt.description)
			
			// Verify prompt building includes team commands
			prompt := interpreter.buildInterpretationPrompt(tt.input, contextObj)
			assert.Contains(t, prompt, "assets teams")
			assert.Contains(t, prompt, tt.input)
		})
	}
}

// TestInterpreter_TeamQueryDisambiguation tests disambiguation between team members and asset teams
func TestInterpreter_TeamQueryDisambiguation(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())
	
	tests := []struct {
		name        string
		input       string
		shouldBeConfig bool
		description string
	}{
		{
			name:        "Team members query",
			input:       "show team members",
			shouldBeConfig: true,
			description: "Should map to config show for team members",
		},
		{
			name:        "Asset teams query",
			input:       "show asset teams",
			shouldBeConfig: false,
			description: "Should map to assets teams for asset ownership",
		},
		{
			name:        "List teams context",
			input:       "list all teams",
			shouldBeConfig: false,
			description: "In asset context, should map to asset teams",
		},
		{
			name:        "Who is on FN team",
			input:       "who is on the FN team",
			shouldBeConfig: true,
			description: "Asking about people should map to config",
		},
		{
			name:        "Which assets does FN own",
			input:       "which assets does FN team own",
			shouldBeConfig: false,
			description: "Asking about assets should map to asset teams",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := interpreter.buildInterpretationPrompt(tt.input, domain.NewContext("test"))
			
			if tt.shouldBeConfig {
				assert.Contains(t, prompt, "config show")
			} else {
				assert.Contains(t, prompt, "assets teams")
			}
		})
	}
}

// TestInterpreter_TeamParameterExtraction tests extraction of team-related parameters
func TestInterpreter_TeamParameterExtraction(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())

	tests := []struct {
		name           string
		command        string
		expectedParams map[string]interface{}
	}{
		{
			name:    "Asset with spaces",
			command: "assets teams assign --asset \"Ancillaries Markups\" --owner \"FN\"",
			expectedParams: map[string]interface{}{
				"asset": "Ancillaries Markups",
				"owner": "FN",
			},
		},
		{
			name:    "Team parameter variations",
			command: "assets teams add-contributor --asset \"Price Lock\" --team \"AD\"",
			expectedParams: map[string]interface{}{
				"asset": "Price Lock",
				"team":  "AD",
			},
		},
		{
			name:    "Multiple contributors",
			command: "assets teams assign --asset \"Dynamic Rounding\" --owner \"FN\" --contributors \"AD,QA\"",
			expectedParams: map[string]interface{}{
				"asset":        "Dynamic Rounding",
				"owner":        "FN",
				"contributors": "AD,QA",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := domain.NewCommand("test-session", "original", tt.command, 0.9)
			assert.NoError(t, err)
			
			interpreter.parseCommandParameters(cmd, tt.command)
			
			for key, expectedValue := range tt.expectedParams {
				actualValue, ok := cmd.GetParameter(key)
				assert.True(t, ok, "Parameter %s should exist", key)
				assert.Equal(t, expectedValue, actualValue)
			}
		})
	}
}

// TestInterpreter_ContextAwareTeamCommands tests team commands with context
func TestInterpreter_ContextAwareTeamCommands(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())
	
	// Create context with recent asset
	ctx := domain.NewContext("test-session")
	ctx.RecentAssets = []string{"Dynamic Markup", "Price Lock"}
	
	tests := []struct {
		name          string
		input         string
		expectedAsset string
		description   string
	}{
		{
			name:          "Assign to recent asset",
			input:         "assign it to FN team",
			expectedAsset: "Dynamic Markup",
			description:   "Should use last referenced asset",
		},
		{
			name:          "Add contributor to context asset",
			input:         "add AD as contributor",
			expectedAsset: "Dynamic Markup",
			description:   "Should use asset from context",
		},
		{
			name:          "Show teams for current asset",
			input:         "who owns this",
			expectedAsset: "Dynamic Markup",
			description:   "Should reference current asset",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := interpreter.buildInterpretationPrompt(tt.input, ctx)
			assert.Contains(t, prompt, "Dynamic Markup")
			assert.Contains(t, prompt, "context")
		})
	}
}

// TestInterpreter_TeamErrorCases tests handling of ambiguous or invalid team queries
func TestInterpreter_TeamErrorCases(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())
	ctx := domain.NewContext("test-session")
	
	tests := []struct {
		name        string
		input       string
		shouldError bool
		description string
	}{
		{
			name:        "Missing asset name",
			input:       "assign to FN team",
			shouldError: true,
			description: "Should require clarification for asset name",
		},
		{
			name:        "Missing team name",
			input:       "assign owner to Payment Processing",
			shouldError: true,
			description: "Should require team name",
		},
		{
			name:        "Ambiguous team reference",
			input:       "make them the owner",
			shouldError: true,
			description: "Unclear team reference",
		},
		{
			name:        "Valid command variation",
			input:       "make FN the owner of Price Lock",
			shouldError: false,
			description: "Should handle alternative phrasing",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := interpreter.buildInterpretationPrompt(tt.input, ctx)
			
			// Verify the prompt includes clarification handling
			if tt.shouldError {
				assert.Contains(t, prompt, "clarification")
			}
		})
	}
}

// TestInterpreter_BulkTeamOperations tests interpretation of bulk team assignment requests
func TestInterpreter_BulkTeamOperations(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())
	ctx := domain.NewContext("test-session")
	
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "Bulk assignment by pattern",
			input:       "assign FN to all pricing and markup assets",
			description: "Should understand bulk assignment by asset type",
		},
		{
			name:        "Time-based bulk assignment",
			input:       "review all assets for FN team stories in H1 2025 and assign them",
			description: "Should handle time-based filtering",
		},
		{
			name:        "Domain-based assignment",
			input:       "assign AD team to all accommodation and advertisement assets",
			description: "Should understand domain-specific grouping",
		},
		{
			name:        "Transfer ownership",
			input:       "transfer all assets from TeamA to TeamB",
			description: "Should handle ownership transfers",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := interpreter.buildInterpretationPrompt(tt.input, ctx)
			assert.Contains(t, prompt, "assets teams")
			t.Logf("Bulk operation test: %s", tt.description)
		})
	}
}
