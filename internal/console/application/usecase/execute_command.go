package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain/ports"
)

// ExecuteCommandUseCase handles command execution
type ExecuteCommandUseCase struct {
	executor ports.CommandExecutor
}

// NewExecuteCommandUseCase creates a new execute command use case
func NewExecuteCommandUseCase(executor ports.CommandExecutor) *ExecuteCommandUseCase {
	return &ExecuteCommandUseCase{
		executor: executor,
	}
}

// Execute runs a command and returns the result
func (uc *ExecuteCommandUseCase) Execute(ctx context.Context, command *domain.Command) (*domain.CommandResult, error) {
	// Pre-execution validation
	if err := uc.preValidate(command); err != nil {
		return nil, err
	}

	// Start timing
	startTime := time.Now()

	// Validate command with executor
	if err := uc.executor.ValidateCommand(command); err != nil {
		duration := time.Since(startTime)
		return domain.NewCommandResult(command.ID, false, nil, err, duration), err
	}

	// Execute the command
	result, err := uc.executor.Execute(ctx, command)
	if err != nil {
		duration := time.Since(startTime)
		return domain.NewCommandResult(command.ID, false, nil, err, duration), err
	}

	// Post-process the result
	uc.postProcessResult(result, command)

	return result, nil
}

// ValidateCommand validates a command before execution
func (uc *ExecuteCommandUseCase) ValidateCommand(command *domain.Command) error {
	// Basic validation
	if err := command.Validate(); err != nil {
		return fmt.Errorf("invalid command: %w", err)
	}

	// Delegate to executor for specific validation
	return uc.executor.ValidateCommand(command)
}

// GetAvailableCommands returns all available commands
func (uc *ExecuteCommandUseCase) GetAvailableCommands() []ports.CommandInfo {
	return uc.executor.GetAvailableCommands()
}

// preValidate performs pre-execution validation
func (uc *ExecuteCommandUseCase) preValidate(command *domain.Command) error {
	// Check basic command structure
	if err := command.Validate(); err != nil {
		return ports.NewValidationError("command", err.Error())
	}

	// Check confidence threshold
	if command.Confidence < 0.5 {
		return ports.NewValidationError("confidence",
			fmt.Sprintf("command confidence too low: %.2f", command.Confidence))
	}

	// Check if command has been interpreted
	if command.Interpreted == "" {
		return ports.NewValidationError("interpreted", "command has not been interpreted")
	}

	return nil
}

// postProcessResult applies post-processing to the result
func (uc *ExecuteCommandUseCase) postProcessResult(result *domain.CommandResult, command *domain.Command) {
	// Add command metadata to result if not present
	if result.CommandID == "" {
		result.CommandID = command.ID
	}

	// Ensure success flag is set correctly
	if result.Error != nil && result.Success {
		result.Success = false
	}

	// Calculate duration if not set
	if result.Duration == 0 && command.Timestamp.Before(time.Now()) {
		result.Duration = time.Since(command.Timestamp)
	}
}
