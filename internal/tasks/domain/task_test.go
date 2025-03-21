package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTask(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		summary  string
		project  string
		sprint   string
		platform string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid task",
			key:      "TEST-1",
			summary:  "Test task",
			project:  "TEST",
			sprint:   "Sprint 1",
			platform: "jira",
			wantErr:  false,
		},
		{
			name:     "empty key",
			key:      "",
			summary:  "Test task",
			project:  "TEST",
			sprint:   "Sprint 1",
			platform: "jira",
			wantErr:  true,
			errMsg:   ErrEmptyKey.Error(),
		},
		{
			name:     "empty summary",
			key:      "TEST-1",
			summary:  "",
			project:  "TEST",
			sprint:   "Sprint 1",
			platform: "jira",
			wantErr:  true,
			errMsg:   ErrEmptySummary.Error(),
		},
		{
			name:     "empty project",
			key:      "TEST-1",
			summary:  "Test task",
			project:  "",
			sprint:   "Sprint 1",
			platform: "jira",
			wantErr:  true,
			errMsg:   ErrEmptyProject.Error(),
		},
		{
			name:     "empty sprint",
			key:      "TEST-1",
			summary:  "Test task",
			project:  "TEST",
			sprint:   "",
			platform: "jira",
			wantErr:  true,
			errMsg:   ErrEmptySprint.Error(),
		},
		{
			name:     "empty platform",
			key:      "TEST-1",
			summary:  "Test task",
			project:  "TEST",
			sprint:   "Sprint 1",
			platform: "",
			wantErr:  true,
			errMsg:   ErrEmptyPlatform.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewTask(tt.key, tt.summary, tt.project, tt.sprint, tt.platform)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.key, task.Key)
			assert.Equal(t, tt.summary, task.Summary)
			assert.Equal(t, tt.project, task.Project)
			assert.Equal(t, tt.sprint, task.Sprint)
			assert.Equal(t, tt.platform, task.Platform)
			assert.Equal(t, TaskStatusTodo, task.Status)
			assert.Equal(t, TaskTypeTask, task.Type)
			assert.Equal(t, TaskPriorityMedium, task.Priority)
			assert.Equal(t, 1, task.Version)
			assert.False(t, task.CreatedAt.IsZero())
			assert.False(t, task.UpdatedAt.IsZero())
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	task, err := NewTask("TEST-1", "Test task", "TEST", "Sprint 1", "jira")
	require.NoError(t, err)

	tests := []struct {
		name        string
		status      TaskStatus
		wantErr     bool
		errMsg      string
		expectedVer int
	}{
		{
			name:        "valid status todo",
			status:      TaskStatusTodo,
			wantErr:     false,
			expectedVer: 2,
		},
		{
			name:        "valid status in progress",
			status:      TaskStatusInProgress,
			wantErr:     false,
			expectedVer: 3,
		},
		{
			name:        "valid status done",
			status:      TaskStatusDone,
			wantErr:     false,
			expectedVer: 4,
		},
		{
			name:        "valid status blocked",
			status:      TaskStatusBlocked,
			wantErr:     false,
			expectedVer: 5,
		},
		{
			name:    "invalid status",
			status:  "INVALID",
			wantErr: true,
			errMsg:  ErrInvalidStatus.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := task.UpdateStatus(tt.status)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.status, task.Status)
			assert.Equal(t, tt.expectedVer, task.Version)
			assert.False(t, task.UpdatedAt.IsZero())
		})
	}
}

func TestUpdateType(t *testing.T) {
	task, err := NewTask("TEST-1", "Test task", "TEST", "Sprint 1", "jira")
	require.NoError(t, err)

	tests := []struct {
		name        string
		taskType    TaskType
		wantErr     bool
		errMsg      string
		expectedVer int
	}{
		{
			name:        "valid type story",
			taskType:    TaskTypeStory,
			wantErr:     false,
			expectedVer: 2,
		},
		{
			name:        "valid type task",
			taskType:    TaskTypeTask,
			wantErr:     false,
			expectedVer: 3,
		},
		{
			name:        "valid type bug",
			taskType:    TaskTypeBug,
			wantErr:     false,
			expectedVer: 4,
		},
		{
			name:        "valid type epic",
			taskType:    TaskTypeEpic,
			wantErr:     false,
			expectedVer: 5,
		},
		{
			name:        "valid type subtask",
			taskType:    TaskTypeSubtask,
			wantErr:     false,
			expectedVer: 6,
		},
		{
			name:     "invalid type",
			taskType: "INVALID",
			wantErr:  true,
			errMsg:   ErrInvalidType.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := task.UpdateType(tt.taskType)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.taskType, task.Type)
			assert.Equal(t, tt.expectedVer, task.Version)
			assert.False(t, task.UpdatedAt.IsZero())
		})
	}
}

func TestUpdatePriority(t *testing.T) {
	task, err := NewTask("TEST-1", "Test task", "TEST", "Sprint 1", "jira")
	require.NoError(t, err)

	tests := []struct {
		name        string
		priority    TaskPriority
		wantErr     bool
		errMsg      string
		expectedVer int
	}{
		{
			name:        "valid priority highest",
			priority:    TaskPriorityHighest,
			wantErr:     false,
			expectedVer: 2,
		},
		{
			name:        "valid priority high",
			priority:    TaskPriorityHigh,
			wantErr:     false,
			expectedVer: 3,
		},
		{
			name:        "valid priority medium",
			priority:    TaskPriorityMedium,
			wantErr:     false,
			expectedVer: 4,
		},
		{
			name:        "valid priority low",
			priority:    TaskPriorityLow,
			wantErr:     false,
			expectedVer: 5,
		},
		{
			name:        "valid priority lowest",
			priority:    TaskPriorityLowest,
			wantErr:     false,
			expectedVer: 6,
		},
		{
			name:     "invalid priority",
			priority: "INVALID",
			wantErr:  true,
			errMsg:   ErrInvalidPriority.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := task.UpdatePriority(tt.priority)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.priority, task.Priority)
			assert.Equal(t, tt.expectedVer, task.Version)
			assert.False(t, task.UpdatedAt.IsZero())
		})
	}
}

func TestUpdateDescription(t *testing.T) {
	task, err := NewTask("TEST-1", "Test task", "TEST", "Sprint 1", "jira")
	require.NoError(t, err)

	description := "Updated description"
	task.UpdateDescription(description)

	assert.Equal(t, description, task.Description)
	assert.Equal(t, 2, task.Version)
	assert.False(t, task.UpdatedAt.IsZero())
}

func TestStatusChecks(t *testing.T) {
	task, err := NewTask("TEST-1", "Test task", "TEST", "Sprint 1", "jira")
	require.NoError(t, err)

	t.Run("initial status is todo", func(t *testing.T) {
		assert.False(t, task.IsDone())
		assert.False(t, task.IsInProgress())
		assert.False(t, task.IsBlocked())
	})

	t.Run("status done", func(t *testing.T) {
		err := task.UpdateStatus(TaskStatusDone)
		require.NoError(t, err)
		assert.True(t, task.IsDone())
		assert.False(t, task.IsInProgress())
		assert.False(t, task.IsBlocked())
	})

	t.Run("status in progress", func(t *testing.T) {
		err := task.UpdateStatus(TaskStatusInProgress)
		require.NoError(t, err)
		assert.False(t, task.IsDone())
		assert.True(t, task.IsInProgress())
		assert.False(t, task.IsBlocked())
	})

	t.Run("status blocked", func(t *testing.T) {
		err := task.UpdateStatus(TaskStatusBlocked)
		require.NoError(t, err)
		assert.False(t, task.IsDone())
		assert.False(t, task.IsInProgress())
		assert.True(t, task.IsBlocked())
	})
}
