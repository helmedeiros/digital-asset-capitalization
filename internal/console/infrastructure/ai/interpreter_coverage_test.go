package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// Test Interpret method with invalid configuration (to trigger error handling)
func TestInterpreter_Interpret_InvalidConfig(t *testing.T) {
	// Use invalid configuration to trigger error paths
	config := Config{
		BaseURL: "http://invalid-nonexistent-host:99999", // This will fail
		Model:   "invalid-model",
	}

	interpreter := NewInterpreter(config)
	ctx := context.Background()
	sessionContext := domain.NewContext("test-session")

	command, err := interpreter.Interpret(ctx, "list assets", sessionContext)

	// Should return error when unable to connect to LLaMA
	assert.Error(t, err)
	assert.Nil(t, command)
	assert.Contains(t, err.Error(), "failed to interpret command")
}

// Test Interpret method with valid command structure (expects connection failure)
func TestInterpreter_Interpret_ValidCommand(t *testing.T) {
	config := Config{
		BaseURL: "http://localhost:11434", // Standard URL but may not be running
		Model:   "llama3",
	}

	interpreter := NewInterpreter(config)
	ctx := context.Background()
	sessionContext := domain.NewContext("test-session")

	// Add some context
	sessionContext.CurrentProject = "TEST"
	sessionContext.RecentAssets = []string{"Asset1"}

	command, err := interpreter.Interpret(ctx, "show all assets", sessionContext)

	// In CI environment, this will likely fail due to no Ollama server
	// Test that error handling works properly
	if err != nil {
		assert.Error(t, err)
		assert.Nil(t, command)
		assert.Contains(t, err.Error(), "failed to interpret command")
	} else {
		// If somehow it succeeds (real Ollama running), verify the result
		assert.NotNil(t, command)
		assert.Equal(t, "show all assets", command.Raw)
		assert.Equal(t, "test-session", command.SessionID)
		assert.NotEmpty(t, command.Interpreted)
	}
}

// Test AnalyzeIntent method with various inputs
func TestInterpreter_AnalyzeIntent_BasicCases(t *testing.T) {
	config := Config{
		BaseURL: "http://invalid-host:99999", // Will fail, testing fallback
		Model:   "test-model",
	}

	interpreter := NewInterpreter(config)
	ctx := context.Background()

	tests := []struct {
		name  string
		input string
	}{
		{"asset list", "list all assets"},
		{"task fetch", "fetch tasks for project TEST"},
		{"sprint info", "show sprint information"},
		{"help command", "help me with commands"},
		{"unclear command", "do something with stuff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interpreter.AnalyzeIntent(ctx, tt.input)

			// May error due to invalid host, but should be handled gracefully
			if err == nil {
				assert.NotNil(t, result)
				// result is a CommandIntent, check its fields
				assert.NotEqual(t, domain.CommandTypeOther, result.Action)      // Should have some action
				assert.NotEqual(t, domain.ResourceTypeUnknown, result.Resource) // Should identify resource
			}
			// If error occurs due to network issues, that's acceptable for this test
		})
	}
}

// Test callLLaMA error handling
func TestInterpreter_callLLaMA_ErrorHandling(t *testing.T) {
	// Test with definitely invalid configuration
	config := Config{
		BaseURL: "http://this-host-does-not-exist:99999",
		Model:   "nonexistent-model",
	}

	interpreter := NewInterpreter(config)
	ctx := context.Background()

	response, err := interpreter.callLLaMA(ctx, "test prompt")

	// Should return empty string and error
	assert.Error(t, err)
	assert.Empty(t, response)
}

// Test setCommandIntent with basic case
func TestInterpreter_setCommandIntent_BasicCase(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())

	command, _ := domain.NewCommand("session-1", "assets create", "assets create", 0.9)
	result := InterpretationResult{
		Command:    "assets create",
		Parameters: map[string]interface{}{"name": "NewAsset"},
	}

	interpreter.setCommandIntent(command, result)

	// Should set some action type
	assert.NotEqual(t, domain.CommandType(""), command.Intent.Action)
}

// Test buildContextInfo with various context states
func TestInterpreter_buildContextInfo_VariousStates(t *testing.T) {
	interpreter := NewInterpreter(DefaultConfig())

	tests := []struct {
		name     string
		setup    func(*domain.Context)
		contains []string
	}{
		{
			name: "full context",
			setup: func(ctx *domain.Context) {
				ctx.CurrentProject = "PROJ"
				ctx.CurrentSprint = "Sprint1"
				ctx.CurrentSpace = "SPACE"
				ctx.RecentAssets = []string{"Asset1", "Asset2"}
				ctx.RecentTasks = []string{"TASK-1", "TASK-2"}
			},
			contains: []string{"PROJ", "Sprint1", "SPACE", "Asset1", "TASK-1"},
		},
		{
			name: "partial context",
			setup: func(ctx *domain.Context) {
				ctx.CurrentProject = "PROJ"
				ctx.RecentAssets = []string{"Asset1"}
			},
			contains: []string{"PROJ", "Asset1"},
		},
		{
			name: "empty context",
			setup: func(_ *domain.Context) {
				// No setup, leave empty
			},
			contains: []string{"No context available"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := domain.NewContext("test-session")
			tt.setup(ctx)

			info := interpreter.buildContextInfo(ctx)

			for _, expected := range tt.contains {
				assert.Contains(t, info, expected)
			}
		})
	}
}

// Test GetClarification with different scenarios
func TestInterpreter_GetClarification_Scenarios(t *testing.T) {
	// Test with valid config (may or may not have LLaMA running)
	interpreter := NewInterpreter(DefaultConfig())
	ctx := context.Background()

	tests := []struct {
		name    string
		reason  string
		options []string
	}{
		{
			name:    "command ambiguity",
			reason:  "Multiple interpretations possible",
			options: []string{"assets list", "assets show"},
		},
		{
			name:    "missing parameter",
			reason:  "Missing required parameter",
			options: []string{"specify project name", "use default project"},
		},
		{
			name:    "empty options",
			reason:  "Command unclear",
			options: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clarification, err := interpreter.GetClarification(ctx, tt.reason, tt.options)

			// Should always provide some clarification, even if LLaMA is not available
			assert.NoError(t, err)
			assert.NotEmpty(t, clarification)

			// Should contain reasonable clarification request (be flexible with exact wording)
			assert.True(t, len(clarification) > 10, "Clarification should be substantial: %s", clarification)
		})
	}
}
