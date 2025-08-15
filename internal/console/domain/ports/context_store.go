package ports

import (
	"context"
	"errors"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// ContextStore defines the interface for storing and retrieving session contexts
type ContextStore interface {
	// Save stores or updates a context
	Save(ctx context.Context, sessionContext *domain.Context) error

	// Load retrieves a context by session ID
	Load(ctx context.Context, sessionID string) (*domain.Context, error)

	// Update atomically updates a context
	Update(ctx context.Context, sessionID string, updateFn func(*domain.Context) error) error

	// Delete removes a context
	Delete(ctx context.Context, sessionID string) error

	// List returns all active session IDs
	List(ctx context.Context) ([]string, error)

	// CleanupExpired removes contexts that have exceeded the timeout
	CleanupExpired(ctx context.Context, timeout int) error
}

// Common errors for context store operations
var (
	ErrContextNotFound = errors.New("context not found")
	ErrContextExpired  = errors.New("context has expired")
	ErrStoreFull       = errors.New("context store is full")
)

// StoreError represents errors during store operations
type StoreError struct {
	Operation string
	SessionID string
	Err       error
}

func (e *StoreError) Error() string {
	return e.Operation + " failed for session " + e.SessionID + ": " + e.Err.Error()
}

func (e *StoreError) Unwrap() error {
	return e.Err
}

// NewStoreError creates a new store error
func NewStoreError(operation, sessionID string, err error) *StoreError {
	return &StoreError{
		Operation: operation,
		SessionID: sessionID,
		Err:       err,
	}
}
