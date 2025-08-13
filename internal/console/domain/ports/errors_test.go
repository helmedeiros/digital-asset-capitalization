package ports

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutionError(t *testing.T) {
	err := NewExecutionError("test command", "test reason", "test help")

	assert.Equal(t, "test command", err.Command)
	assert.Equal(t, "test reason", err.Reason)
	assert.Equal(t, "test help", err.Help)
	assert.Equal(t, "test reason", err.Error())
}

func TestValidationError(t *testing.T) {
	// Test with field
	err := NewValidationError("field", "message")
	assert.Equal(t, "field", err.Field)
	assert.Equal(t, "message", err.Message)
	assert.Equal(t, "field: message", err.Error())

	// Test without field
	err = NewValidationError("", "message only")
	assert.Equal(t, "", err.Field)
	assert.Equal(t, "message only", err.Message)
	assert.Equal(t, "message only", err.Error())
}

func TestCommandInfo(t *testing.T) {
	paramInfo := ParameterInfo{
		Name:        "test-param",
		Description: "Test parameter",
		Required:    true,
		Type:        "string",
		Default:     "default-value",
	}

	cmdInfo := CommandInfo{
		Command:     "test command",
		Description: "Test command description",
		Examples:    []string{"example1", "example2"},
		Parameters:  []ParameterInfo{paramInfo},
	}

	assert.Equal(t, "test command", cmdInfo.Command)
	assert.Equal(t, "Test command description", cmdInfo.Description)
	assert.Len(t, cmdInfo.Examples, 2)
	assert.Len(t, cmdInfo.Parameters, 1)
	assert.Equal(t, "test-param", cmdInfo.Parameters[0].Name)
}

func TestStoreError(t *testing.T) {
	originalErr := errors.New("original error")
	storeErr := NewStoreError("save", "session-123", originalErr)

	assert.Equal(t, "save", storeErr.Operation)
	assert.Equal(t, "session-123", storeErr.SessionID)
	assert.Equal(t, originalErr, storeErr.Err)

	// Test Error() method
	expected := "save failed for session session-123: original error"
	assert.Equal(t, expected, storeErr.Error())

	// Test Unwrap() method
	assert.Equal(t, originalErr, storeErr.Unwrap())
}

func TestContextStoreErrors(t *testing.T) {
	assert.NotNil(t, ErrContextNotFound)
	assert.Equal(t, "context not found", ErrContextNotFound.Error())

	assert.NotNil(t, ErrContextExpired)
	assert.Equal(t, "context has expired", ErrContextExpired.Error())

	assert.NotNil(t, ErrStoreFull)
	assert.Equal(t, "context store is full", ErrStoreFull.Error())
}
