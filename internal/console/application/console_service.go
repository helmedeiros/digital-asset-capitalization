package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain/ports"
)

// ConsoleService orchestrates the console session lifecycle
type ConsoleService struct {
	interpreter     ports.AIInterpreter
	executor        ports.CommandExecutor
	contextStore    ports.ContextStore
	sessionTimeout  time.Duration
	cleanupInterval time.Duration
}

// NewConsoleService creates a new console service
func NewConsoleService(
	interpreter ports.AIInterpreter,
	executor ports.CommandExecutor,
	contextStore ports.ContextStore,
) *ConsoleService {
	return &ConsoleService{
		interpreter:     interpreter,
		executor:        executor,
		contextStore:    contextStore,
		sessionTimeout:  30 * time.Minute,
		cleanupInterval: 5 * time.Minute,
	}
}

// ProcessResult contains the result of processing user input
type ProcessResult struct {
	Success               bool
	Command               *domain.Command
	Result                *domain.CommandResult
	Output                interface{}
	Error                 string
	Help                  string
	RequiresClarification bool
	ClarificationPrompt   string
	Options               []string
	Duration              time.Duration
}

// StartSession initializes a new console session
func (s *ConsoleService) StartSession(ctx context.Context) (*domain.Context, error) {
	sessionID := generateSessionID()
	sessionContext := domain.NewContext(sessionID)

	if err := s.contextStore.Save(ctx, sessionContext); err != nil {
		return nil, fmt.Errorf("failed to save session context: %w", err)
	}

	return sessionContext, nil
}

// ProcessInput handles user input and returns the response
func (s *ConsoleService) ProcessInput(ctx context.Context, sessionID string, input string) (*ProcessResult, error) {
	// Load session context
	sessionContext, err := s.contextStore.Load(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ports.ErrContextNotFound) {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to load session context: %w", err)
	}

	// Check if session has expired
	if sessionContext.IsExpired(s.sessionTimeout) {
		return nil, fmt.Errorf("session has expired")
	}

	// Start timing
	startTime := time.Now()

	// Interpret the command
	command, err := s.interpreter.Interpret(ctx, input, sessionContext)
	if err != nil {
		// Handle interpretation errors
		if intErr, ok := err.(*ports.InterpretationError); ok && len(intErr.Options) > 0 {
			// Need clarification
			clarification, _ := s.interpreter.GetClarification(ctx, intErr.Reason, intErr.Options)
			return &ProcessResult{
				Success:               false,
				RequiresClarification: true,
				ClarificationPrompt:   clarification,
				Options:               intErr.Options,
			}, nil
		}
		return nil, fmt.Errorf("failed to interpret command: %w", err)
	}

	// Add command to history
	sessionContext.AddCommand(*command)

	// Check if command needs clarification
	if command.RequiresClarification() {
		clarification, _ := s.interpreter.GetClarification(ctx,
			"Low confidence in interpretation",
			[]string{command.Interpreted})
		return &ProcessResult{
			Success:               false,
			RequiresClarification: true,
			ClarificationPrompt:   clarification,
			Command:               command,
		}, nil
	}

	// Execute the command
	result, err := s.executor.Execute(ctx, command)
	if err != nil {
		// Check if it's a validation error
		if valErr, ok := err.(*ports.ValidationError); ok {
			return &ProcessResult{
				Success: false,
				Error:   valErr.Error(),
				Command: command,
				Help:    "Please check the command parameters and try again.",
			}, nil
		}

		// Check if it's an execution error
		if execErr, ok := err.(*ports.ExecutionError); ok {
			return &ProcessResult{
				Success: false,
				Error:   execErr.Error(),
				Command: command,
				Help:    execErr.Help,
			}, nil
		}

		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	// Add result to context
	sessionContext.AddCommandResult(*result)

	// Update context based on command type and result
	s.updateContextFromResult(sessionContext, command, result)

	// Save updated context
	if err := s.contextStore.Save(ctx, sessionContext); err != nil {
		// Log error but don't fail the command
		fmt.Printf("Warning: failed to save updated context: %v\n", err)
	}

	// Calculate duration
	duration := time.Since(startTime)

	return &ProcessResult{
		Success:  true,
		Command:  command,
		Result:   result,
		Output:   result.Output,
		Duration: duration,
	}, nil
}

// EndSession terminates a console session
func (s *ConsoleService) EndSession(ctx context.Context, sessionID string) error {
	return s.contextStore.Delete(ctx, sessionID)
}

// GetSessionContext retrieves the current session context
func (s *ConsoleService) GetSessionContext(ctx context.Context, sessionID string) (*domain.Context, error) {
	return s.contextStore.Load(ctx, sessionID)
}

// GetAvailableCommands returns all available commands
func (s *ConsoleService) GetAvailableCommands() []ports.CommandInfo {
	return s.executor.GetAvailableCommands()
}

// updateContextFromResult updates the session context based on command execution results
func (s *ConsoleService) updateContextFromResult(ctx *domain.Context, cmd *domain.Command, result *domain.CommandResult) {
	// This is a simplified version - in practice, you'd check the actual result type
	switch cmd.Intent.Resource {
	case domain.ResourceTypeAsset:
		if cmd.Intent.Action == domain.CommandTypeCreate || cmd.Intent.Action == domain.CommandTypeRead {
			// Update asset context if result contains asset data
			if assetName, ok := cmd.GetStringParameter("name"); ok {
				ctx.UpdateAssetContext(assetName, result.Output)
			}
		}
	case domain.ResourceTypeTask:
		if taskKey, ok := cmd.GetStringParameter("key"); ok {
			ctx.UpdateTaskContext(taskKey, result.Output)
		}
	case domain.ResourceTypeSprint:
		if sprint, ok := cmd.GetStringParameter("sprint"); ok {
			ctx.SetCurrentSprint(sprint)
		}
		if project, ok := cmd.GetStringParameter("project"); ok {
			ctx.SetCurrentProject(project)
		}
	}
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}
