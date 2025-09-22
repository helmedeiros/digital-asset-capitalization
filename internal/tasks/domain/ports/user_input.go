package ports

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

// UserInput defines the interface for handling user interactions
type UserInput interface {
	// Confirm asks the user for a yes/no confirmation
	Confirm(format string, args ...interface{}) (bool, error)
}

// SprintCandidate represents a sprint candidate for selection
type SprintCandidate struct {
	Sprint ports.Sprint
	Reason string // Why this sprint was matched (e.g., "exact match", "name contains 'Panama'")
}

// SprintSelectionPort defines the interface for interactive sprint selection
type SprintSelectionPort interface {
	// SelectSprint presents sprint options to user and returns selected sprint
	// Returns nil if user cancels selection
	SelectSprint(candidates []SprintCandidate) (*ports.Sprint, error)
}
