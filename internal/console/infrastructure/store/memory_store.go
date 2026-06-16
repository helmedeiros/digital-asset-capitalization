package store

import (
	"context"
	"sync"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain/ports"
)

// MemoryStore implements in-memory context storage
type MemoryStore struct {
	contexts    map[string]*domain.Context
	mu          sync.RWMutex
	maxSessions int
	sessionTTL  time.Duration
}

// Config holds configuration for the memory store
type Config struct {
	MaxSessions int
	SessionTTL  time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		MaxSessions: 100,
		SessionTTL:  30 * time.Minute,
	}
}

// NewMemoryStore creates a new in-memory context store
func NewMemoryStore(config Config) *MemoryStore {
	store := &MemoryStore{
		contexts:    make(map[string]*domain.Context),
		maxSessions: config.MaxSessions,
		sessionTTL:  config.SessionTTL,
	}

	// Start cleanup routine
	go store.cleanupRoutine()

	return store
}

// Save stores or updates a context
func (s *MemoryStore) Save(_ context.Context, sessionContext *domain.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if we're at capacity and this is a new session
	if len(s.contexts) >= s.maxSessions {
		if _, exists := s.contexts[sessionContext.SessionID]; !exists {
			return ports.ErrStoreFull
		}
	}

	// Deep copy to avoid race conditions
	contextCopy := s.copyContext(sessionContext)
	s.contexts[sessionContext.SessionID] = contextCopy

	return nil
}

// Load retrieves a context by session ID
func (s *MemoryStore) Load(_ context.Context, sessionID string) (*domain.Context, error) {
	s.mu.RLock()
	sessionContext, exists := s.contexts[sessionID]
	s.mu.RUnlock()

	if !exists {
		return nil, ports.ErrContextNotFound
	}

	// Check if expired
	if sessionContext.IsExpired(s.sessionTTL) {
		s.mu.Lock()
		delete(s.contexts, sessionID)
		s.mu.Unlock()
		return nil, ports.ErrContextExpired
	}

	// Return a copy to avoid race conditions
	return s.copyContext(sessionContext), nil
}

// Update atomically updates a context
func (s *MemoryStore) Update(_ context.Context, sessionID string, updateFn func(*domain.Context) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionContext, exists := s.contexts[sessionID]
	if !exists {
		return ports.ErrContextNotFound
	}

	// Check if expired
	if sessionContext.IsExpired(s.sessionTTL) {
		delete(s.contexts, sessionID)
		return ports.ErrContextExpired
	}

	// Apply the update function
	if err := updateFn(sessionContext); err != nil {
		return err
	}

	// Update last activity
	sessionContext.LastActivity = time.Now()

	return nil
}

// Delete removes a context
func (s *MemoryStore) Delete(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.contexts, sessionID)
	return nil
}

// List returns all active session IDs
func (s *MemoryStore) List(_ context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionIDs := make([]string, 0, len(s.contexts))
	for sessionID, sessionContext := range s.contexts {
		// Only include non-expired sessions
		if !sessionContext.IsExpired(s.sessionTTL) {
			sessionIDs = append(sessionIDs, sessionID)
		}
	}

	return sessionIDs, nil
}

// CleanupExpired removes contexts that have exceeded the timeout
func (s *MemoryStore) CleanupExpired(_ context.Context, timeoutMinutes int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	timeout := time.Duration(timeoutMinutes) * time.Minute
	now := time.Now()

	for sessionID, sessionContext := range s.contexts {
		if now.Sub(sessionContext.LastActivity) > timeout {
			delete(s.contexts, sessionID)
		}
	}

	return nil
}

// GetStats returns statistics about the store
func (s *MemoryStore) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := Stats{
		TotalSessions:   len(s.contexts),
		MaxSessions:     s.maxSessions,
		SessionTTL:      s.sessionTTL,
		ExpiredSessions: 0,
	}

	now := time.Now()
	for _, sessionContext := range s.contexts {
		if now.Sub(sessionContext.LastActivity) > s.sessionTTL {
			stats.ExpiredSessions++
		}
	}

	return stats
}

// copyContext creates a deep copy of a context to avoid race conditions
func (s *MemoryStore) copyContext(original *domain.Context) *domain.Context {
	contextCopy := domain.NewContext(original.SessionID)
	contextCopy.StartTime = original.StartTime
	contextCopy.LastActivity = original.LastActivity
	contextCopy.CurrentProject = original.CurrentProject
	contextCopy.CurrentSprint = original.CurrentSprint
	contextCopy.CurrentSpace = original.CurrentSpace
	contextCopy.PreferredFormat = original.PreferredFormat
	contextCopy.Verbosity = original.Verbosity

	// Copy commands
	contextCopy.Commands = make([]domain.Command, len(original.Commands))
	for i, cmd := range original.Commands {
		contextCopy.Commands[i] = cmd
		// Deep copy parameters
		if cmd.Parameters != nil {
			contextCopy.Commands[i].Parameters = make(map[string]interface{})
			for k, v := range cmd.Parameters {
				contextCopy.Commands[i].Parameters[k] = v
			}
		}
	}

	// Copy command results
	contextCopy.CommandResults = make(map[string]domain.CommandResult)
	for k, v := range original.CommandResults {
		contextCopy.CommandResults[k] = v
	}

	// Copy recent entities
	contextCopy.RecentAssets = make([]string, len(original.RecentAssets))
	copy(contextCopy.RecentAssets, original.RecentAssets)

	contextCopy.RecentTasks = make([]string, len(original.RecentTasks))
	copy(contextCopy.RecentTasks, original.RecentTasks)

	// Copy variables
	if original.Variables != nil {
		contextCopy.Variables = make(map[string]interface{})
		for k, v := range original.Variables {
			contextCopy.Variables[k] = v
		}
	}

	// Note: LastAsset, LastTask, LastTeam are interfaces,
	// so we just copy the reference (shallow copy for these)
	contextCopy.LastAsset = original.LastAsset
	contextCopy.LastTask = original.LastTask
	contextCopy.LastTeam = original.LastTeam

	return contextCopy
}

// cleanupRoutine runs periodically to clean up expired sessions.
func (s *MemoryStore) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.sweepExpired(time.Now())
	}
}

// sweepExpired removes contexts older than sessionTTL. Extracted from
// cleanupRoutine so the sweep logic is unit-testable without driving
// a 5-minute ticker.
func (s *MemoryStore) sweepExpired(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for sessionID, sessionContext := range s.contexts {
		if now.Sub(sessionContext.LastActivity) > s.sessionTTL {
			delete(s.contexts, sessionID)
		}
	}
}

// Stats contains statistics about the store
type Stats struct {
	TotalSessions   int
	ExpiredSessions int
	MaxSessions     int
	SessionTTL      time.Duration
}
