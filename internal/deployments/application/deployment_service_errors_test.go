package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain"
)

// All of these target wrap/error branches missed by the table-driven
// tests in deployment_service_test.go: NewDeployment failures, repo
// FindByTaskKey / FindByTimeRange / ListAll wrappers, the
// "asset resolver not configured" guard, the "invalid time range"
// guard, and the descending-sort comparator that only runs when
// there are 2+ items to swap.

func TestDeploymentService_RecordDeployment_NewDeploymentValidationWraps(t *testing.T) {
	t.Parallel()
	// Empty version → domain.NewDeployment returns "version is required",
	// which the service wraps as "failed to create deployment".
	svc := NewDeploymentService(new(MockDeploymentRepository), new(MockAssetResolver))
	_, err := svc.RecordDeployment(context.Background(), RecordDeploymentInput{
		TaskKeys:    []string{"TASK-1"},
		Environment: domain.EnvironmentProduction,
		Version:     "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create deployment")
}

// GetDeploymentsByTask

func TestDeploymentService_GetDeploymentsByTask_FindByTaskKeyErrorWraps(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := new(MockDeploymentRepository)
	repo.On("FindByTaskKey", ctx, "TASK-1").
		Return([]*domain.Deployment(nil), errors.New("disk"))

	svc := NewDeploymentService(repo, new(MockAssetResolver))
	_, err := svc.GetDeploymentsByTask(ctx, "TASK-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find deployments for task TASK-1")
	repo.AssertExpectations(t)
}

func TestDeploymentService_GetDeploymentsByTask_SortsDescending(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	older, err := domain.NewDeployment([]string{"TASK-1"}, domain.EnvironmentProduction, "v1")
	require.NoError(t, err)
	newer, err := domain.NewDeployment([]string{"TASK-1"}, domain.EnvironmentProduction, "v2")
	require.NoError(t, err)
	// NewDeployment stamps DeployedAt with time.Now; override so the
	// comparator has something to swap.
	older.DeployedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer.DeployedAt = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	repo := new(MockDeploymentRepository)
	repo.On("FindByTaskKey", ctx, "TASK-1").
		Return([]*domain.Deployment{older, newer}, nil)

	svc := NewDeploymentService(repo, new(MockAssetResolver))
	got, err := svc.GetDeploymentsByTask(ctx, "TASK-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.True(t, got[0].DeployedAt.After(got[1].DeployedAt),
		"expected descending order, got %v then %v", got[0].DeployedAt, got[1].DeployedAt)
}

// GetDeploymentsByAsset

func TestDeploymentService_GetDeploymentsByAsset_NoResolverFails(t *testing.T) {
	t.Parallel()
	svc := NewDeploymentService(new(MockDeploymentRepository), nil)
	_, err := svc.GetDeploymentsByAsset(context.Background(), "PaymentGateway")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "asset resolver not configured")
}

func TestDeploymentService_GetDeploymentsByAsset_ListAllErrorWraps(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := new(MockDeploymentRepository)
	resolver := new(MockAssetResolver)
	repo.On("ListAll", ctx).Return([]*domain.Deployment(nil), errors.New("disk"))

	svc := NewDeploymentService(repo, resolver)
	_, err := svc.GetDeploymentsByAsset(ctx, "PaymentGateway")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list deployments")
	repo.AssertExpectations(t)
}

// GetDeploymentsTimeline

func TestDeploymentService_GetDeploymentsTimeline_InvalidTimeRange(t *testing.T) {
	t.Parallel()
	svc := NewDeploymentService(new(MockDeploymentRepository), new(MockAssetResolver))
	// To == zero (since From=0 and To=0 → invalid range, per TimeRange.IsValid).
	_, err := svc.GetDeploymentsTimeline(context.Background(), time.Time{}, time.Time{}, nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid time range")
}

func TestDeploymentService_GetDeploymentsTimeline_FindByTimeRangeErrorWraps(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := new(MockDeploymentRepository)
	repo.On("FindByTimeRange", ctx, mock.AnythingOfType("domain.TimeRange")).
		Return([]*domain.Deployment(nil), errors.New("disk"))

	svc := NewDeploymentService(repo, new(MockAssetResolver))
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	_, err := svc.GetDeploymentsTimeline(ctx, from, to, nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find deployments")
	repo.AssertExpectations(t)
}

func TestDeploymentService_GetDeploymentsTimeline_FilteredByEnvironmentErrorWraps(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := new(MockDeploymentRepository)
	env := domain.EnvironmentProduction
	repo.On("FindByEnvironmentAndTimeRange", ctx, env, mock.AnythingOfType("domain.TimeRange")).
		Return([]*domain.Deployment(nil), errors.New("disk"))

	svc := NewDeploymentService(repo, new(MockAssetResolver))
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	_, err := svc.GetDeploymentsTimeline(ctx, from, to, &env, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find deployments")
	repo.AssertExpectations(t)
}

// GetDeploymentStatistics

func TestDeploymentService_GetDeploymentStatistics_FindByTimeRangeErrorWraps(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := new(MockDeploymentRepository)
	repo.On("FindByTimeRange", ctx, mock.AnythingOfType("domain.TimeRange")).
		Return([]*domain.Deployment(nil), errors.New("disk"))

	svc := NewDeploymentService(repo, new(MockAssetResolver))
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	_, err := svc.GetDeploymentStatistics(ctx, from, to)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find deployments")
	repo.AssertExpectations(t)
}

// ListAllDeployments

func TestDeploymentService_ListAllDeployments_ListAllErrorWraps(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := new(MockDeploymentRepository)
	repo.On("ListAll", ctx).Return([]*domain.Deployment(nil), errors.New("disk"))

	svc := NewDeploymentService(repo, new(MockAssetResolver))
	_, err := svc.ListAllDeployments(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list deployments")
	repo.AssertExpectations(t)
}

func TestDeploymentService_ListAllDeployments_SortsDescending(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	older, err := domain.NewDeployment([]string{"TASK-1"}, domain.EnvironmentProduction, "v1")
	require.NoError(t, err)
	newer, err := domain.NewDeployment([]string{"TASK-2"}, domain.EnvironmentProduction, "v2")
	require.NoError(t, err)
	older.DeployedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer.DeployedAt = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	repo := new(MockDeploymentRepository)
	repo.On("ListAll", ctx).Return([]*domain.Deployment{older, newer}, nil)

	svc := NewDeploymentService(repo, new(MockAssetResolver))
	got, err := svc.ListAllDeployments(ctx)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.True(t, got[0].DeployedAt.After(got[1].DeployedAt))
}
