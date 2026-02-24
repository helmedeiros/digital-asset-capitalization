package domain

import (
	"fmt"
	"time"
)

// SprintLock represents a lock on a classified sprint to prevent accidental re-classification
type SprintLock struct {
	Project   string    `json:"project"`
	Sprint    string    `json:"sprint"`
	LockedAt  time.Time `json:"locked_at"`
	TaskCount int       `json:"task_count"`
}

// NewSprintLock creates a new SprintLock
func NewSprintLock(project, sprint string, taskCount int) *SprintLock {
	return &SprintLock{
		Project:   project,
		Sprint:    sprint,
		LockedAt:  time.Now(),
		TaskCount: taskCount,
	}
}

// Key returns the unique key for the sprint lock
func (l *SprintLock) Key() string {
	return SprintLockKey(l.Project, l.Sprint)
}

// SprintLockKey returns a composite key for a project+sprint pair
func SprintLockKey(project, sprint string) string {
	return fmt.Sprintf("%s::%s", project, sprint)
}
