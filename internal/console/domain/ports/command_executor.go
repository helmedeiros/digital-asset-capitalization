package ports

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// CommandExecutor defines the interface for executing parsed commands
type CommandExecutor interface {
	// Execute runs the command and returns the result
	Execute(ctx context.Context, command *domain.Command) (*domain.CommandResult, error)

	// ValidateCommand checks if a command can be executed
	ValidateCommand(command *domain.Command) error

	// GetAvailableCommands returns all available commands for help/suggestions
	GetAvailableCommands() []CommandInfo
}

// CommandInfo provides information about an available command
type CommandInfo struct {
	Command     string
	Description string
	Examples    []string
	Parameters  []ParameterInfo
}

// ParameterInfo describes a command parameter
type ParameterInfo struct {
	Name        string
	Description string
	Required    bool
	Type        string // string, int, bool, etc.
	Default     interface{}
}

// ExecutionError represents errors during command execution
type ExecutionError struct {
	Command string
	Reason  string
	Help    string // Helpful message for the user
}

func (e *ExecutionError) Error() string {
	return e.Reason
}

// NewExecutionError creates a new execution error
func NewExecutionError(command, reason, help string) *ExecutionError {
	return &ExecutionError{
		Command: command,
		Reason:  reason,
		Help:    help,
	}
}

// ValidationError represents command validation errors
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Message
	}
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
