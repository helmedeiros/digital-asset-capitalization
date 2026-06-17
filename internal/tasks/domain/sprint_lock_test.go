package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSprintLockKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		project  string
		sprint   string
		expected string
	}{
		{
			name:     "simple key",
			project:  "COP",
			sprint:   "Sprint 1",
			expected: "COP::Sprint 1",
		},
		{
			name:     "key with special characters in sprint",
			project:  "FN",
			sprint:   "The Hulk (Q1 2025)",
			expected: "FN::The Hulk (Q1 2025)",
		},
		{
			name:     "empty project",
			project:  "",
			sprint:   "Sprint 1",
			expected: "::Sprint 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SprintLockKey(tt.project, tt.sprint)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewSprintLock(t *testing.T) {
	t.Parallel()
	before := time.Now()
	lock := NewSprintLock("COP", "Sprint 1", 5)
	after := time.Now()

	assert.Equal(t, "COP", lock.Project)
	assert.Equal(t, "Sprint 1", lock.Sprint)
	assert.Equal(t, 5, lock.TaskCount)
	assert.True(t, lock.LockedAt.After(before) || lock.LockedAt.Equal(before))
	assert.True(t, lock.LockedAt.Before(after) || lock.LockedAt.Equal(after))
}

func TestSprintLock_Key(t *testing.T) {
	t.Parallel()
	lock := &SprintLock{
		Project:   "COP",
		Sprint:    "Sprint 1",
		LockedAt:  time.Now(),
		TaskCount: 3,
	}

	assert.Equal(t, "COP::Sprint 1", lock.Key())
}
