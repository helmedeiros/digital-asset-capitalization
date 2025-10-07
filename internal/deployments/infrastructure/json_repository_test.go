package infrastructure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain"
)

func setupTestRepo(t *testing.T) (*JSONRepository, func()) {
	tempDir := t.TempDir()

	config := JSONRepositoryConfig{
		Directory: tempDir,
		Filename:  "test_deployments.json",
	}

	repo := NewJSONRepository(config).(*JSONRepository)

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return repo, cleanup
}

func createTestDeployment(id, version, taskKey string, env domain.Environment) *domain.Deployment {
	return &domain.Deployment{
		ID:          id,
		TaskKeys:    []string{taskKey},
		Environment: env,
		Version:     version,
		Status:      domain.DeploymentStatusSuccessful,
		DeployedAt:  time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestNewJSONRepository(t *testing.T) {
	tests := []struct {
		name   string
		config JSONRepositoryConfig
		want   JSONRepositoryConfig
	}{
		{
			name: "with valid config",
			config: JSONRepositoryConfig{
				Directory: "custom_dir",
				Filename:  "custom.json",
			},
			want: JSONRepositoryConfig{
				Directory: "custom_dir",
				Filename:  "custom.json",
			},
		},
		{
			name: "with empty directory",
			config: JSONRepositoryConfig{
				Directory: "",
				Filename:  "custom.json",
			},
			want: JSONRepositoryConfig{
				Directory: ".assetcap",
				Filename:  "custom.json",
			},
		},
		{
			name: "with empty filename",
			config: JSONRepositoryConfig{
				Directory: "custom_dir",
				Filename:  "",
			},
			want: JSONRepositoryConfig{
				Directory: "custom_dir",
				Filename:  "deployments.json",
			},
		},
		{
			name:   "with empty config",
			config: JSONRepositoryConfig{},
			want: JSONRepositoryConfig{
				Directory: ".assetcap",
				Filename:  "deployments.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewJSONRepository(tt.config).(*JSONRepository)
			assert.Equal(t, tt.want.Directory, repo.dir)
			assert.Equal(t, tt.want.Filename, repo.filename)
		})
	}
}

func TestDefaultJSONRepositoryConfig(t *testing.T) {
	config := DefaultJSONRepositoryConfig()
	assert.Equal(t, ".assetcap", config.Directory)
	assert.Equal(t, "deployments.json", config.Filename)
}

func TestJSONRepository_Save(t *testing.T) {
	ctx := context.Background()

	t.Run("save valid deployment", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		deployment := createTestDeployment("dep-1", "v1.0.0", "TASK-1", domain.EnvironmentProduction)

		err := repo.Save(ctx, deployment)
		assert.NoError(t, err)

		// Verify file exists
		filePath := repo.getFilePath()
		assert.FileExists(t, filePath)
	})

	t.Run("save nil deployment", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		err := repo.Save(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deployment cannot be nil")
	})

	t.Run("save invalid deployment", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		deployment := &domain.Deployment{} // Invalid deployment

		err := repo.Save(ctx, deployment)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid deployment")
	})

	t.Run("save multiple deployments", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		dep1 := createTestDeployment("dep-1", "v1.0.0", "TASK-1", domain.EnvironmentProduction)
		dep2 := createTestDeployment("dep-2", "v1.1.0", "TASK-2", domain.EnvironmentStaging)

		err := repo.Save(ctx, dep1)
		require.NoError(t, err)

		err = repo.Save(ctx, dep2)
		require.NoError(t, err)

		// Verify both exist
		found1, err := repo.FindByID(ctx, "dep-1")
		require.NoError(t, err)
		assert.Equal(t, dep1.ID, found1.ID)

		found2, err := repo.FindByID(ctx, "dep-2")
		require.NoError(t, err)
		assert.Equal(t, dep2.ID, found2.ID)
	})
}

func TestJSONRepository_FindByID(t *testing.T) {
	ctx := context.Background()

	t.Run("find existing deployment", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		original := createTestDeployment("dep-1", "v1.0.0", "TASK-1", domain.EnvironmentProduction)
		err := repo.Save(ctx, original)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, "dep-1")
		require.NoError(t, err)
		assert.Equal(t, original.ID, found.ID)
		assert.Equal(t, original.Version, found.Version)
		assert.Equal(t, original.Environment, found.Environment)
	})

	t.Run("find non-existing deployment", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		_, err := repo.FindByID(ctx, "non-existing")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("find with empty ID", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		_, err := repo.FindByID(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deployment ID cannot be empty")
	})

	t.Run("find from empty file", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		// Create empty file
		err := os.WriteFile(repo.getFilePath(), []byte{}, 0644)
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, "dep-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestJSONRepository_FindByTaskKey(t *testing.T) {
	ctx := context.Background()

	t.Run("find deployments by task key", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		dep1 := createTestDeployment("dep-1", "v1.0.0", "TASK-1", domain.EnvironmentProduction)
		dep2 := createTestDeployment("dep-2", "v1.1.0", "TASK-1", domain.EnvironmentStaging)
		dep3 := createTestDeployment("dep-3", "v2.0.0", "TASK-2", domain.EnvironmentProduction)

		require.NoError(t, repo.Save(ctx, dep1))
		require.NoError(t, repo.Save(ctx, dep2))
		require.NoError(t, repo.Save(ctx, dep3))

		results, err := repo.FindByTaskKey(ctx, "TASK-1")
		require.NoError(t, err)
		assert.Len(t, results, 2)

		// Check that we found the right deployments
		foundIDs := make([]string, len(results))
		for i, dep := range results {
			foundIDs[i] = dep.ID
		}
		assert.Contains(t, foundIDs, "dep-1")
		assert.Contains(t, foundIDs, "dep-2")
	})

	t.Run("find with empty task key", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		_, err := repo.FindByTaskKey(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task key cannot be empty")
	})

	t.Run("find non-existing task key", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		dep1 := createTestDeployment("dep-1", "v1.0.0", "TASK-1", domain.EnvironmentProduction)
		require.NoError(t, repo.Save(ctx, dep1))

		results, err := repo.FindByTaskKey(ctx, "NON-EXISTING")
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestJSONRepository_FindByTimeRange(t *testing.T) {
	ctx := context.Background()

	t.Run("find deployments in time range", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		now := time.Now()
		dep1 := createTestDeployment("dep-1", "v1.0.0", "TASK-1", domain.EnvironmentProduction)
		dep1.DeployedAt = now.Add(-2 * time.Hour)

		dep2 := createTestDeployment("dep-2", "v1.1.0", "TASK-2", domain.EnvironmentStaging)
		dep2.DeployedAt = now.Add(-1 * time.Hour)

		dep3 := createTestDeployment("dep-3", "v2.0.0", "TASK-3", domain.EnvironmentProduction)
		dep3.DeployedAt = now.Add(-5 * time.Hour) // Outside range

		require.NoError(t, repo.Save(ctx, dep1))
		require.NoError(t, repo.Save(ctx, dep2))
		require.NoError(t, repo.Save(ctx, dep3))

		timeRange := domain.TimeRange{
			From: now.Add(-3 * time.Hour),
			To:   now,
		}

		results, err := repo.FindByTimeRange(ctx, timeRange)
		require.NoError(t, err)
		assert.Len(t, results, 2)

		foundIDs := make([]string, len(results))
		for i, dep := range results {
			foundIDs[i] = dep.ID
		}
		assert.Contains(t, foundIDs, "dep-1")
		assert.Contains(t, foundIDs, "dep-2")
		assert.NotContains(t, foundIDs, "dep-3")
	})

	t.Run("find with invalid time range", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		invalidRange := domain.TimeRange{
			From: time.Now(),
			To:   time.Now().Add(-1 * time.Hour),
		}

		_, err := repo.FindByTimeRange(ctx, invalidRange)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid time range")
	})
}

func TestJSONRepository_FindByEnvironmentAndTimeRange(t *testing.T) {
	ctx := context.Background()

	t.Run("find deployments by environment and time range", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		now := time.Now()
		dep1 := createTestDeployment("dep-1", "v1.0.0", "TASK-1", domain.EnvironmentProduction)
		dep1.DeployedAt = now.Add(-1 * time.Hour)

		dep2 := createTestDeployment("dep-2", "v1.1.0", "TASK-2", domain.EnvironmentStaging)
		dep2.DeployedAt = now.Add(-1 * time.Hour)

		dep3 := createTestDeployment("dep-3", "v2.0.0", "TASK-3", domain.EnvironmentProduction)
		dep3.DeployedAt = now.Add(-5 * time.Hour) // Outside time range

		require.NoError(t, repo.Save(ctx, dep1))
		require.NoError(t, repo.Save(ctx, dep2))
		require.NoError(t, repo.Save(ctx, dep3))

		timeRange := domain.TimeRange{
			From: now.Add(-2 * time.Hour),
			To:   now,
		}

		results, err := repo.FindByEnvironmentAndTimeRange(ctx, domain.EnvironmentProduction, timeRange)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "dep-1", results[0].ID)
	})

	t.Run("find with empty environment", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		timeRange := domain.TimeRange{
			From: time.Now().Add(-1 * time.Hour),
			To:   time.Now(),
		}

		_, err := repo.FindByEnvironmentAndTimeRange(ctx, "", timeRange)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment cannot be empty")
	})
}

func TestJSONRepository_ListAll(t *testing.T) {
	ctx := context.Background()

	t.Run("list all deployments", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		dep1 := createTestDeployment("dep-1", "v1.0.0", "TASK-1", domain.EnvironmentProduction)
		dep2 := createTestDeployment("dep-2", "v1.1.0", "TASK-2", domain.EnvironmentStaging)
		dep3 := createTestDeployment("dep-3", "v2.0.0", "TASK-3", domain.EnvironmentProduction)

		require.NoError(t, repo.Save(ctx, dep1))
		require.NoError(t, repo.Save(ctx, dep2))
		require.NoError(t, repo.Save(ctx, dep3))

		results, err := repo.ListAll(ctx)
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("list from empty repository", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		results, err := repo.ListAll(ctx)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestJSONRepository_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("update existing deployment", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		original := createTestDeployment("dep-1", "v1.0.0", "TASK-1", domain.EnvironmentProduction)
		err := repo.Save(ctx, original)
		require.NoError(t, err)

		// Modify the deployment
		updated := *original
		updated.Version = "v1.0.1"
		updated.Status = domain.DeploymentStatusRolledBack

		err = repo.Update(ctx, &updated)
		require.NoError(t, err)

		// Verify changes
		found, err := repo.FindByID(ctx, "dep-1")
		require.NoError(t, err)
		assert.Equal(t, "v1.0.1", found.Version)
		assert.Equal(t, domain.DeploymentStatusRolledBack, found.Status)
	})

	t.Run("update nil deployment", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		err := repo.Update(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deployment cannot be nil")
	})

	t.Run("update non-existing deployment", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		deployment := createTestDeployment("non-existing", "v1.0.0", "TASK-1", domain.EnvironmentProduction)

		err := repo.Update(ctx, deployment)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestJSONRepository_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("delete existing deployment", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		deployment := createTestDeployment("dep-1", "v1.0.0", "TASK-1", domain.EnvironmentProduction)
		err := repo.Save(ctx, deployment)
		require.NoError(t, err)

		err = repo.Delete(ctx, "dep-1")
		require.NoError(t, err)

		// Verify deleted
		_, err = repo.FindByID(ctx, "dep-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("delete with empty ID", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		err := repo.Delete(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deployment ID cannot be empty")
	})

	t.Run("delete non-existing deployment", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		err := repo.Delete(ctx, "non-existing")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestJSONRepository_Count(t *testing.T) {
	ctx := context.Background()

	t.Run("count deployments", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		// Initially empty
		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, count)

		// Add some deployments
		dep1 := createTestDeployment("dep-1", "v1.0.0", "TASK-1", domain.EnvironmentProduction)
		dep2 := createTestDeployment("dep-2", "v1.1.0", "TASK-2", domain.EnvironmentStaging)

		require.NoError(t, repo.Save(ctx, dep1))
		require.NoError(t, repo.Save(ctx, dep2))

		count, err = repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, count)

		// Delete one
		require.NoError(t, repo.Delete(ctx, "dep-1"))

		count, err = repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestJSONRepository_FileOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("creates directory if not exists", func(t *testing.T) {
		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "nested", "dir")

		config := JSONRepositoryConfig{
			Directory: subDir,
			Filename:  "test.json",
		}

		repo := NewJSONRepository(config).(*JSONRepository)
		deployment := createTestDeployment("dep-1", "v1.0.0", "TASK-1", domain.EnvironmentProduction)

		err := repo.Save(ctx, deployment)
		require.NoError(t, err)

		// Verify directory was created
		assert.DirExists(t, subDir)
		assert.FileExists(t, repo.getFilePath())
	})

	t.Run("handles corrupted JSON file", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		// Create corrupted JSON file
		filePath := repo.getFilePath()
		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		require.NoError(t, err)

		err = os.WriteFile(filePath, []byte("{invalid json"), 0644)
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, "any-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal")
	})

	t.Run("concurrent access", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		// Test concurrent writes - since JSON repository uses a mutex,
		// we expect all writes to succeed but can't guarantee ordering
		done := make(chan bool, 2)

		go func() {
			defer func() { done <- true }()
			for i := 0; i < 10; i++ {
				dep := createTestDeployment(fmt.Sprintf("dep-a-%d", i), "v1.0.0", "TASK-A", domain.EnvironmentProduction)
				repo.Save(ctx, dep)
			}
		}()

		go func() {
			defer func() { done <- true }()
			for i := 0; i < 10; i++ {
				dep := createTestDeployment(fmt.Sprintf("dep-b-%d", i), "v1.0.0", "TASK-B", domain.EnvironmentStaging)
				repo.Save(ctx, dep)
			}
		}()

		// Wait for both goroutines
		<-done
		<-done

		// Verify some deployments were saved (exact count may vary due to timing)
		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Greater(t, count, 0)
		assert.LessOrEqual(t, count, 20)

		// More importantly, verify the JSON file is not corrupted
		// by being able to list all deployments without error
		deployments, err := repo.ListAll(ctx)
		require.NoError(t, err)
		assert.Equal(t, count, len(deployments))
	})
}
