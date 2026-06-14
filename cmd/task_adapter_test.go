package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tasksdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

func newSampleTask(key string, sprints ...string) *tasksdomain.Task {
	t := &tasksdomain.Task{
		Key:       key,
		Summary:   "Sample",
		Status:    tasksdomain.TaskStatusInProgress,
		Type:      tasksdomain.TaskTypeStory,
		Priority:  tasksdomain.TaskPriorityMedium,
		WorkType:  tasksdomain.WorkTypeDevelopment,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Labels:    []string{"cap-asset-payment-gateway"},
		Epic:      "EPIC-1",
	}
	if len(sprints) > 0 {
		t.Sprint = sprints[0]
	}
	return t
}

func TestTaskServiceAdapter_ClassifyTasks(t *testing.T) {
	ctx := context.Background()

	t.Run("empty project and empty sprint rejected", func(t *testing.T) {
		adapter := &TaskServiceAdapter{service: &MockTaskService{}}
		_, err := adapter.ClassifyTasks(ctx, "", "S1", false)
		require.Error(t, err)
		_, err = adapter.ClassifyTasks(ctx, "FN", "", false)
		require.Error(t, err)
	})

	t.Run("service error is wrapped", func(t *testing.T) {
		mock := &MockTaskService{}
		mock.On("ClassifyTasks", ctx, tasksdomain.ClassifyTasksInput{
			Project: "FN", Sprint: "S1", DryRun: true, Apply: false,
		}).Return(errors.New("boom"))
		adapter := &TaskServiceAdapter{service: mock}
		_, err := adapter.ClassifyTasks(ctx, "FN", "S1", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to classify tasks")
	})

	t.Run("classify ok but follow-up GetTasks fails -> short message map", func(t *testing.T) {
		mock := &MockTaskService{}
		mock.On("ClassifyTasks", ctx, tasksdomain.ClassifyTasksInput{
			Project: "FN", Sprint: "S1", DryRun: false, Apply: true,
		}).Return(nil)
		mock.On("GetTasks", ctx, "FN", "S1").Return(nil, errors.New("read failed"))
		adapter := &TaskServiceAdapter{service: mock}
		got, err := adapter.ClassifyTasks(ctx, "FN", "S1", true)
		require.NoError(t, err)
		m, ok := got.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "classified", m["status"])
		assert.Equal(t, true, m["applied"])
		// short branch carries only the basic keys, no counts.
		_, hasCounts := m["total_tasks"]
		assert.False(t, hasCounts)
	})

	t.Run("happy path counts labeled vs unlabeled and deduplicates asset labels", func(t *testing.T) {
		mock := &MockTaskService{}
		mock.On("ClassifyTasks", ctx, tasksdomain.ClassifyTasksInput{
			Project: "FN", Sprint: "S1", DryRun: true, Apply: false,
		}).Return(nil)

		labeledA := &tasksdomain.Task{Key: "T-1", Labels: []string{"cap-asset-pay"}}
		labeledB := &tasksdomain.Task{Key: "T-2", Labels: []string{"cap-asset-pay"}}
		labeledC := &tasksdomain.Task{Key: "T-3", Labels: []string{"cap-asset-search"}}
		unlabeled := &tasksdomain.Task{Key: "T-4", Labels: []string{"some-other-label"}}

		mock.On("GetTasks", ctx, "FN", "S1").
			Return([]*tasksdomain.Task{labeledA, labeledB, labeledC, unlabeled}, nil)

		adapter := &TaskServiceAdapter{service: mock}
		got, err := adapter.ClassifyTasks(ctx, "FN", "S1", false)
		require.NoError(t, err)
		m, ok := got.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, 4, m["total_tasks"])
		assert.Equal(t, 3, m["labeled_tasks"])
		assert.Equal(t, 1, m["unlabeled_tasks"])
		uniq, ok := m["unique_assets"].([]string)
		require.True(t, ok)
		assert.ElementsMatch(t, []string{"cap-asset-pay", "cap-asset-search"}, uniq)
	})
}

func TestTaskServiceAdapter_InspectTask(t *testing.T) {
	ctx := context.Background()

	t.Run("empty key rejected", func(t *testing.T) {
		adapter := &TaskServiceAdapter{service: &MockTaskService{}}
		_, err := adapter.InspectTask(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task key is required")
	})

	t.Run("service error wraps as not-found", func(t *testing.T) {
		mock := &MockTaskService{}
		mock.On("GetTaskByKey", ctx, "MISSING").Return(nil, errors.New("not found in jira"))
		adapter := &TaskServiceAdapter{service: mock}
		_, err := adapter.InspectTask(ctx, "MISSING")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task not found")
	})

	t.Run("happy path returns the task as a detail map", func(t *testing.T) {
		mock := &MockTaskService{}
		mock.On("GetTaskByKey", ctx, "FN-1").Return(newSampleTask("FN-1", "S1"), nil)
		adapter := &TaskServiceAdapter{service: mock}
		got, err := adapter.InspectTask(ctx, "FN-1")
		require.NoError(t, err)
		m, ok := got.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "FN-1", m["key"])
	})
}
