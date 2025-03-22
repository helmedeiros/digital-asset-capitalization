package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTaskRepository struct {
	findByProjectAndSprintFunc func(ctx context.Context, project, sprint string) ([]*domain.Task, error)
}

func (m *mockTaskRepository) Save(ctx context.Context, task *domain.Task) error {
	return errors.New("not implemented")
}

func (m *mockTaskRepository) FindByKey(ctx context.Context, key string) (*domain.Task, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTaskRepository) FindByProjectAndSprint(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	if m.findByProjectAndSprintFunc != nil {
		return m.findByProjectAndSprintFunc(ctx, project, sprint)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTaskRepository) FindByProject(ctx context.Context, project string) ([]*domain.Task, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTaskRepository) FindBySprint(ctx context.Context, sprint string) ([]*domain.Task, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTaskRepository) FindByPlatform(ctx context.Context, platform string) ([]*domain.Task, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTaskRepository) FindAll(ctx context.Context) ([]*domain.Task, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTaskRepository) Delete(ctx context.Context, key string) error {
	return errors.New("not implemented")
}

func (m *mockTaskRepository) DeleteByProjectAndSprint(ctx context.Context, project, sprint string) error {
	return errors.New("not implemented")
}

func TestNewFetchTasksHandler(t *testing.T) {
	repo := &mockTaskRepository{}
	handler := NewFetchTasksHandler(repo)
	assert.NotNil(t, handler, "Handler should not be nil")
	assert.Equal(t, repo, handler.taskRepository, "Repository should be set")
}

func TestFetchTasksHandler_Handle(t *testing.T) {
	ctx := context.Background()

	t.Run("empty project", func(t *testing.T) {
		handler := NewFetchTasksHandler(&mockTaskRepository{})
		err := handler.Handle(ctx, FetchTasksCommand{
			Project:  "",
			Sprint:   "Sprint 1",
			Platform: "jira",
		})
		require.Error(t, err, "Should return error")
		assert.Contains(t, err.Error(), "project is required", "Error message should indicate project is required")
	})

	t.Run("empty platform", func(t *testing.T) {
		handler := NewFetchTasksHandler(&mockTaskRepository{})
		err := handler.Handle(ctx, FetchTasksCommand{
			Project:  "TEST",
			Sprint:   "Sprint 1",
			Platform: "",
		})
		require.Error(t, err, "Should return error")
		assert.Contains(t, err.Error(), "platform is required", "Error message should indicate platform is required")
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockTaskRepository{
			findByProjectAndSprintFunc: func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
				return nil, errors.New("repository error")
			},
		}
		handler := NewFetchTasksHandler(repo)
		err := handler.Handle(ctx, FetchTasksCommand{
			Project:  "TEST",
			Sprint:   "Sprint 1",
			Platform: "jira",
		})
		require.Error(t, err, "Should return error")
		assert.Contains(t, err.Error(), "failed to fetch tasks", "Error message should indicate fetch failure")
	})

	t.Run("successful fetch", func(t *testing.T) {
		now := time.Now()
		tasks := []*domain.Task{
			{
				Key:       "TEST-1",
				Summary:   "Test Task",
				Status:    domain.TaskStatusInProgress,
				Project:   "TEST",
				Sprint:    "Sprint 1",
				Platform:  "jira",
				CreatedAt: now,
				UpdatedAt: now,
				Version:   1,
			},
		}

		repo := &mockTaskRepository{
			findByProjectAndSprintFunc: func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
				assert.Equal(t, "TEST", project, "Project should match")
				assert.Equal(t, "Sprint 1", sprint, "Sprint should match")
				return tasks, nil
			},
		}

		handler := NewFetchTasksHandler(repo)
		err := handler.Handle(ctx, FetchTasksCommand{
			Project:  "TEST",
			Sprint:   "Sprint 1",
			Platform: "jira",
		})
		require.NoError(t, err, "Should not return error")
	})
}
