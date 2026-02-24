package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// SprintLockStorage implements SprintLockRepository using a JSON file
type SprintLockStorage struct {
	dir  string
	file string
}

// NewSprintLockStorage creates a new SprintLockStorage instance
func NewSprintLockStorage(dir, file string) *SprintLockStorage {
	return &SprintLockStorage{
		dir:  dir,
		file: file,
	}
}

// FindLock returns the lock for a project+sprint pair, or nil if not locked
func (s *SprintLockStorage) FindLock(_ context.Context, project, sprint string) (*domain.SprintLock, error) {
	locks, err := s.loadLocks()
	if err != nil {
		return nil, fmt.Errorf("failed to load sprint locks: %w", err)
	}

	key := domain.SprintLockKey(project, sprint)
	lock, exists := locks[key]
	if !exists {
		return nil, nil
	}

	return lock, nil
}

// SaveLock persists a sprint lock
func (s *SprintLockStorage) SaveLock(_ context.Context, lock *domain.SprintLock) error {
	if lock == nil {
		return fmt.Errorf("sprint lock cannot be nil")
	}

	locks, err := s.loadLocks()
	if err != nil {
		return fmt.Errorf("failed to load sprint locks: %w", err)
	}

	locks[lock.Key()] = lock

	return s.saveLocks(locks)
}

func (s *SprintLockStorage) loadLocks() (map[string]*domain.SprintLock, error) {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(s.dir, s.file)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*domain.SprintLock), nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var locks map[string]*domain.SprintLock
	if err := json.Unmarshal(data, &locks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sprint locks: %w", err)
	}

	return locks, nil
}

func (s *SprintLockStorage) saveLocks(locks map[string]*domain.SprintLock) error {
	data, err := json.MarshalIndent(locks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sprint locks: %w", err)
	}

	filePath := filepath.Join(s.dir, s.file)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Ensure SprintLockStorage implements SprintLockRepository
var _ ports.SprintLockRepository = (*SprintLockStorage)(nil)
