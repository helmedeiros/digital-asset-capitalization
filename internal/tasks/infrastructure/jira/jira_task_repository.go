package jira

import (
	"context"
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// TaskRepository implements the ports.JiraTaskRepository interface for Jira
type TaskRepository struct {
	client Client
}

// NewRepository creates a new Jira repository instance
func NewRepository() (*TaskRepository, error) {
	config, err := NewConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira configuration: %w", err)
	}

	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	return &TaskRepository{
		client: client,
	}, nil
}

// Save saves or updates a task
func (r *TaskRepository) Save(_ context.Context, _ *domain.Task) error {
	// TODO: Implement task saving in Jira
	return fmt.Errorf("not implemented")
}

// FindByKey finds a task by its key
func (r *TaskRepository) FindByKey(_ context.Context, _ string) (*domain.Task, error) {
	// TODO: Implement task retrieval by key in Jira
	return nil, fmt.Errorf("not implemented")
}

// FindByProjectAndSprint finds all tasks for a given project and sprint
func (r *TaskRepository) FindByProjectAndSprint(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	return r.client.FetchTasks(ctx, project, sprint)
}

// FindByProject finds all tasks for a given project
func (r *TaskRepository) FindByProject(_ context.Context, _ string) ([]*domain.Task, error) {
	// TODO: Implement task retrieval by project in Jira
	return nil, fmt.Errorf("not implemented")
}

// FindBySprint finds all tasks for a given sprint
func (r *TaskRepository) FindBySprint(_ context.Context, _ string) ([]*domain.Task, error) {
	// TODO: Implement task retrieval by sprint in Jira
	return nil, fmt.Errorf("not implemented")
}

// FindByPlatform finds all tasks from a specific platform
func (r *TaskRepository) FindByPlatform(_ context.Context, _ string) ([]*domain.Task, error) {
	// TODO: Implement task retrieval by platform in Jira
	return nil, fmt.Errorf("not implemented")
}

// FindAll returns all tasks
func (r *TaskRepository) FindAll(_ context.Context) ([]*domain.Task, error) {
	// TODO: Implement all tasks retrieval in Jira
	return nil, fmt.Errorf("not implemented")
}

// Delete deletes a task by its key
func (r *TaskRepository) Delete(_ context.Context, _ string) error {
	// TODO: Implement task deletion in Jira
	return fmt.Errorf("not implemented")
}

// DeleteByProjectAndSprint deletes all tasks for a given project and sprint
func (r *TaskRepository) DeleteByProjectAndSprint(_ context.Context, _, _ string) error {
	// TODO: Implement task deletion by project and sprint in Jira
	return fmt.Errorf("not implemented")
}

// UpdateLabels updates the labels of a task in the remote repository
func (r *TaskRepository) UpdateLabels(ctx context.Context, taskKey string, labels []string) error {
	return r.client.UpdateLabels(ctx, taskKey, labels)
}

// Ensure Repository implements ports.Repository
var _ ports.TaskRepository = (*TaskRepository)(nil)
