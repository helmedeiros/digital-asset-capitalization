package executor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// Mock implementations for testing
type MockAssetService struct {
	mock.Mock
}

func (m *MockAssetService) ListAssets(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

func (m *MockAssetService) CreateAsset(ctx context.Context, name, description string) (interface{}, error) {
	args := m.Called(ctx, name, description)
	return args.Get(0), args.Error(1)
}

func (m *MockAssetService) GetAsset(ctx context.Context, name string) (interface{}, error) {
	args := m.Called(ctx, name)
	return args.Get(0), args.Error(1)
}

func (m *MockAssetService) UpdateAsset(ctx context.Context, name, description string) (interface{}, error) {
	args := m.Called(ctx, name, description)
	return args.Get(0), args.Error(1)
}

func (m *MockAssetService) DeleteAsset(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockAssetService) SyncAssets(ctx context.Context, space, label string) (interface{}, error) {
	args := m.Called(ctx, space, label)
	return args.Get(0), args.Error(1)
}

func (m *MockAssetService) EnrichAsset(ctx context.Context, name, field string) (interface{}, error) {
	args := m.Called(ctx, name, field)
	return args.Get(0), args.Error(1)
}

func (m *MockAssetService) GenerateKeywords(ctx context.Context, name string) (interface{}, error) {
	args := m.Called(ctx, name)
	return args.Get(0), args.Error(1)
}

// Team management methods
func (m *MockAssetService) AssignTeamOwner(ctx context.Context, asset, team string) (interface{}, error) {
	args := m.Called(ctx, asset, team)
	return args.Get(0), args.Error(1)
}

func (m *MockAssetService) AddContributingTeam(ctx context.Context, asset, team string) (interface{}, error) {
	args := m.Called(ctx, asset, team)
	return args.Get(0), args.Error(1)
}

func (m *MockAssetService) RemoveContributingTeam(ctx context.Context, asset, team string) (interface{}, error) {
	args := m.Called(ctx, asset, team)
	return args.Get(0), args.Error(1)
}

func (m *MockAssetService) ShowTeamAssignments(ctx context.Context, asset string) (interface{}, error) {
	args := m.Called(ctx, asset)
	return args.Get(0), args.Error(1)
}

func (m *MockAssetService) ListTeamAssignments(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

// Advanced asset operations
func (m *MockAssetService) SyncAndEnrich(ctx context.Context, space, label string, keywords bool, fields []string) (interface{}, error) {
	args := m.Called(ctx, space, label, keywords, fields)
	return args.Get(0), args.Error(1)
}

func TestNewCommandExecutor(t *testing.T) {
	mockAsset := &MockAssetService{}

	executor := NewCommandExecutor(mockAsset, nil, nil, nil, nil)

	assert.NotNil(t, executor)
	assert.Equal(t, mockAsset, executor.assetService)
}

func TestCommandExecutor_Execute_Assets(t *testing.T) {
	mockAsset := &MockAssetService{}
	executor := NewCommandExecutor(mockAsset, nil, nil, nil, nil)
	ctx := context.Background()

	// Test assets list command
	command, _ := domain.NewCommand("session-1", "assets list", "assets list", 0.9)

	mockAsset.On("ListAssets", ctx).Return([]string{"asset1", "asset2"}, nil)

	result, err := executor.Execute(ctx, command)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockAsset.AssertExpectations(t)
}

func TestCommandExecutor_Execute_Help(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)
	ctx := context.Background()

	// Help is a single word command, not "help action"
	command, _ := domain.NewCommand("session-1", "help", "help", 0.9)

	result, err := executor.Execute(ctx, command)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	assert.True(t, result.Success)
	resultMap, ok := result.Output.(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, resultMap, "message")
	assert.Contains(t, resultMap, "commands")
}

func TestCommandExecutor_Execute_ContextShow(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)
	ctx := context.Background()

	command, _ := domain.NewCommand("session-1", "context show", "context show", 0.9)

	result, err := executor.Execute(ctx, command)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestCommandExecutor_Execute_Assets_Create(t *testing.T) {
	mockAsset := &MockAssetService{}
	executor := NewCommandExecutor(mockAsset, nil, nil, nil, nil)
	ctx := context.Background()

	// Test assets create command
	command, _ := domain.NewCommand("session-1", "assets create", "assets create", 0.9)
	command.AddParameter("name", "Test Asset")
	command.AddParameter("description", "Test Description")

	mockAsset.On("CreateAsset", ctx, "Test Asset", "Test Description").Return(map[string]string{"status": "created"}, nil)

	result, err := executor.Execute(ctx, command)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	mockAsset.AssertExpectations(t)
}

func TestCommandExecutor_Execute_UnknownCommand(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)
	ctx := context.Background()

	command, _ := domain.NewCommand("session-1", "unknown action", "unknown action", 0.9)

	result, err := executor.Execute(ctx, command)

	assert.Error(t, err)
	assert.NotNil(t, result) // CommandResult is still returned even on error
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "Unknown resource")
}

func TestCommandExecutor_Validation_EdgeCases(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)

	// Test nil command
	err := executor.ValidateCommand(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command cannot be nil")

	// Test single word commands that should fail format validation
	singleWordCommand, _ := domain.NewCommand("session-1", "invalid", "invalid", 0.9)
	err = executor.ValidateCommand(singleWordCommand)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command must have at least resource and action")

	// Test valid two-word command
	validCommand, _ := domain.NewCommand("session-1", "assets list", "assets list", 0.9)
	err = executor.ValidateCommand(validCommand)
	assert.NoError(t, err)
}

func TestCommandExecutor_GetAvailableCommands(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)

	commands := executor.GetAvailableCommands()

	assert.NotEmpty(t, commands)
	// Should contain asset commands
	found := false
	for _, cmd := range commands {
		if cmd.Command == "assets list" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should contain assets list command")
}

func TestCommandExecutor_ValidateCommand(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)

	// Test nil command
	err := executor.ValidateCommand(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command cannot be nil")

	// Test valid command
	command, _ := domain.NewCommand("session-1", "assets list", "assets list", 0.9)
	err = executor.ValidateCommand(command)
	assert.NoError(t, err)

	// Test invalid resource
	command, _ = domain.NewCommand("session-1", "invalid action", "invalid action", 0.9)
	err = executor.ValidateCommand(command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource")
}

func TestCommandExecutor_Execute_Exit(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)
	ctx := context.Background()

	command, _ := domain.NewCommand("session-1", "exit", "exit", 0.9)

	result, err := executor.Execute(ctx, command)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	outputMap, ok := result.Output.(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, "exit", outputMap["action"])
}

func TestCommandExecutor_Execute_Assets_WithoutService(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil) // nil asset service
	ctx := context.Background()

	command, _ := domain.NewCommand("session-1", "assets list", "assets list", 0.9)

	result, err := executor.Execute(ctx, command)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "asset service not available")
}

func TestCommandExecutor_Execute_EmptyCommand(t *testing.T) {
	// Test domain-level validation for empty interpretation (returns nil command)
	command, err := domain.NewCommand("session-1", "raw input", "", 0.9)
	assert.Error(t, err)
	assert.Nil(t, command)
	assert.Contains(t, err.Error(), "interpreted command is required")
}

func TestCommandExecutor_Execute_RequiresAction(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)
	ctx := context.Background()

	// Test that assets command requires an action
	command, _ := domain.NewCommand("session-1", "assets", "assets", 0.9)

	result, err := executor.Execute(ctx, command)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "requires an action")
}

func TestCommandExecutor_Execute_Assets_Show(t *testing.T) {
	mockAsset := &MockAssetService{}
	executor := NewCommandExecutor(mockAsset, nil, nil, nil, nil)
	ctx := context.Background()

	// Test assets show command
	command, _ := domain.NewCommand("session-1", "assets show", "assets show", 0.9)
	command.AddParameter("name", "Test Asset")

	mockAsset.On("GetAsset", ctx, "Test Asset").Return(map[string]string{"name": "Test Asset"}, nil)

	result, err := executor.Execute(ctx, command)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	mockAsset.AssertExpectations(t)
}

func TestCommandExecutor_Execute_Assets_Update(t *testing.T) {
	mockAsset := &MockAssetService{}
	executor := NewCommandExecutor(mockAsset, nil, nil, nil, nil)
	ctx := context.Background()

	// Test assets update command
	command, _ := domain.NewCommand("session-1", "assets update", "assets update", 0.9)
	command.AddParameter("name", "Test Asset")
	command.AddParameter("description", "Updated description")

	mockAsset.On("UpdateAsset", ctx, "Test Asset", "Updated description").Return(map[string]string{"status": "updated"}, nil)

	result, err := executor.Execute(ctx, command)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	mockAsset.AssertExpectations(t)
}

func TestCommandExecutor_Execute_Assets_Delete(t *testing.T) {
	mockAsset := &MockAssetService{}
	executor := NewCommandExecutor(mockAsset, nil, nil, nil, nil)
	ctx := context.Background()

	// Test assets delete command
	command, _ := domain.NewCommand("session-1", "assets delete", "assets delete", 0.9)
	command.AddParameter("name", "Test Asset")

	mockAsset.On("DeleteAsset", ctx, "Test Asset").Return(nil)

	result, err := executor.Execute(ctx, command)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	mockAsset.AssertExpectations(t)
}

func TestCommandExecutor_Execute_Context_Clear(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)
	ctx := context.Background()

	command, _ := domain.NewCommand("session-1", "context clear", "context clear", 0.9)

	result, err := executor.Execute(ctx, command)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestCommandExecutor_Execute_Assets_Sync(t *testing.T) {
	mockAsset := &MockAssetService{}
	executor := NewCommandExecutor(mockAsset, nil, nil, nil, nil)
	ctx := context.Background()

	// Test assets sync command
	command, _ := domain.NewCommand("session-1", "assets sync", "assets sync", 0.9)
	command.AddParameter("space", "TEST")
	command.AddParameter("label", "test-label")

	mockAsset.On("SyncAssets", ctx, "TEST", "test-label").Return(map[string]string{"status": "synced"}, nil)

	result, err := executor.Execute(ctx, command)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	mockAsset.AssertExpectations(t)
}

func TestCommandExecutor_Execute_Assets_Enrich(t *testing.T) {
	mockAsset := &MockAssetService{}
	executor := NewCommandExecutor(mockAsset, nil, nil, nil, nil)
	ctx := context.Background()

	// Test assets enrich command
	command, _ := domain.NewCommand("session-1", "assets enrich", "assets enrich", 0.9)
	command.AddParameter("name", "Test Asset")
	command.AddParameter("field", "description")

	mockAsset.On("EnrichAsset", ctx, "Test Asset", "description").Return(map[string]string{"status": "enriched"}, nil)

	result, err := executor.Execute(ctx, command)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	mockAsset.AssertExpectations(t)
}

func TestCommandExecutor_Execute_Assets_Keywords(t *testing.T) {
	mockAsset := &MockAssetService{}
	executor := NewCommandExecutor(mockAsset, nil, nil, nil, nil)
	ctx := context.Background()

	// Test assets keywords command
	command, _ := domain.NewCommand("session-1", "assets keywords", "assets keywords", 0.9)
	command.AddParameter("name", "Test Asset")

	mockAsset.On("GenerateKeywords", ctx, "Test Asset").Return(map[string]interface{}{"keywords": []string{"key1", "key2"}}, nil)

	result, err := executor.Execute(ctx, command)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	mockAsset.AssertExpectations(t)
}

func TestCommandExecutor_Execute_Assets_InvalidAction(t *testing.T) {
	mockAsset := &MockAssetService{}
	executor := NewCommandExecutor(mockAsset, nil, nil, nil, nil)
	ctx := context.Background()

	// Test invalid assets action
	command, _ := domain.NewCommand("session-1", "assets invalid", "assets invalid", 0.9)

	result, err := executor.Execute(ctx, command)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "Unknown asset action")
}

func TestCommandExecutor_Execute_Assets_MissingName(t *testing.T) {
	mockAsset := &MockAssetService{}
	executor := NewCommandExecutor(mockAsset, nil, nil, nil, nil)
	ctx := context.Background()

	// Test assets create without name
	command, _ := domain.NewCommand("session-1", "assets create", "assets create", 0.9)
	// Don't add name parameter

	result, err := executor.Execute(ctx, command)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "asset name is required")
}

func TestCommandExecutor_ValidateResourceSpecific(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)

	// Test assets command validation
	command, _ := domain.NewCommand("session-1", "assets create", "assets create", 0.9)
	// Missing name parameter - should fail validation
	err := executor.ValidateCommand(command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")

	// Test with name parameter - should pass
	command.AddParameter("name", "Test Asset")
	err = executor.ValidateCommand(command)
	assert.NoError(t, err)
}

func TestCommandExecutor_isValidResource(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)

	// Test valid resources
	assert.True(t, executor.isValidResource("assets"))
	assert.True(t, executor.isValidResource("tasks"))
	assert.True(t, executor.isValidResource("help"))
	assert.True(t, executor.isValidResource("exit"))

	// Test invalid resource
	assert.False(t, executor.isValidResource("invalid"))
}

func TestCommandExecutor_isValidAction(t *testing.T) {
	executor := NewCommandExecutor(nil, nil, nil, nil, nil)

	// Test valid actions for assets
	assert.True(t, executor.isValidAction("assets", "list"))
	assert.True(t, executor.isValidAction("assets", "create"))
	assert.True(t, executor.isValidAction("assets", "show"))

	// Test invalid action for assets
	assert.False(t, executor.isValidAction("assets", "invalid"))

	// Test action for non-existent resource
	assert.False(t, executor.isValidAction("nonexistent", "list"))
}
