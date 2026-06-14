package infrastructure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain"
)

// captureRepo records every Save into an in-memory slice and (optionally)
// fails after a configured number of successful saves. Hand-rolled so
// the table tests can inspect every generated deployment without
// dragging testify/mock plumbing through what is otherwise a small
// pure-Go data-generation surface.
type captureRepo struct {
	mu          sync.Mutex
	saved       []*domain.Deployment
	failAfter   int
	failWithErr error
}

func (r *captureRepo) Save(_ context.Context, d *domain.Deployment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failAfter > 0 && len(r.saved) >= r.failAfter {
		return r.failWithErr
	}
	r.saved = append(r.saved, d)
	return nil
}

// The MockDataProvider only ever calls Save on the repository, but the
// port interface is wider. Stub the rest so captureRepo satisfies
// ports.DeploymentRepository.
func (r *captureRepo) FindByID(context.Context, string) (*domain.Deployment, error) { return nil, nil }
func (r *captureRepo) FindByTaskKey(context.Context, string) ([]*domain.Deployment, error) {
	return nil, nil
}
func (r *captureRepo) FindByTimeRange(context.Context, domain.TimeRange) ([]*domain.Deployment, error) {
	return nil, nil
}
func (r *captureRepo) FindByEnvironmentAndTimeRange(context.Context, domain.Environment, domain.TimeRange) ([]*domain.Deployment, error) {
	return nil, nil
}
func (r *captureRepo) ListAll(context.Context) ([]*domain.Deployment, error) { return nil, nil }
func (r *captureRepo) Update(context.Context, *domain.Deployment) error      { return nil }
func (r *captureRepo) Delete(context.Context, string) error                  { return nil }
func (r *captureRepo) Count(context.Context) (int, error)                    { return len(r.saved), nil }

func TestNewMockDataProvider(t *testing.T) {
	repo := &captureRepo{}
	p := NewMockDataProvider(repo)
	require.NotNil(t, p)
	assert.Same(t, repo, p.repo, "provider should hold the repo it was constructed with")
	require.NotNil(t, p.rand)
}

func TestDefaultMockDataConfig(t *testing.T) {
	cfg := DefaultMockDataConfig()
	assert.Equal(t, 10, cfg.Count)
	assert.Equal(t, []string{"FN", "AD", "MZN"}, cfg.Projects)
	assert.True(t, cfg.StartDate.Before(cfg.EndDate))
	// StartDate is ~1 month before EndDate.
	delta := cfg.EndDate.Sub(cfg.StartDate)
	assert.Greater(t, delta, 25*24*time.Hour, "start should be ~1 month before end")
	assert.Less(t, delta, 35*24*time.Hour, "start should be ~1 month before end")
}

func TestGenerateMockDeployments_HappyPath(t *testing.T) {
	repo := &captureRepo{}
	provider := NewMockDataProvider(repo)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	cfg := MockDataConfig{
		Count:     12,
		StartDate: start,
		EndDate:   end,
		Projects:  []string{"FN", "AD"},
	}

	require.NoError(t, provider.GenerateMockDeployments(context.Background(), cfg))
	require.Len(t, repo.saved, 12)

	versionRE := regexp.MustCompile(`^v\d+\.\d+\.\d+(-rc\d+)?$`)
	shaRE := regexp.MustCompile(`^[a-f0-9]{7}$`)
	envSet := map[domain.Environment]bool{
		domain.EnvironmentProduction:  true,
		domain.EnvironmentStaging:     true,
		domain.EnvironmentQA:          true,
		domain.EnvironmentDevelopment: true,
	}

	for i, d := range repo.saved {
		assert.NotEmpty(t, d.ID, "deployment %d", i)
		assert.NotEmpty(t, d.TaskKeys, "deployment %d should carry at least one task key", i)
		assert.LessOrEqual(t, len(d.TaskKeys), 5, "deployment %d task count capped at 5", i)
		for _, key := range d.TaskKeys {
			assert.Regexp(t, `^(FN|AD)-\d+$`, key, "task key must be project-prefixed and numeric")
		}
		assert.Regexp(t, versionRE, d.Version, "version must look semver-ish")
		assert.Regexp(t, shaRE, d.CommitSHA, "commit SHA must be 7 lowercase hex chars")
		assert.True(t, envSet[d.Environment], "environment %q must be one of the supported set", d.Environment)
		assert.False(t, d.DeployedAt.Before(start), "deployedAt %s should be >= start", d.DeployedAt)
		assert.False(t, d.DeployedAt.After(end), "deployedAt %s should be <= end", d.DeployedAt)
		assert.NotEmpty(t, d.DeployedBy)

		if d.Status == domain.DeploymentStatusRolledBack {
			require.NotNil(t, d.Metadata, "rolled-back deployments include rollback metadata")
			require.NotNil(t, d.Metadata.RollbackFrom)
			assert.True(t, strings.HasPrefix(*d.Metadata.RollbackFrom, "v"))
		}

		if d.DeployedBy == "github-actions" {
			require.NotNil(t, d.Metadata)
			assert.Contains(t, d.Metadata.PipelineURL, "github.com/org/repo/actions/runs/")
		}
	}
}

func TestGenerateMockDeployments_ZeroConfigUsesDefaults(t *testing.T) {
	repo := &captureRepo{}
	provider := NewMockDataProvider(repo)

	// Empty config triggers every fallback branch: count, start, end,
	// projects. Use Count=0 explicitly to hit the count fallback too.
	require.NoError(t, provider.GenerateMockDeployments(context.Background(), MockDataConfig{}))

	assert.Len(t, repo.saved, 10, "zero count should fall back to default 10")
	// Project fallback is {"FN","AD","MZN"} -- every generated task key
	// must be prefixed with one of those.
	projectRE := regexp.MustCompile(`^(FN|AD|MZN)-\d+$`)
	for _, d := range repo.saved {
		require.NotEmpty(t, d.TaskKeys)
		for _, key := range d.TaskKeys {
			assert.Regexp(t, projectRE, key)
		}
	}
}

func TestGenerateMockDeployments_PropagatesRepoError(t *testing.T) {
	wantErr := errors.New("disk full")
	repo := &captureRepo{failAfter: 3, failWithErr: wantErr}
	provider := NewMockDataProvider(repo)

	err := provider.GenerateMockDeployments(context.Background(), MockDataConfig{
		Count:     10,
		StartDate: time.Now().AddDate(0, 0, -7),
		EndDate:   time.Now(),
		Projects:  []string{"FN"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save mock deployment")
	assert.ErrorIs(t, err, wantErr)
	assert.Len(t, repo.saved, 3, "should have stopped at the first save failure")
}

func TestGenerateSampleMockFile(t *testing.T) {
	// GenerateSampleMockFile hard-codes the JSONRepository's directory
	// to ".", so chdir into a temp dir, run, and verify the file
	// shows up at that filename inside it. The defer restores the
	// original working directory so we don't leak state across tests.
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	require.NoError(t, GenerateSampleMockFile(context.Background(), "sample.json"))

	info, err := os.Stat(filepath.Join(dir, "sample.json"))
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "sample file should be non-empty")
}
