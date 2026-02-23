package main

import (
	"context"

	"github.com/stretchr/testify/mock"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	assetsdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	sprintusecase "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application/usecase"
	sprintdomain "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	tasksdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	taskports "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// MockAssetService is a mock implementation of AssetService
type MockAssetService struct {
	mock.Mock
}

func (m *MockAssetService) CreateAsset(name, description string) error {
	args := m.Called(name, description)
	return args.Error(0)
}

func (m *MockAssetService) ListAssets() ([]*assetsdomain.Asset, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*assetsdomain.Asset), args.Error(1)
}

func (m *MockAssetService) GetAsset(name string) (*assetsdomain.Asset, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*assetsdomain.Asset), args.Error(1)
}

func (m *MockAssetService) UpdateAsset(name, description, why, benefits, how, metrics string) error {
	args := m.Called(name, description, why, benefits, how, metrics)
	return args.Error(0)
}

func (m *MockAssetService) UpdateDocumentation(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockAssetService) IncrementTaskCount(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockAssetService) DecrementTaskCount(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockAssetService) GenerateKeywords(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockAssetService) EnrichAsset(name, field string) error {
	args := m.Called(name, field)
	return args.Error(0)
}

func (m *MockAssetService) DeleteAsset(name string, deleteConfluencePage bool) error {
	args := m.Called(name, deleteConfluencePage)
	return args.Error(0)
}

func (m *MockAssetService) SyncFromConfluence(space, label string, debug bool) (*assetsdomain.SyncResult, error) {
	args := m.Called(space, label, debug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*assetsdomain.SyncResult), args.Error(1)
}

func (m *MockAssetService) AssignTeam(assetName, owningTeam string, contributingTeams []string) error {
	args := m.Called(assetName, owningTeam, contributingTeams)
	return args.Error(0)
}

func (m *MockAssetService) GetAssetTeams() ([]assetsapp.AssetTeamInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]assetsapp.AssetTeamInfo), args.Error(1)
}

func (m *MockAssetService) GetAssetTeamInfo(assetName string) (*assetsapp.AssetTeamInfo, error) {
	args := m.Called(assetName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*assetsapp.AssetTeamInfo), args.Error(1)
}

func (m *MockAssetService) AddContributingTeam(assetName, teamName string) error {
	args := m.Called(assetName, teamName)
	return args.Error(0)
}

func (m *MockAssetService) RemoveContributingTeam(assetName, teamName string) error {
	args := m.Called(assetName, teamName)
	return args.Error(0)
}

func (m *MockAssetService) PublishToConfluence(ctx context.Context, assetName, spaceKey, parentPageID string, dryRun, debug bool) (*assetsapp.PublishToConfluenceResult, error) {
	args := m.Called(ctx, assetName, spaceKey, parentPageID, dryRun, debug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*assetsapp.PublishToConfluenceResult), args.Error(1)
}

func (m *MockAssetService) UpdateConfluencePage(ctx context.Context, assetName string, dryRun, debug bool) (*assetsapp.PublishToConfluenceResult, error) {
	args := m.Called(ctx, assetName, dryRun, debug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*assetsapp.PublishToConfluenceResult), args.Error(1)
}

// MockTaskService is a mock implementation of TaskService
type MockTaskService struct {
	mock.Mock
}

func (m *MockTaskService) FetchTasks(ctx context.Context, project, sprint, platform string) error {
	args := m.Called(ctx, project, sprint, platform)
	return args.Error(0)
}

func (m *MockTaskService) FetchTaskByKey(ctx context.Context, key, platform string) error {
	args := m.Called(ctx, key, platform)
	return args.Error(0)
}

func (m *MockTaskService) GetTasks(ctx context.Context, project, sprint string) ([]*tasksdomain.Task, error) {
	args := m.Called(ctx, project, sprint)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*tasksdomain.Task), args.Error(1)
}

func (m *MockTaskService) GetTasksByAsset(ctx context.Context, asset string) ([]*tasksdomain.Task, error) {
	args := m.Called(ctx, asset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*tasksdomain.Task), args.Error(1)
}

func (m *MockTaskService) GetTaskByKey(ctx context.Context, key string) (*tasksdomain.Task, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tasksdomain.Task), args.Error(1)
}

func (m *MockTaskService) ClassifyTasks(ctx context.Context, input tasksdomain.ClassifyTasksInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockTaskService) GetLocalRepository() taskports.TaskRepository {
	args := m.Called()
	return args.Get(0).(taskports.TaskRepository)
}

// MockSprintService is a mock implementation of SprintService
type MockSprintService struct {
	mock.Mock
}

func (m *MockSprintService) ProcessJiraIssues(project, sprint, override string) (string, error) {
	args := m.Called(project, sprint, override)
	return args.String(0), args.Error(1)
}

func (m *MockSprintService) ProcessSprint(project string, sprint *sprintdomain.Sprint) error {
	args := m.Called(project, sprint)
	return args.Error(0)
}

func (m *MockSprintService) ProcessTeamIssues(team *sprintdomain.Team) error {
	args := m.Called(team)
	return args.Error(0)
}

func (m *MockSprintService) ListSprints(project, period string) (*sprintusecase.ListSprintsResult, error) {
	args := m.Called(project, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sprintusecase.ListSprintsResult), args.Error(1)
}

func (m *MockSprintService) ProcessJiraIssuesWithStrategy(project, sprint, override string, useSprintBounded bool) (string, error) {
	args := m.Called(project, sprint, override, useSprintBounded)
	return args.String(0), args.Error(1)
}

// MockConfigService is defined in config_commands_test.go to avoid duplication
