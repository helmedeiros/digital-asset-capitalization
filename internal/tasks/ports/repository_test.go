package ports

import (
	"context"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepositoryInterface ensures that all repository implementations satisfy the interface
func TestRepositoryInterface(t *testing.T) {
	// This test ensures that the interface is properly defined
	// and that all required methods are present
	var _ Repository = (*mockRepository)(nil)
}

// mockRepository is a mock implementation of the Repository interface
// used for testing purposes
type mockRepository struct{}

func (m *mockRepository) Save(ctx context.Context, task *domain.Task) error {
	return nil
}

func (m *mockRepository) FindByKey(ctx context.Context, key string) (*domain.Task, error) {
	return nil, nil
}

func (m *mockRepository) FindByProjectAndSprint(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	return nil, nil
}

func (m *mockRepository) FindByProject(ctx context.Context, project string) ([]*domain.Task, error) {
	return nil, nil
}

func (m *mockRepository) FindBySprint(ctx context.Context, sprint string) ([]*domain.Task, error) {
	return nil, nil
}

func (m *mockRepository) FindByPlatform(ctx context.Context, platform string) ([]*domain.Task, error) {
	return nil, nil
}

func (m *mockRepository) FindAll(ctx context.Context) ([]*domain.Task, error) {
	return nil, nil
}

func (m *mockRepository) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockRepository) DeleteByProjectAndSprint(ctx context.Context, project, sprint string) error {
	return nil
}

// TestRepositoryInterfaceCompliance verifies that the interface methods
// have the correct signatures and return types
func TestRepositoryInterfaceCompliance(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepository{}

	// Test Save
	err := repo.Save(ctx, &domain.Task{})
	require.NoError(t, err)

	// Test FindByKey
	task, err := repo.FindByKey(ctx, "test-key")
	require.NoError(t, err)
	assert.Nil(t, task)

	// Test FindByProjectAndSprint
	tasks, err := repo.FindByProjectAndSprint(ctx, "test-project", "test-sprint")
	require.NoError(t, err)
	assert.Nil(t, tasks)

	// Test FindByProject
	tasks, err = repo.FindByProject(ctx, "test-project")
	require.NoError(t, err)
	assert.Nil(t, tasks)

	// Test FindBySprint
	tasks, err = repo.FindBySprint(ctx, "test-sprint")
	require.NoError(t, err)
	assert.Nil(t, tasks)

	// Test FindByPlatform
	tasks, err = repo.FindByPlatform(ctx, "test-platform")
	require.NoError(t, err)
	assert.Nil(t, tasks)

	// Test FindAll
	tasks, err = repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Nil(t, tasks)

	// Test Delete
	err = repo.Delete(ctx, "test-key")
	require.NoError(t, err)

	// Test DeleteByProjectAndSprint
	err = repo.DeleteByProjectAndSprint(ctx, "test-project", "test-sprint")
	require.NoError(t, err)
}
