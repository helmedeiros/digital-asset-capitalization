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

func TestTask_UpdateWorkType(t *testing.T) {
	task, err := NewTask("TEST-1", "Test Task", "TEST", "Sprint 1", "web")
	require.NoError(t, err)

	initialVersion := task.Version
	initialUpdatedAt := task.UpdatedAt

	t.Run("should update to valid work type - maintenance", func(t *testing.T) {
		err := task.UpdateWorkType(WorkTypeMaintenance)
		assert.NoError(t, err, "Should update to maintenance work type without error")
		assert.Equal(t, WorkTypeMaintenance, task.WorkType, "Work type should be set to maintenance")
		assert.Equal(t, initialVersion+1, task.Version, "Version should increment")
		assert.True(t, task.UpdatedAt.After(initialUpdatedAt), "UpdatedAt should be updated")
	})

	t.Run("should update to valid work type - discovery", func(t *testing.T) {
		currentVersion := task.Version
		currentUpdatedAt := task.UpdatedAt

		err := task.UpdateWorkType(WorkTypeDiscovery)
		assert.NoError(t, err, "Should update to discovery work type without error")
		assert.Equal(t, WorkTypeDiscovery, task.WorkType, "Work type should be set to discovery")
		assert.Equal(t, currentVersion+1, task.Version, "Version should increment")
		assert.True(t, task.UpdatedAt.After(currentUpdatedAt), "UpdatedAt should be updated")
	})

	t.Run("should update to valid work type - development", func(t *testing.T) {
		currentVersion := task.Version
		currentUpdatedAt := task.UpdatedAt

		err := task.UpdateWorkType(WorkTypeDevelopment)
		assert.NoError(t, err, "Should update to development work type without error")
		assert.Equal(t, WorkTypeDevelopment, task.WorkType, "Work type should be set to development")
		assert.Equal(t, currentVersion+1, task.Version, "Version should increment")
		assert.True(t, task.UpdatedAt.After(currentUpdatedAt), "UpdatedAt should be updated")
	})

	t.Run("should return error for invalid work type", func(t *testing.T) {
		currentWorkType := task.WorkType
		currentVersion := task.Version
		currentUpdatedAt := task.UpdatedAt

		err := task.UpdateWorkType(WorkType("invalid-work-type"))
		assert.Error(t, err, "Should return error for invalid work type")
		assert.ErrorIs(t, err, ErrInvalidWorkType, "Should return ErrInvalidWorkType")

		// State should remain unchanged
		assert.Equal(t, currentWorkType, task.WorkType, "Work type should remain unchanged")
		assert.Equal(t, currentVersion, task.Version, "Version should remain unchanged")
		assert.Equal(t, currentUpdatedAt, task.UpdatedAt, "UpdatedAt should remain unchanged")
	})

	t.Run("should handle empty work type", func(t *testing.T) {
		currentWorkType := task.WorkType
		currentVersion := task.Version

		err := task.UpdateWorkType(WorkType(""))
		assert.Error(t, err, "Should return error for empty work type")
		assert.ErrorIs(t, err, ErrInvalidWorkType, "Should return ErrInvalidWorkType")
		assert.Equal(t, currentWorkType, task.WorkType, "Work type should remain unchanged")
		assert.Equal(t, currentVersion, task.Version, "Version should remain unchanged")
	})

	t.Run("should handle all valid work types", func(t *testing.T) {
		validWorkTypes := []WorkType{
			WorkTypeMaintenance,
			WorkTypeDiscovery,
			WorkTypeDevelopment,
		}

		for _, workType := range validWorkTypes {
			err := task.UpdateWorkType(workType)
			assert.NoError(t, err, "Should accept valid work type: %s", workType)
			assert.Equal(t, workType, task.WorkType, "Work type should be set correctly")
		}
	})

	t.Run("should handle case sensitivity", func(t *testing.T) {
		currentWorkType := task.WorkType
		currentVersion := task.Version

		// Test uppercase version of valid work type
		err := task.UpdateWorkType(WorkType("CAP-MAINTENANCE"))
		assert.Error(t, err, "Should be case sensitive")
		assert.ErrorIs(t, err, ErrInvalidWorkType, "Should return ErrInvalidWorkType")
		assert.Equal(t, currentWorkType, task.WorkType, "Work type should remain unchanged")
		assert.Equal(t, currentVersion, task.Version, "Version should remain unchanged")
	})
}
