package domain

import "errors"

// Common domain errors
var (
	// Command errors
	ErrInvalidCommand   = errors.New("invalid command")
	ErrCommandNotFound  = errors.New("command not found")
	ErrLowConfidence    = errors.New("low confidence in command interpretation")
	ErrAmbiguousCommand = errors.New("ambiguous command")

	// Context errors
	ErrInvalidContext  = errors.New("invalid context")
	ErrContextExpired  = errors.New("context has expired")
	ErrNoActiveSession = errors.New("no active session")

	// Parameter errors
	ErrMissingParameter  = errors.New("missing required parameter")
	ErrInvalidParameter  = errors.New("invalid parameter value")
	ErrTooManyParameters = errors.New("too many parameters")
)
