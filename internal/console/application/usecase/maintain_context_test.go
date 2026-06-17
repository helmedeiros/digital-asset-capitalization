package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// MockContextStore for testing
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
	if args.Get(0) == nil {
		return args.Error(0)
	}

	// Execute the updateFn with a test context
	testCtx := &domain.Context{
		SessionID:      sessionID,
		Commands:       []domain.Command{},
		CommandResults: make(map[string]domain.CommandResult),
	}
	if updateFn != nil {
		updateFn(testCtx)
	}
	return args.Error(0)
}

func (m *MockContextStore) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockContextStore) CleanupExpired(ctx context.Context, timeout int) error {
	args := m.Called(ctx, timeout)
	return args.Error(0)
}

func (m *MockContextStore) List(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func TestNewMaintainContextUseCase(t *testing.T) {
	t.Parallel()
	mockStore := new(MockContextStore)
	useCase := NewMaintainContextUseCase(mockStore)

	assert.NotNil(t, useCase)
	assert.Equal(t, mockStore, useCase.contextStore)
}

func TestMaintainContextUseCase_UpdateContext(t *testing.T) {
	t.Parallel()
	mockStore := new(MockContextStore)
	useCase := NewMaintainContextUseCase(mockStore)
	ctx := context.Background()

	// Create test command and result
	command := &domain.Command{
		ID:          "cmd-123",
		Raw:         "show assets",
		Interpreted: "assets list",
		SessionID:   "session-123",
		Confidence:  0.95,
		Timestamp:   time.Now(),
		Parameters:  map[string]interface{}{"limit": 10},
	}

	result := &domain.CommandResult{
		CommandID: "cmd-123",
		Success:   true,
		Output:    map[string]interface{}{"count": 5},
		Error:     nil,
		Duration:  100 * time.Millisecond,
	}

	// Mock the store update
	mockStore.On("Update", ctx, "session-123", mock.AnythingOfType("func(*domain.Context) error")).Return(nil)

	err := useCase.UpdateContext(ctx, "session-123", command, result)

	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestMaintainContextUseCase_UpdateContext_StoreError(t *testing.T) {
	t.Parallel()
	mockStore := new(MockContextStore)
	useCase := NewMaintainContextUseCase(mockStore)
	ctx := context.Background()

	command := &domain.Command{
		ID:          "cmd-123",
		Raw:         "show assets",
		Interpreted: "assets list",
		SessionID:   "session-123",
		Confidence:  0.95,
		Timestamp:   time.Now(),
	}

	result := &domain.CommandResult{
		CommandID: "cmd-123",
		Success:   true,
		Output:    map[string]interface{}{"count": 5},
		Duration:  100 * time.Millisecond,
	}

	// Mock store to return error
	expectedErr := assert.AnError
	mockStore.On("Update", ctx, "session-123", mock.AnythingOfType("func(*domain.Context) error")).Return(expectedErr)

	err := useCase.UpdateContext(ctx, "session-123", command, result)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockStore.AssertExpectations(t)
}

func TestMaintainContextUseCase_UpdateContext_WithEmptyCommand(t *testing.T) {
	t.Parallel()
	mockStore := new(MockContextStore)
	useCase := NewMaintainContextUseCase(mockStore)
	ctx := context.Background()

	// Create minimal command and result
	command := &domain.Command{
		ID: "cmd-empty",
	}

	result := &domain.CommandResult{
		CommandID: "cmd-empty",
		Success:   false,
	}

	// Mock the store update
	mockStore.On("Update", ctx, "session-456", mock.AnythingOfType("func(*domain.Context) error")).Return(nil)

	err := useCase.UpdateContext(ctx, "session-456", command, result)

	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestMaintainContextUseCase_UpdateContext_WithNilParameters(t *testing.T) {
	t.Parallel()
	mockStore := new(MockContextStore)
	useCase := NewMaintainContextUseCase(mockStore)
	ctx := context.Background()

	command := &domain.Command{
		ID:          "cmd-nil",
		Raw:         "test command",
		Interpreted: "test",
		SessionID:   "session-123",
		Confidence:  0.8,
		Timestamp:   time.Now(),
		Parameters:  nil, // nil parameters
	}

	result := &domain.CommandResult{
		CommandID: "cmd-nil",
		Success:   true,
		Output:    nil, // nil data
		Duration:  50 * time.Millisecond,
	}

	mockStore.On("Update", ctx, "session-nil", mock.AnythingOfType("func(*domain.Context) error")).Return(nil)

	err := useCase.UpdateContext(ctx, "session-nil", command, result)

	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}
