package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTask(t *testing.T) {
	t.Parallel()
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
			name:     "valid task with comma-separated sprints",
			key:      "TEST-2",
			summary:  "Multi-sprint task",
			project:  "TEST",
			sprint:   "Sprint 1, Sprint 2",
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

			// Test that sprints array is populated from sprint string
			if tt.sprint != "" {
				sprints := task.GetSprints()
				assert.NotEmpty(t, sprints)
				if tt.sprint == "Sprint 1, Sprint 2" {
					assert.Equal(t, []string{"Sprint 1", "Sprint 2"}, sprints)
				} else {
					assert.Equal(t, []string{tt.sprint}, sprints)
				}
			}
		})
	}
}

func TestNewTaskWithSprints(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		key      string
		summary  string
		project  string
		sprints  []string
		platform string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid task with single sprint",
			key:      "TEST-1",
			summary:  "Test task",
			project:  "TEST",
			sprints:  []string{"Sprint 1"},
			platform: "jira",
			wantErr:  false,
		},
		{
			name:     "valid task with multiple sprints",
			key:      "TEST-2",
			summary:  "Multi-sprint task",
			project:  "TEST",
			sprints:  []string{"Sprint 1", "Sprint 2", "Sprint 3"},
			platform: "jira",
			wantErr:  false,
		},
		{
			name:     "empty key",
			key:      "",
			summary:  "Test task",
			project:  "TEST",
			sprints:  []string{"Sprint 1"},
			platform: "jira",
			wantErr:  true,
			errMsg:   ErrEmptyKey.Error(),
		},
		{
			name:     "empty summary",
			key:      "TEST-1",
			summary:  "",
			project:  "TEST",
			sprints:  []string{"Sprint 1"},
			platform: "jira",
			wantErr:  true,
			errMsg:   ErrEmptySummary.Error(),
		},
		{
			name:     "empty project",
			key:      "TEST-1",
			summary:  "Test task",
			project:  "",
			sprints:  []string{"Sprint 1"},
			platform: "jira",
			wantErr:  true,
			errMsg:   ErrEmptyProject.Error(),
		},
		{
			name:     "empty sprints",
			key:      "TEST-1",
			summary:  "Test task",
			project:  "TEST",
			sprints:  []string{},
			platform: "jira",
			wantErr:  true,
			errMsg:   ErrEmptySprint.Error(),
		},
		{
			name:     "empty platform",
			key:      "TEST-1",
			summary:  "Test task",
			project:  "TEST",
			sprints:  []string{"Sprint 1"},
			platform: "",
			wantErr:  true,
			errMsg:   ErrEmptyPlatform.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewTaskWithSprints(tt.key, tt.summary, tt.project, tt.sprints, tt.platform)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.key, task.Key)
			assert.Equal(t, tt.summary, task.Summary)
			assert.Equal(t, tt.project, task.Project)
			assert.Equal(t, tt.sprints, task.Sprints)
			assert.Equal(t, tt.platform, task.Platform)
			assert.Equal(t, TaskStatusTodo, task.Status)
			assert.Equal(t, TaskTypeTask, task.Type)
			assert.Equal(t, TaskPriorityMedium, task.Priority)
			assert.Equal(t, 1, task.Version)
			assert.False(t, task.CreatedAt.IsZero())
			assert.False(t, task.UpdatedAt.IsZero())

			// Test that Sprint field is set correctly for backward compatibility
			expectedSprintStr := ""
			if len(tt.sprints) > 0 {
				if len(tt.sprints) == 1 {
					expectedSprintStr = tt.sprints[0]
				} else {
					expectedSprintStr = "Sprint 1, Sprint 2, Sprint 3"
				}
			}
			assert.Equal(t, expectedSprintStr, task.Sprint)
		})
	}
}

func TestGetSprints(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		task            *Task
		expectedSprints []string
	}{
		{
			name: "task with sprints array",
			task: &Task{
				Sprints: []string{"Sprint 1", "Sprint 2"},
				Sprint:  "Sprint 1, Sprint 2",
			},
			expectedSprints: []string{"Sprint 1", "Sprint 2"},
		},
		{
			name: "task with only sprint string",
			task: &Task{
				Sprint: "Sprint 1, Sprint 2",
			},
			expectedSprints: []string{"Sprint 1", "Sprint 2"},
		},
		{
			name: "task with single sprint string",
			task: &Task{
				Sprint: "Sprint 1",
			},
			expectedSprints: []string{"Sprint 1"},
		},
		{
			name:            "task with empty sprint data",
			task:            &Task{},
			expectedSprints: []string{},
		},
		{
			name: "task with whitespace in sprint string",
			task: &Task{
				Sprint: " Sprint 1 , Sprint 2 , Sprint 3 ",
			},
			expectedSprints: []string{"Sprint 1", "Sprint 2", "Sprint 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sprints := tt.task.GetSprints()
			assert.Equal(t, tt.expectedSprints, sprints)
		})
	}
}

func TestGetPrimarySprint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		task            *Task
		expectedPrimary string
	}{
		{
			name: "task with multiple sprints",
			task: &Task{
				Sprints: []string{"Sprint 1", "Sprint 2", "Sprint 3"},
			},
			expectedPrimary: "Sprint 1",
		},
		{
			name: "task with single sprint",
			task: &Task{
				Sprints: []string{"Sprint 1"},
			},
			expectedPrimary: "Sprint 1",
		},
		{
			name: "task with sprint string only",
			task: &Task{
				Sprint: "Sprint 1, Sprint 2",
			},
			expectedPrimary: "Sprint 1",
		},
		{
			name:            "task with no sprints",
			task:            &Task{},
			expectedPrimary: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary := tt.task.GetPrimarySprint()
			assert.Equal(t, tt.expectedPrimary, primary)
		})
	}
}

func TestHasSprint(t *testing.T) {
	t.Parallel()
	task := &Task{
		Sprints: []string{"Sprint 1", "Sprint 2", "Sprint 3"},
		Sprint:  "Sprint 1, Sprint 2, Sprint 3",
	}

	tests := []struct {
		name       string
		sprintName string
		expectHas  bool
	}{
		{
			name:       "has first sprint",
			sprintName: "Sprint 1",
			expectHas:  true,
		},
		{
			name:       "has middle sprint",
			sprintName: "Sprint 2",
			expectHas:  true,
		},
		{
			name:       "has last sprint",
			sprintName: "Sprint 3",
			expectHas:  true,
		},
		{
			name:       "does not have sprint",
			sprintName: "Sprint 4",
			expectHas:  false,
		},
		{
			name:       "empty sprint name",
			sprintName: "",
			expectHas:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := task.HasSprint(tt.sprintName)
			assert.Equal(t, tt.expectHas, result)
		})
	}
}

func TestSetSprints(t *testing.T) {
	t.Parallel()
	task := &Task{
		Key:     "TEST-1",
		Version: 1,
	}

	newSprints := []string{"New Sprint 1", "New Sprint 2"}
	task.SetSprints(newSprints)

	assert.Equal(t, newSprints, task.Sprints)
	assert.Equal(t, "New Sprint 1, New Sprint 2", task.Sprint)
	assert.Equal(t, 2, task.Version)
	assert.False(t, task.UpdatedAt.IsZero())
}

func TestAddSprint(t *testing.T) {
	t.Parallel()
	task := &Task{
		Key:     "TEST-1",
		Sprints: []string{"Sprint 1"},
		Sprint:  "Sprint 1",
		Version: 1,
	}

	// Add new sprint
	task.AddSprint("Sprint 2")
	assert.Equal(t, []string{"Sprint 1", "Sprint 2"}, task.Sprints)
	assert.Equal(t, "Sprint 1, Sprint 2", task.Sprint)
	assert.Equal(t, 2, task.Version)

	// Add existing sprint (should not change anything)
	oldVersion := task.Version
	task.AddSprint("Sprint 1")
	assert.Equal(t, []string{"Sprint 1", "Sprint 2"}, task.Sprints)
	assert.Equal(t, oldVersion, task.Version)
}

func TestRemoveSprint(t *testing.T) {
	t.Parallel()
	task := &Task{
		Key:     "TEST-1",
		Sprints: []string{"Sprint 1", "Sprint 2", "Sprint 3"},
		Sprint:  "Sprint 1, Sprint 2, Sprint 3",
		Version: 1,
	}

	// Remove middle sprint
	task.RemoveSprint("Sprint 2")
	assert.Equal(t, []string{"Sprint 1", "Sprint 3"}, task.Sprints)
	assert.Equal(t, "Sprint 1, Sprint 3", task.Sprint)
	assert.Equal(t, 2, task.Version)

	// Remove non-existing sprint (should not change anything)
	oldVersion := task.Version
	task.RemoveSprint("Sprint 4")
	assert.Equal(t, []string{"Sprint 1", "Sprint 3"}, task.Sprints)
	assert.Equal(t, oldVersion, task.Version)
}

func TestJSONMarshalUnmarshal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		task *Task
	}{
		{
			name: "task with sprints array",
			task: &Task{
				Key:     "TEST-1",
				Summary: "Test task",
				Project: "TEST",
				Sprints: []string{"Sprint 1", "Sprint 2"},
				Sprint:  "Sprint 1, Sprint 2",
			},
		},
		{
			name: "task with only sprint string (legacy format)",
			task: &Task{
				Key:     "TEST-2",
				Summary: "Legacy task",
				Project: "TEST",
				Sprint:  "Sprint 1, Sprint 2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.task)
			require.NoError(t, err)

			// Unmarshal from JSON
			var unmarshaled Task
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			// Verify data integrity
			assert.Equal(t, tt.task.Key, unmarshaled.Key)
			assert.Equal(t, tt.task.Summary, unmarshaled.Summary)
			assert.Equal(t, tt.task.Project, unmarshaled.Project)

			// Verify sprint consistency
			expectedSprints := tt.task.GetSprints()
			actualSprints := unmarshaled.GetSprints()
			assert.Equal(t, expectedSprints, actualSprints)

			// Verify both Sprint and Sprints fields are populated
			assert.NotEmpty(t, unmarshaled.Sprint)
			assert.NotEmpty(t, unmarshaled.Sprints)
		})
	}
}

func TestJSONBackwardCompatibility(t *testing.T) {
	t.Parallel()
	// Test unmarshaling legacy JSON format (only sprint field)
	legacyJSON := `{
		"key": "TEST-1",
		"summary": "Legacy task",
		"project": "TEST",
		"sprint": "Sprint 1, Sprint 2",
		"platform": "jira",
		"status": "TODO",
		"type": "TASK",
		"priority": "MEDIUM",
		"work_type": "",
		"labels": [],
		"epic": "",
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-01T00:00:00Z",
		"version": 1
	}`

	var task Task
	err := json.Unmarshal([]byte(legacyJSON), &task)
	require.NoError(t, err)

	// Verify that Sprints array is populated from Sprint string
	assert.Equal(t, "Sprint 1, Sprint 2", task.Sprint)
	assert.Equal(t, []string{"Sprint 1", "Sprint 2"}, task.Sprints)
	assert.True(t, task.HasSprint("Sprint 1"))
	assert.True(t, task.HasSprint("Sprint 2"))
	assert.Equal(t, "Sprint 1", task.GetPrimarySprint())
}

func TestJSONForwardCompatibility(t *testing.T) {
	t.Parallel()
	// Test marshaling new format and ensuring Sprint field is populated
	task := &Task{
		Key:     "TEST-1",
		Summary: "New task",
		Project: "TEST",
		Sprints: []string{"Sprint 1", "Sprint 2", "Sprint 3"},
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify both fields are present in JSON
	assert.Equal(t, "Sprint 1, Sprint 2, Sprint 3", result["sprint"])

	sprintsField, ok := result["sprints"].([]interface{})
	require.True(t, ok)
	assert.Len(t, sprintsField, 3)
	assert.Equal(t, "Sprint 1", sprintsField[0])
	assert.Equal(t, "Sprint 2", sprintsField[1])
	assert.Equal(t, "Sprint 3", sprintsField[2])
}

func TestUpdateStatus(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	task, err := NewTask("TEST-1", "Test task", "TEST", "Sprint 1", "jira")
	require.NoError(t, err)

	description := "Updated description"
	task.UpdateDescription(description)

	assert.Equal(t, description, task.Description)
	assert.Equal(t, 2, task.Version)
	assert.False(t, task.UpdatedAt.IsZero())
}

func TestStatusChecks(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestNewTaskWithoutSprint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		key         string
		summary     string
		project     string
		platform    string
		expectError bool
		errorType   error
	}{
		{
			name:        "valid task without sprint",
			key:         "TEST-1",
			summary:     "Test task",
			project:     "TEST",
			platform:    "JIRA",
			expectError: false,
		},
		{
			name:        "empty key",
			key:         "",
			summary:     "Test task",
			project:     "TEST",
			platform:    "JIRA",
			expectError: true,
			errorType:   ErrEmptyKey,
		},
		{
			name:        "empty summary",
			key:         "TEST-1",
			summary:     "",
			project:     "TEST",
			platform:    "JIRA",
			expectError: true,
			errorType:   ErrEmptySummary,
		},
		{
			name:        "empty project",
			key:         "TEST-1",
			summary:     "Test task",
			project:     "",
			platform:    "JIRA",
			expectError: true,
			errorType:   ErrEmptyProject,
		},
		{
			name:        "empty platform",
			key:         "TEST-1",
			summary:     "Test task",
			project:     "TEST",
			platform:    "",
			expectError: true,
			errorType:   ErrEmptyPlatform,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewTaskWithoutSprint(tt.key, tt.summary, tt.project, tt.platform)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorType, err)
				assert.Nil(t, task)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, task)
				assert.Equal(t, tt.key, task.Key)
				assert.Equal(t, tt.summary, task.Summary)
				assert.Equal(t, tt.project, task.Project)
				assert.Equal(t, tt.platform, task.Platform)
				assert.Equal(t, "", task.Sprint)
				assert.Empty(t, task.Sprints)
				assert.Equal(t, TaskStatusTodo, task.Status)
				assert.Equal(t, TaskTypeTask, task.Type)
				assert.Equal(t, TaskPriorityMedium, task.Priority)
			}
		})
	}
}
