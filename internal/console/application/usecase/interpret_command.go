package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain/ports"
)

// InterpretCommandUseCase handles natural language command interpretation
type InterpretCommandUseCase struct {
	interpreter ports.AIInterpreter
}

// NewInterpretCommandUseCase creates a new interpret command use case
func NewInterpretCommandUseCase(interpreter ports.AIInterpreter) *InterpretCommandUseCase {
	return &InterpretCommandUseCase{
		interpreter: interpreter,
	}
}

// Execute interprets user input into a structured command
func (uc *InterpretCommandUseCase) Execute(ctx context.Context, input string, sessionContext *domain.Context) (*domain.Command, error) {
	// Validate input
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("input cannot be empty")
	}

	// Check for special commands that don't need AI interpretation
	if cmd := uc.handleSpecialCommands(input, sessionContext); cmd != nil {
		return cmd, nil
	}

	// Call AI interpreter
	command, err := uc.interpreter.Interpret(ctx, input, sessionContext)
	if err != nil {
		return nil, err
	}

	return command, nil
}

// handleSpecialCommands handles commands that don't need AI interpretation
func (uc *InterpretCommandUseCase) handleSpecialCommands(input string, context *domain.Context) *domain.Command {
	lowerInput := strings.ToLower(input)

	// Exit commands
	if lowerInput == "exit" || lowerInput == "quit" || lowerInput == "bye" {
		cmd, _ := domain.NewCommand(context.SessionID, input, "exit", 1.0)
		cmd.SetIntent(domain.CommandTypeOther, domain.ResourceTypeContext, "")
		return cmd
	}

	// Help commands
	if lowerInput == "help" || lowerInput == "?" {
		cmd, _ := domain.NewCommand(context.SessionID, input, "help", 1.0)
		cmd.SetIntent(domain.CommandTypeHelp, domain.ResourceTypeUnknown, "")
		return cmd
	}

	return nil
}
