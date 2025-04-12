package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

func TestRandomClassifier_ClassifyTask(t *testing.T) {
	classifier := NewRandomClassifier()

	// Create a test task
	task, err := domain.NewTask("TEST-1", "Test task", "TEST", "Sprint 1", "JIRA")
	assert.NoError(t, err)

	// Classify the task
	workType, err := classifier.ClassifyTask(task)
	assert.NoError(t, err)

	// Verify the work type is one of the expected values
	validWorkTypes := map[domain.WorkType]bool{
		domain.WorkTypeMaintenance: true,
		domain.WorkTypeDiscovery:   true,
		domain.WorkTypeDevelopment: true,
	}

	assert.True(t, validWorkTypes[workType], "Work type should be one of the valid types")
}

func TestRandomClassifier_ClassifyTasks(t *testing.T) {
	classifier := NewRandomClassifier()

	// Create test tasks
	tasks := make([]*domain.Task, 3)
	for i := 0; i < 3; i++ {
		task, err := domain.NewTask(
			"TEST-"+string(rune('1'+i)),
			"Test task",
			"TEST",
			"Sprint 1",
			"JIRA",
		)
		assert.NoError(t, err)
		tasks[i] = task
	}

	// Classify the tasks
	workTypes, err := classifier.ClassifyTasks(tasks)
	assert.NoError(t, err)

	// Verify we got classifications for all tasks
	assert.Equal(t, len(tasks), len(workTypes))

	// Verify each work type is valid
	validWorkTypes := map[domain.WorkType]bool{
		domain.WorkTypeMaintenance: true,
		domain.WorkTypeDiscovery:   true,
		domain.WorkTypeDevelopment: true,
	}

	for _, workType := range workTypes {
		assert.True(t, validWorkTypes[workType], "Work type should be one of the valid types")
	}
}
