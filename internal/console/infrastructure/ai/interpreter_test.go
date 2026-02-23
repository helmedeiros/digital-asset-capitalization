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
	assert.Equal(t, "llama4", config.Model)
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
		Model:   "llama4",
	}

	assert.Equal(t, "http://localhost:11434", config.BaseURL)
	assert.Equal(t, "llama4", config.Model)
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
		Model:   "llama4",
	})
	ctx := context.Background()

	clarification, err := interpreter.GetClarification(ctx, "ambiguous command", []string{"option1", "option2"})

	assert.NoError(t, err)
	assert.Equal(t, "Could you please clarify what you meant?", clarification)
}
