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
	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain/ports"
)

// Mock Repository
type MockDeploymentRepository struct {
	mock.Mock
}

func (m *MockDeploymentRepository) Save(ctx context.Context, deployment *domain.Deployment) error {
	args := m.Called(ctx, deployment)
	return args.Error(0)
}

func (m *MockDeploymentRepository) Update(ctx context.Context, deployment *domain.Deployment) error {
	args := m.Called(ctx, deployment)
	return args.Error(0)
}

func (m *MockDeploymentRepository) FindByID(ctx context.Context, id string) (*domain.Deployment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Deployment), args.Error(1)
}

func (m *MockDeploymentRepository) FindByTaskKey(ctx context.Context, taskKey string) ([]*domain.Deployment, error) {
	args := m.Called(ctx, taskKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Deployment), args.Error(1)
}

func (m *MockDeploymentRepository) FindByTimeRange(ctx context.Context, timeRange domain.TimeRange) ([]*domain.Deployment, error) {
	args := m.Called(ctx, timeRange)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Deployment), args.Error(1)
}

func (m *MockDeploymentRepository) FindByEnvironmentAndTimeRange(ctx context.Context, environment domain.Environment, timeRange domain.TimeRange) ([]*domain.Deployment, error) {
	args := m.Called(ctx, environment, timeRange)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Deployment), args.Error(1)
}

func (m *MockDeploymentRepository) ListAll(ctx context.Context) ([]*domain.Deployment, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Deployment), args.Error(1)
}

func (m *MockDeploymentRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDeploymentRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

// Mock Asset Resolver
type MockAssetResolver struct {
	mock.Mock
}

func (m *MockAssetResolver) ResolveAssetsForTasks(ctx context.Context, taskKeys []string) ([]ports.AssetInfo, error) {
	args := m.Called(ctx, taskKeys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.AssetInfo), args.Error(1)
}

func (m *MockAssetResolver) ResolveAssetsForTask(ctx context.Context, taskKey string) ([]string, error) {
	args := m.Called(ctx, taskKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func TestNewDeploymentService(t *testing.T) {
	mockRepo := new(MockDeploymentRepository)
	mockResolver := new(MockAssetResolver)

	service := NewDeploymentService(mockRepo, mockResolver)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockResolver, service.assetResolver)
}

func TestDeploymentService_RecordDeployment(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     RecordDeploymentInput
		setupMock func(*MockDeploymentRepository, *MockAssetResolver)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "successful deployment recording",
			input: RecordDeploymentInput{
				TaskKeys:    []string{"TASK-1", "TASK-2"},
				Environment: domain.EnvironmentProduction,
				Version:     "v1.0.0",
				DeployedBy:  "CI/CD",
				CommitSHA:   "abc123",
			},
			setupMock: func(repo *MockDeploymentRepository, _ *MockAssetResolver) {
				repo.On("Save", ctx, mock.AnythingOfType("*domain.Deployment")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "repository save error",
			input: RecordDeploymentInput{
				TaskKeys:    []string{"TASK-1"},
				Environment: domain.EnvironmentProduction,
				Version:     "v1.0.0",
			},
			setupMock: func(repo *MockDeploymentRepository, _ *MockAssetResolver) {
				repo.On("Save", ctx, mock.AnythingOfType("*domain.Deployment")).Return(errors.New("save failed"))
			},
			wantErr: true,
			errMsg:  "save failed",
		},
		{
			name: "invalid input - empty task keys",
			input: RecordDeploymentInput{
				TaskKeys:    []string{},
				Environment: domain.EnvironmentProduction,
				Version:     "v1.0.0",
			},
			setupMock: func(_ *MockDeploymentRepository, _ *MockAssetResolver) {
				// No mock setup needed as validation fails before repo call
			},
			wantErr: true,
			errMsg:  "at least one task key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockDeploymentRepository)
			mockResolver := new(MockAssetResolver)
			tt.setupMock(mockRepo, mockResolver)

			service := NewDeploymentService(mockRepo, mockResolver)
			deployment, err := service.RecordDeployment(ctx, tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, deployment)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, deployment)
				assert.Equal(t, tt.input.TaskKeys, deployment.TaskKeys)
				assert.Equal(t, tt.input.Environment, deployment.Environment)
				assert.Equal(t, tt.input.Version, deployment.Version)
				assert.Equal(t, tt.input.DeployedBy, deployment.DeployedBy)
				assert.Equal(t, tt.input.CommitSHA, deployment.CommitSHA)
			}

			mockRepo.AssertExpectations(t)
			mockResolver.AssertExpectations(t)
		})
	}
}

func TestDeploymentService_UpdateDeploymentStatus(t *testing.T) {
	ctx := context.Background()
	deploymentID := "dep-123"

	tests := []struct {
		name      string
		status    domain.DeploymentStatus
		setupMock func(*MockDeploymentRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "successful status update",
			status: domain.DeploymentStatusSuccessful,
			setupMock: func(repo *MockDeploymentRepository) {
				deployment := &domain.Deployment{
					ID:          deploymentID,
					TaskKeys:    []string{"TASK-1"},
					Environment: domain.EnvironmentProduction,
					Version:     "v1.0.0",
					Status:      domain.DeploymentStatusInProgress,
				}
				repo.On("FindByID", ctx, deploymentID).Return(deployment, nil)
				repo.On("Update", ctx, mock.AnythingOfType("*domain.Deployment")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "deployment not found",
			status: domain.DeploymentStatusSuccessful,
			setupMock: func(repo *MockDeploymentRepository) {
				repo.On("FindByID", ctx, deploymentID).Return(nil, errors.New("not found"))
			},
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name:   "update error",
			status: domain.DeploymentStatusFailed,
			setupMock: func(repo *MockDeploymentRepository) {
				deployment := &domain.Deployment{
					ID:          deploymentID,
					TaskKeys:    []string{"TASK-1"},
					Environment: domain.EnvironmentProduction,
					Version:     "v1.0.0",
					Status:      domain.DeploymentStatusInProgress,
				}
				repo.On("FindByID", ctx, deploymentID).Return(deployment, nil)
				repo.On("Update", ctx, mock.AnythingOfType("*domain.Deployment")).Return(errors.New("update failed"))
			},
			wantErr: true,
			errMsg:  "update failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockDeploymentRepository)
			mockResolver := new(MockAssetResolver)
			tt.setupMock(mockRepo)

			service := NewDeploymentService(mockRepo, mockResolver)
			err := service.UpdateDeploymentStatus(ctx, deploymentID, tt.status)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDeploymentService_GetDeploymentByID(t *testing.T) {
	ctx := context.Background()
	deploymentID := "dep-123"

	deployment := &domain.Deployment{
		ID:          deploymentID,
		TaskKeys:    []string{"TASK-1"},
		Environment: domain.EnvironmentProduction,
		Version:     "v1.0.0",
		Status:      domain.DeploymentStatusSuccessful,
	}

	mockRepo := new(MockDeploymentRepository)
	mockResolver := new(MockAssetResolver)

	mockRepo.On("FindByID", ctx, deploymentID).Return(deployment, nil)

	service := NewDeploymentService(mockRepo, mockResolver)
	result, err := service.GetDeploymentByID(ctx, deploymentID)

	assert.NoError(t, err)
	assert.Equal(t, deployment, result)

	mockRepo.AssertExpectations(t)
}

func TestDeploymentService_GetDeploymentsByTask(t *testing.T) {
	ctx := context.Background()
	taskKey := "TASK-1"

	tests := []struct {
		name      string
		setupMock func(*MockDeploymentRepository, *MockAssetResolver)
		wantErr   bool
	}{
		{
			name: "get deployments by task",
			setupMock: func(repo *MockDeploymentRepository, _ *MockAssetResolver) {
				deployments := []*domain.Deployment{
					{
						ID:          "dep-1",
						TaskKeys:    []string{taskKey},
						Environment: domain.EnvironmentProduction,
						Version:     "v1.0.0",
					},
				}
				repo.On("FindByTaskKey", ctx, taskKey).Return(deployments, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockDeploymentRepository)
			mockResolver := new(MockAssetResolver)
			tt.setupMock(mockRepo, mockResolver)

			service := NewDeploymentService(mockRepo, mockResolver)
			results, err := service.GetDeploymentsByTask(ctx, taskKey)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, results)
			}

			mockRepo.AssertExpectations(t)
			mockResolver.AssertExpectations(t)
		})
	}
}

func TestDeploymentService_GetDeploymentsByAsset(t *testing.T) {
	ctx := context.Background()
	assetName := "TestAsset"

	mockRepo := new(MockDeploymentRepository)
	mockResolver := new(MockAssetResolver)

	deployments := []*domain.Deployment{
		{
			ID:          "dep-1",
			TaskKeys:    []string{"TASK-1", "TASK-2"},
			Environment: domain.EnvironmentProduction,
			Version:     "v1.0.0",
		},
		{
			ID:          "dep-2",
			TaskKeys:    []string{"TASK-3"},
			Environment: domain.EnvironmentStaging,
			Version:     "v1.0.1",
		},
	}

	mockRepo.On("ListAll", ctx).Return(deployments, nil)

	// Mock asset resolver to return the assets for the deployments
	// The service calls ResolveAssetsForTasks with all task keys from each deployment
	mockResolver.On("ResolveAssetsForTasks", ctx, []string{"TASK-1", "TASK-2"}).Return([]ports.AssetInfo{
		{Name: assetName, TaskCount: 1, TaskKeys: []string{"TASK-1"}},
		{Name: "OtherAsset", TaskCount: 1, TaskKeys: []string{"TASK-2"}},
	}, nil)

	mockResolver.On("ResolveAssetsForTasks", ctx, []string{"TASK-3"}).Return([]ports.AssetInfo{
		{Name: assetName, TaskCount: 1, TaskKeys: []string{"TASK-3"}},
	}, nil)

	service := NewDeploymentService(mockRepo, mockResolver)
	results, err := service.GetDeploymentsByAsset(ctx, assetName)

	assert.NoError(t, err)
	assert.Len(t, results, 2)

	mockRepo.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
}

func TestDeploymentService_GetDeploymentsTimeline(t *testing.T) {
	ctx := context.Background()
	from := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 9, 30, 23, 59, 59, 0, time.UTC)

	deployments := []*domain.Deployment{
		{
			ID:          "dep-1",
			TaskKeys:    []string{"TASK-1"},
			Environment: domain.EnvironmentProduction,
			Version:     "v1.0.0",
			DeployedAt:  time.Date(2025, 9, 10, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:          "dep-2",
			TaskKeys:    []string{"TASK-2"},
			Environment: domain.EnvironmentProduction,
			Version:     "v1.0.1",
			DeployedAt:  time.Date(2025, 9, 10, 14, 0, 0, 0, time.UTC),
		},
		{
			ID:          "dep-3",
			TaskKeys:    []string{"TASK-3"},
			Environment: domain.EnvironmentStaging,
			Version:     "v1.0.2",
			DeployedAt:  time.Date(2025, 9, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	mockRepo := new(MockDeploymentRepository)
	mockResolver := new(MockAssetResolver)

	timeRange := domain.TimeRange{From: from, To: to}
	mockRepo.On("FindByTimeRange", ctx, timeRange).Return(deployments, nil)

	service := NewDeploymentService(mockRepo, mockResolver)
	timeline, err := service.GetDeploymentsTimeline(ctx, from, to, nil, false)

	assert.NoError(t, err)
	assert.NotNil(t, timeline)

	mockRepo.AssertExpectations(t)
}

func TestDeploymentService_GetDeploymentStatistics(t *testing.T) {
	ctx := context.Background()
	from := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 9, 30, 23, 59, 59, 0, time.UTC)
	deployments := []*domain.Deployment{
		{
			ID:          "dep-1",
			TaskKeys:    []string{"TASK-1", "TASK-2"},
			Environment: domain.EnvironmentProduction,
			Version:     "v1.0.0",
		},
		{
			ID:          "dep-2",
			TaskKeys:    []string{"TASK-3"},
			Environment: domain.EnvironmentStaging,
			Version:     "v1.0.1",
		},
		{
			ID:          "dep-3",
			TaskKeys:    []string{"TASK-1", "TASK-4"},
			Environment: domain.EnvironmentProduction,
			Version:     "v1.0.2",
		},
	}

	mockRepo := new(MockDeploymentRepository)
	mockResolver := new(MockAssetResolver)

	// The service calls FindByTimeRange, not ListAll
	timeRange := domain.TimeRange{From: from, To: to}
	mockRepo.On("FindByTimeRange", ctx, timeRange).Return(deployments, nil)

	service := NewDeploymentService(mockRepo, mockResolver)
	stats, err := service.GetDeploymentStatistics(ctx, from, to)

	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 3, stats.TotalDeployments)
	assert.Equal(t, 2, stats.ByEnvironment["production"])
	assert.Equal(t, 1, stats.ByEnvironment["staging"])
	assert.Len(t, stats.UniqueTaskKeys, 4)

	mockRepo.AssertExpectations(t)
}

func TestDeploymentService_ListAllDeployments(t *testing.T) {
	ctx := context.Background()

	deployments := []*domain.Deployment{
		{
			ID:          "dep-1",
			TaskKeys:    []string{"TASK-1"},
			Environment: domain.EnvironmentProduction,
			Version:     "v1.0.0",
		},
		{
			ID:          "dep-2",
			TaskKeys:    []string{"TASK-2"},
			Environment: domain.EnvironmentStaging,
			Version:     "v1.0.1",
		},
	}

	mockRepo := new(MockDeploymentRepository)
	mockResolver := new(MockAssetResolver)

	mockRepo.On("ListAll", ctx).Return(deployments, nil)

	service := NewDeploymentService(mockRepo, mockResolver)
	results, err := service.ListAllDeployments(ctx)

	assert.NoError(t, err)
	assert.Equal(t, deployments, results)

	mockRepo.AssertExpectations(t)
}
