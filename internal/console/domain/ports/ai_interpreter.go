package ports

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// AIInterpreter defines the interface for natural language interpretation
type AIInterpreter interface {
	// Interpret converts natural language input to a structured command
	Interpret(ctx context.Context, input string, sessionContext *domain.Context) (*domain.Command, error)

	// GetClarification generates a clarifying question when input is ambiguous
	GetClarification(ctx context.Context, ambiguity string, options []string) (string, error)

	// AnalyzeIntent performs deeper analysis of user intent
	AnalyzeIntent(ctx context.Context, input string) (*domain.CommandIntent, error)
}

// InterpretationError represents errors during interpretation
type InterpretationError struct {
	Input   string
	Reason  string
	Options []string // Possible interpretations
}

func (e *InterpretationError) Error() string {
	return e.Reason
}

// NewInterpretationError creates a new interpretation error
func NewInterpretationError(input, reason string, options []string) *InterpretationError {
	return &InterpretationError{
		Input:   input,
		Reason:  reason,
		Options: options,
	}
}
