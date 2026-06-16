package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// TestExecuteCommandUseCase_ValidateCommand_DomainInvalidShortCircuits
// covers the previously-untested branch where command.Validate() itself
// fails (missing ID), so the executor is never reached.
func TestExecuteCommandUseCase_ValidateCommand_DomainInvalidShortCircuits(t *testing.T) {
	mockExecutor := new(MockCommandExecutor)
	// Intentionally no .On("ValidateCommand") — the test asserts the
	// short-circuit prevents that call.
	useCase := NewExecuteCommandUseCase(mockExecutor)

	// Missing ID makes Command.Validate() fail with "command ID is required".
	err := useCase.ValidateCommand(&domain.Command{
		SessionID:   "s",
		Raw:         "x",
		Interpreted: "y",
		Confidence:  0.9,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid command")
	assert.Contains(t, err.Error(), "command ID is required")
	mockExecutor.AssertNotCalled(t, "ValidateCommand")
}
