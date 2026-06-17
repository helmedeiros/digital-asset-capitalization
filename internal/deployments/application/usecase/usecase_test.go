package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain/ports"
)

// stubRepo and stubResolver are minimal hand-rolled implementations of
// the deployment repository + asset resolver ports. They let the use
// cases run through a real *application.DeploymentService without
// pulling in testify/mock boilerplate from the parent package.
type stubRepo struct {
	saved             []*domain.Deployment
	saveErr           error
	listAllResult     []*domain.Deployment
	listAllErr        error
	findByTaskResult  []*domain.Deployment
	findByTaskErr     error
	findByRangeResult []*domain.Deployment
	findByRangeErr    error
	findByEnvResult   []*domain.Deployment
	findByEnvErr      error
}

func (r *stubRepo) Save(_ context.Context, d *domain.Deployment) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = append(r.saved, d)
	return nil
}
func (r *stubRepo) Update(context.Context, *domain.Deployment) error { return nil }
func (r *stubRepo) Delete(context.Context, string) error             { return nil }
func (r *stubRepo) Count(context.Context) (int, error)               { return len(r.saved), nil }
func (r *stubRepo) FindByID(context.Context, string) (*domain.Deployment, error) {
	return nil, nil
}
func (r *stubRepo) FindByTaskKey(_ context.Context, _ string) ([]*domain.Deployment, error) {
	return r.findByTaskResult, r.findByTaskErr
}
func (r *stubRepo) FindByTimeRange(_ context.Context, _ domain.TimeRange) ([]*domain.Deployment, error) {
	return r.findByRangeResult, r.findByRangeErr
}
func (r *stubRepo) FindByEnvironmentAndTimeRange(_ context.Context, _ domain.Environment, _ domain.TimeRange) ([]*domain.Deployment, error) {
	return r.findByEnvResult, r.findByEnvErr
}
func (r *stubRepo) ListAll(context.Context) ([]*domain.Deployment, error) {
	return r.listAllResult, r.listAllErr
}

type stubResolver struct {
	infoForTasks []ports.AssetInfo
	infoErr      error
}

func (r *stubResolver) ResolveAssetsForTasks(_ context.Context, _ []string) ([]ports.AssetInfo, error) {
	return r.infoForTasks, r.infoErr
}
func (r *stubResolver) ResolveAssetsForTask(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func newService(t *testing.T, repo ports.DeploymentRepository, resolver ports.AssetResolver) *application.DeploymentService {
	t.Helper()
	svc := application.NewDeploymentService(repo, resolver)
	require.NotNil(t, svc)
	return svc
}

// ----- RecordDeploymentUseCase -----

func TestRecordDeploymentUseCase_Execute(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("nil service is reported as not configured", func(t *testing.T) {
		uc := &RecordDeploymentUseCase{}
		got, err := uc.Execute(ctx, application.RecordDeploymentInput{})
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "deployment service not configured")
	})

	t.Run("missing task keys", func(t *testing.T) {
		uc := NewRecordDeploymentUseCase(newService(t, &stubRepo{}, &stubResolver{}))
		_, err := uc.Execute(ctx, application.RecordDeploymentInput{
			Environment: domain.EnvironmentProduction,
			Version:     "v1",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task key")
	})

	t.Run("missing environment", func(t *testing.T) {
		uc := NewRecordDeploymentUseCase(newService(t, &stubRepo{}, &stubResolver{}))
		_, err := uc.Execute(ctx, application.RecordDeploymentInput{
			TaskKeys: []string{"PROJ-1"},
			Version:  "v1",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "environment")
	})

	t.Run("missing version", func(t *testing.T) {
		uc := NewRecordDeploymentUseCase(newService(t, &stubRepo{}, &stubResolver{}))
		_, err := uc.Execute(ctx, application.RecordDeploymentInput{
			TaskKeys:    []string{"PROJ-1"},
			Environment: domain.EnvironmentProduction,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "version")
	})

	t.Run("happy path returns the saved deployment", func(t *testing.T) {
		repo := &stubRepo{}
		uc := NewRecordDeploymentUseCase(newService(t, repo, &stubResolver{}))
		got, err := uc.Execute(ctx, application.RecordDeploymentInput{
			TaskKeys:    []string{"PROJ-1", "PROJ-2"},
			Environment: domain.EnvironmentProduction,
			Version:     "v1.2.3",
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "v1.2.3", got.Version)
		assert.Len(t, repo.saved, 1, "service should have written exactly one deployment")
	})

	t.Run("service save error is wrapped", func(t *testing.T) {
		repo := &stubRepo{saveErr: errors.New("disk full")}
		uc := NewRecordDeploymentUseCase(newService(t, repo, &stubResolver{}))
		_, err := uc.Execute(ctx, application.RecordDeploymentInput{
			TaskKeys:    []string{"PROJ-1"},
			Environment: domain.EnvironmentProduction,
			Version:     "v1",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to record deployment")
	})
}

// ----- GetDeploymentHistoryUseCase -----

func mustNewDeployment(t *testing.T, key string, env domain.Environment) *domain.Deployment {
	t.Helper()
	d, err := domain.NewDeployment([]string{key}, env, "v1.0.0")
	require.NoError(t, err)
	return d
}

func TestGetDeploymentHistoryUseCase_Execute(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("nil service is reported as not configured", func(t *testing.T) {
		uc := &GetDeploymentHistoryUseCase{}
		_, err := uc.Execute(ctx, HistoryFilter{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "deployment service not configured")
	})

	t.Run("filter by task key returns wrapped deployments respecting limit", func(t *testing.T) {
		deps := []*domain.Deployment{
			mustNewDeployment(t, "PROJ-1", domain.EnvironmentProduction),
			mustNewDeployment(t, "PROJ-1", domain.EnvironmentProduction),
			mustNewDeployment(t, "PROJ-1", domain.EnvironmentProduction),
		}
		repo := &stubRepo{findByTaskResult: deps}
		uc := NewGetDeploymentHistoryUseCase(newService(t, repo, &stubResolver{}))

		got, err := uc.Execute(ctx, HistoryFilter{TaskKey: "PROJ-1", Limit: 2})
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Same(t, deps[0], got[0].Deployment)
	})

	t.Run("filter by task key surfaces repository errors", func(t *testing.T) {
		repo := &stubRepo{findByTaskErr: errors.New("boom")}
		uc := NewGetDeploymentHistoryUseCase(newService(t, repo, &stubResolver{}))
		_, err := uc.Execute(ctx, HistoryFilter{TaskKey: "PROJ-1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get deployments for task")
	})

	t.Run("no filter walks ListAll and respects environment + limit", func(t *testing.T) {
		// ListAllDeployments sorts by DeployedAt descending, so pin the
		// DeployedAt values explicitly to make the post-filter order
		// deterministic. prod is the most recent, then anotherProd,
		// then staging.
		baseT := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		prod := mustNewDeployment(t, "PROJ-1", domain.EnvironmentProduction)
		prod.DeployedAt = baseT.Add(3 * time.Hour)
		staging := mustNewDeployment(t, "PROJ-2", domain.EnvironmentStaging)
		staging.DeployedAt = baseT.Add(2 * time.Hour)
		anotherProd := mustNewDeployment(t, "PROJ-3", domain.EnvironmentProduction)
		anotherProd.DeployedAt = baseT.Add(1 * time.Hour)
		repo := &stubRepo{listAllResult: []*domain.Deployment{prod, staging, anotherProd}}
		uc := NewGetDeploymentHistoryUseCase(newService(t, repo, &stubResolver{}))

		env := domain.EnvironmentProduction
		got, err := uc.Execute(ctx, HistoryFilter{Environment: &env, Limit: 1})
		require.NoError(t, err)
		require.Len(t, got, 1, "limit trims after environment filter")
		assert.Same(t, prod, got[0].Deployment, "after env filter the most-recent prod deployment should win")
	})

	t.Run("no filter wraps ListAll errors", func(t *testing.T) {
		repo := &stubRepo{listAllErr: errors.New("repo down")}
		uc := NewGetDeploymentHistoryUseCase(newService(t, repo, &stubResolver{}))
		_, err := uc.Execute(ctx, HistoryFilter{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list all deployments")
	})
}

// ----- GetDeploymentsTimelineUseCase -----

func TestGetDeploymentsTimelineUseCase_Execute(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	t.Run("nil service is reported as not configured", func(t *testing.T) {
		uc := &GetDeploymentsTimelineUseCase{}
		_, err := uc.Execute(ctx, TimelineInput{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "deployment service not configured")
	})

	t.Run("missing dates return validation error", func(t *testing.T) {
		uc := NewGetDeploymentsTimelineUseCase(newService(t, &stubRepo{}, &stubResolver{}))
		_, err := uc.Execute(ctx, TimelineInput{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "from and to dates")
	})

	t.Run("inverted dates return validation error", func(t *testing.T) {
		uc := NewGetDeploymentsTimelineUseCase(newService(t, &stubRepo{}, &stubResolver{}))
		_, err := uc.Execute(ctx, TimelineInput{From: to, To: from})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "to date must be after from date")
	})

	t.Run("happy path returns timeline + statistics + period string", func(t *testing.T) {
		dep := mustNewDeployment(t, "PROJ-1", domain.EnvironmentProduction)
		dep.DeployedAt = from.AddDate(0, 0, 5)
		repo := &stubRepo{
			findByRangeResult: []*domain.Deployment{dep},
		}
		uc := NewGetDeploymentsTimelineUseCase(newService(t, repo, &stubResolver{}))

		got, err := uc.Execute(ctx, TimelineInput{From: from, To: to})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.NotEmpty(t, got.Timeline)
		assert.NotNil(t, got.Statistics)
		assert.Equal(t, "2026-01-01 to 2026-03-01", got.Period)
	})

	t.Run("environment filter delegates to env-scoped repository call", func(t *testing.T) {
		dep := mustNewDeployment(t, "PROJ-1", domain.EnvironmentStaging)
		dep.DeployedAt = from.AddDate(0, 0, 7)
		repo := &stubRepo{
			findByEnvResult: []*domain.Deployment{dep},
		}
		uc := NewGetDeploymentsTimelineUseCase(newService(t, repo, &stubResolver{}))

		env := domain.EnvironmentStaging
		got, err := uc.Execute(ctx, TimelineInput{From: from, To: to, Environment: &env})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.NotEmpty(t, got.Timeline)
	})

	t.Run("timeline repository error is wrapped", func(t *testing.T) {
		repo := &stubRepo{findByRangeErr: errors.New("repo down")}
		uc := NewGetDeploymentsTimelineUseCase(newService(t, repo, &stubResolver{}))
		_, err := uc.Execute(ctx, TimelineInput{From: from, To: to})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get deployments timeline")
	})
}
