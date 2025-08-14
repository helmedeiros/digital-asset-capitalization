package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// Additional mock services for comprehensive testing
type MockTaskService struct {
	mock.Mock
}

func (m *MockTaskService) FetchTasks(ctx context.Context, project, sprint string) (interface{}, error) {
	args := m.Called(ctx, project, sprint)
	return args.Get(0), args.Error(1)
}

func (m *MockTaskService) ShowTasks(ctx context.Context, project, sprint string) (interface{}, error) {
	args := m.Called(ctx, project, sprint)
	return args.Get(0), args.Error(1)
}

func (m *MockTaskService) ClassifyTasks(ctx context.Context, project, sprint string, apply bool) (interface{}, error) {
	args := m.Called(ctx, project, sprint, apply)
	return args.Get(0), args.Error(1)
}

func (m *MockTaskService) InspectTask(ctx context.Context, key string) (interface{}, error) {
	args := m.Called(ctx, key)
	return args.Get(0), args.Error(1)
}

type MockSprintService struct {
	mock.Mock
}

func (m *MockSprintService) ListSprints(ctx context.Context, project, period string) (interface{}, error) {
	args := m.Called(ctx, project, period)
	return args.Get(0), args.Error(1)
}

func (m *MockSprintService) AllocateSprint(ctx context.Context, project, sprint string, bounded bool) (interface{}, error) {
	args := m.Called(ctx, project, sprint, bounded)
	return args.Get(0), args.Error(1)
}

type MockInvestmentService struct {
	mock.Mock
}

func (m *MockInvestmentService) CalculateInvestment(ctx context.Context, asset, project string, sprints []string) (interface{}, error) {
	args := m.Called(ctx, asset, project, sprints)
	return args.Get(0), args.Error(1)
}

func (m *MockInvestmentService) ListInvestments(ctx context.Context, project string) (interface{}, error) {
	args := m.Called(ctx, project)
	return args.Get(0), args.Error(1)
}

func (m *MockInvestmentService) ShowRates(ctx context.Context, project string) (interface{}, error) {
	args := m.Called(ctx, project)
	return args.Get(0), args.Error(1)
}

type MockConfigService struct {
	mock.Mock
}

func (m *MockConfigService) InitConfig(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

func (m *MockConfigService) ShowConfig(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

func (m *MockConfigService) ValidateConfig(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

func (m *MockConfigService) SyncTeam(ctx context.Context, project string) (interface{}, error) {
	args := m.Called(ctx, project)
	return args.Get(0), args.Error(1)
}

// Test Task Commands (0% coverage)
func TestCommandExecutor_ExecuteTaskCommand(t *testing.T) {
	mockTaskService := &MockTaskService{}
	executor := NewCommandExecutor(nil, mockTaskService, nil, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name        string
		command     *domain.Command
		setupMock   func()
		expectError bool
	}{
		{
			name: "tasks fetch",
			command: &domain.Command{
				ID:          "cmd-tasks-fetch",
				Raw:         "fetch tasks",
				Interpreted: "tasks fetch",
				Parameters: map[string]interface{}{
					"project": "TEST",
					"sprint":  "Sprint 1",
				},
				SessionID: "session-1",
			},
			setupMock: func() {
				mockTaskService.On("FetchTasks", ctx, "TEST", "Sprint 1").Return([]string{"TASK-1", "TASK-2"}, nil)
			},
			expectError: false,
		},
		{
			name: "tasks show",
			command: &domain.Command{
				ID:          "cmd-tasks-show",
				Raw:         "show tasks",
				Interpreted: "tasks show",
				Parameters: map[string]interface{}{
					"project": "TEST",
					"sprint":  "Sprint 1",
				},
				SessionID: "session-1",
			},
			setupMock: func() {
				mockTaskService.On("ShowTasks", ctx, "TEST", "Sprint 1").Return([]string{"TASK-1", "TASK-2"}, nil)
			},
			expectError: false,
		},
		{
			name: "tasks classify",
			command: &domain.Command{
				ID:          "cmd-tasks-classify",
				Raw:         "classify tasks",
				Interpreted: "tasks classify",
				Parameters: map[string]interface{}{
					"project": "TEST",
					"sprint":  "Sprint 1",
					"apply":   true,
				},
				SessionID: "session-1",
			},
			setupMock: func() {
				mockTaskService.On("ClassifyTasks", ctx, "TEST", "Sprint 1", true).Return("Classification complete", nil)
			},
			expectError: false,
		},
		{
			name: "tasks inspect",
			command: &domain.Command{
				ID:          "cmd-tasks-inspect",
				Raw:         "inspect task",
				Interpreted: "tasks inspect",
				Parameters: map[string]interface{}{
					"key": "TASK-123",
				},
				SessionID: "session-1",
			},
			setupMock: func() {
				mockTaskService.On("InspectTask", ctx, "TASK-123").Return(map[string]string{"key": "TASK-123"}, nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := executor.Execute(ctx, tt.command)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.Success)
			}

			mockTaskService.ExpectedCalls = nil // Clear for next test
		})
	}
}

// Test Sprint Commands (0% coverage)
func TestCommandExecutor_ExecuteSprintCommand(t *testing.T) {
	mockSprintService := &MockSprintService{}
	executor := NewCommandExecutor(nil, nil, mockSprintService, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name        string
		command     *domain.Command
		setupMock   func()
		expectError bool
	}{
		{
			name: "sprint list",
			command: &domain.Command{
				ID:          "cmd-sprint-list",
				Raw:         "list sprints",
				Interpreted: "sprint list",
				Parameters: map[string]interface{}{
					"project": "TEST",
					"period":  "Q1 2025",
				},
				SessionID: "session-1",
			},
			setupMock: func() {
				mockSprintService.On("ListSprints", ctx, "TEST", "Q1 2025").Return([]string{"Sprint 1", "Sprint 2"}, nil)
			},
			expectError: false,
		},
		{
			name: "sprint allocate",
			command: &domain.Command{
				ID:          "cmd-sprint-allocate",
				Raw:         "allocate sprint",
				Interpreted: "sprint allocate",
				Parameters: map[string]interface{}{
					"project":        "TEST",
					"sprint":         "Sprint 1",
					"sprint-bounded": true,
				},
				SessionID: "session-1",
			},
			setupMock: func() {
				mockSprintService.On("AllocateSprint", ctx, "TEST", "Sprint 1", true).Return("Sprint allocated", nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := executor.Execute(ctx, tt.command)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.Success)
			}

			mockSprintService.ExpectedCalls = nil
		})
	}
}

// Test Investment Commands (0% coverage)
func TestCommandExecutor_ExecuteInvestmentCommand(t *testing.T) {
	mockInvestmentService := &MockInvestmentService{}
	executor := NewCommandExecutor(nil, nil, nil, mockInvestmentService, nil)
	ctx := context.Background()

	tests := []struct {
		name        string
		command     *domain.Command
		setupMock   func()
		expectError bool
	}{
		{
			name: "investment calculate",
			command: &domain.Command{
				ID:          "cmd-investment-calculate",
				Raw:         "calculate investment",
				Interpreted: "investment calculate",
				Parameters: map[string]interface{}{
					"asset":   "Payment Processing",
					"project": "TEST",
					"sprints": "Sprint 1,Sprint 2",
				},
				SessionID: "session-1",
			},
			setupMock: func() {
				mockInvestmentService.On("CalculateInvestment", ctx, "Payment Processing", "TEST", []string{"Sprint 1", "Sprint 2"}).Return("Investment calculated", nil)
			},
			expectError: false,
		},
		{
			name: "investment list",
			command: &domain.Command{
				ID:          "cmd-investment-list",
				Raw:         "list investments",
				Interpreted: "investment list",
				Parameters: map[string]interface{}{
					"project": "TEST",
				},
				SessionID: "session-1",
			},
			setupMock: func() {
				mockInvestmentService.On("ListInvestments", ctx, "TEST").Return([]string{"Investment 1", "Investment 2"}, nil)
			},
			expectError: false,
		},
		{
			name: "investment show-rates",
			command: &domain.Command{
				ID:          "cmd-investment-rates",
				Raw:         "show investment rates",
				Interpreted: "investment show-rates",
				Parameters: map[string]interface{}{
					"project": "TEST",
				},
				SessionID: "session-1",
			},
			setupMock: func() {
				mockInvestmentService.On("ShowRates", ctx, "TEST").Return(map[string]float64{"rate1": 0.5, "rate2": 0.3}, nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := executor.Execute(ctx, tt.command)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.Success)
			}

			mockInvestmentService.ExpectedCalls = nil
		})
	}
}

// Test Config Commands (0% coverage)
func TestCommandExecutor_ExecuteConfigCommand(t *testing.T) {
	mockConfigService := &MockConfigService{}
	executor := NewCommandExecutor(nil, nil, nil, nil, mockConfigService)
	ctx := context.Background()

	tests := []struct {
		name        string
		command     *domain.Command
		setupMock   func()
		expectError bool
	}{
		{
			name: "config init",
			command: &domain.Command{
				ID:          "cmd-config-init",
				Raw:         "init config",
				Interpreted: "config init",
				Parameters:  map[string]interface{}{},
				SessionID:   "session-1",
			},
			setupMock: func() {
				mockConfigService.On("InitConfig", ctx).Return("Config initialized", nil)
			},
			expectError: false,
		},
		{
			name: "config show",
			command: &domain.Command{
				ID:          "cmd-config-show",
				Raw:         "show config",
				Interpreted: "config show",
				Parameters:  map[string]interface{}{},
				SessionID:   "session-1",
			},
			setupMock: func() {
				mockConfigService.On("ShowConfig", ctx).Return(map[string]string{"jira_url": "https://test.atlassian.net"}, nil)
			},
			expectError: false,
		},
		{
			name: "config validate",
			command: &domain.Command{
				ID:          "cmd-config-validate",
				Raw:         "validate config",
				Interpreted: "config validate",
				Parameters:  map[string]interface{}{},
				SessionID:   "session-1",
			},
			setupMock: func() {
				mockConfigService.On("ValidateConfig", ctx).Return("Config is valid", nil)
			},
			expectError: false,
		},
		{
			name: "config sync-team",
			command: &domain.Command{
				ID:          "cmd-config-sync-team",
				Raw:         "sync team",
				Interpreted: "config sync-team",
				Parameters: map[string]interface{}{
					"project": "TEST",
				},
				SessionID: "session-1",
			},
			setupMock: func() {
				mockConfigService.On("SyncTeam", ctx, "TEST").Return("Team synced", nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := executor.Execute(ctx, tt.command)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.Success)
			}

			mockConfigService.ExpectedCalls = nil
		})
	}
}

// Test Context Commands
func TestCommandExecutor_ExecuteContextCommand(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name        string
		command     *domain.Command
		expectError bool
	}{
		{
			name: "context show",
			command: &domain.Command{
				ID:          "cmd-context-show",
				Raw:         "show context",
				Interpreted: "context show",
				Parameters:  map[string]interface{}{},
				SessionID:   "session-1",
			},
			expectError: false,
		},
		{
			name: "context clear",
			command: &domain.Command{
				ID:          "cmd-context-clear",
				Raw:         "clear context",
				Interpreted: "context clear",
				Parameters:  map[string]interface{}{},
				SessionID:   "session-1",
			},
			expectError: false,
		},
		{
			name: "context unknown",
			command: &domain.Command{
				ID:          "cmd-context-unknown",
				Raw:         "context unknown",
				Interpreted: "context unknown",
				Parameters:  map[string]interface{}{},
				SessionID:   "session-1",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(ctx, tt.command)

			if tt.expectError {
				assert.Error(t, err)
				assert.False(t, result.Success)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.Success)
			}
		})
	}
}

// Test routeCommand edge cases
func TestCommandExecutor_RouteCommand_ErrorCases(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name        string
		command     *domain.Command
		expectError bool
	}{
		{
			name: "empty command",
			command: &domain.Command{
				ID:          "cmd-empty",
				Raw:         "",
				Interpreted: "",
				SessionID:   "session-1",
			},
			expectError: true,
		},
		{
			name: "unknown resource",
			command: &domain.Command{
				ID:          "cmd-unknown",
				Raw:         "unknown action",
				Interpreted: "unknown action",
				SessionID:   "session-1",
			},
			expectError: true,
		},
		{
			name: "assets without action",
			command: &domain.Command{
				ID:          "cmd-assets-no-action",
				Raw:         "assets",
				Interpreted: "assets",
				SessionID:   "session-1",
			},
			expectError: true,
		},
		{
			name: "tasks without action",
			command: &domain.Command{
				ID:          "cmd-tasks-no-action",
				Raw:         "tasks",
				Interpreted: "tasks",
				SessionID:   "session-1",
			},
			expectError: true,
		},
		{
			name: "sprint without action",
			command: &domain.Command{
				ID:          "cmd-sprint-no-action",
				Raw:         "sprint",
				Interpreted: "sprint",
				SessionID:   "session-1",
			},
			expectError: true,
		},
		{
			name: "investment without action",
			command: &domain.Command{
				ID:          "cmd-investment-no-action",
				Raw:         "investment",
				Interpreted: "investment",
				SessionID:   "session-1",
			},
			expectError: true,
		},
		{
			name: "config without action",
			command: &domain.Command{
				ID:          "cmd-config-no-action",
				Raw:         "config",
				Interpreted: "config",
				SessionID:   "session-1",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(ctx, tt.command)

			if tt.expectError {
				assert.Error(t, err)
				assert.False(t, result.Success)
			} else {
				assert.NoError(t, err)
				assert.True(t, result.Success)
			}
		})
	}
}

// Test validateResourceSpecific coverage
func TestCommandExecutor_ValidateResourceSpecificCoverage(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)

	tests := []struct {
		name        string
		command     *domain.Command
		expectError bool
	}{
		{
			name: "assets create without name",
			command: &domain.Command{
				ID:          "cmd-assets-create",
				Raw:         "create asset",
				Interpreted: "assets create",
				Parameters:  map[string]interface{}{},
				SessionID:   "session-1",
			},
			expectError: true,
		},
		{
			name: "assets create with name",
			command: &domain.Command{
				ID:          "cmd-assets-create",
				Raw:         "create asset",
				Interpreted: "assets create",
				Parameters: map[string]interface{}{
					"name": "Test Asset",
				},
				SessionID: "session-1",
			},
			expectError: false,
		},
		{
			name: "tasks fetch without project",
			command: &domain.Command{
				ID:          "cmd-tasks-fetch",
				Raw:         "fetch tasks",
				Interpreted: "tasks fetch",
				Parameters:  map[string]interface{}{},
				SessionID:   "session-1",
			},
			expectError: true,
		},
		{
			name: "tasks fetch with project",
			command: &domain.Command{
				ID:          "cmd-tasks-fetch",
				Raw:         "fetch tasks",
				Interpreted: "tasks fetch",
				Parameters: map[string]interface{}{
					"project": "TEST",
				},
				SessionID: "session-1",
			},
			expectError: false,
		},
		{
			name: "sprint list (no specific validation)",
			command: &domain.Command{
				ID:          "cmd-sprint-list",
				Raw:         "list sprints",
				Interpreted: "sprint list",
				Parameters:  map[string]interface{}{},
				SessionID:   "session-1",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateCommand(tt.command)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test service error scenarios
func TestCommandExecutor_ServiceErrors(t *testing.T) {
	mockTaskService := &MockTaskService{}
	executor := NewCommandExecutor(nil, mockTaskService, nil, nil, nil)
	ctx := context.Background()

	command := &domain.Command{
		ID:          "cmd-tasks-fetch",
		Raw:         "fetch tasks",
		Interpreted: "tasks fetch",
		Parameters: map[string]interface{}{
			"project": "TEST",
			"sprint":  "Sprint 1",
		},
		SessionID: "session-1",
	}

	// Mock service error
	mockTaskService.On("FetchTasks", ctx, "TEST", "Sprint 1").Return(nil, errors.New("service unavailable"))

	result, err := executor.Execute(ctx, command)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)

	mockTaskService.AssertExpectations(t)
}
