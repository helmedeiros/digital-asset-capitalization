package application

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain/ports"
)

// Mock implementations for testing
type MockAIInterpreter struct {
	mock.Mock
}

func (m *MockAIInterpreter) Interpret(ctx context.Context, input string, sessionContext *domain.Context) (*domain.Command, error) {
	args := m.Called(ctx, input, sessionContext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Command), args.Error(1)
}

func (m *MockAIInterpreter) GetClarification(ctx context.Context, reason string, options []string) (string, error) {
	args := m.Called(ctx, reason, options)
	return args.String(0), args.Error(1)
}

func (m *MockAIInterpreter) AnalyzeIntent(ctx context.Context, input string) (*domain.CommandIntent, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CommandIntent), args.Error(1)
}

type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) Execute(ctx context.Context, command *domain.Command) (*domain.CommandResult, error) {
	args := m.Called(ctx, command)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CommandResult), args.Error(1)
}

func (m *MockCommandExecutor) ValidateCommand(command *domain.Command) error {
	args := m.Called(command)
	return args.Error(0)
}

func (m *MockCommandExecutor) GetAvailableCommands() []ports.CommandInfo {
	args := m.Called()
	return args.Get(0).([]ports.CommandInfo)
}

type MockContextStore struct {
	mock.Mock
}

func (m *MockContextStore) Save(ctx context.Context, sessionContext *domain.Context) error {
	args := m.Called(ctx, sessionContext)
	return args.Error(0)
}

func (m *MockContextStore) Load(ctx context.Context, sessionID string) (*domain.Context, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Context), args.Error(1)
}

func (m *MockContextStore) Update(ctx context.Context, sessionID string, updateFn func(*domain.Context) error) error {
	args := m.Called(ctx, sessionID, updateFn)
	return args.Error(0)
}

func (m *MockContextStore) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockContextStore) List(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockContextStore) CleanupExpired(ctx context.Context, timeout int) error {
	args := m.Called(ctx, timeout)
	return args.Error(0)
}

func TestNewConsoleService(t *testing.T) {
	service := NewConsoleService(nil, nil, nil)

	require.NotNil(t, service)
	assert.Equal(t, 30*time.Minute, service.sessionTimeout)
	assert.Equal(t, 5*time.Minute, service.cleanupInterval)
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()

	assert.NotEmpty(t, id1)
	assert.Contains(t, id1, "session-")
	assert.Len(t, id1, len("session-")+19) // session- + nanosecond timestamp
}

func TestConsoleService_StartSession(t *testing.T) {
	mockStore := &MockContextStore{}
	service := NewConsoleService(nil, nil, mockStore)
	ctx := context.Background()

	// Mock successful save
	mockStore.On("Save", ctx, mock.AnythingOfType("*domain.Context")).Return(nil)

	sessionContext, err := service.StartSession(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, sessionContext)
	assert.NotEmpty(t, sessionContext.SessionID)
	mockStore.AssertExpectations(t)
}

func TestProcessResult_Structure(t *testing.T) {
	result := &ProcessResult{
		Success:               true,
		Command:               nil,
		Result:                nil,
		Output:                "test output",
		Error:                 "",
		Help:                  "test help",
		RequiresClarification: false,
		ClarificationPrompt:   "",
		Options:               []string{"option1", "option2"},
		Duration:              100 * time.Millisecond,
	}

	assert.True(t, result.Success)
	assert.Equal(t, "test output", result.Output)
	assert.Equal(t, "test help", result.Help)
	assert.False(t, result.RequiresClarification)
	assert.Len(t, result.Options, 2)
	assert.Equal(t, 100*time.Millisecond, result.Duration)
}

func TestConsoleService_EndSession(t *testing.T) {
	mockStore := &MockContextStore{}
	service := NewConsoleService(nil, nil, mockStore)
	ctx := context.Background()

	// Mock successful delete
	mockStore.On("Delete", ctx, "test-session").Return(nil)

	err := service.EndSession(ctx, "test-session")

	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestConsoleService_GetSessionContext(t *testing.T) {
	mockStore := &MockContextStore{}
	service := NewConsoleService(nil, nil, mockStore)
	ctx := context.Background()

	expectedContext := domain.NewContext("test-session")
	mockStore.On("Load", ctx, "test-session").Return(expectedContext, nil)

	sessionContext, err := service.GetSessionContext(ctx, "test-session")

	assert.NoError(t, err)
	assert.Equal(t, expectedContext, sessionContext)
	mockStore.AssertExpectations(t)
}

func TestConsoleService_ProcessInput_SessionNotFound(t *testing.T) {
	mockStore := &MockContextStore{}
	service := NewConsoleService(nil, nil, mockStore)
	ctx := context.Background()

	// Mock context not found
	mockStore.On("Load", ctx, "invalid-session").Return((*domain.Context)(nil), ports.ErrContextNotFound)

	result, err := service.ProcessInput(ctx, "invalid-session", "test input")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "session not found")
	mockStore.AssertExpectations(t)
}

func TestConsoleService_ProcessInput_LoadError(t *testing.T) {
	mockStore := &MockContextStore{}
	service := NewConsoleService(nil, nil, mockStore)
	ctx := context.Background()

	// Mock generic load error
	mockStore.On("Load", ctx, "test-session").Return((*domain.Context)(nil), assert.AnError)

	result, err := service.ProcessInput(ctx, "test-session", "test input")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to load session context")
	mockStore.AssertExpectations(t)
}

func TestConsoleService_StartSession_SaveError(t *testing.T) {
	mockStore := &MockContextStore{}
	service := NewConsoleService(nil, nil, mockStore)
	ctx := context.Background()

	// Mock save error
	mockStore.On("Save", ctx, mock.AnythingOfType("*domain.Context")).Return(assert.AnError)

	sessionContext, err := service.StartSession(ctx)

	assert.Error(t, err)
	assert.Nil(t, sessionContext)
	assert.Contains(t, err.Error(), "failed to save session context")
	mockStore.AssertExpectations(t)
}

func TestConsoleService_ProcessInput_ExpiredSession(t *testing.T) {
	mockStore := &MockContextStore{}
	service := NewConsoleService(nil, nil, mockStore)
	ctx := context.Background()

	// Create an expired context
	expiredContext := domain.NewContext("expired-session")
	expiredContext.LastActivity = time.Now().Add(-2 * time.Hour) // 2 hours ago

	mockStore.On("Load", ctx, "expired-session").Return(expiredContext, nil)

	result, err := service.ProcessInput(ctx, "expired-session", "test input")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "session has expired")
	mockStore.AssertExpectations(t)
}

func TestConsoleService_ProcessInput_InterpretationError(t *testing.T) {
	mockInterpreter := &MockAIInterpreter{}
	mockStore := &MockContextStore{}
	service := NewConsoleService(mockInterpreter, nil, mockStore)
	ctx := context.Background()

	// Mock valid session
	sessionContext := domain.NewContext("test-session")
	mockStore.On("Load", ctx, "test-session").Return(sessionContext, nil)

	// Mock interpretation error
	mockInterpreter.On("Interpret", ctx, "test input", sessionContext).Return((*domain.Command)(nil), assert.AnError)

	result, err := service.ProcessInput(ctx, "test-session", "test input")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to interpret command")
	mockStore.AssertExpectations(t)
	mockInterpreter.AssertExpectations(t)
}

func TestConsoleService_ProcessInput_ExecutionError(t *testing.T) {
	mockInterpreter := &MockAIInterpreter{}
	mockExecutor := &MockCommandExecutor{}
	mockStore := &MockContextStore{}
	service := NewConsoleService(mockInterpreter, mockExecutor, mockStore)
	ctx := context.Background()

	// Mock valid session
	sessionContext := domain.NewContext("test-session")
	mockStore.On("Load", ctx, "test-session").Return(sessionContext, nil)

	// Mock successful interpretation
	command, _ := domain.NewCommand("test-session", "assets list", "assets list", 0.9)
	mockInterpreter.On("Interpret", ctx, "test input", sessionContext).Return(command, nil)

	// Mock execution error
	mockExecutor.On("Execute", ctx, command).Return((*domain.CommandResult)(nil), assert.AnError)

	result, err := service.ProcessInput(ctx, "test-session", "test input")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to execute command")
	mockStore.AssertExpectations(t)
	mockInterpreter.AssertExpectations(t)
	mockExecutor.AssertExpectations(t)
}

func TestConsoleService_ProcessInput_SuccessfulExecution(t *testing.T) {
	mockInterpreter := &MockAIInterpreter{}
	mockExecutor := &MockCommandExecutor{}
	mockStore := &MockContextStore{}
	service := NewConsoleService(mockInterpreter, mockExecutor, mockStore)
	ctx := context.Background()

	// Mock valid session
	sessionContext := domain.NewContext("test-session")
	mockStore.On("Load", ctx, "test-session").Return(sessionContext, nil)

	// Mock successful interpretation
	command, _ := domain.NewCommand("test-session", "assets list", "assets list", 0.9)
	mockInterpreter.On("Interpret", ctx, "test input", sessionContext).Return(command, nil)

	// Mock successful execution
	result := &domain.CommandResult{
		Success: true,
		Output:  "Test output",
	}
	mockExecutor.On("Execute", ctx, command).Return(result, nil)

	// Mock store save for session update
	mockStore.On("Save", ctx, sessionContext).Return(nil)

	processResult, err := service.ProcessInput(ctx, "test-session", "test input")

	assert.NoError(t, err)
	assert.NotNil(t, processResult)
	assert.True(t, processResult.Success)
	assert.Equal(t, command, processResult.Command)
	assert.Equal(t, result, processResult.Result)
	assert.Equal(t, "Test output", processResult.Output)
	mockStore.AssertExpectations(t)
	mockInterpreter.AssertExpectations(t)
	mockExecutor.AssertExpectations(t)
}

func TestConsoleService_ProcessInput_ValidationError(t *testing.T) {
	mockInterpreter := &MockAIInterpreter{}
	mockExecutor := &MockCommandExecutor{}
	mockStore := &MockContextStore{}
	service := NewConsoleService(mockInterpreter, mockExecutor, mockStore)
	ctx := context.Background()

	// Mock valid session
	sessionContext := domain.NewContext("test-session")
	mockStore.On("Load", ctx, "test-session").Return(sessionContext, nil)

	// Mock successful interpretation
	command, _ := domain.NewCommand("test-session", "assets create", "assets create", 0.9)
	mockInterpreter.On("Interpret", ctx, "test input", sessionContext).Return(command, nil)

	// Mock validation error
	valErr := ports.NewValidationError("test", "test reason")
	mockExecutor.On("Execute", ctx, command).Return((*domain.CommandResult)(nil), valErr)

	processResult, err := service.ProcessInput(ctx, "test-session", "test input")

	assert.NoError(t, err)
	assert.NotNil(t, processResult)
	assert.False(t, processResult.Success)
	assert.Contains(t, processResult.Error, "test reason")
	assert.Equal(t, "Please check the command parameters and try again.", processResult.Help)
	mockStore.AssertExpectations(t)
	mockInterpreter.AssertExpectations(t)
	mockExecutor.AssertExpectations(t)
}

func TestConsoleService_ProcessInput_ExecutionErrorType(t *testing.T) {
	mockInterpreter := &MockAIInterpreter{}
	mockExecutor := &MockCommandExecutor{}
	mockStore := &MockContextStore{}
	service := NewConsoleService(mockInterpreter, mockExecutor, mockStore)
	ctx := context.Background()

	// Mock valid session
	sessionContext := domain.NewContext("test-session")
	mockStore.On("Load", ctx, "test-session").Return(sessionContext, nil)

	// Mock successful interpretation
	command, _ := domain.NewCommand("test-session", "assets list", "assets list", 0.9)
	mockInterpreter.On("Interpret", ctx, "test input", sessionContext).Return(command, nil)

	// Mock execution error
	execErr := ports.NewExecutionError("assets list", "service unavailable", "try again later")
	mockExecutor.On("Execute", ctx, command).Return((*domain.CommandResult)(nil), execErr)

	processResult, err := service.ProcessInput(ctx, "test-session", "test input")

	assert.NoError(t, err)
	assert.NotNil(t, processResult)
	assert.False(t, processResult.Success)
	assert.Contains(t, processResult.Error, "service unavailable")
	assert.Equal(t, "try again later", processResult.Help)
	mockStore.AssertExpectations(t)
	mockInterpreter.AssertExpectations(t)
	mockExecutor.AssertExpectations(t)
}

func TestConsoleService_GetAvailableCommands(t *testing.T) {
	mockExecutor := &MockCommandExecutor{}
	service := NewConsoleService(nil, mockExecutor, nil)

	expectedCommands := []ports.CommandInfo{
		{Command: "assets list", Description: "List all assets"},
		{Command: "assets create", Description: "Create a new asset"},
	}

	mockExecutor.On("GetAvailableCommands").Return(expectedCommands)

	commands := service.GetAvailableCommands()

	assert.Equal(t, expectedCommands, commands)
	mockExecutor.AssertExpectations(t)
}
