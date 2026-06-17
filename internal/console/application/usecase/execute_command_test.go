package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain/ports"
)

// MockCommandExecutor for testing
type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) ValidateCommand(command *domain.Command) error {
	args := m.Called(command)
	return args.Error(0)
}

func (m *MockCommandExecutor) Execute(ctx context.Context, command *domain.Command) (*domain.CommandResult, error) {
	args := m.Called(ctx, command)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CommandResult), args.Error(1)
}

func (m *MockCommandExecutor) GetAvailableCommands() []ports.CommandInfo {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]ports.CommandInfo)
}

func TestNewExecuteCommandUseCase(t *testing.T) {
	t.Parallel()
	mockExecutor := new(MockCommandExecutor)
	useCase := NewExecuteCommandUseCase(mockExecutor)

	assert.NotNil(t, useCase)
	assert.Equal(t, mockExecutor, useCase.executor)
}

func TestExecuteCommandUseCase_Execute_Success(t *testing.T) {
	t.Parallel()
	mockExecutor := new(MockCommandExecutor)
	useCase := NewExecuteCommandUseCase(mockExecutor)
	ctx := context.Background()

	command := &domain.Command{
		ID:          "cmd-123",
		Raw:         "show assets",
		Interpreted: "assets list",
		Confidence:  0.95,
		Timestamp:   time.Now(),
		SessionID:   "session-123",
		Parameters:  map[string]interface{}{"limit": 10},
	}

	expectedResult := &domain.CommandResult{
		CommandID: "cmd-123",
		Success:   true,
		Output:    map[string]interface{}{"assets": []string{"asset1", "asset2"}},
		Duration:  50 * time.Millisecond,
	}

	// Mock successful validation and execution
	mockExecutor.On("ValidateCommand", command).Return(nil)
	mockExecutor.On("Execute", ctx, command).Return(expectedResult, nil)

	result, err := useCase.Execute(ctx, command)

	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	mockExecutor.AssertExpectations(t)
}

func TestExecuteCommandUseCase_Execute_PreValidationFailure(t *testing.T) {
	t.Parallel()
	mockExecutor := new(MockCommandExecutor)
	useCase := NewExecuteCommandUseCase(mockExecutor)
	ctx := context.Background()

	// Command with low confidence (should fail pre-validation)
	command := &domain.Command{
		ID:          "cmd-low-confidence",
		Raw:         "show something",
		Interpreted: "unknown",
		Confidence:  0.3, // Below 0.5 threshold
		Timestamp:   time.Now(),
		SessionID:   "session-123",
	}

	result, err := useCase.Execute(ctx, command)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "confidence")
}

func TestExecuteCommandUseCase_Execute_ValidationFailure(t *testing.T) {
	t.Parallel()
	mockExecutor := new(MockCommandExecutor)
	useCase := NewExecuteCommandUseCase(mockExecutor)
	ctx := context.Background()

	command := &domain.Command{
		ID:          "cmd-invalid",
		Raw:         "invalid command",
		Interpreted: "invalid",
		Confidence:  0.95,
		Timestamp:   time.Now(),
		SessionID:   "session-123",
	}

	validationErr := errors.New("command not supported")
	mockExecutor.On("ValidateCommand", command).Return(validationErr)

	result, err := useCase.Execute(ctx, command)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, validationErr, result.Error)
	mockExecutor.AssertExpectations(t)
}

func TestExecuteCommandUseCase_Execute_ExecutionFailure(t *testing.T) {
	t.Parallel()
	mockExecutor := new(MockCommandExecutor)
	useCase := NewExecuteCommandUseCase(mockExecutor)
	ctx := context.Background()

	command := &domain.Command{
		ID:          "cmd-exec-fail",
		Raw:         "show assets",
		Interpreted: "assets list",
		Confidence:  0.95,
		Timestamp:   time.Now(),
		SessionID:   "session-123",
	}

	execErr := errors.New("database connection failed")

	// Mock successful validation but execution failure
	mockExecutor.On("ValidateCommand", command).Return(nil)
	mockExecutor.On("Execute", ctx, command).Return(nil, execErr)

	result, err := useCase.Execute(ctx, command)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, execErr, result.Error)
	mockExecutor.AssertExpectations(t)
}

func TestExecuteCommandUseCase_ValidateCommand(t *testing.T) {
	t.Parallel()
	mockExecutor := new(MockCommandExecutor)
	useCase := NewExecuteCommandUseCase(mockExecutor)

	tests := []struct {
		name        string
		command     *domain.Command
		mockError   error
		expectError bool
	}{
		{
			name: "valid command",
			command: &domain.Command{
				ID:          "cmd-valid",
				Raw:         "show assets",
				Interpreted: "assets list",
				SessionID:   "session-123",
				Confidence:  0.95,
				Timestamp:   time.Now(),
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "executor validation error",
			command: &domain.Command{
				ID:          "cmd-invalid",
				Raw:         "invalid command",
				Interpreted: "invalid",
				SessionID:   "session-123",
				Confidence:  0.95,
				Timestamp:   time.Now(),
			},
			mockError:   errors.New("unsupported command"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor.On("ValidateCommand", tt.command).Return(tt.mockError)

			err := useCase.ValidateCommand(tt.command)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockExecutor.AssertExpectations(t)
			// Clear the mock for the next test
			mockExecutor.ExpectedCalls = nil
		})
	}
}

func TestExecuteCommandUseCase_GetAvailableCommands(t *testing.T) {
	t.Parallel()
	mockExecutor := new(MockCommandExecutor)
	useCase := NewExecuteCommandUseCase(mockExecutor)

	expectedCommands := []ports.CommandInfo{
		{
			Command:     "assets",
			Description: "Manage digital assets",
			Examples:    []string{"show assets", "create asset"},
		},
		{
			Command:     "tasks",
			Description: "Manage tasks",
			Examples:    []string{"list tasks", "create task"},
		},
	}

	mockExecutor.On("GetAvailableCommands").Return(expectedCommands)

	commands := useCase.GetAvailableCommands()

	assert.Equal(t, expectedCommands, commands)
	mockExecutor.AssertExpectations(t)
}

func TestExecuteCommandUseCase_PreValidate(t *testing.T) {
	t.Parallel()
	mockExecutor := new(MockCommandExecutor)
	useCase := NewExecuteCommandUseCase(mockExecutor)

	tests := []struct {
		name          string
		command       *domain.Command
		expectError   bool
		errorContains string
	}{
		{
			name: "valid command",
			command: &domain.Command{
				ID:          "cmd-valid",
				Raw:         "show assets",
				Interpreted: "assets list",
				SessionID:   "session-123",
				Confidence:  0.95,
				Timestamp:   time.Now(),
			},
			expectError: false,
		},
		{
			name: "low confidence",
			command: &domain.Command{
				ID:          "cmd-low-conf",
				Raw:         "show something",
				Interpreted: "unknown",
				SessionID:   "session-123",
				Confidence:  0.3,
				Timestamp:   time.Now(),
			},
			expectError:   true,
			errorContains: "confidence",
		},
		{
			name: "not interpreted",
			command: &domain.Command{
				ID:          "cmd-not-interpreted",
				Raw:         "show assets",
				Interpreted: "", // Empty interpreted
				SessionID:   "session-123",
				Confidence:  0.95,
				Timestamp:   time.Now(),
			},
			expectError:   true,
			errorContains: "interpreted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := useCase.preValidate(tt.command)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecuteCommandUseCase_PostProcessResult(t *testing.T) {
	t.Parallel()
	mockExecutor := new(MockCommandExecutor)
	useCase := NewExecuteCommandUseCase(mockExecutor)

	// Test setting command ID
	t.Run("sets command ID when missing", func(t *testing.T) {
		command := &domain.Command{
			ID:        "cmd-123",
			Timestamp: time.Now().Add(-100 * time.Millisecond),
		}
		result := &domain.CommandResult{
			CommandID: "", // Missing command ID
			Success:   true,
		}

		useCase.postProcessResult(result, command)

		assert.Equal(t, "cmd-123", result.CommandID)
	})

	// Test fixing success flag when error exists
	t.Run("fixes success flag when error exists", func(t *testing.T) {
		command := &domain.Command{ID: "cmd-123"}
		result := &domain.CommandResult{
			Success: true,
			Error:   errors.New("some error"),
		}

		useCase.postProcessResult(result, command)

		assert.False(t, result.Success)
	})

	// Test duration calculation
	t.Run("calculates duration when not set", func(t *testing.T) {
		command := &domain.Command{
			ID:        "cmd-123",
			Timestamp: time.Now().Add(-200 * time.Millisecond),
		}
		result := &domain.CommandResult{
			Duration: 0, // No duration set
		}

		useCase.postProcessResult(result, command)

		assert.Greater(t, result.Duration, time.Duration(0))
		assert.LessOrEqual(t, result.Duration, 300*time.Millisecond)
	})

	// Test no changes when values are already set
	t.Run("preserves existing values", func(t *testing.T) {
		command := &domain.Command{ID: "cmd-123"}
		result := &domain.CommandResult{
			CommandID: "existing-id",
			Success:   true,
			Duration:  50 * time.Millisecond,
		}

		originalCommandID := result.CommandID
		originalSuccess := result.Success
		originalDuration := result.Duration

		useCase.postProcessResult(result, command)

		assert.Equal(t, originalCommandID, result.CommandID)
		assert.Equal(t, originalSuccess, result.Success)
		assert.Equal(t, originalDuration, result.Duration)
	})
}
