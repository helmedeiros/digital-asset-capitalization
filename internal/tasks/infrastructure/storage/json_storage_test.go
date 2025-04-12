package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// testDir is a temporary directory for test files
const testDir = "testdata"
const testFile = "test_tasks.json"

type testHelper struct {
	storage *JSONStorage
	dir     string
}

func setupTest(t *testing.T) *testHelper {
	t.Helper()

	// Create a unique test directory for each test
	dir := filepath.Join(testDir, t.Name())

	// Clean up any existing test data
	_ = os.RemoveAll(dir)

	// Create test directory
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err, "Failed to create test directory")

	storage := NewJSONStorage(dir, testFile)

	return &testHelper{
		storage: storage,
		dir:     dir,
	}
}

func (h *testHelper) cleanup(t *testing.T) {
	t.Helper()
	err := os.RemoveAll(h.dir)
	assert.NoError(t, err, "Failed to cleanup test directory")
}

func (h *testHelper) createTestTask(key, summary, project, sprint string) *domain.Task {
	task, err := domain.NewTask(key, summary, project, sprint, "test")
	if err != nil {
		panic(err)
	}
	return task
}

func TestJSONStorage_Save(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should save task successfully", func(t *testing.T) {
		task := h.createTestTask("TEST-1", "Test Task", "TEST", "Sprint 1")

		err := h.storage.Save(context.Background(), task)
		require.NoError(t, err, "Failed to save task")

		// Verify file exists
		filePath := filepath.Join(h.dir, testFile)
		_, err = os.Stat(filePath)
		require.NoError(t, err, "Expected file to exist")

		// Load and verify contents
		loaded, err := h.storage.FindByKey(context.Background(), task.Key)
		require.NoError(t, err, "Failed to load task")
		assert.Equal(t, task.Key, loaded.Key)
		assert.Equal(t, task.Summary, loaded.Summary)
		assert.Equal(t, task.Project, loaded.Project)
		assert.Equal(t, task.Sprint, loaded.Sprint)
	})

	t.Run("should update existing task", func(t *testing.T) {
		task := h.createTestTask("TEST-1", "Test Task", "TEST", "Sprint 1")
		err := h.storage.Save(context.Background(), task)
		require.NoError(t, err, "Failed to save initial task")

		// Update task
		task.UpdateDescription("Updated description")
		err = h.storage.Save(context.Background(), task)
		require.NoError(t, err, "Failed to update task")

		// Verify update
		loaded, err := h.storage.FindByKey(context.Background(), task.Key)
		require.NoError(t, err, "Failed to load updated task")
		assert.Equal(t, "Updated description", loaded.Description)
	})

	t.Run("should handle nil task", func(t *testing.T) {
		err := h.storage.Save(context.Background(), nil)
		assert.Error(t, err, "Expected error for nil task")
	})
}

func TestJSONStorage_FindByKey(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should find existing task", func(t *testing.T) {
		task := h.createTestTask("TEST-1", "Test Task", "TEST", "Sprint 1")
		err := h.storage.Save(context.Background(), task)
		require.NoError(t, err, "Failed to save task")

		loaded, err := h.storage.FindByKey(context.Background(), task.Key)
		require.NoError(t, err, "Failed to find task")
		assert.Equal(t, task.Key, loaded.Key)
	})

	t.Run("should return error for non-existent task", func(t *testing.T) {
		_, err := h.storage.FindByKey(context.Background(), "NON-EXISTENT")
		assert.Error(t, err, "Expected error for non-existent task")
	})

	t.Run("should return error for empty key", func(t *testing.T) {
		_, err := h.storage.FindByKey(context.Background(), "")
		assert.Error(t, err, "Expected error for empty key")
	})
}

func TestJSONStorage_FindByProjectAndSprint(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should find tasks for project and sprint", func(t *testing.T) {
		// Create test tasks
		task1 := h.createTestTask("TEST-1", "Task 1", "TEST", "Sprint 1")
		task2 := h.createTestTask("TEST-2", "Task 2", "TEST", "Sprint 1")
		task3 := h.createTestTask("TEST-3", "Task 3", "TEST", "Sprint 2")

		// Save tasks
		err := h.storage.Save(context.Background(), task1)
		require.NoError(t, err, "Failed to save task1")
		err = h.storage.Save(context.Background(), task2)
		require.NoError(t, err, "Failed to save task2")
		err = h.storage.Save(context.Background(), task3)
		require.NoError(t, err, "Failed to save task3")

		// Find tasks for TEST project and Sprint 1
		tasks, err := h.storage.FindByProjectAndSprint(context.Background(), "TEST", "Sprint 1")
		require.NoError(t, err, "Failed to find tasks")
		assert.Len(t, tasks, 2, "Expected 2 tasks")

		// Verify task contents
		taskKeys := make(map[string]bool)
		for _, task := range tasks {
			taskKeys[task.Key] = true
		}
		assert.True(t, taskKeys["TEST-1"], "Expected TEST-1")
		assert.True(t, taskKeys["TEST-2"], "Expected TEST-2")
	})

	t.Run("should return empty slice when no tasks found", func(t *testing.T) {
		tasks, err := h.storage.FindByProjectAndSprint(context.Background(), "NON-EXISTENT", "Sprint 1")
		require.NoError(t, err, "Failed to find tasks")
		assert.Empty(t, tasks, "Expected empty slice")
	})
}

func TestJSONStorage_FindByProject(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should find all tasks for project", func(t *testing.T) {
		// Create test tasks
		task1 := h.createTestTask("TEST-1", "Task 1", "TEST", "Sprint 1")
		task2 := h.createTestTask("TEST-2", "Task 2", "TEST", "Sprint 2")
		task3 := h.createTestTask("OTHER-1", "Task 3", "OTHER", "Sprint 1")

		// Save tasks
		err := h.storage.Save(context.Background(), task1)
		require.NoError(t, err, "Failed to save task1")
		err = h.storage.Save(context.Background(), task2)
		require.NoError(t, err, "Failed to save task2")
		err = h.storage.Save(context.Background(), task3)
		require.NoError(t, err, "Failed to save task3")

		// Find tasks for TEST project
		tasks, err := h.storage.FindByProject(context.Background(), "TEST")
		require.NoError(t, err, "Failed to find tasks")
		assert.Len(t, tasks, 2, "Expected 2 tasks")

		// Verify task contents
		taskKeys := make(map[string]bool)
		for _, task := range tasks {
			taskKeys[task.Key] = true
		}
		assert.True(t, taskKeys["TEST-1"], "Expected TEST-1")
		assert.True(t, taskKeys["TEST-2"], "Expected TEST-2")
	})
}

func TestJSONStorage_FindBySprint(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should find all tasks for sprint", func(t *testing.T) {
		// Create test tasks
		task1 := h.createTestTask("TEST-1", "Task 1", "TEST", "Sprint 1")
		task2 := h.createTestTask("OTHER-1", "Task 2", "OTHER", "Sprint 1")
		task3 := h.createTestTask("TEST-2", "Task 3", "TEST", "Sprint 2")

		// Save tasks
		err := h.storage.Save(context.Background(), task1)
		require.NoError(t, err, "Failed to save task1")
		err = h.storage.Save(context.Background(), task2)
		require.NoError(t, err, "Failed to save task2")
		err = h.storage.Save(context.Background(), task3)
		require.NoError(t, err, "Failed to save task3")

		// Find tasks for Sprint 1
		tasks, err := h.storage.FindBySprint(context.Background(), "Sprint 1")
		require.NoError(t, err, "Failed to find tasks")
		assert.Len(t, tasks, 2, "Expected 2 tasks")

		// Verify task contents
		taskKeys := make(map[string]bool)
		for _, task := range tasks {
			taskKeys[task.Key] = true
		}
		assert.True(t, taskKeys["TEST-1"], "Expected TEST-1")
		assert.True(t, taskKeys["OTHER-1"], "Expected OTHER-1")
	})
}

func TestJSONStorage_FindByPlatform(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should find all tasks for platform", func(t *testing.T) {
		// Create test tasks with different platforms
		task1 := h.createTestTask("TEST-1", "Task 1", "TEST", "Sprint 1")
		task2 := h.createTestTask("TEST-2", "Task 2", "TEST", "Sprint 1")
		task3 := h.createTestTask("TEST-3", "Task 3", "TEST", "Sprint 1")
		task3.Platform = "other"

		// Save tasks
		err := h.storage.Save(context.Background(), task1)
		require.NoError(t, err, "Failed to save task1")
		err = h.storage.Save(context.Background(), task2)
		require.NoError(t, err, "Failed to save task2")
		err = h.storage.Save(context.Background(), task3)
		require.NoError(t, err, "Failed to save task3")

		// Find tasks for test platform
		tasks, err := h.storage.FindByPlatform(context.Background(), "test")
		require.NoError(t, err, "Failed to find tasks")
		assert.Len(t, tasks, 2, "Expected 2 tasks")

		// Verify task contents
		taskKeys := make(map[string]bool)
		for _, task := range tasks {
			taskKeys[task.Key] = true
		}
		assert.True(t, taskKeys["TEST-1"], "Expected TEST-1")
		assert.True(t, taskKeys["TEST-2"], "Expected TEST-2")
	})
}

func TestJSONStorage_FindAll(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should find all tasks", func(t *testing.T) {
		// Create test tasks
		task1 := h.createTestTask("TEST-1", "Task 1", "TEST", "Sprint 1")
		task2 := h.createTestTask("TEST-2", "Task 2", "TEST", "Sprint 2")
		task3 := h.createTestTask("OTHER-1", "Task 3", "OTHER", "Sprint 1")

		// Save tasks
		err := h.storage.Save(context.Background(), task1)
		require.NoError(t, err, "Failed to save task1")
		err = h.storage.Save(context.Background(), task2)
		require.NoError(t, err, "Failed to save task2")
		err = h.storage.Save(context.Background(), task3)
		require.NoError(t, err, "Failed to save task3")

		// Find all tasks
		tasks, err := h.storage.FindAll(context.Background())
		require.NoError(t, err, "Failed to find tasks")
		assert.Len(t, tasks, 3, "Expected 3 tasks")

		// Verify task contents
		taskKeys := make(map[string]bool)
		for _, task := range tasks {
			taskKeys[task.Key] = true
		}
		assert.True(t, taskKeys["TEST-1"], "Expected TEST-1")
		assert.True(t, taskKeys["TEST-2"], "Expected TEST-2")
		assert.True(t, taskKeys["OTHER-1"], "Expected OTHER-1")
	})
}

func TestJSONStorage_Delete(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should delete existing task", func(t *testing.T) {
		task := h.createTestTask("TEST-1", "Test Task", "TEST", "Sprint 1")
		err := h.storage.Save(context.Background(), task)
		require.NoError(t, err, "Failed to save task")

		err = h.storage.Delete(context.Background(), task.Key)
		require.NoError(t, err, "Failed to delete task")

		_, err = h.storage.FindByKey(context.Background(), task.Key)
		assert.Error(t, err, "Expected error when finding deleted task")
	})

	t.Run("should return error for non-existent task", func(t *testing.T) {
		err := h.storage.Delete(context.Background(), "NON-EXISTENT")
		assert.Error(t, err, "Expected error for non-existent task")
	})

	t.Run("should return error for empty key", func(t *testing.T) {
		err := h.storage.Delete(context.Background(), "")
		assert.Error(t, err, "Expected error for empty key")
	})
}

func TestJSONStorage_DeleteByProjectAndSprint(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should delete tasks for project and sprint", func(t *testing.T) {
		// Create test tasks
		task1 := h.createTestTask("TEST-1", "Task 1", "TEST", "Sprint 1")
		task2 := h.createTestTask("TEST-2", "Task 2", "TEST", "Sprint 1")
		task3 := h.createTestTask("TEST-3", "Task 3", "TEST", "Sprint 2")
		task4 := h.createTestTask("OTHER-1", "Task 4", "OTHER", "Sprint 1")

		// Save tasks
		err := h.storage.Save(context.Background(), task1)
		require.NoError(t, err, "Failed to save task1")
		err = h.storage.Save(context.Background(), task2)
		require.NoError(t, err, "Failed to save task2")
		err = h.storage.Save(context.Background(), task3)
		require.NoError(t, err, "Failed to save task3")
		err = h.storage.Save(context.Background(), task4)
		require.NoError(t, err, "Failed to save task4")

		// Delete tasks for TEST project and Sprint 1
		err = h.storage.DeleteByProjectAndSprint(context.Background(), "TEST", "Sprint 1")
		require.NoError(t, err, "Failed to delete tasks")

		// Verify remaining tasks
		tasks, err := h.storage.FindAll(context.Background())
		require.NoError(t, err, "Failed to find tasks")
		assert.Len(t, tasks, 2, "Expected 2 tasks")

		// Verify task contents
		taskKeys := make(map[string]bool)
		for _, task := range tasks {
			taskKeys[task.Key] = true
		}
		assert.True(t, taskKeys["TEST-3"], "Expected TEST-3")
		assert.True(t, taskKeys["OTHER-1"], "Expected OTHER-1")
	})
}

func TestJSONStorage_EdgeCases(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should handle empty task map", func(t *testing.T) {
		// Save empty map
		err := h.storage.saveTasks(make(map[string]*domain.Task))
		require.NoError(t, err, "Failed to save empty map")

		// Load and verify
		tasks, err := h.storage.loadTasks()
		require.NoError(t, err, "Failed to load empty map")
		assert.Empty(t, tasks, "Expected empty map")
	})

	t.Run("should handle large number of tasks", func(t *testing.T) {
		// Create many tasks
		tasks := make(map[string]*domain.Task)
		for i := 0; i < 1000; i++ {
			task := h.createTestTask(
				fmt.Sprintf("TEST-%d", i),
				fmt.Sprintf("Task %d", i),
				"TEST",
				"Sprint 1",
			)
			tasks[task.Key] = task
		}

		// Save tasks
		err := h.storage.saveTasks(tasks)
		require.NoError(t, err, "Failed to save many tasks")

		// Load and verify
		loaded, err := h.storage.loadTasks()
		require.NoError(t, err, "Failed to load many tasks")
		assert.Len(t, loaded, 1000, "Expected 1000 tasks")
	})

	t.Run("should handle invalid JSON file", func(t *testing.T) {
		// Write invalid JSON to file
		filePath := filepath.Join(h.dir, testFile)
		err := os.WriteFile(filePath, []byte("invalid json"), 0644)
		require.NoError(t, err, "Failed to write invalid JSON")

		// Try to load tasks
		_, err = h.storage.loadTasks()
		assert.Error(t, err, "Expected error loading invalid JSON")
	})
}
