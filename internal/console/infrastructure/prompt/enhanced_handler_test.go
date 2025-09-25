package prompt

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// Test methods that don't require complex service mocking

func TestEnhancedHandler_EndSession_NoService(t *testing.T) {
	handler := NewEnhancedHandler(nil)
	handler.sessionContext = nil

	ctx := context.Background()
	err := handler.endSession(ctx)

	assert.NoError(t, err)
}

func TestEnhancedHandler_HandleEnhancedInput_EmptyInput(t *testing.T) {
	handler := NewEnhancedHandler(nil)
	ctx := context.Background()

	err := handler.handleEnhancedInput(ctx, "")
	assert.NoError(t, err)
}

func TestEnhancedHandler_HandleSpecialCommands_WithoutService(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))
	handler := NewEnhancedHandler(nil)

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"help command", "help", true},
		{"history command", "history", true},
		{"clear command", "clear", true},
		{"HELP uppercase", "HELP", true},
		{"regular command", "show assets", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.handleSpecialCommands(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnhancedHandler_AddToHistory_WithoutService(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	// Test that addToHistory doesn't panic when there's no prompt session
	assert.NotPanics(t, func() {
		handler.addToHistory("test input", "test output", 0, true)
	})
}

func TestEnhancedHandler_SpinnerMethods_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	// Test spinner methods don't panic
	assert.NotPanics(t, func() {
		handler.startProcessingIndicator("Testing...")
	})

	assert.NotPanics(t, func() {
		handler.stopProcessingIndicator()
	})

	// Test stopping when no spinner is active
	assert.NotPanics(t, func() {
		handler.stopProcessingIndicator()
	})
}

func TestEnhancedHandler_DisplayMethods_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	// Test that all display methods don't panic
	assert.NotPanics(t, func() {
		handler.displayEnhancedWelcome()
	})

	assert.NotPanics(t, func() {
		handler.displayEnhancedGoodbye()
	})

	assert.NotPanics(t, func() {
		handler.displayEnhancedError("test error")
	})

	assert.NotPanics(t, func() {
		handler.displayEnhancedInfo("test info")
	})

	assert.NotPanics(t, func() {
		handler.displayEnhancedSuccess("test success")
	})

	assert.NotPanics(t, func() {
		handler.displayEnhancedHelp()
	})

	assert.NotPanics(t, func() {
		handler.clearScreen()
	})

	assert.NotPanics(t, func() {
		handler.displayCommandHistory()
	})
}

func TestEnhancedHandler_DisplayResult_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	testResults := []*application.ProcessResult{
		{
			Success: true,
			Output:  nil,
		},
		{
			Success: true,
			Output:  "simple string output",
		},
		{
			Success: true,
			Output: map[string]interface{}{
				"name":   "Test Asset",
				"status": "active",
			},
		},
		{
			Success: true,
			Output: []interface{}{
				map[string]interface{}{"name": "Asset1"},
				map[string]interface{}{"name": "Asset2"},
			},
		},
	}

	for i, result := range testResults {
		t.Run(fmt.Sprintf("result_%d", i), func(t *testing.T) {
			assert.NotPanics(t, func() {
				handler.displayEnhancedResult(result)
			})
		})
	}
}

func TestEnhancedHandler_DisplayContext_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	testContext := domain.NewContext("test-session")

	assert.NotPanics(t, func() {
		handler.displayEnhancedContext(testContext)
	})
}

func TestEnhancedHandler_DisplayClarification_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	assert.NotPanics(t, func() {
		handler.displayEnhancedClarification("Please clarify", []string{"option1", "option2"})
	})
}

func TestEnhancedHandler_CompletePromptWithInput_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	assert.NotPanics(t, func() {
		handler.completePromptWithInput("test input")
	})
}

func TestEnhancedHandler_DisplayMapOutput_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	testMaps := []map[string]interface{}{
		{
			"name":   "Test Asset",
			"status": "active",
		},
		{
			"assets": []interface{}{
				map[string]interface{}{"name": "Asset1"},
				map[string]interface{}{"name": "Asset2"},
			},
		},
		{}, // empty map
	}

	for i, testMap := range testMaps {
		t.Run(fmt.Sprintf("map_%d", i), func(t *testing.T) {
			assert.NotPanics(t, func() {
				handler.displayEnhancedMapOutput(testMap)
			})
		})
	}
}

func TestEnhancedHandler_DisplayListOutput_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	testLists := [][]interface{}{
		{"item1", "item2", "item3"},
		{
			map[string]interface{}{"command": "help", "description": "Show help"},
			map[string]interface{}{"command": "exit", "description": "Exit console"},
		},
		{}, // empty list
	}

	for i, testList := range testLists {
		t.Run(fmt.Sprintf("list_%d", i), func(t *testing.T) {
			assert.NotPanics(t, func() {
				handler.displayEnhancedListOutput(testList)
			})
		})
	}
}
